package internal

import "fmt"

// Code 是签到错误的分类码，用于通知用户具体失败原因
type Code int

const (
	CodeOK           Code = 0
	E001CredNotFound Code = 1 // 本机未找到登录凭据
	E002TokenExpired Code = 2 // token 失效，需重新提取
	E003RiskControl  Code = 3 // 9074 风控，设备指纹不匹配
	E004Network      Code = 4 // 网络错误
	E005BizError     Code = 5 // 业务未知错误
	E006TaskRegister Code = 6 // 计划任务注册失败
)

// CheckinError 统一错误结构
type CheckinError struct {
	Code     Code
	Platform string
	Message  string
}

func (e *CheckinError) Error() string { return e.Message }

func NewCheckinError(code Code, platform, msg string) *CheckinError {
	return &CheckinError{Code: code, Platform: platform, Message: msg}
}

// UserMessage 返回给用户看的中文说明（含错误码与处理建议）
func (e *CheckinError) UserMessage() string {
	switch e.Code {
	case E001CredNotFound:
		return fmt.Sprintf("%s：未找到登录凭据，请先在本机登录 %s 后重新运行本程序进行安装", e.Platform, e.Platform)
	case E002TokenExpired:
		return fmt.Sprintf("%s：登录已失效(token过期)，请重新运行本程序进行安装以刷新凭据", e.Platform)
	case E003RiskControl:
		return fmt.Sprintf("%s：被风控拦截(9074)，设备指纹不匹配，请重新运行本程序以重新提取设备信息", e.Platform)
	case E004Network:
		return fmt.Sprintf("%s：网络错误，请检查网络连接后重试", e.Platform)
	case E005BizError:
		return fmt.Sprintf("%s：签到返回未知错误 - %s", e.Platform, e.Message)
	case E006TaskRegister:
		return fmt.Sprintf("计划任务注册失败：%s", e.Message)
	default:
		return fmt.Sprintf("%s：未知错误 - %s", e.Platform, e.Message)
	}
}
