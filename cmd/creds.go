package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"daily-checkin/extract"
	"daily-checkin/platform"
)

// credItem 凭据面板里的一行
type credItem struct {
	Platform string // trae / miyoushe / workbuddy
	Label    string // 账号昵称
	Detail   string // 状态+过期时间等
	File     string // 凭据文件路径（用于删除）
}

// Creds 凭据管理面板：总览所有平台凭据状态，支持重新登录/重新配置/删除。
// 用法：daily-checkin.exe creds
func Creds() {
	reader := bufio.NewReader(os.Stdin)
	for {
		items := buildCredItems()
		printCredItems(items)
		fmt.Println()
		fmt.Print("输入操作（编号+r=重新登录/配置，编号+d=删除，q=退出）：")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" || line == "q" || line == "Q" {
			return
		}
		if len(line) < 2 {
			fmt.Println("格式错误，示例：1r 或 2d")
			continue
		}
		op := line[len(line)-1]
		idx, err := strconv.Atoi(line[:len(line)-1])
		if err != nil || idx < 1 || idx > len(items) {
			fmt.Println("编号无效")
			continue
		}
		item := items[idx-1]
		switch op {
		case 'r', 'R':
			reconfigCred(item)
		case 'd', 'D':
			deleteCred(reader, item)
		default:
			fmt.Println("未知操作，仅支持 r/d/q")
		}
	}
}

func buildCredItems() []credItem {
	var items []credItem

	// Trae 多账号
	for _, a := range extract.LoadTraeAuths() {
		label := a.Nickname
		if label == "" {
			label = a.UID
		}
		status := "✅ 正常"
		if a.ExpiresAt > 0 {
			exp := time.Unix(a.ExpiresAt, 0)
			days := int(time.Until(exp).Hours() / 24)
			if days < 0 {
				status = "❌ Token 已过期"
			} else if days <= 2 {
				status = fmt.Sprintf("⚠️ Token 即将过期（剩 %d 天）", days)
			} else {
				status = fmt.Sprintf("✅ 正常（Token 剩 %d 天）", days)
			}
		}
		items = append(items, credItem{
			Platform: "trae", Label: label, Detail: status, File: a.SrcFile,
		})
	}

	// 米游社
	if c, err := platform.LoadMiyousheCookie(); err == nil {
		preview := c
		if len(preview) > 20 {
			preview = preview[:20] + "..."
		}
		items = append(items, credItem{
			Platform: "miyoushe", Label: "米游社·原神", Detail: "✅ 已配置 Cookie (" + preview + ")",
			File: platform.MiyousheCookiePath(),
		})
	} else {
		items = append(items, credItem{
			Platform: "miyoushe", Label: "米游社·原神", Detail: "❌ 未配置 Cookie",
		})
	}

	// WorkBuddy（本机桌面端登录态，无需手动维护）
	if _, err := extract.LoadWorkBuddyToken(); err == nil {
		items = append(items, credItem{
			Platform: "workbuddy", Label: "WorkBuddy", Detail: "✅ 本机桌面端登录态正常",
		})
	} else {
		items = append(items, credItem{
			Platform: "workbuddy", Label: "WorkBuddy", Detail: "❌ 未检测到本机登录态（请先登录 WorkBuddy 桌面端）",
		})
	}

	return items
}

func printCredItems(items []credItem) {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  凭据状态总览")
	fmt.Println("========================================")
	for i, it := range items {
		fmt.Printf("[%d] %s - %s\n", i+1, it.Label, it.Detail)
	}
}

func reconfigCred(item credItem) {
	switch item.Platform {
	case "trae":
		fmt.Println("正在为 Trae 账号「" + item.Label + "」重新登录...")
		if Login() {
			fmt.Println("✅ 重新登录成功")
		}
	case "miyoushe":
		fmt.Println("正在为米游社重新配置 Cookie...")
		if Miyoushe() {
			fmt.Println("✅ Cookie 重新配置成功")
		}
	case "workbuddy":
		fmt.Println("WorkBuddy 凭据来自本机桌面端登录态，无需手动维护。")
		fmt.Println("如失效，请在 WorkBuddy 桌面端重新登录，本工具会自动读取。")
	}
}

func deleteCred(reader *bufio.Reader, item credItem) {
	if item.File == "" {
		fmt.Println("该凭据无本机文件可删除（" + item.Label + "）")
		return
	}
	fmt.Printf("确认删除「%s」的凭据文件 %s？[y/N]：", item.Label, item.File)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans != "y" && ans != "yes" {
		fmt.Println("已取消")
		return
	}
	if err := os.Remove(item.File); err != nil {
		fmt.Println("删除失败：" + err.Error())
		return
	}
	fmt.Println("✅ 已删除")
}
