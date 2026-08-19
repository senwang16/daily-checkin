//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow      = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	procShowWindow            = user32.NewProc("ShowWindow")
)

// hideConsole 在静默签到（run）模式下隐藏控制台窗口，避免计划任务/自启启动时黑框一闪而过。
// 仅当进程拥有独立控制台（计划任务等脱离终端启动）时隐藏；
// 用户在终端里手动运行时会共享父控制台，此时不隐藏，避免误隐藏用户终端。
func hideConsole() {
	var pids [2]uint32
	n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	if n > 1 {
		return
	}
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, 0) // SW_HIDE
	}
}
