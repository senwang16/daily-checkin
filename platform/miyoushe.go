package platform

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"daily-checkin/internal"
)

// 米游社原神签到 API（社区公开，MihoyoBBSTools 同款）
const (
	mysActID     = "e202009291139501" // 原神签到活动 ID
	mysAPIBase   = "https://api-takumi.mihoyo.com"
	mysRolesURL  = mysAPIBase + "/binding/api/getUserGameRolesByCookie?game_biz=hk4e_cn"
	mysSignBase  = "https://api-takumi.mihoyo.com/event/bbs_sign_reward"
	mysInfoURL   = mysSignBase + "/info?act_id=" + mysActID + "&region=%s&uid=%s"
	mysSignURL   = mysSignBase + "/sign"
	mysAppVer    = "2.34.1"
	mysClientTyp = "5"
	// DS v1 签名盐（米游社签到接口长期使用，社区公开）
	mysDSSalt = "9nQiU3AV0rJSIBWgdynfoGMGKaklfbM7"
)

// miyousheCookieFile 保存用户粘贴的 Cookie
type miyousheCookieFile struct {
	Cookie string `json:"cookie"`
}

func miyousheCookiePath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "daily-checkin", "miyoushe.json")
}

// LoadMiyousheCookie 读取米游社 Cookie（环境变量优先，便于云端）
func LoadMiyousheCookie() (string, error) {
	if c := os.Getenv("MIYOUSHE_COOKIE"); c != "" {
		return c, nil
	}
	data, err := os.ReadFile(miyousheCookiePath())
	if err != nil {
		return "", err
	}
	var f miyousheCookieFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", err
	}
	if strings.TrimSpace(f.Cookie) == "" {
		return "", fmt.Errorf("cookie 为空")
	}
	return f.Cookie, nil
}

// SaveMiyousheCookie 保存 Cookie 到本机
func SaveMiyousheCookie(cookie string) error {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "daily-checkin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(miyousheCookieFile{Cookie: cookie}, "", "  ")
	return os.WriteFile(miyousheCookiePath(), data, 0o600)
}

// mysDS 生成米游社 v1 版 DS 签名：md5(salt=盐&t=秒时间戳&r=6位随机数)
func mysDS() string {
	t := time.Now().Unix()
	r := rand.Intn(899999) + 100000 // 100000-999999
	s := fmt.Sprintf("salt=%s&t=%d&r=%d", mysDSSalt, t, r)
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%d,%d,%x", t, r, sum)
}

func mysHeaders(cookie string, withDS bool) map[string]string {
	h := map[string]string{
		"Cookie":           cookie,
		"User-Agent":       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) miHoYoBBS/" + mysAppVer,
		"Accept":           "application/json, text/plain, */*",
		"Referer":          "https://webstatic.mihoyo.com/",
		"X-Requested-With": "com.mihoyo.hyperion",
	}
	if withDS {
		h["DS"] = mysDS()
		h["x-rpc-app_version"] = mysAppVer
		h["x-rpc-client_type"] = mysClientTyp
		h["x-rpc-device_id"] = mysDeviceID(cookie)
	}
	return h
}

// mysDeviceID 从 cookie 派生一个稳定 device id（避免随机触发风控）
func mysDeviceID(cookie string) string {
	sum := md5.Sum([]byte(cookie))
	return fmt.Sprintf("%x", sum)
}

// gameRole 原神角色信息
type gameRole struct {
	Region   string `json:"region"`
	GameUID  string `json:"game_uid"`
	Nickname string `json:"nickname"`
	Level    int    `json:"level"`
}

// mysRetcode 米游社接口业务码
func mysRetcode(resp map[string]interface{}) int {
	return int(toFloat(resp["retcode"]))
}

func mysMessage(resp map[string]interface{}) string {
	if m, ok := resp["message"].(string); ok {
		return m
	}
	return ""
}

// Miyoushe 米游社（原神）平台实现
type Miyoushe struct{}

func (Miyoushe) Name() string        { return "miyoushe" }
func (Miyoushe) DisplayName() string { return "米游社·原神" }

func (Miyoushe) Detect() bool {
	_, err := LoadMiyousheCookie()
	return err == nil
}

