package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"daily-checkin/notify"
	"daily-checkin/platform"
	"daily-checkin/scheduler"
	"daily-checkin/store"
)

// Run 执行签到。daemon=true 时进入常驻定时器（降级模式）。返回 true 表示全部成功。
func Run(daemon bool) bool {
	if daemon {
		daemonLoop()
		return true
	}
	return runOnce()
}

func runOnce() bool {
	cfg, err := store.LoadConfig()
	if err != nil {
		cfg = &store.Config{}
	}
	// 云端/命令行覆盖：环境变量优先（无需本机配置文件）
	if p := os.Getenv("CHECKIN_PLATFORMS"); p != "" {
		cfg.Platforms = strings.Split(p, ",")
	}
	if t := os.Getenv("CHECKIN_TIME"); t != "" {
		cfg.Time = t
	}
	if len(cfg.Platforms) == 0 {
		store.AppendLog("未配置平台（请运行安装向导或设置 CHECKIN_PLATFORMS 环境变量）")
		fmt.Println("未配置平台（请运行安装向导或设置 CHECKIN_PLATFORMS 环境变量）")
		return false
	}
	if cfg.Time == "" {
		cfg.Time = "09:30"
	}
	var fails []string
	for _, name := range cfg.Platforms {
		name = strings.TrimSpace(name)
		p := platform.Get(name)
		if p == nil {
			line := "[FAIL] 未知平台: " + name + "（请检查配置中的平台名）"
			store.AppendLog(line)
			fmt.Println(line)
			fails = append(fails, "未知平台: "+name)
			continue
		}
		if e := p.Checkin(); e != nil {
			line := "[FAIL] " + e.UserMessage()
			store.AppendLog(line)
			fmt.Println(line)
			fails = append(fails, e.UserMessage())
		} else {
			line := "[OK] " + p.DisplayName() + " 签到成功"
			store.AppendLog(line)
			fmt.Println(line)
		}
	}
	if len(fails) > 0 {
		notify.Notify("每日签到失败", strings.Join(fails, "\n"))
		fmt.Println("签到存在失败项")
		return false
	}
	store.AppendLog("[INFO] 全部平台签到成功")
	fmt.Println("全部平台签到成功")
	return true
}

func daemonLoop() {
	runOnce()
	for {
		t := "09:30"
		if cfg, err := store.LoadConfig(); err == nil && cfg.Time != "" {
			t = cfg.Time
		}
		waitUntil(t)
		runOnce()
	}
}

func waitUntil(t string) {
	for {
		now := time.Now()
		var hh, mm int
		if _, err := fmt.Sscanf(t, "%d:%d", &hh, &mm); err != nil {
			hh, mm = 9, 30
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		d := time.Until(next)
		if d <= 0 {
			return
		}
		// 分段睡眠：系统休眠唤醒后单调钟停滞，分段重算避免错过定时
		if d > 30*time.Minute {
			time.Sleep(30 * time.Minute)
			continue
		}
		time.Sleep(d)
		return
	}
}

// Status 打印当前安装与登录状态
func Status() {
	cfg, err := store.LoadConfig()
	if err != nil {
		fmt.Println("未安装。直接双击本程序打开安装向导。")
		return
	}
	fmt.Println("已安装，每天 " + cfg.Time + " 签到：" + strings.Join(cfg.Platforms, ", "))
	for _, name := range cfg.Platforms {
		p := platform.Get(name)
		if p == nil {
			continue
		}
		state := "已登录"
		if !p.Detect() {
			state = "未检测到登录凭据"
		}
		fmt.Println(" - " + p.DisplayName() + "：" + state)
	}
}

// Uninstall 卸载
func Uninstall() {
	_ = scheduler.Remove()
	if err := store.RemoveConfig(); err != nil {
		fmt.Println("移除配置失败：" + err.Error())
	} else {
		fmt.Println("已卸载自动签到（计划任务/自启已清理）。")
	}
}
