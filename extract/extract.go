package extract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"daily-checkin/internal"
)

// 常量（移植自旧 trae_checkin.py，已实测可用）
const (
	CLIENT_ID   = "en1oxy7wnw8j9n"
	APP_VERSION = "0.1.50"
	OAUTH_HOST  = "https://api.trae.com.cn"
	EP_EXCHANGE = "/cloudide/api/v3/trae/oauth/ExchangeToken"
	EP_USERINFO = "/cloudide/api/v3/trae/GetUserInfo"
)

// TraeAuth 是 Trae 登录凭据
type TraeAuth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UID          string `json:"uid"`
	Nickname     string `json:"nickname"`
	DeviceID     string `json:"device_id"`
	ExpiresAt    int64  `json:"expires_at"`
	CreatedAt    int64  `json:"created_at"`
	// FromEnv 标记凭据来自环境变量（云端模式），刷新后无法回写，故跳过自动续期
	FromEnv bool `json:"-"`
	// SrcFile 凭据来源文件（多账号回写定位），不序列化
	SrcFile string `json:"-"`
}

// NeedRefresh token 是否即将过期（提前 12 小时刷新）。
// ExpiresAt 缺失（如环境变量单项注入）时不刷新，直接使用现有 token，避免发送注定失败的刷新请求。
func (a *TraeAuth) NeedRefresh() bool {
	if a.ExpiresAt <= 0 {
		return false
	}
	return time.Now().Unix() > a.ExpiresAt-12*3600
}

// Refresh 用 refresh_token 换取新 access_token（轮换）
func (a *TraeAuth) Refresh() error {
	body := map[string]interface{}{
		"ClientID":     CLIENT_ID,
		"RefreshToken": a.RefreshToken,
		"ClientSecret": "-",
		"UserID":       "",
	}
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
		"User-Agent":   "Trae/" + APP_VERSION,
	}
	code, resp, err := internal.NewHTTPClient().PostJSON(OAUTH_HOST+EP_EXCHANGE, headers, body)
	if err != nil {
		return err
	}
	if code >= 400 {
		return fmt.Errorf("exchange http %d", code)
	}
	res, ok := resp["Result"].(map[string]interface{})
	if !ok || res["Token"] == nil {
		return fmt.Errorf("exchange 返回无 Token: %v", resp)
	}
	a.AccessToken = res["Token"].(string)
	if rt, ok := res["RefreshToken"].(string); ok && rt != "" {
		a.RefreshToken = rt
	}
	if exp, ok := res["TokenExpireAt"].(float64); ok && exp > 0 {
		e := int64(exp)
		if e > 1e12 {
			e /= 1000
		}
		if e > time.Now().Unix() {
			a.ExpiresAt = e
		}
	} else if dur, ok := res["TokenExpireDuration"].(float64); ok && dur > 0 {
		a.ExpiresAt = time.Now().Unix() + int64(dur)
	} else {
		a.ExpiresAt = time.Now().Unix() + 1209600
	}
	return nil
}

func localAppData() string { return os.Getenv("LOCALAPPDATA") }

// LoadTraeAuth 从本机搜索 Trae 登录凭据（返回第一个账号）。
// 多账号场景请用 LoadTraeAuths()。云端模式优先读取环境变量。
func LoadTraeAuth() (*TraeAuth, error) {
	auths := LoadTraeAuths()
	if len(auths) == 0 {
		return nil, fmt.Errorf("未找到 Trae 登录凭据(请运行 daily-checkin.exe login 登录，或设置 TRAE_AUTH_JSON 环境变量)")
	}
	return auths[0], nil
}

// SaveTraeAuth 持久化凭据（供续期回写）。按 uid 存到 auths/ 子目录，支持多账号。
func SaveTraeAuth(a *TraeAuth) error {
	if a.UID != "" {
		return saveTraeAuthTo(authsDir(), "trae-"+a.UID+".json", a)
	}
	// 无 uid 时回退旧单文件（兼容老逻辑）
	return saveTraeAuthTo(filepath.Join(localAppData(), "daily-checkin"), "trae_auth.json", a)
}

func authsDir() string {
	return filepath.Join(localAppData(), "daily-checkin", "auths")
}

func saveTraeAuthTo(dir, name string, a *TraeAuth) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(a, "", "  ")
	p := filepath.Join(dir, name)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// SaveTraeAuthBack 按凭据来源文件回写（多账号刷新 token 后写回原文件）。
// 来源是环境变量或未知时回退到按 uid 保存。
func SaveTraeAuthBack(a *TraeAuth) error {
	if a.SrcFile != "" && !isEnvSource(a.SrcFile) {
		data, _ := json.MarshalIndent(a, "", "  ")
		tmp := a.SrcFile + ".tmp"
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			return err
		}
		return os.Rename(tmp, a.SrcFile)
	}
	return SaveTraeAuth(a)
}

func isEnvSource(src string) bool {
	return len(src) >= 4 && src[:4] == "env:"
}

