package cmd

import "fmt"

// PrintUsage 打印命令行用法（供 main 和 menu 调用）
func PrintUsage() {
	fmt.Println("用法:")
	fmt.Println("  daily-checkin.exe            打开交互式安装向导")
	fmt.Println("  daily-checkin.exe menu       功能主菜单（所有功能列出来，选编号执行）")
	fmt.Println("  daily-checkin.exe run        立即执行一次签到")
	fmt.Println("  daily-checkin.exe run --daemon  常驻进程，按配置时间定时签到")
	fmt.Println("  daily-checkin.exe status     查看安装与登录状态")
	fmt.Println("  daily-checkin.exe login      登录 Trae，生成签到凭证（浏览器手机号/验证码）")
	fmt.Println("  daily-checkin.exe miyoushe   配置米游社 Cookie（原神签到，浏览器抓取）")
	fmt.Println("  daily-checkin.exe creds      凭据管理面板（查看/更新/删除所有凭据）")
	fmt.Println("  daily-checkin.exe notify     配置强通知（Server酱微信 / SMTP 邮箱）")
	fmt.Println("  daily-checkin.exe install --platforms=trae,workbuddy --time=10:00  命令行安装")
	fmt.Println("  daily-checkin.exe export     导出本机凭据（供云端配置）")
	fmt.Println("  daily-checkin.exe uninstall  卸载自动签到")
}
