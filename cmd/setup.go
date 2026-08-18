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
	fmt.Printf("本机登录状态: Trae=%v  WorkBuddy=%v\n", boolStr(traeOK), boolStr(wbOK))
	fmt.Println()

	t := prompt(reader, "请输入每天签到时间（HH:MM，回车默认 10:00）：", "10:00")

	enableTrae := yesNo(reader, "是否启用 Trae（每日 +200 积分）", traeOK)
	enableWB := yesNo(reader, "是否启用 WorkBuddy（每日 +100 积分）", wbOK)

	var plats []string
	if enableTrae {
		plats = append(plats, "trae")
	}
	if enableWB {
		plats = append(plats, "workbuddy")
	}
	if len(plats) == 0 {
		fmt.Println("错误：至少选择一个平台")
		fmt.Println("配置未完成。")
		pause()
		return false
	}

	fmt.Println()
	fmt.Printf("即将安装：每天 %s 自动签到，平台：%s\n", t, strings.Join(plats, ", "))
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
