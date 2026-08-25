package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"daily-checkin/platform"
)

// Setup 交互式命令行安装（双击 exe 默认进入）。
// 不依赖任何 GUI 库，每台 Windows 都能稳定运行。
func Setup() bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("===================================")
	fmt.Println("  每日签到 - 安装向导")
	fmt.Println("  双击即可配置，无需 GUI 库")
	fmt.Println("===================================")
	fmt.Println()

	traeOK := platform.Trae{}.Detect()
	wbOK := platform.WorkBuddy{}.Detect()
	mysOK := platform.Miyoushe{}.Detect()
	fmt.Printf("本机登录状态: Trae=%v  WorkBuddy=%v  米游社=%v\n", boolStr(traeOK), boolStr(wbOK), boolStr(mysOK))
	fmt.Println()

	// Trae 未登录时引导登录（浏览器手机号/验证码），否则无法自动签到
	if !traeOK {
		fmt.Println("检测到 Trae 未登录，无法自动签到。")
		if yesNo(reader, "是否现在登录 Trae（将打开浏览器）", true) {
			if Login(false) {
				traeOK = platform.Trae{}.Detect()
			} else {
				fmt.Println("登录未完成，本次跳过 Trae 启用。")
			}
		}
		fmt.Println()
	}
	// WorkBuddy 凭证来自本机桌面端登录态，无独立登录流程，仅提示
	if !wbOK {
		fmt.Println("检测到 WorkBuddy 未登录：签到凭证来自本机 WorkBuddy 桌面端登录态。")
		fmt.Println("请先安装并登录 WorkBuddy 桌面端，再重新运行本向导。")
		fmt.Println()
	}

	t := prompt(reader, "请输入每天签到时间（HH:MM，回车默认 10:00）：", "10:00")

	enableTrae := yesNo(reader, "是否启用 Trae（每日 +200 积分）", traeOK)
	enableWB := yesNo(reader, "是否启用 WorkBuddy（每日 +100 积分）", wbOK)
	enableMys := yesNo(reader, "是否启用 米游社·原神（每日签到福利，需先运行 daily-checkin.exe miyoushe 配置 Cookie）", mysOK)

	var plats []string
	if enableTrae {
		plats = append(plats, "trae")
	}
	if enableWB {
		plats = append(plats, "workbuddy")
	}
	if enableMys {
		plats = append(plats, "miyoushe")
	}
	if len(plats) == 0 {
		fmt.Println("错误：至少选择一个平台")
		fmt.Println("配置未完成。")
		pause()
		return false
	}

	fmt.Println()
	fmt.Printf("即将安装：每天 %s 自动签到，平台：%s\n", t, strings.Join(plats, ", "))
	fmt.Println("提示：若未「以管理员身份运行」，定时任务可能注册失败，将自动改用「开机自启」方式，功能不受影响。")
	ok := Install(plats, t)
	pause()
	return ok
}

func prompt(r *bufio.Reader, tip, def string) string {
	fmt.Print(tip)
	s, _ := r.ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	return s
}

func yesNo(r *bufio.Reader, tip string, def bool) bool {
	yes := "Y"
	if !def {
		yes = "N"
	}
	fmt.Printf("%s [Y/N]（回车默认 %s）：", tip, yes)
	s, _ := r.ReadString('\n')
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return def
	}
	return s == "Y" || s == "YES" || s == "是"
}

func boolStr(b bool) string {
	if b {
		return "已登录"
	}
	return "未登录"
}

func pause() {
	fmt.Println()
	fmt.Println("按回车键退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
