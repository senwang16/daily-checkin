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

	traeAuths := extract.LoadTraeAuths()

	var aha string
	var aerr error
	aha, aerr = extract.FindAhaDeviceID()
	if aerr != nil {
		// 回退：用任一账号的 device_id（通常就是真实 aha 设备 ID，多账号共用）
		for _, a := range traeAuths {
			if a.DeviceID != "" {
				aha = a.DeviceID
				aerr = nil
				break
			}
		}
	}
	if aerr != nil {
		fmt.Println("# TRAE_AHA_ID：未找到（请确认本机安装并启动过 Trae App，或任一账号凭据含 device_id）")
	} else {
		fmt.Printf("TRAE_AHA_ID=%s\n", aha)
	}
	fmt.Println()

	if len(traeAuths) == 0 {
		fmt.Println("# TRAE_AUTH_JSON：未找到（请先运行 daily-checkin.exe login 登录至少一个 Trae 账号）")
	} else {
		fmt.Printf("# TRAE_AUTH_JSON（整个 JSON 作为 secret 值，可多行，已含全部 %d 个账号）：\n", len(traeAuths))
		data, _ := json.MarshalIndent(traeAuths, "", "  ")
		fmt.Printf("TRAE_AUTH_JSON=%s\n", string(data))
	}
	fmt.Println()

	fmt.Println("CHECKIN_PLATFORMS=trae,workbuddy")
	fmt.Println()
	fmt.Println("# ===== 云端模式另见独立项目（本仓库已不含云端功能） =====")
}
