package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Menu 交互式主菜单：列出所有功能，输入编号进入，完成后返回菜单。
// 用法：daily-checkin.exe menu
func Menu() {
	reader := bufio.NewReader(os.Stdin)
	for {
		printMenu()
		fmt.Print("请输入编号：")
		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)
		fmt.Println()

		switch choice {
		case "1":
			Setup()
		case "2":
			Run(false)
			pauseMenu(reader)
		case "3":
			Status()
			pauseMenu(reader)
		case "4":
			Creds()
		case "5":
			Login()
			pauseMenu(reader)
		case "6":
			Miyoushe()
			pauseMenu(reader)
		case "7":
			NotifySetup()
			pauseMenu(reader)
		case "8":
			Uninstall()
			pauseMenu(reader)
		case "9":
			PrintUsage()
			pauseMenu(reader)
		case "0", "q", "Q", "exit":
			fmt.Println("再见！")
			return
		default:
			fmt.Println("无效编号，请重新输入。")
			pauseMenu(reader)
		}
	}
}

func printMenu() {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  每日签到 - 功能菜单")
	fmt.Println("========================================")
	fmt.Println("[1] 安装/修改每日自动签到（选平台、设时间）")
	fmt.Println("[2] 立即执行一次签到")
	fmt.Println("[3] 查看当前状态（已装哪些、登录状态）")
	fmt.Println("[4] 凭据管理（查看/更新/删除所有账号）")
	fmt.Println("[5] 登录 Trae（可多次，支持多账号）")
	fmt.Println("[6] 配置米游社 Cookie（原神签到）")
	fmt.Println("[7] 配置强通知（Server酱微信 / 邮箱）")
	fmt.Println("[8] 卸载自动签到")
	fmt.Println("[9] 查看命令行用法")
	fmt.Println("[0] 退出")
	fmt.Println()
}

func pauseMenu(reader *bufio.Reader) {
	fmt.Println()
	fmt.Print("按回车键返回菜单...")
	reader.ReadString('\n')
}
