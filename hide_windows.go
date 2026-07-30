//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// hideWindow 让子进程（nvidia-smi）在 Windows 上不弹控制台窗口
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// openBrowser 用默认浏览器打开 URL
func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
