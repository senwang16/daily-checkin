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
		// 静默签到：计划任务/自启启动时隐藏控制台窗口，避免黑框一闪而过；
		// 用户在终端手动运行时共享父控制台，不会误隐藏。
		hideConsole()
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
	case "login":
		// 交互式登录，生成 trae_auth.json（浏览器手机号/验证码）
		// --new-device：用独立随机设备身份登录（多账号必需，避免同设备互踢）
		newDevice := false
		for _, a := range args[1:] {
			if a == "--new-device" {
				newDevice = true
			}
		}
		if !cmd.Login(newDevice) {
			os.Exit(1)
		}
	case "miyoushe":
		// 交互式配置米游社 Cookie（浏览器无痕模式抓取）
		if !cmd.Miyoushe() {
			os.Exit(1)
		}
	case "creds":
		// 凭据管理面板：总览所有平台凭据状态，支持重新登录/配置/删除
		cmd.Creds()
	case "notify":
		// 配置强通知渠道（Server酱微信 / SMTP 邮箱）
		if !cmd.NotifySetup() {
			os.Exit(1)
		}
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
	case "menu":
		// 交互式主菜单：列出所有功能，输入编号进入
		cmd.Menu()
	default:
		// 未知子命令：打印帮助并退出，避免误入交互式向导
		fmt.Fprintln(os.Stderr, "未知命令: "+args[0])
		cmd.PrintUsage()
		os.Exit(2)
	}
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