// LoadTraeAuths 加载本机所有 Trae 账号（多账号支持）。
// 来源：auths/*.json（新，按 uid 命名）+ 旧单文件 trae_auth.json / auth.json（向后兼容）。
// 云端模式：环境变量仅注入单个账号。
func LoadTraeAuths() []*TraeAuth {
	var out []*TraeAuth
	seen := map[string]bool{}

	add := func(a *TraeAuth, fromFile string) {
		if a == nil || a.AccessToken == "" {
			return
		}
		key := a.UID
		if key == "" {
			key = fromFile
		}
		if seen[key] {
			return
		}
		seen[key] = true
		a.SrcFile = fromFile
		out = append(out, a)
	}

	// 1) 环境变量（云端单账号）
	if j := os.Getenv("TRAE_AUTH_JSON"); j != "" {
		var a TraeAuth
		if json.Unmarshal([]byte(j), &a) == nil && a.AccessToken != "" {
			a.FromEnv = true
			add(&a, "env:TRAE_AUTH_JSON")
			return out
		}
	}
	if t := os.Getenv("TRAE_ACCESS_TOKEN"); t != "" {
		a := &TraeAuth{
			AccessToken:  t,
			RefreshToken: os.Getenv("TRAE_REFRESH_TOKEN"),
			UID:          os.Getenv("TRAE_UID"),
			DeviceID:     os.Getenv("TRAE_DEVICE_ID"),
			FromEnv:      true,
		}
		add(a, "env:TRAE_ACCESS_TOKEN")
		return out
	}

	// 2) auths/*.json（新多账号目录）
	if entries, _ := filepath.Glob(filepath.Join(authsDir(), "*.json")); len(entries) > 0 {
		for _, p := range entries {
			if data, err := os.ReadFile(p); err == nil {
				var a TraeAuth
				if json.Unmarshal(data, &a) == nil && a.AccessToken != "" {
					add(&a, p)
				}
			}
		}
	}

	// 3) 旧单文件（向后兼容）
	for _, p := range []string{
		filepath.Join(localAppData(), "daily-checkin", "trae_auth.json"),
		filepath.Join(localAppData(), "daily-checkin", "auth.json"),
	} {
		if data, err := os.ReadFile(p); err == nil {
			var a TraeAuth
			if json.Unmarshal(data, &a) == nil && a.AccessToken != "" {
				add(&a, p)
			}
		}
	}
	return out
}

// FindAhaDeviceID 从 Trae App 日志提取真实 aha 设备 ID（风控白名单指纹）
// 来源：%APPDATA%/TRAE SOLO CN/logs/aha_electron_*.log（旧版）或 logs/aha_log/aha_electron_*.log（新版）。
// 云端模式可用环境变量覆盖。仅匹配显式 device_id 字段，避免把日志里的时间戳/消息 ID 等长数字误当设备指纹。
func FindAhaDeviceID() (string, error) {
	if id := os.Getenv("TRAE_AHA_ID"); id != "" {
		return id, nil
	}
	logDir := filepath.Join(os.Getenv("APPDATA"), "TRAE SOLO CN", "logs")
	// 兼容新旧版本日志路径（新版 Trae 日志在 logs\aha_log\ 子目录）
	patterns := []string{
		filepath.Join(logDir, "aha_electron_*.log"),
		filepath.Join(logDir, "aha_log", "aha_electron_*.log"),
	}
	seen := map[string]bool{}
	var entries []string
	for _, p := range patterns {
		ms, _ := filepath.Glob(p)
		for _, m := range ms {
			if !seen[m] {
				seen[m] = true
				entries = append(entries, m)
			}
		}
	}
	re := regexp.MustCompile(`device_id["'\s:=]+(\d{10,20})`)
	for _, f := range entries {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if m := re.FindSubmatch(data); m != nil {
			return string(m[1]), nil
		}
	}
	return "", fmt.Errorf("未在 Trae 日志找到 aha 设备 ID")
}

// FindTraeMachineID 读取 Trae 桌面端本机 machineid（UUID）。
// 登录时复用真实设备身份，避免随机 device_id 被服务端当成新设备。
func FindTraeMachineID() string {
	p := filepath.Join(os.Getenv("APPDATA"), "TRAE SOLO CN", "machineid")
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// LoadWorkBuddyToken 读取 WorkBuddy 桌面端明文登录态 token。云端模式可用环境变量覆盖。
func LoadWorkBuddyToken() (string, error) {
	if t := os.Getenv("WORKBUDDY_TOKEN"); t != "" {
		return t, nil
	}
	p := filepath.Join(localAppData(), "CodeBuddyExtension", "Data", "Public", "auth", "workbuddy-desktop.info")
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	var d struct {
		Auth struct {
			AccessToken string `json:"accessToken"`
		} `json:"auth"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return "", err
	}
	if d.Auth.AccessToken == "" {
		return "", fmt.Errorf("token 为空")
	}
	return d.Auth.AccessToken, nil
}
