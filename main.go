package main

import (
	"fmt"
	"os"
	"strings"

	"daily-checkin/cmd"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		// 双击 exe 默认进入交互式命令行安装向导（不依赖 GUI 库）
		if !cmd.Setup() {
			os.Exit(1)
		}
		return
	}
	switch args[0] {
	case "run":
		daemon := false
		for _, a := range args[1:] {
			if a == "--daemon" {
				daemon = true
			}
		}
		if !cmd.Run(daemon) {
			os.Exit(1)
		}
	case "status":
		cmd.Status()
	case "install":
		plats, t := parseInstallArgs(args[1:])
		if !cmd.Install(plats, t) {
			os.Exit(1)
		}
	case "export":
		// 导出本机凭据，供写入 GitHub Secrets（云端模式）
		cmd.ExportSecrets()
	case "uninstall":
		cmd.Uninstall()
	default:
		// 未知子命令：打印帮助并退出，避免误入交互式向导
		fmt.Fprintln(os.Stderr, "未知命令: "+args[0])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println("用法:")
	fmt.Println("  daily-checkin.exe            打开交互式安装向导")
	fmt.Println("  daily-checkin.exe run        立即执行一次签到")
	fmt.Println("  daily-checkin.exe run --daemon  常驻进程，按配置时间定时签到")
	fmt.Println("  daily-checkin.exe status     查看安装与登录状态")
	fmt.Println("  daily-checkin.exe install --platforms=trae,workbuddy --time=10:00  命令行安装")
	fmt.Println("  daily-checkin.exe export    导出本机凭据（供云端配置）")
	fmt.Println("  daily-checkin.exe uninstall  卸载自动签到")
}

// parseInstallArgs 解析 --platforms=trae,workbuddy --time=09:30
func parseInstallArgs(args []string) ([]string, string) {
	var plats []string
	t := ""
	for _, a := range args {
		if strings.HasPrefix(a, "--platforms=") {
			plats = strings.Split(strings.TrimPrefix(a, "--platforms="), ",")
		} else if strings.HasPrefix(a, "--time=") {
			t = strings.TrimPrefix(a, "--time=")
		}
	}
	return plats, t
}
