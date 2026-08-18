package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"daily-checkin/platform"
	"daily-checkin/scheduler"
	"daily-checkin/store"
	"github.com/lxn/walk"
)

func guiLogPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "daily-checkin", "gui_error.log")
}

func logGuiError(msg string) {
	_ = os.MkdirAll(filepath.Dir(guiLogPath()), 0o755)
	f, err := os.OpenFile(guiLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		_, _ = f.WriteString(time.Now().Format("2006-01-02 15:04:05") + " " + msg + "\n")
		f.Close()
	}
}

// Run 打开安装向导（双击 exe 默认进入）
func Run() {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("GUI 启动异常: %v", r)
			logGuiError(msg)
			walk.MsgBox(nil, "每日签到 · 错误", msg+"\n\n详细日志见: "+guiLogPath(), walk.MsgBoxOK|walk.MsgBoxIconError)
		}
	}()
	mw, err := walk.NewMainWindow()
	if err != nil {
		msg := "无法创建主窗口: " + err.Error()
		logGuiError(msg)
		walk.MsgBox(nil, "每日签到 · 错误", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}
	mw.SetTitle("每日签到 · 安装向导")
	mw.SetSize(walk.Size{Width: 480, Height: 360})
	mw.SetLayout(walk.NewVBoxLayout())

	if l, e := walk.NewLabel(mw); e == nil {
		l.SetText("检测到本机已登录的签到平台，请勾选要启用的，并设置每天签到时间：")
	}

	cbTrae, _ := walk.NewCheckBox(mw)
	cbTrae.SetText("Trae（每日 +200 积分）")
	cbTrae.SetChecked(platform.Trae{}.Detect())

	cbWB, _ := walk.NewCheckBox(mw)
	cbWB.SetText("WorkBuddy（每日 +100 积分）")
	cbWB.SetChecked(platform.WorkBuddy{}.Detect())

	if l, e := walk.NewLabel(mw); e == nil {
		l.SetText("每天签到时间（HH:MM，24 小时制）：")
	}
	te, _ := walk.NewLineEdit(mw)
	te.SetText("09:30")

	statusLbl, _ := walk.NewLabel(mw)
	statusLbl.SetText("")

	btn, _ := walk.NewPushButton(mw)
	btn.SetText("安装并启用自动签到")
	btn.Clicked().Attach(func() {
		var plats []string
		if cbTrae.Checked() {
			plats = append(plats, "trae")
		}
		if cbWB.Checked() {
			plats = append(plats, "workbuddy")
		}
		if len(plats) == 0 {
			statusLbl.SetText("请至少勾选一个平台")
			return
		}
		t := te.Text()
		if !validTime(t) {
			statusLbl.SetText("时间格式错误，应为 HH:MM（如 09:30）")
			return
		}
		if err := store.SaveConfig(&store.Config{Time: t, Platforms: plats}); err != nil {
			statusLbl.SetText("保存配置失败：" + err.Error())
			return
		}
		exe, _ := os.Executable()
		if err := scheduler.RegisterDaily(exe, t, plats); err != nil {
			statusLbl.SetText("注册计划任务失败：" + err.Error())
			return
		}
		statusLbl.SetText(fmt.Sprintf("✅ 安装成功！已设置为每天 %s 自动签到。", t))
	})

	mw.Show()
	mw.Run()
}

func validTime(t string) bool {
	re := regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
	return re.MatchString(t)
}
