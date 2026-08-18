package platform

import (
	"daily-checkin/extract"
	"daily-checkin/internal"
)

const (
	wbHost          = "https://www.codebuddy.cn/v2/billing/meter/daily-checkin"
	wbCodeAlreadyIn = 10001 // WorkBuddy 签到接口「今日已签到」业务码（Python 参考实现确认）
)

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
		if code == 401 {
			return internal.NewCheckinError(internal.E002TokenExpired, "WorkBuddy", "token 失效")
		}
		if code == 429 {
			return internal.NewCheckinError(internal.E007RateLimited, "WorkBuddy", "请求过于频繁(429)")
		}
		return internal.NewCheckinError(internal.E004Network, "WorkBuddy", e.Error())
	}
	rc := int(toFloat(resp["code"]))
	if rc == 0 {
		return nil
	}
	// 幂等：显式业务码 10001 表示今日已签到，视为成功（Python 参考实现确认）
	if rc == wbCodeAlreadyIn {
		return nil
	}
	return internal.NewCheckinError(internal.E005BizError, "WorkBuddy", bizMsg(resp))
}
