package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"daily-checkin/extract"
	"daily-checkin/internal"
)

// Login 交互式登录，生成 trae_auth.json（移植自旧 trae_login.py）
// 流程：生成 machine_id/device_id -> 构造登录链接 -> 浏览器手机号/验证码登录
// -> 粘贴回调链接 -> ExchangeToken 换 access token -> GetUserInfo 拿 uid -> 落盘
func Login() bool {
	sep := strings.Repeat("=", 60)
	fmt.Println(sep)
	fmt.Println("  TRAE Work(SOLO) 登录 - 生成签到凭证")
	fmt.Println(sep)
	fmt.Println()

	// 复用本机真实设备身份（machineid + 日志里的 aha 设备 ID），
	// 避免随机 device_id 被服务端当成新设备、触发多端登录限制；取不到时回退随机。
	machineID := extract.FindTraeMachineID()
	deviceID := ""
	if id, e := extract.FindAhaDeviceID(); e == nil {
		deviceID = id
	}
	if machineID == "" {
		machineID = randHex(16)
	}
	if deviceID == "" {
		deviceID = machineID
	}
	loginURL := buildLoginURL(machineID, deviceID)

	fmt.Println("步骤：")
	fmt.Println("  1. 在浏览器打开下面的链接，用手机号/验证码登录")
	fmt.Println("  2. 登录成功后浏览器会跳到打不开的 127.0.0.1 地址")
	fmt.Println("  3. 复制浏览器地址栏的【完整链接】粘贴到这里")
	fmt.Println()
	fmt.Println("请在浏览器打开：")
	fmt.Println()
	fmt.Println("  " + loginURL)
	fmt.Println()
	if machineID != "" && deviceID != "" {
		fmt.Printf("[*] 使用本机真实设备身份登录（machine_id=%s device_id=%s）\n", machineID, deviceID)
	}
	openBrowser(loginURL)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("粘贴回调链接（回车完成登录）：")
	callback, _ := reader.ReadString('\n')
	callback = strings.TrimSpace(callback)
	if callback == "" {
		fmt.Fprintln(os.Stderr, "未输入回调链接，已取消")
		return false
	}

	u, err := url.Parse(callback)
	if err != nil {
		fmt.Fprintln(os.Stderr, "回调链接无法解析: "+err.Error())
		return false
	}
	qs := u.Query()
	refreshToken := qs.Get("refreshToken")
	userInfo := parseJSONParam(qs.Get("userInfo"))
	userJwt := parseJSONParam(qs.Get("userJwt"))

	uid := strOf(userInfo["UserID"])
	nickname := strOf(userInfo["ScreenName"])
	jwtToken := strOf(userJwt["Token"])
	jwtRefresh := strOf(userJwt["RefreshToken"])
	if refreshToken == "" {
		refreshToken = jwtRefresh
	}

	token, newRefresh, expiresAt := "", refreshToken, int64(0)
	if refreshToken != "" {
		// ExchangeToken（access token + refreshToken 轮换）
		body := map[string]interface{}{
			"ClientID":     extract.CLIENT_ID,
			"RefreshToken": refreshToken,
			"ClientSecret": "-",
			"UserID":       "",
		}
		headers := map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "Trae/" + extract.APP_VERSION,
		}
		_, resp, e := internal.NewHTTPClient().PostJSON(extract.OAUTH_HOST+extract.EP_EXCHANGE, headers, body)
		if e != nil {
			fmt.Fprintln(os.Stderr, "ExchangeToken 失败: "+e.Error())
			return false
		}
		result, _ := resp["Result"].(map[string]interface{})
		token = strOf(result["Token"])
		if token == "" {
			fmt.Fprintln(os.Stderr, "ExchangeToken 失败: "+jsonStr(resp))
			return false
		}
		newRefresh = strOf(result["RefreshToken"])
		if newRefresh == "" {
			newRefresh = refreshToken
		}
		expiresAt = parseExpiry(result)
		fmt.Printf("[*] ExchangeToken 成功: Token %s...\n", token[:min(12, len(token))])
	} else {
		// 兜底：无 refreshToken 时直接用 userJwt 的 Token
		token = jwtToken
		expiresAt = parseExpiry(userJwt)
		if token == "" {
			fmt.Fprintln(os.Stderr, "回调链接缺少 refreshToken，且 userJwt 也没有 Token")
			return false
		}
		fmt.Println("[*] 无 refreshToken，使用 userJwt 的 Token 兜底")
	}

	// GetUserInfo 确认
	if ui, e := getUserInfo(token); e == nil {
		if id := strOf(ui["UserID"]); id != "" {
			uid = id
			nickname = strOf(ui["ScreenName"])
		}
	}
	if uid == "" {
		fmt.Fprintln(os.Stderr, "未能获取 uid，请检查 token 是否有效")
		return false
	}

	auth := &extract.TraeAuth{
		AccessToken:  token,
		RefreshToken: newRefresh,
		UID:          uid,
		Nickname:     nickname,
		DeviceID:     deviceID,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now().Unix(),
	}
	if err := extract.SaveTraeAuth(auth); err != nil {
		fmt.Fprintln(os.Stderr, "保存凭据失败: "+err.Error())
		return false
	}

	fmt.Println()
	fmt.Println(sep)
	fmt.Printf("  登录成功，凭证已保存到 %%LOCALAPPDATA%%\\daily-checkin\\auths\\trae-%s.json\n", uid)
	fmt.Printf("  用户: %s (uid: %s)\n", nickname, uid)
	fmt.Println("  多账号：再次运行 daily-checkin.exe login 登录另一个号即可，会自动并存")
	fmt.Println("  接下来运行 daily-checkin.exe install 即可设置自动签到")
	fmt.Println(sep)
	return true
}

