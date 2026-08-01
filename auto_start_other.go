//go:build !windows

package main

// 非 Windows 平台无自启动支持
func autoStartEnabled() bool { return false }

func setAutoStart(bool) {}
