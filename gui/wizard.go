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
	. "github.com/lxn/walk/declarative"
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

	var mw *walk.MainWindow
	var cbTrae, cbWB *walk.CheckBox
	var te *walk.LineEdit
	var statusLbl *walk.Label

	_, err := MainWindow{
		AssignTo: &mw,
		Title:    "每日签到 · 安装向导",
		MinSize:  Size{Width: 480, Height: 360},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: "检测到本机已登录的签到平台，请勾选要启用的，并设置每天签到时间：",
			},
			CheckBox{
				AssignTo: &cbTrae,
				Text:     "Trae（每日 +200 积分）",
				Checked:  platform.Trae{}.Detect(),
			},
			CheckBox{
				AssignTo: &cbWB,
				Text:     "WorkBuddy（每日 +100 积分）",
				Checked:  platform.WorkBuddy{}.Detect(),
			},
			Label{
				Text: "每天签到时间（HH:MM，24 小时制）：",
			},
			LineEdit{
				AssignTo: &te,
				Text:     "09:30",
			},
			Label{
				AssignTo: &statusLbl,
				Text:     "",
			},
			PushButton{
				Text: "安装并启用自动签到",
				OnClicked: func() {
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
				},
			},
		},
	}.Run()
	if err != nil {
		msg := "无法创建主窗口: " + err.Error()
		logGuiError(msg)
		walk.MsgBox(nil, "每日签到 · 错误", msg+"\n\n详细日志见: "+guiLogPath(), walk.MsgBoxOK|walk.MsgBoxIconError)
	}
}

func validTime(t string) bool {
	re := regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
	return re.MatchString(t)
}
