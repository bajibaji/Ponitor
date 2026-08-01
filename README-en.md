# Ponitor

> [中文](./README.md)

[![Build](https://github.com/bajibaji/Ponitor/actions/workflows/build.yml/badge.svg)](https://github.com/bajibaji/Ponitor/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/bajibaji/Ponitor)](https://github.com/bajibaji/Ponitor/releases)
[![License](https://img.shields.io/github/license/bajibaji/Ponitor)](./LICENSE)
[![Go](https://img.shields.io/badge/go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)

**A pixel-font CRT terminal-style LAN system monitor that turns your old phone into a dedicated second-screen for your PC — with near-zero impact on your host.**

A single-binary Go system monitor that replicates the look and feel of a vintage CRT terminal in your browser: scan lines, phosphor glow, character bloom — all there. Real-time CPU, memory, GPU, and network traffic. Runs silently in the system tray with a right-click menu for scan interval and theme switching.

**Turn your old phone into a dedicated performance monitor.** Run the binary on your PC, open the browser on an old phone over the same LAN — you get a real-time "second screen" showing CPU, memory, GPU, and network traffic.

**Screenshots:**

| Landscape (iPhone 13 Pro) | Portrait |
|---|---|
| <img src="./screenshot.png" width="360" alt="landscape"> | <img src="./screenshot2.png" width="180" alt="portrait"> |

| Old phone (iPhone 4 landscape) |
|---|
| <img src="./screenshot3.png" alt="iPhone 4 landscape"> |

> Tip: On old phones the Safari address bar takes up some height — swipe up to hide it and the dashboard fills the whole screen automatically (adaptive card height, no manual tuning needed).

## Features

- **Authentic pixel CRT terminal** — bundles the open-source [Cubic 11](https://github.com/ACh-K/Cubic-11) pixel font (SIL OFL 1.1), complete with scan lines, phosphor glow, and a blinking cursor. No font install needed on phones
- **Every GPU vendor supported** — NVIDIA via `nvidia-smi` (utilization / VRAM / temperature), AMD / Intel via PDH performance counters + WMI (the same data source as Task Manager)
- **Zero-dependency single binary** — no Go, Node, or runtime to install; double-click and run. The frontend is static HTML embedded right into the binary
- **Near-invisible host footprint** — ~20 MB resident RAM, near-zero idle CPU; wakes every 2s to sample then deep-sleeps, completely unnoticeable in daily use
- **Phone browser, zero install** — connect the phone to the same WiFi, open the page, and it becomes a dedicated monitor screen
- **Tray menu live tuning** — scan interval (0.5s / 1s / 2s / 5s), 4 themes, card height (adaptive / fixed presets for phone portrait/landscape), username, language (auto / 中文 / English), and run-at-startup switch instantly; settings persist to `config.json`
- **Auto alerts** — >70% usage turns yellow, >85% turns red
- **Landscape adaptive** — rotates to a 2×2 grid layout on phone landscape, filling the whole screen

### Performance footprint

| Metric | Value |
|--------|-------|
| Resident RAM | ~20 MB |
| Idle CPU | ~0.01% (measured avg 0.007%, peak 0.03% at 1s polling; lower at the default 2s; deep-sleeps otherwise) |
| Sample interval | 0.5s / 1s / 2s / 5s adjustable |

## Quick Start

### 1. Download (recommended)

Grab the latest `Ponitor_<version>_windows_x64.exe` (or `windows_arm`) from [Releases](https://github.com/bajibaji/Ponitor/releases) and double-click to run.

### 2. Open on phone

- Connect the phone to the same WiFi
- Open `http://<YOUR_PC_LAN_IP>:8080` in the browser

A system tray icon appears — right-click it for (menu language follows the system, switchable in the menu):

- **Open Webpage** — open the dashboard in the default browser
- **Scan Interval** — 0.5s / 1s / 2s / 5s (applied live, no page reload)
- **Theme** — Matrix Green / Amber / Cyber Blue / Classic Mono (applied live)
- **Card Height** — Auto (Adaptive, default) / Standard (180) / Medium (150) / Compact (110) / +10 / -10, fits phone portrait/landscape; Auto computes from the screen height so everything fits in one view
- **Username** — reset to system username / custom (opens a settings page; defaults to your PC account name)
- **Language** — Auto (System) / 中文 / English, applies instantly
- **Run at Startup** — toggle auto-start with Windows
- **Author & Version** — grey read-only info (author DanJuan + version number)
- **Quit**

Settings persist to `config.json` and survive restarts.

### 3. Stop

Pick **Quit** from the tray menu. If the tray is not visible (e.g. remote desktop), end the `Ponitor_*.exe` process from Task Manager.

## Build from source

```bash
# Requires Go 1.26+ (Windows)
go build -ldflags="-H=windowsgui -s -w -X main.version=0.5.0" -o Ponitor_0.5.0.exe .
# Or double-click rebuild.bat — auto-names the output Ponitor_<version>.exe from the git tag
```

Launch: just double-click `Ponitor_*.exe` to run.

### Build via GitHub Actions

Push a `v*` tag (or trigger manually) to build both Windows artifacts — x64 + arm — and attach them to a Release:

```bash
git tag v0.4.3
git push origin v0.4.3
```

Naming: `Ponitor_<version>_windows_<arch>.exe`, e.g. `Ponitor_v0.4.3_windows_x64.exe` (x64 / arm).

## Metrics

| Metric | NVIDIA | AMD / Intel |
|--------|--------|-------------|
| GPU utilization | ✅ `nvidia-smi` | ✅ PDH counters (same source as Task Manager) |
| GPU VRAM / temperature | ✅ | ❌ no reliable source (shows N/A) |
| GPU name | ✅ | ✅ WMI |
| CPU usage / cores | `gopsutil/cpu` |
| Memory usage / percent | `gopsutil/mem` |
| Network I/O rate | `gopsutil/net` |

## Project Structure

```
├── main.go          # Backend: data collection + HTTP API + tray menu
├── dashboard.html   # Frontend: terminal-style dashboard (embedded in binary)
├── hide_windows.go  # Windows: hide nvidia-smi console + open browser
├── hide_other.go    # Non-Windows: no-op stubs
├── gpu_windows.go   # Windows: GPU collection (nvidia-smi or PDH+WMI)
├── gpu_other.go     # Non-Windows: GPU stub
├── Cubic_11.woff2   # Pixel font (SIL OFL 1.1, embedded in binary)
├── OFL.txt          # Font open-source license
├── icon.ico         # Tray icon (embedded)
├── config.json      # Persisted settings (interval / theme / card height / username / language)
├── go.mod / go.sum  # Go module dependencies
├── .github/workflows/build.yml  # Windows CI build (triggered by v* tags)
├── Ponitor_*.exe    # Compiled binaries (Ponitor_<version>.exe)
├── rebuild.bat      # Rebuild + start (Windows)
├── README.md        # Chinese version
└── README-en.md     # This file
```

## Tech Stack

- **Backend**: Go + [gopsutil/v4](https://github.com/shirou/gopsutil) + [getlantern/systray](https://github.com/getlantern/systray)
- **Frontend**: Vanilla HTML/CSS/JS, zero dependencies, CRT terminal aesthetic
- **API**: HTTP JSON — `/api/cpu` `/api/mem` `/api/gpu` `/api/network` `/api/config` `/api/theme` `/api/interval` `/api/cardheight` `/api/username`
- **GPU**: NVIDIA `nvidia-smi` + Windows PDH performance counters + WMI

## License

Released under the [MIT License](./LICENSE). The bundled pixel font [Cubic 11](https://github.com/ACh-K/Cubic-11) is licensed under the [SIL Open Font License 1.1](./OFL.txt).
