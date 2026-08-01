//go:build !windows

package main

import "os/exec"

// hideWindow 非 Windows 平台无需处理
func hideWindow(cmd *exec.Cmd) {}

// openBrowser 用默认浏览器打开 URL（Linux: xdg-open，macOS: open）
func openBrowser(url string) {
	for _, c := range []string{"xdg-open", "open"} {
		if err := exec.Command(c, url).Start(); err == nil {
			return
		}
	}
}

// ensureSingleInstance 非 Windows 平台不做单实例限制
func ensureSingleInstance() bool { return false }
