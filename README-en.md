# Ponitor

> [中文](./README.md)

**A CRT terminal-style LAN system monitor — with near-zero impact on your host.**

A single-binary Go system monitor that replicates the look and feel of a vintage CRT terminal in your browser: scan lines, phosphor glow, character bloom — all there. Real-time CPU, memory, GPU, and network traffic. Runs silently in the system tray with a right-click menu for scan interval and theme switching.

**Turn your old phone into a dedicated performance monitor.** Run the binary on your PC, open the browser on an old phone over the same LAN — you get a real-time "second screen" showing CPU, memory, GPU, and network traffic.

<p align="center"><img src="./screenshot.png" alt="screenshot"></p>

## Features

- **Authentic pixel CRT terminal** — bundles the open-source [Cubic 11](https://github.com/ACh-K/Cubic-11) pixel font (SIL OFL 1.1), complete with scan lines, phosphor glow, and a blinking cursor. No font install needed on phones
- **Every GPU vendor supported** — NVIDIA via `nvidia-smi` (utilization / VRAM / temperature), AMD / Intel via PDH performance counters + WMI (the same data source as Task Manager)
- **Zero-dependency single binary** — no Go, Node, or runtime to install; double-click and run. The frontend is static HTML embedded right into the binary
- **Near-invisible host footprint** — wakes every 2s to sample, deep-sleeps the rest; ~20 MB RAM, near-zero idle CPU
- **Phone browser, zero install** — connect the phone to the same WiFi, open the page, and it becomes a dedicated monitor screen
- **Tray menu live tuning** — scan interval (0.5s / 1s / 2s / 5s) and 4 themes (Matrix Green / Amber / Cyber Blue / Classic Mono) switch instantly; settings persist to `config.json`
- **Auto alerts** — >70% usage turns yellow, >85% turns red
- **Landscape adaptive** — rotates to a 2×2 grid layout on phone landscape, filling the whole screen

## Quick Start

### 1. Download (recommended)

Grab the latest `Ponitor_<version>_windows_amd64.exe` (or arm64) from [Releases](https://github.com/bajibaji/Ponitor/releases) and double-click to run.

### 2. Open on phone

- Connect the phone to the same WiFi
- Open `http://<YOUR_PC_LAN_IP>:8080` in the browser

A system tray icon appears — right-click it for:

- **Open Webpage** — open the dashboard in the default browser
- **Scan Interval** — 0.5s / 1s / 2s / 5s (applied live, no page reload)
- **Theme** — Matrix Green / Amber / Cyber Blue / Classic Mono (applied live)
- **Quit**

Settings persist to `config.json` and survive restarts.

### 3. Stop

Pick **Quit** from the tray menu.

## Build from source

```bash
# Requires Go 1.26+ (Windows)
go build -ldflags="-H=windowsgui -s -w" -o Ponitor_v0.3.exe .
# Or double-click rebuild.bat — auto-names the output Ponitor_<version>.exe from the git tag
```

Launch: `start.bat` auto-picks the newest `Ponitor_*.exe` and runs it silently in the background.

### Build via GitHub Actions

Push a `v*` tag (or trigger manually) to build both Windows artifacts — amd64 + arm64 — and attach them to a Release:

```bash
git tag v0.4
git push origin v0.4
```

Naming: `Ponitor_<version>_windows_<arch>.exe`, e.g. `Ponitor_v0.4_windows_x64.exe` (x64 / arm).

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
monitor/
├── main.go          # Backend: data collection + HTTP API + tray menu
├── dashboard.html   # Frontend: terminal-style dashboard (embedded in binary)
├── hide_windows.go  # Windows: hide nvidia-smi console + open browser
├── hide_other.go    # Non-Windows: no-op stubs
├── gpu_windows.go   # Windows: GPU collection (nvidia-smi or PDH+WMI)
├── gpu_other.go     # Non-Windows: GPU stub
├── Cubic_11.woff2   # Pixel font (SIL OFL 1.1, embedded in binary)
├── OFL.txt          # Font open-source license
├── icon.ico         # Tray icon (embedded)
├── config.json      # Persisted settings (interval + theme)
├── go.mod / go.sum  # Go module dependencies
├── .github/workflows/build.yml  # Windows CI build (triggered by v* tags)
├── Ponitor_*.exe    # Compiled binaries (Ponitor_<version>.exe)
├── start.bat        # Start (Windows)
├── rebuild.bat      # Rebuild + start (Windows)
├── README.md        # Chinese version
└── README-en.md     # This file
```

## Tech Stack

- **Backend**: Go + [gopsutil/v4](https://github.com/shirou/gopsutil) + [getlantern/systray](https://github.com/getlantern/systray)
- **Frontend**: Vanilla HTML/CSS/JS, zero dependencies, CRT terminal aesthetic
- **API**: HTTP JSON — `/api/cpu` `/api/mem` `/api/gpu` `/api/network` `/api/config` `/api/theme` `/api/interval`
- **GPU**: NVIDIA `nvidia-smi` + Windows PDH performance counters + WMI
