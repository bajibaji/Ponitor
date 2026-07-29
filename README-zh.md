# Ponitor

> [English](./README.md)

一个 Go 编写的单二进制系统性能监视器，通过局域网提供 CPU、内存、网络和 GPU 数据的终端风格 Web 仪表盘。无运行时依赖，<5MB 二进制，~14MB 内存。

**把旧手机变成 PC 的专属性能监视器。** 在 PC 上运行这个二进制文件，同一局域网内的旧手机打开浏览器 —— 你就多了一块实时显示 CPU、内存、GPU、网络流量的"小屏幕"。

<p align="center"><img src="./screenshot.png" alt="screenshot"></p>

## 性能

| 指标 | 数值 |
|------|------|
| 二进制体积 | <5 MB（`-ldflags="-s -w"` 编译） |
| 运行时内存 | ~14 MB |
| CPU 开销 | 基本为零（每 2 秒采集一次，其余时间休眠） |
| 运行环境 | 无依赖，不装 Go、不装 Node、不装任何运行时 |
| 前端 | 纯静态 HTML，手机无需安装 App，浏览器打开即用 |

## 快速开始

### 1. 编译

```bash
# 确保已安装 Go 1.26+
go build -ldflags="-s -w" -o monitor.exe .
# 或者直接双击 rebuild.bat
```

### 2. 启动

```bash
monitor.exe
# 或者直接双击 start.bat
```

### 3. 在手机上打开

- 手机连同一个 WiFi
- 浏览器访问 `http://你的PC内网IP:8080`

`start.bat` 启动后会自动打印出完整 URL。

### 停止

- 关掉命令行窗口
- 或双击 `stop.bat`

### 编译到其他平台

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o monitor .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o monitor .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o monitor .
```

## 监控项

| 指标 | 数据来源 |
|------|----------|
| CPU 使用率 | `gopsutil/cpu` |
| 内存使用量/占比 | `gopsutil/mem` |
| GPU 利用率/VRAM/温度 | `nvidia-smi` |
| 网络收发速率 | `gopsutil/net` |

- 每 2 秒刷新一次
- 使用率 >70% 黄色预警，>85% 红色告警
- 横屏自动切换 2×2 网格布局，适配手机横屏

## 项目结构

```
monitor/
├── main.go          # 后端：采集 + HTTP API
├── dashboard.html   # 前端：终端风格仪表盘（嵌入二进制）
├── go.mod / go.sum  # Go 模块依赖
├── monitor.exe      # 编译产物
├── start.bat        # 启动
├── stop.bat         # 停止
├── rebuild.bat      # 重新编译并启动
├── README.md        # 英文版说明
└── README-zh.md     # 本文件
```

## 技术栈

- **后端**: Go + [gopsutil/v4](https://github.com/shirou/gopsutil)
- **前端**: 纯 HTML/CSS/JS，零依赖，模拟 CRT 终端风格
- **API**: HTTP JSON，`/api/cpu` `/api/mem` `/api/gpu` `/api/network`
- **跨平台**: Linux / macOS 上也能编译运行（GPU 部分需 NVIDIA 显卡 + nvidia-smi）
