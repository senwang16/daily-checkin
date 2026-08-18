package cmd

import (
	"encoding/json"
	"fmt"

	"daily-checkin/extract"
)

// ExportSecrets 提取本机签到凭据并打印，供复制到 GitHub 仓库 Secrets。
// 用法：daily-checkin.exe export
func ExportSecrets() {
	fmt.Println("# ===== 复制以下值到 GitHub 仓库 Secrets =====")
	fmt.Println("# 路径：仓库 Settings → Secrets and variables → Actions → New repository secret")
	fmt.Println()

	wb, werr := extract.LoadWorkBuddyToken()
	if werr != nil {
		fmt.Println("# WorkBuddy Token：未找到（请先在本机登录 WorkBuddy 桌面端）")
	} else {
		fmt.Printf("WORKBUDDY_TOKEN=%s\n", wb)
	}
	fmt.Println()

	trae, terr := extract.LoadTraeAuth()

	aha, aerr := extract.FindAhaDeviceID()
	if aerr != nil {
		// 回退：用 trae_auth.json 里的 device_id（通常就是真实 aha 设备 ID）
		if trae != nil && trae.DeviceID != "" {
			aha = trae.DeviceID
			aerr = nil
		}
	}
	if aerr != nil {
		fmt.Println("# TRAE_AHA_ID：未找到（请确认本机安装并启动过 Trae App，或 trae_auth.json 含 device_id）")
	} else {
		fmt.Printf("TRAE_AHA_ID=%s\n", aha)
	}
	fmt.Println()

	if terr != nil {
		fmt.Println("# TRAE_AUTH_JSON：未找到（请将旧项目 auth.json 放到 %LOCALAPPDATA%\\daily-checkin\\trae_auth.json）")
	} else {
		fmt.Println("# TRAE_AUTH_JSON（整个 JSON 作为 secret 值，可多行）：")
		data, _ := json.MarshalIndent(trae, "", "  ")
		fmt.Printf("TRAE_AUTH_JSON=%s\n", string(data))
	}
	fmt.Println()

	fmt.Println("CHECKIN_PLATFORMS=trae,workbuddy")
	fmt.Println()
	fmt.Println("# ===== 云端模式另见独立项目（本仓库已不含云端功能） =====")
}
