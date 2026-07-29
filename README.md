# Ponitor

A single-binary Go system monitor that exposes CPU, memory, network and GPU metrics over LAN via a terminal-style web dashboard. No runtime, ~14MB RAM.

**把旧手机变成 PC 的专属性能监视器。** 在 PC 上运行这个二进制文件，同一局域网内的旧手机打开浏览器 —— 你就多了一块实时显示 CPU、内存、GPU、网络流量的"小屏幕"。

![screenshot](./screenshot.png)

## 怎么用

### 1. 下载/编译

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
├── start.bat        # 启动（Windows）
├── stop.bat         # 停止（Windows）
└── rebuild.bat      # 重新编译并启动（Windows）
```

## 技术栈

- **后端**: Go + [gopsutil/v4](https://github.com/shirou/gopsutil)
- **前端**: 纯 HTML/CSS/JS，零依赖，模拟 CRT 终端风格
- **API**: HTTP JSON，`/api/cpu` `/api/mem` `/api/gpu` `/api/network`
- **跨平台**: Linux / macOS 上也能编译运行（GPU 部分需 NVIDIA 显卡 + nvidia-smi）