func buildLoginURL(machineID, deviceID string) string {
	params := url.Values{}
	params.Set("login_version", "1")
	params.Set("auth_from", "solo")
	params.Set("login_channel", "native_ide")
	params.Set("plugin_version", "2.3.62834")
	params.Set("auth_type", "local")
	params.Set("client_id", extract.CLIENT_ID)
	params.Set("redirect", "0")
	params.Set("login_trace_id", randHex(8))
	params.Set("auth_callback_url", "http://127.0.0.1:18080/authorize")
	params.Set("machine_id", machineID)
	params.Set("device_id", deviceID)
	params.Set("x_device_id", deviceID)
	params.Set("x_machine_id", machineID)
	params.Set("x_device_brand", "PC")
	params.Set("x_device_type", "PC")
	params.Set("x_os_version", "1.0")
	params.Set("x_app_version", extract.APP_VERSION)
	params.Set("x_app_type", "stable")
	return "https://www.trae.cn/authorization?" + params.Encode()
}

func getUserInfo(token string) (map[string]interface{}, error) {
	body := map[string]interface{}{"ReqSource": "IDE", "IDEVersion": extract.APP_VERSION}
	headers := map[string]string{
		"Content-Type":    "application/json",
		"x-cloudide-token": token,
		"User-Agent":      "Trae/" + extract.APP_VERSION,
	}
	_, resp, err := internal.NewHTTPClient().PostJSON(extract.OAUTH_HOST+extract.EP_USERINFO, headers, body)
	if err != nil {
		return nil, err
	}
	if r, ok := resp["Result"].(map[string]interface{}); ok {
		return r, nil
	}
	return resp, nil
}

// parseJSONParam 解析回调里 URL 编码的 JSON 参数（容错两层 unquote）
func parseJSONParam(raw string) map[string]interface{} {
	if raw == "" {
		return nil
	}
	for _, val := range []string{raw, unquoteOnce(raw)} {
		var obj map[string]interface{}
		if json.Unmarshal([]byte(val), &obj) == nil && obj != nil {
			return obj
		}
	}
	return nil
}

func unquoteOnce(s string) string {
	if u, err := url.QueryUnescape(s); err == nil {
		return u
	}
	return s
}

// parseExpiry 解析 TokenExpireAt（毫秒/秒自适应），缺失时按 14 天兜底
func parseExpiry(m map[string]interface{}) int64 {
	if exp, ok := m["TokenExpireAt"].(float64); ok && exp > 0 {
		e := int64(exp)
		if e > 1e12 {
			e /= 1000
		}
		if e > time.Now().Unix() {
			return e
		}
	}
	if dur, ok := m["TokenExpireDuration"].(float64); ok && dur > 0 {
		return time.Now().Unix() + int64(dur)
	}
	return time.Now().Unix() + 1209600
}

func strOf(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func jsonStr(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// openBrowser 尽力在默认浏览器打开登录链接（失败静默，用户可手动复制）
func openBrowser(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err != nil {
		fmt.Println("（自动打开浏览器失败，请手动复制上面的链接）")
		return
	}
	fmt.Println("已尝试在默认浏览器打开，若未弹出请手动复制上面的链接。")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
