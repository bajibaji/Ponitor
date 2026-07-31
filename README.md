# Ponitor

> [English](./README-en.md)

**CRT 终端风格的局域网性能监视器，对宿主机性能的影响几乎可以忽略不计。**

一个 Go 编写的单二进制系统监视器，在浏览器里模拟出老式 CRT 终端的显示效果 —— 扫描线、荧光余晖、字符辉光一应俱全，实时展示 CPU、内存、GPU 和网络流量。从系统托盘静默运行，右键菜单即可切换扫描间隔和主题。

~14 MB 内存，<5 MB 二进制体积，大部分时间 CPU 占用**基本为零**。不需要装 Go、Node 或任何运行时，不需要在手机上装 App。适合长期挂在后台，把旧手机变成一块永远不关机的专属性能监视屏。

**把旧手机变成 PC 的专属性能监视器。** 在 PC 上运行这个二进制文件，同一局域网内的旧手机打开浏览器 —— 多一块实时显示 CPU、内存、GPU、网络流量的"小屏幕"。

<p align="center"><img src="./screenshot.png" alt="screenshot"></p>

## 性能开销

| 指标 | 数值 |
|------|------|
| 二进制体积 | <5 MB（`-ldflags="-s -w"` 编译） |
| 运行时内存 | ~14 MB |
| CPU 开销 | **几乎为零**（默认每 2 秒唤醒采集一次，其余时间深度休眠） |
| 运行环境 | **零依赖** — 不装 Go、不装 Node、不装任何运行时 |
| 前端 | 纯静态 HTML，**手机无需安装 App**，浏览器打开即用 |

## 快速开始

### 1. 编译

```bash
# 确保已安装 Go 1.26+（Windows）
go build -ldflags="-H=windowsgui -s -w" -o Ponitor_v0.3.exe .
# 或者直接双击 rebuild.bat（自动从 git tag 取版本号生成 Ponitor_<版本>.exe）
```

### 2. 启动

```bash
Ponitor_v0.3.exe
# 或者直接双击 start.bat（自动找到最新的 Ponitor_*.exe）
```

### 3. 在手机上打开

- 手机连同一个 WiFi
- 浏览器访问 `http://你的PC内网IP:8080`

`start.bat` 会在后台静默启动（无窗口），任务栏通知区出现托盘图标 —— 右键菜单（从上到下）：

- **打开网页** —— 用默认浏览器打开仪表盘
- **扫描间隔** —— 0.5s / 1s / 2s / 5s（即时生效，无需刷新页面）
- **主题** —— 黑绿 Matrix / 琥珀金 Amber / 赛博蓝 Cyber Blue / 黑白 Classic Mono（即时生效）
- **退出**

设置持久化到 `config.json`，重启后保留。

### 停止

- 关掉命令行窗口
- 或双击 `stop.bat`

### 编译到其他平台

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o Ponitor_v0.3_linux_amd64 .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o Ponitor_v0.3_darwin_amd64 .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o Ponitor_v0.3_darwin_arm64 .
```

### 通过 GitHub Actions 编译全平台

推送 `v*` 标签（或手动触发）即可自动编译 Windows / Linux / macOS 的 amd64 + arm64 共 6 个产物，并上传到 Release：

```bash
git tag v0.3
git push origin v0.3
```

产物命名：`Ponitor_<版本>_<系统>_<架构>.<扩展名>`，如 `Ponitor_v0.3_windows_amd64.exe`。

## 监控项

| 指标 | 数据来源 |
|------|----------|
| CPU 使用率 | `gopsutil/cpu` |
| 内存使用量/占比 | `gopsutil/mem` |
| GPU 利用率/VRAM/温度 | `nvidia-smi` |
| 网络收发速率 | `gopsutil/net` |

- 扫描间隔可通过托盘菜单切换（0.5s / 1s / 2s / 5s），后端采样与前端刷新均即时生效
- 使用率 >70% 黄色预警，>85% 红色告警
- 横屏自动切换 2×2 网格布局，适配手机横屏
- 4 种可切换主题：黑绿 Matrix、琥珀金 Amber、赛博蓝 Cyber Blue、黑白 Classic Mono

## 项目结构

```
monitor/
├── main.go          # 后端：采集 + HTTP API + 托盘菜单
├── dashboard.html   # 前端：终端风格仪表盘（嵌入二进制）
├── hide_windows.go  # Windows：隐藏 nvidia-smi 控制台 + 打开浏览器
├── hide_other.go    # 非 Windows：空实现
├── icon.ico         # 托盘图标（嵌入）
├── config.json      # 持久化设置（间隔 + 主题）
├── go.mod / go.sum  # Go 模块依赖
├── .github/workflows/build.yml  # 全平台 CI 编译（tag v* 触发）
├── Ponitor_*.exe    # 编译产物（Ponitor_<版本>.exe）
├── start.bat        # 启动
├── stop.bat         # 停止
├── rebuild.bat      # 重新编译并启动
├── README.md        # 本文件
└── README-en.md     # 英文版说明
```

## 技术栈

- **后端**: Go + [gopsutil/v4](https://github.com/shirou/gopsutil) + [getlantern/systray](https://github.com/getlantern/systray)
- **前端**: 纯 HTML/CSS/JS，零依赖，模拟 CRT 终端风格
- **API**: HTTP JSON，`/api/cpu` `/api/mem` `/api/gpu` `/api/network` `/api/config` `/api/theme` `/api/interval`
- **跨平台**: Linux / macOS 上也能编译运行（GPU 部分需 NVIDIA 显卡 + nvidia-smi）
