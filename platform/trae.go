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
	_, err := extract.LoadTraeAuth()
	return err == nil
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
	auth, err := extract.LoadTraeAuth()
	if err != nil {
		return internal.NewCheckinError(internal.E001CredNotFound, "Trae", err.Error())
	}
	// 设备指纹：优先本机日志提取的真实 aha ID（绝不写死兜底，避免共享指纹触发 9074）
	devID := auth.DeviceID
	if realID, e := extract.FindAhaDeviceID(); e == nil && realID != "" {
		devID = realID
	}
	if devID == "" {
		return internal.NewCheckinError(internal.E001CredNotFound, "Trae",
			"未找到本机 aha 设备指纹，请确认已安装并登录 Trae 桌面端（或手动设置 TRAE_AHA_ID 环境变量）")
	}
	// token 续期（云端环境变量注入的凭据无法回写，跳过续期避免作废旧 refresh_token）
	if !auth.FromEnv && auth.NeedRefresh() {
		if re := auth.Refresh(); re != nil {
			store.AppendLog("[WARN] Trae token 刷新失败: " + re.Error())
		} else {
			_ = extract.SaveTraeAuth(auth)
		}
	}
	client := internal.NewHTTPClient()
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
