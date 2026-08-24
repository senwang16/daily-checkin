package platform

import (
	"encoding/json"

	"daily-checkin/extract"
	"daily-checkin/internal"
	"daily-checkin/store"
)

const (
	ugHost      = "https://api.trae.cn"
	epStatus    = "/trae/api/v2/ug/checkin_credits/status"
	epClaim     = "/trae/api/v2/ug/checkin_credits/claim"
	osVersion   = "10.0.26200"
	appVersion  = "0.1.50"
)

// Trae 平台实现
type Trae struct{}

func (Trae) Name() string      { return "trae" }
func (Trae) DisplayName() string { return "Trae" }

func (Trae) Detect() bool {
	return len(extract.LoadTraeAuths()) > 0
}

func ugHeaders(token, deviceID string) map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/json",
		"User-Agent":    "Trae/" + appVersion,
		"Authorization": "Cloud-IDE-JWT " + token,
		"X-User-Region": "CN",
		"X-Device-Id":   deviceID,
		"x-device-type": "windows",
		"x-os-version":  osVersion,
		"x-app-version": appVersion,
	}
}

func (Trae) Checkin() *internal.CheckinError {
	auths := extract.LoadTraeAuths()
	if len(auths) == 0 {
		return internal.NewCheckinError(internal.E001CredNotFound, "Trae",
			"未找到 Trae 登录凭据，请运行 daily-checkin.exe login 登录")
	}
	// 设备指纹：全账号共用本机真实 aha ID（同一台电脑），避免共享/随机指纹触发 9074
	devID := ""
	if realID, e := extract.FindAhaDeviceID(); e == nil && realID != "" {
		devID = realID
	}
	if devID == "" {
		return internal.NewCheckinError(internal.E001CredNotFound, "Trae",
			"未找到本机 aha 设备指纹，请确认已安装并登录 Trae 桌面端（或手动设置 TRAE_AHA_ID 环境变量）")
	}
	client := internal.NewHTTPClient()
	var fails []string
	okCount := 0
	for _, auth := range auths {
		name := auth.Nickname
		if name == "" {
			name = auth.UID
		}
		if ce := checkinOneTrae(client, auth, devID); ce != nil {
			fails = append(fails, name+": "+ce.UserMessage())
		} else {
			okCount++
			store.AppendLog("[OK] Trae 账号 " + name + " 签到成功")
		}
	}
	if len(fails) > 0 {
		if okCount > 0 {
			store.AppendLog("[WARN] Trae 部分账号失败: " + joinFails(fails))
		}
		return internal.NewCheckinError(internal.E005BizError, "Trae", joinFails(fails))
	}
	return nil
}

func joinFails(fails []string) string {
	s := ""
	for i, f := range fails {
		if i > 0 {
			s += "; "
		}
		s += f
	}
	return s
}

// checkinOneTrae 对单个 Trae 账号执行签到
func checkinOneTrae(client *internal.HTTPClient, auth *extract.TraeAuth, devID string) *internal.CheckinError {
	// token 续期（云端环境变量注入的凭据无法回写，跳过续期避免作废旧 refresh_token）
	if !auth.FromEnv && auth.NeedRefresh() {
		if re := auth.Refresh(); re != nil {
			store.AppendLog("[WARN] Trae token 刷新失败(" + auth.UID + "): " + re.Error())
		} else {
			_ = extract.SaveTraeAuthBack(auth)
		}
	}
	h := ugHeaders(auth.AccessToken, devID)
	// 1. 查状态
	code, resp, e := client.PostJSON(ugHost+epStatus, h, map[string]interface{}{})
	if e != nil {
		if code == 401 {
			return internal.NewCheckinError(internal.E002TokenExpired, "Trae", "token 失效")
		}
		if code == 429 {
			return internal.NewCheckinError(internal.E007RateLimited, "Trae", "请求过于频繁(429)")
		}
		return internal.NewCheckinError(internal.E004Network, "Trae", e.Error())
	}
	checked, _ := resp["checked_in"].(bool)
	if checked {
		return nil
	}
	if enable, _ := resp["enable"].(bool); enable == false {
		return nil
	}
	// 2. claim（带重试）
	code, resp, e = client.PostJSONRetry(ugHost+epClaim, h, map[string]interface{}{}, 2)
	if e != nil {
		if code == 401 {
			return internal.NewCheckinError(internal.E002TokenExpired, "Trae", "token 失效")
		}
		if code == 429 {
			return internal.NewCheckinError(internal.E007RateLimited, "Trae", "请求过于频繁(429)")
		}
		return internal.NewCheckinError(internal.E004Network, "Trae", e.Error())
	}
	rc := int(toFloat(resp["code"]))
	if rc == 0 {
		return nil
	}
	// 幂等：claim 返回已签到同样视为成功
	if checked, _ := resp["checked_in"].(bool); checked {
		return nil
	}
	if rc == 9074 {
		return internal.NewCheckinError(internal.E003RiskControl, "Trae", "操作太过频繁")
	}
	return internal.NewCheckinError(internal.E005BizError, "Trae", bizMsg(resp))
}

func bizMsg(resp map[string]interface{}) string {
	if m, ok := resp["message"].(string); ok && m != "" {
		return m
	}
	if m, ok := resp["msg"].(string); ok && m != "" {
		return m
	}
	return ""
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}
