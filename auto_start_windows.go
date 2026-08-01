//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// autoStartEnabled 检查注册表 Run 键中是否存在 Ponitor 自启动项
func autoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue("Ponitor")
	return err == nil
}

// setAutoStart 写入/删除自启动项（值为 exe 完整路径）
func setAutoStart(on bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	if on {
		exe, err := os.Executable()
		if err != nil {
			return
		}
		_ = k.SetStringValue("Ponitor", exe)
	} else {
		_ = k.DeleteValue("Ponitor")
	}
}
