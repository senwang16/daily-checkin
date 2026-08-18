package notify

import "github.com/go-toast/toast"

// Notify 弹出 Windows 通知中心 Toast（失败通知带具体原因）
func Notify(title, msg string) {
	n := toast.Notification{
		AppID:   "DailyCheckin",
		Title:   title,
		Message: msg,
	}
	_ = n.Push()
}
