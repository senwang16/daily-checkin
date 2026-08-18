package platform

import (
	"strings"

	"daily-checkin/extract"
	"daily-checkin/internal"
)

const wbHost = "https://www.codebuddy.cn/v2/billing/meter/daily-checkin"

// WorkBuddy 平台实现
type WorkBuddy struct{}

func (WorkBuddy) Name() string       { return "workbuddy" }
func (WorkBuddy) DisplayName() string { return "WorkBuddy" }

func (WorkBuddy) Detect() bool {
	_, err := extract.LoadWorkBuddyToken()
	return err == nil
}

func (WorkBuddy) Checkin() *internal.CheckinError {
	token, err := extract.LoadWorkBuddyToken()
	if err != nil {
		return internal.NewCheckinError(internal.E001CredNotFound, "WorkBuddy", err.Error())
	}
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}
	client := internal.NewHTTPClient()
	code, resp, e := client.PostJSONRetry(wbHost, headers, map[string]interface{}{}, 2)
	if e != nil {
		return internal.NewCheckinError(internal.E004Network, "WorkBuddy", e.Error())
	}
	if code == 401 {
		return internal.NewCheckinError(internal.E002TokenExpired, "WorkBuddy", "token 失效")
	}
	rc := int(toFloat(resp["code"]))
	if rc == 0 {
		return nil
	}
	msg := bizMsg(resp)
	// 幂等：今天已签到视为成功（接口会返回非 0 业务码 + 提示文案）
	if strings.Contains(msg, "已签到") || strings.Contains(msg, "今天") || strings.Contains(msg, "明天") {
		return nil
	}
	return internal.NewCheckinError(internal.E005BizError, "WorkBuddy", msg)
}
