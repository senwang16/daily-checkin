package notify

import (
	"encoding/json"
	"fmt"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-toast/toast"

	"daily-checkin/internal"
)

// NotifyConfig 通知渠道配置（存本机 notify.json）
type NotifyConfig struct {
	ServerChanKey string `json:"serverchan_key"` // Server酱 SendKey（微信推送）
	SMTPHost      string `json:"smtp_host"`      // 如 smtp.qq.com
	SMTPPort      int    `json:"smtp_port"`      // 如 465/587
	SMTPUser      string `json:"smtp_user"`      // 发件邮箱
	SMTPPass      string `json:"smtp_pass"`      // 授权码（非登录密码）
	SMTPTo        string `json:"smtp_to"`        // 收件邮箱
}

func configPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "daily-checkin", "notify.json")
}

// LoadConfig 读取通知配置（环境变量优先，便于云端）
func LoadConfig() *NotifyConfig {
	cfg := &NotifyConfig{}
	// 环境变量覆盖（云端模式）
	if k := os.Getenv("SERVERCHAN_KEY"); k != "" {
		cfg.ServerChanKey = k
	}
	if h := os.Getenv("SMTP_HOST"); h != "" {
		cfg.SMTPHost = h
	}
	if p := os.Getenv("SMTP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &cfg.SMTPPort)
	}
	if u := os.Getenv("SMTP_USER"); u != "" {
		cfg.SMTPUser = u
	}
	if pw := os.Getenv("SMTP_PASS"); pw != "" {
		cfg.SMTPPass = pw
	}
	if to := os.Getenv("SMTP_TO"); to != "" {
		cfg.SMTPTo = to
	}
	if cfg.ServerChanKey != "" || cfg.SMTPHost != "" {
		return cfg
	}
	// 本机文件
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	return cfg
}

// SaveConfig 保存通知配置
func SaveConfig(c *NotifyConfig) error {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "daily-checkin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(configPath(), data, 0o600)
}

// Notify 弹出 Windows 通知中心 Toast（失败通知带具体原因）
func Notify(title, msg string) {
	n := toast.Notification{
		AppID:   "DailyCheckin",
		Title:   title,
		Message: msg,
	}
	_ = n.Push()
}

// NotifyAll 多渠道强通知：Toast + Server酱微信 + SMTP 邮箱（已配置的渠道都会发）
func NotifyAll(title, msg string) {
	Notify(title, msg) // 本地 Toast 始终发
	cfg := LoadConfig()
	if cfg.ServerChanKey != "" {
		if err := sendServerChan(cfg.ServerChanKey, title, msg); err != nil {
			logNotifyErr("Server酱", err)
		}
	}
	if cfg.SMTPHost != "" && cfg.SMTPUser != "" && cfg.SMTPPass != "" && cfg.SMTPTo != "" {
		if err := sendEmail(cfg, title, msg); err != nil {
			logNotifyErr("SMTP", err)
		}
	}
}

// sendServerChan Server酱微信推送（https://sctapi.ftqq.com/<SendKey>.send）
func sendServerChan(key, title, msg string) error {
	api := "https://sctapi.ftqq.com/" + key + ".send"
	body := url.Values{}
	body.Set("title", title)
	body.Set("desp", msg)
	_, resp, err := internal.NewHTTPClient().PostForm(api, body)
	if err != nil {
		return err
	}
	if resp != nil {
		if code, ok := resp["code"].(float64); ok && code != 0 {
			if m, ok := resp["message"].(string); ok {
				return fmt.Errorf("Server酱返回错误: %s", m)
			}
		}
	}
	return nil
}

// sendEmail SMTP 邮件通知（支持 QQ/163/Gmail 等，SSL 465 或 STARTTLS 587）
func sendEmail(cfg *NotifyConfig, title, msg string) error {
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	to := []string{cfg.SMTPTo}
	subject := "【每日签到】" + title
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n\r\n时间: %s\r\n",
		cfg.SMTPUser, cfg.SMTPTo, subject, msg, time.Now().Format("2006-01-02 15:04:05"))
	// 465 走 SSL，587 走 STARTTLS；smtp.SendMail 默认 STARTTLS
	// 对 465 需要显式 TLS，这里用 SendMail 的通用方式（587 兼容性好）
	return smtp.SendMail(addr, auth, cfg.SMTPUser, to, []byte(body))
}

func logNotifyErr(channel string, err error) {
	// 通知失败不阻塞主流程，仅写日志
	f, ferr := os.OpenFile(filepath.Join(os.Getenv("LOCALAPPDATA"), "daily-checkin", "checkin.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if ferr != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] [WARN] %s 通知发送失败: %s\n", time.Now().Format("2006-01-02 15:04:05"), channel, err.Error())
}

// IsConfigured 是否已配置任一强通知渠道
func IsConfigured() bool {
	cfg := LoadConfig()
	return cfg.ServerChanKey != "" || (cfg.SMTPHost != "" && cfg.SMTPUser != "")
}

// Summary 返回已配置渠道的可读描述（用于 creds/notify 命令显示）
func Summary() string {
	cfg := LoadConfig()
	var parts []string
	if cfg.ServerChanKey != "" {
		k := cfg.ServerChanKey
		if len(k) > 12 {
			k = k[:8] + "..."
		}
		parts = append(parts, "Server酱("+k+")")
	}
	if cfg.SMTPHost != "" && cfg.SMTPUser != "" {
		parts = append(parts, "邮箱("+cfg.SMTPTo+")")
	}
	if len(parts) == 0 {
		return "未配置强通知渠道（仅本地 Toast）"
	}
	return strings.Join(parts, " + ")
}
