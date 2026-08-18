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
	ahaDeviceID = "1927881068695851" // 兜底指纹；优先用本机日志提取的真 aha ID
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
	// 设备指纹：优先本机日志提取的真实 aha ID
	devID := auth.DeviceID
	if realID, e := extract.FindAhaDeviceID(); e == nil && realID != "" {
		devID = realID
	}
	if devID == "" {
		devID = ahaDeviceID
	}
	// token 续期
	if auth.NeedRefresh() {
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
		return internal.NewCheckinError(internal.E004Network, "Trae", e.Error())
	}
	if code == 401 {
		return internal.NewCheckinError(internal.E002TokenExpired, "Trae", "token 失效")
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
		return internal.NewCheckinError(internal.E004Network, "Trae", e.Error())
	}
	if code == 401 {
		return internal.NewCheckinError(internal.E002TokenExpired, "Trae", "token 失效")
	}
	rc := int(toFloat(resp["code"]))
	if rc == 0 {
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
