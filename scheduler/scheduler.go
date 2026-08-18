package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"daily-checkin/internal"
)

const (
	taskName      = "DailyCheckin"      // 每日定时任务
	logonTaskName = "DailyCheckin-Logon" // 登录触发任务（覆盖关机再开机场景）
)

func runSchtasks(args ...string) (string, error) {
	cmd := exec.Command("schtasks", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RegisterDaily 注册自动签到。
// 主任务：每天定时跑（登录时运行，无需密码）。
// 登录任务：每次用户登录即跑一次 —— 覆盖「每天关机、开机登录后」自动补签的场景。
// 两者均幂等（今日已签不重复），schtasks 不可用时降级为开机自启常驻。
func RegisterDaily(exe, t string, plats []string) error {
	cmdLine := fmt.Sprintf("\"%s\" run", exe)
	// 1) 每日定时
	if _, err := runSchtasks("/Create", "/TN", taskName, "/TR", cmdLine, "/SC", "DAILY", "/ST", t, "/F"); err != nil {
		// 2) 降级：写 Startup vbs 自启常驻进程（run --daemon）
		if e := registerFallback(exe); e != nil {
			return internal.NewCheckinError(internal.E006TaskRegister, "scheduler",
				fmt.Sprintf("schtasks 失败且降级失败: %v", e))
		}
		return nil
	}
	// 3) 登录触发（确保关机再开机后也能签到）
	_, _ = runSchtasks("/Create", "/TN", logonTaskName, "/TR", cmdLine, "/SC", "ONLOGON", "/F")
	return nil
}

func registerFallback(exe string) error {
	startup := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	if err := os.MkdirAll(startup, 0o755); err != nil {
		return err
	}
	vbs := filepath.Join(startup, "daily_checkin.vbs")
	// vbs 内双引号需写成两个双引号
	content := fmt.Sprintf("CreateObject(\"WScript.Shell\").Run \"\"%s\"\" run --daemon\", 0\n", exe)
	return os.WriteFile(vbs, []byte(content), 0o644)
}

// Remove 移除计划任务与降级自启
func Remove() error {
	_, _ = runSchtasks("/Delete", "/TN", taskName, "/F")
	_, _ = runSchtasks("/Delete", "/TN", logonTaskName, "/F")
	vbs := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "daily_checkin.vbs")
	_ = os.Remove(vbs)
	return nil
}

// IsRegistered 是否已注册（任一任务存在即视为已注册）
func IsRegistered() bool {
	for _, tn := range []string{taskName, logonTaskName} {
		out, err := runSchtasks("/Query", "/TN", tn)
		if err == nil && len(out) > 0 {
			return true
		}
	}
	return false
}
