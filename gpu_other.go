//go:build !windows

package main

// pollGPU 非 Windows 平台不支持 GPU 采集
func pollGPU() *GPUInfo { return nil }