// Checkin 执行原神签到
func (Miyoushe) Checkin() *internal.CheckinError {
	cookie, err := LoadMiyousheCookie()
	if err != nil {
		return internal.NewCheckinError(internal.E001CredNotFound, "米游社",
			"未配置 Cookie，请运行 daily-checkin.exe miyoushe 粘贴 Cookie")
	}
	client := internal.NewHTTPClient()

	// 1. 获取原神角色列表
	code, resp, e := client.Get(mysRolesURL, mysHeaders(cookie, false))
	if e != nil {
		if code == 401 {
			return internal.NewCheckinError(internal.E002TokenExpired, "米游社", "Cookie 失效(401)")
		}
		return internal.NewCheckinError(internal.E004Network, "米游社", "获取角色失败: "+e.Error())
	}
	if rc := mysRetcode(resp); rc != 0 {
		if rc == -100 || rc == 10001 {
			return internal.NewCheckinError(internal.E002TokenExpired, "米游社",
				"Cookie 已失效，请重新获取（浏览器无痕模式登录米游社，F12 抓 Cookie）")
		}
		return internal.NewCheckinError(internal.E005BizError, "米游社",
			fmt.Sprintf("获取角色失败(retcode=%d): %s", rc, mysMessage(resp)))
	}
	roles := parseRoles(resp)
	if len(roles) == 0 {
		return internal.NewCheckinError(internal.E001CredNotFound, "米游社",
			"该账号未绑定原神角色（或 Cookie 不含 account_id/cookie_token）")
	}

	// 2. 逐个角色签到（一般只有一个）
	var fails []string
	for _, role := range roles {
		if ce := signOneRole(client, cookie, role); ce != nil {
			fails = append(fails, role.Nickname+": "+ce.UserMessage())
		}
	}
	if len(fails) > 0 {
		return internal.NewCheckinError(internal.E005BizError, "米游社", strings.Join(fails, "; "))
	}
	return nil
}

func parseRoles(resp map[string]interface{}) []gameRole {
	var roles []gameRole
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return roles
	}
	list, ok := data["list"].([]interface{})
	if !ok {
		return roles
	}
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		roles = append(roles, gameRole{
			Region:   strVal(m["region"]),
			GameUID:  strVal(m["game_uid"]),
			Nickname: strVal(m["nickname"]),
			Level:    int(toFloat(m["level"])),
		})
	}
	return roles
}

func strVal(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// signOneRole 对单个角色：先查 info 是否已签，未签则 sign
func signOneRole(client *internal.HTTPClient, cookie string, role gameRole) *internal.CheckinError {
	// 查签到状态
	infoURL := fmt.Sprintf(mysInfoURL, role.Region, role.GameUID)
	code, resp, e := client.Get(infoURL, mysHeaders(cookie, false))
	if e != nil {
		if code == 429 {
			return internal.NewCheckinError(internal.E007RateLimited, "米游社", "请求过于频繁(429)")
		}
		return internal.NewCheckinError(internal.E004Network, "米游社", "查询签到状态失败: "+e.Error())
	}
	if rc := mysRetcode(resp); rc != 0 {
		return internal.NewCheckinError(internal.E005BizError, "米游社",
			fmt.Sprintf("查询签到状态失败(retcode=%d): %s", rc, mysMessage(resp)))
	}
	if data, ok := resp["data"].(map[string]interface{}); ok {
		if signed, _ := data["is_sign"].(bool); signed {
			return nil // 已签到
		}
	}

	// 执行签到
	body := map[string]interface{}{
		"act_id": mysActID,
		"region": role.Region,
		"uid":    role.GameUID,
	}
	code, resp, e = client.PostJSONRetry(mysSignURL, mysHeaders(cookie, true), body, 2)
	if e != nil {
		if code == 401 {
			return internal.NewCheckinError(internal.E002TokenExpired, "米游社", "Cookie 失效(401)")
		}
		if code == 429 {
			return internal.NewCheckinError(internal.E007RateLimited, "米游社", "请求过于频繁(429)")
		}
		return internal.NewCheckinError(internal.E004Network, "米游社", "签到请求失败: "+e.Error())
	}
	rc := mysRetcode(resp)
	if rc == 0 {
		return nil
	}
	// 幂等：-5003 = 今日已签到
	if rc == -5003 {
		return nil
	}
	return internal.NewCheckinError(internal.E005BizError, "米游社",
		fmt.Sprintf("签到失败(retcode=%d): %s", rc, mysMessage(resp)))
}
