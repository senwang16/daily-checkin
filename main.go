package main

import (
	"os"

	"daily-checkin/cmd"
	"daily-checkin/gui"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		gui.Run()
		return
	}
	switch args[0] {
	case "run":
		daemon := false
		for _, a := range args[1:] {
			if a == "--daemon" {
				daemon = true
			}
		}
		if !cmd.Run(daemon) {
			os.Exit(1)
		}
	case "status":
		cmd.Status()
	case "export":
		// 导出本机凭据，供写入 GitHub Secrets（云端模式）
		cmd.ExportSecrets()
	case "uninstall":
		cmd.Uninstall()
	default:
		gui.Run()
	}
}
