package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"daily-checkin/platform"
)

// Miyoushe 交互式配置米游社 Cookie。
// 用法：daily-checkin.exe miyoushe
func Miyoushe() bool {
	sep := strings.Repeat("=", 60)
	fmt.Println(sep)
	fmt.Println("  米游社·原神签到 - 配置 Cookie")
	fmt.Println(sep)
	fmt.Println()
	fmt.Println("获取 Cookie 步骤：")
	fmt.Println("  1. 浏览器【无痕模式】打开 https://www.miyoushe.com/ys/ 并登录")
	fmt.Println("  2. 按 F12 打开开发者工具 → Network（网络）标签")
	fmt.Println("  3. 刷新页面，在筛选框输入: getUserGameUnreadCount")
	fmt.Println("  4. 点任意一条结果 → Headers → 复制 Request Headers 里的整段 Cookie")
	fmt.Println()

	// 显示当前状态
	if c, err := platform.LoadMiyousheCookie(); err == nil {
		preview := c
		if len(preview) > 30 {
			preview = preview[:30] + "..."
		}
		fmt.Printf("当前已配置 Cookie: %s\n", preview)
	} else {
		fmt.Println("当前未配置 Cookie。")
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("粘贴完整 Cookie（回车确认，直接回车取消）：")
	cookie, _ := reader.ReadString('\n')
	cookie = strings.TrimSpace(cookie)
	if cookie == "" {
		fmt.Println("已取消。")
		return false
	}
	// 基本校验：必须含 account_id 与 cookie_token
	if !strings.Contains(cookie, "account_id") && !strings.Contains(cookie, "ltuid") {
		fmt.Println("警告：Cookie 中未检测到 account_id，可能不完整，仍将保存。")
	}
	if err := platform.SaveMiyousheCookie(cookie); err != nil {
		fmt.Println("保存失败: " + err.Error())
		return false
	}
	fmt.Println()
	fmt.Println("✅ Cookie 已保存到 %LOCALAPPDATA%\\daily-checkin\\miyoushe.json")
	fmt.Println()
	fmt.Print("是否立即测试一次签到？[Y/N]（回车默认 Y）：")
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToUpper(ans))
	if ans == "" || ans == "Y" || ans == "YES" {
		if e := (platform.Miyoushe{}).Checkin(); e != nil {
			fmt.Println("❌ 签到测试失败: " + e.UserMessage())
			return false
		}
		fmt.Println("✅ 签到测试成功（或今日已签到）！")
		fmt.Println("提示：在配置中加入 miyoushe 平台即可每天自动签到。")
	}
	return true
}
