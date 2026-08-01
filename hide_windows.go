//go:build windows

package main

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// hideWindow 让子进程（nvidia-smi）在 Windows 上不弹控制台窗口
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// openBrowser 用默认浏览器打开 URL
func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

// mutexHandle 保持全局互斥体句柄存活（进程退出时系统自动释放）
var mutexHandle windows.Handle

// ensureSingleInstance 全局互斥体实现单实例；已有一个实例运行返回 true
func ensureSingleInstance() bool {
	name, _ := windows.UTF16PtrFromString("Global\\Ponitor")
	h, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		return err == windows.ERROR_ALREADY_EXISTS
	}
	mutexHandle = h // 保持句柄，释放则锁失效
	return false
}
