package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"daily-checkin/notify"
)

// NotifySetup 交互式配置强通知渠道（Server酱微信 + SMTP 邮箱）。
// 用法：daily-checkin.exe notify
func NotifySetup() bool {
	reader := bufio.NewReader(os.Stdin)
	cfg := notify.LoadConfig()

	sep := strings.Repeat("=", 60)
	fmt.Println(sep)
	fmt.Println("  强通知配置（失败时除本地 Toast 外，额外推送）")
	fmt.Println(sep)
	fmt.Println()
	fmt.Printf("当前配置：%s\n", notify.Summary())
	fmt.Println()
	fmt.Println("支持渠道：")
	fmt.Println("  [1] Server酱（微信服务号推送，免费，推荐）")
	fmt.Println("      官网 https://sct.ftqq.com 微信扫码登录后获取 SendKey")
	fmt.Println("  [2] SMTP 邮箱（QQ/163/Gmail 等，需开启 SMTP 并获取授权码）")
	fmt.Println()
	fmt.Print("选择渠道 [1/2]，输入 q 退出，直接回车跳过：")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		fmt.Print("请输入 Server酱 SendKey（以 SCT 开头的一串）：")
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)
		if key == "" {
			fmt.Println("已取消")
			return false
		}
		cfg.ServerChanKey = key
		if err := notify.SaveConfig(cfg); err != nil {
			fmt.Println("保存失败: " + err.Error())
			return false
		}
		fmt.Println("✅ Server酱已配置")
		fmt.Print("是否立即发送一条测试通知？[Y/n]：")
		ans, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(ans)) != "n" {
			notify.NotifyAll("每日签到测试", "这是一条测试通知，配置成功！")
			fmt.Println("已发送，请检查微信服务号")
		}
	case "2":
		fmt.Println("SMTP 配置（以 QQ 邮箱为例：smtp.qq.com:465，授权码在 QQ邮箱→设置→账户 开启）")
		fmt.Print("SMTP 服务器（如 smtp.qq.com）：")
		host, _ := reader.ReadString('\n')
		cfg.SMTPHost = strings.TrimSpace(host)
		fmt.Print("SMTP 端口（SSL=465，STARTTLS=587，默认 587）：")
		portStr, _ := reader.ReadString('\n')
		portStr = strings.TrimSpace(portStr)
		if portStr == "" {
			cfg.SMTPPort = 587
		} else {
			cfg.SMTPPort, _ = strconv.Atoi(portStr)
		}
		fmt.Print("发件邮箱（如 xxx@qq.com）：")
		user, _ := reader.ReadString('\n')
		cfg.SMTPUser = strings.TrimSpace(user)
		fmt.Print("SMTP 授权码（非登录密码）：")
		pass, _ := reader.ReadString('\n')
		cfg.SMTPPass = strings.TrimSpace(pass)
		fmt.Print("收件邮箱（可与发件相同）：")
		to, _ := reader.ReadString('\n')
		cfg.SMTPTo = strings.TrimSpace(to)
		if cfg.SMTPHost == "" || cfg.SMTPUser == "" || cfg.SMTPPass == "" || cfg.SMTPTo == "" {
			fmt.Println("信息不完整，已取消")
			return false
		}
		if err := notify.SaveConfig(cfg); err != nil {
			fmt.Println("保存失败: " + err.Error())
			return false
		}
		fmt.Println("✅ SMTP 邮箱已配置")
		fmt.Print("是否立即发送一条测试通知？[Y/n]：")
		ans, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(ans)) != "n" {
			notify.NotifyAll("每日签到测试", "这是一条测试通知，配置成功！")
			fmt.Println("已发送，请检查邮箱")
		}
	case "q", "Q", "":
		fmt.Println("已退出")
		return true
	default:
		fmt.Println("无效选择")
		return false
	}
	return true
}
