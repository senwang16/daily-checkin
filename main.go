package main

import (
	"os"
	"strings"

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
	case "install":
		plats, t := parseInstallArgs(args[1:])
		if !cmd.Install(plats, t) {
			os.Exit(1)
		}
	case "export":
		// 导出本机凭据，供写入 GitHub Secrets（云端模式）
		cmd.ExportSecrets()
	case "uninstall":
		cmd.Uninstall()
	default:
		gui.Run()
	}
}

// parseInstallArgs 解析 --platforms=trae,workbuddy --time=09:30
func parseInstallArgs(args []string) ([]string, string) {
	var plats []string
	t := ""
	for _, a := range args {
		if strings.HasPrefix(a, "--platforms=") {
			plats = strings.Split(strings.TrimPrefix(a, "--platforms="), ",")
		} else if strings.HasPrefix(a, "--time=") {
			t = strings.TrimPrefix(a, "--time=")
		}
	}
	return plats, t
}
