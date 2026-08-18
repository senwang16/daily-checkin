package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"daily-checkin/scheduler"
	"daily-checkin/store"
)

// Install 命令行安装（GUI 不可用时的兜底方式）。
// platforms 形如 []string{"trae","workbuddy"}；为空则默认两平台全开。
func Install(platforms []string, t string) bool {
	if len(platforms) == 0 {
		platforms = []string{"trae", "workbuddy"}
	}
	if t == "" {
		t = "09:30"
	}
	if !validTime(t) {
		fmt.Println("时间格式错误，应为 HH:MM（如 09:30）")
		return false
	}
	if err := store.SaveConfig(&store.Config{Time: t, Platforms: platforms}); err != nil {
		fmt.Println("保存配置失败:", err)
		return false
	}
	exe, _ := os.Executable()
	if err := scheduler.RegisterDaily(exe, t, platforms); err != nil {
		fmt.Println("注册计划任务失败:", err)
		return false
	}
	fmt.Printf("✅ 安装成功！已设置为每天 %s 自动签到，平台: %s\n", t, strings.Join(platforms, ", "))
	fmt.Println("已注册 [每日定时] + [登录触发] 两个计划任务；每天关机再开机、登录后自动补签。")
	if len(platforms) == 0 {
		platforms = []string{"trae", "workbuddy"}
	}
	return true
}

func validTime(t string) bool {
	re := regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
	return re.MatchString(t)
}
