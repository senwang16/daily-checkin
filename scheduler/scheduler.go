package scheduler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"daily-checkin/internal"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	taskName      = "DailyCheckin"      // 每日定时任务
	logonTaskName = "DailyCheckin-Logon" // 登录触发任务（覆盖关机再开机场景）
)

// decodeGBK 将 schtasks 在中文 Windows 上输出的 GBK 字节解码为 UTF-8 字符串，
// 避免错误信息出现乱码（如「����: �ܾ����ʡ�」）。纯 ASCII / 已为 UTF-8 的内容亦无损。
func decodeGBK(b []byte) string {
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(b)
	if err != nil {
		return string(b)
	}
	return string(decoded)
}

// runSchtasks 执行 schtasks。带 20s 超时：若任务计划程序服务响应慢/被拦，
// 子进程会被强制结束，调用方据此降级，避免向导永久卡在「注册计划任务」这一步。
func runSchtasks(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "schtasks", args...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return decodeGBK(out), fmt.Errorf("schtasks 执行超时(可能 Task Scheduler 服务不可用或被拦截): %w", ctx.Err())
	}
	return decodeGBK(out), err
}

// RegisterDaily 注册自动签到。
// 主任务：每天定时跑（登录时运行，无需密码）。
// 登录任务：每次用户登录即跑一次 —— 覆盖「每天关机、开机登录后」自动补签的场景。
// 两者均幂等（今日已签不重复），schtasks 不可用时降级为开机自启常驻。
func RegisterDaily(exe, t string, plats []string) error {
	cmdLine := fmt.Sprintf("\"%s\" run", exe)
	// 1) 每日定时
	if _, err := runSchtasks("/Create", "/TN", taskName, "/TR", cmdLine, "/SC", "DAILY", "/ST", t, "/F"); err != nil {
		// 2) 降级：schtasks 不可用 → 开机自启常驻进程（run --daemon），自行定时
		if e := registerFallback(exe, true); e != nil {
			return internal.NewCheckinError(internal.E006TaskRegister, "scheduler",
				fmt.Sprintf("schtasks 失败且降级失败: %v", e))
		}
		return nil
	}
	// 主任务成功：清理历史常驻自启，避免与计划任务双跑
	_ = removeFallback()
	// 3) 登录触发（需管理员权限）。失败不阻断整体安装：改用「登录时跑一次」的 Startup 自启兜底
	if out, err := runSchtasks("/Create", "/TN", logonTaskName, "/TR", cmdLine, "/SC", "ONLOGON", "/F"); err != nil {
		if e := registerFallback(exe, false); e != nil {
			return fmt.Errorf("登录触发任务(DailyCheckin-Logon)注册失败: %v; schtasks输出: %s; 已注册每日定时任务但缺少登录兜底（可手动以管理员身份创建）", err, strings.TrimSpace(out))
		}
		fmt.Println("⚠️ 登录触发任务需管理员权限，已自动改用「开机自启(Startup)」方式实现登录后自动补签，功能不受影响。")
	}
	return nil
}

// registerFallback 写开机自启 vbs。daemon=true 时常驻进程自行定时（schtasks 不可用场景）；
// daemon=false 时仅登录时跑一次签到（配合每日定时任务补签，不常驻、不双跑）。
func registerFallback(exe string, daemon bool) error {
	startup := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	if err := os.MkdirAll(startup, 0o755); err != nil {
		return err
	}
	vbs := filepath.Join(startup, "daily_checkin.vbs")
	mode := "run"
	if daemon {
		mode = "run --daemon"
	}
	// vbs 内双引号需写成两个双引号
	content := fmt.Sprintf("CreateObject(\"WScript.Shell\").Run \"\"%s\"\" %s\", 0\n", exe, mode)
	return os.WriteFile(vbs, []byte(content), 0o644)
}

func removeFallback() error {
	vbs := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "daily_checkin.vbs")
	err := os.Remove(vbs)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Remove 移除计划任务与降级自启，返回真实错误
func Remove() error {
	var errs []string
	if _, err := runSchtasks("/Delete", "/TN", taskName, "/F"); err != nil {
		errs = append(errs, fmt.Sprintf("删除每日任务: %v", err))
	}
	if _, err := runSchtasks("/Delete", "/TN", logonTaskName, "/F"); err != nil {
		errs = append(errs, fmt.Sprintf("删除登录任务: %v", err))
	}
	if err := removeFallback(); err != nil {
		errs = append(errs, fmt.Sprintf("删除自启: %v", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
