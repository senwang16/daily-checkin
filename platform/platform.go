package platform

import "daily-checkin/internal"

// Platform 是签到平台统一接口
type Platform interface {
	Name() string
	DisplayName() string
	Detect() bool                          // 本机是否已登录
	Checkin() *internal.CheckinError       // 执行签到，nil=成功
}

var registry = map[string]Platform{}

// Register 注册平台实现
func Register(p Platform) { registry[p.Name()] = p }

// Get 按名称取平台
func Get(name string) Platform { return registry[name] }

// Enabled 按配置名称列表返回平台实例
func Enabled(names []string) []Platform {
	var ps []Platform
	for _, n := range names {
		if p, ok := registry[n]; ok {
			ps = append(ps, p)
		}
	}
	return ps
}

func init() {
	Register(Trae{})
	Register(WorkBuddy{})
}
