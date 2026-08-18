package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config 是安装配置（启用平台 + 时间）
type Config struct {
	Time        string   `json:"time"`
	Platforms   []string `json:"platforms"`
	InstalledAt string   `json:"installed_at"`
}

func dir() string { return filepath.Join(os.Getenv("LOCALAPPDATA"), "daily-checkin") }

func ConfigPath() string { return filepath.Join(dir(), "config.json") }

func LogPath() string { return filepath.Join(dir(), "checkin.log") }

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func SaveConfig(c *Config) error {
	if err := os.MkdirAll(dir(), 0o755); err != nil {
		return err
	}
	c.InstalledAt = time.Now().Format("2006-01-02 15:04:05")
	data, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(ConfigPath(), data, 0o600)
}

func RemoveConfig() error {
	err := os.Remove(ConfigPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// maxLogSize 日志超过该大小后归档为 checkin.log.old
const maxLogSize = 1 << 20 // 1MB

// AppendLog 追加一行签到日志（超限自动轮转）
func AppendLog(line string) {
	_ = os.MkdirAll(dir(), 0o755)
	if fi, err := os.Stat(LogPath()); err == nil && fi.Size() > maxLogSize {
		_ = os.Rename(LogPath(), LogPath()+".old")
	}
	f, err := os.OpenFile(LogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] %s\n", ts, line)
}
