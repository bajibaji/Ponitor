# Ponitor

> [中文](./README.md)

**A CRT terminal-style LAN system monitor — with near-zero impact on your host.**

A single-binary Go system monitor that replicates the look and feel of a vintage CRT terminal in your browser: scan lines, phosphor glow, character bloom — all there. Real-time CPU, memory, GPU, and network traffic. Runs silently in the system tray with a right-click menu for scan interval and theme switching.

~14 MB RAM, <5 MB binary, CPU sits at **near zero** most of the time. No Go, Node, or any runtime to install. No app needed on the phone. Built to sit in the background and turn an old phone into a permanent dedicated performance monitor.

**Turn your old phone into a dedicated performance monitor.** Run the binary on your PC, open the browser on an old phone over the same LAN — you get a real-time "second screen" showing CPU, memory, GPU, and network traffic.

<p align="center"><img src="./screenshot.png" alt="screenshot"></p>

## Performance Overhead

| Metric | Value |
|--------|-------|
| Binary size | <5 MB (compiled with `-ldflags="-s -w"`) |
| Runtime memory | ~14 MB |
| CPU overhead | **Near zero** (wakes every 2s to poll, deep sleeps the rest) |
| Dependencies | **Zero** — no Go, Node, or any runtime needed |
| Frontend | Static HTML — **no app install**, just a browser |

## Quick Start

### 1. Build

```bash
# Requires Go 1.26+ (Windows)
go build -ldflags="-H=windowsgui -s -w" -o Ponitor_v0.3.exe .
# Or double-click rebuild.bat — auto-names the output Ponitor_<version>.exe from the git tag
```

### 2. Run

```bash
Ponitor_v0.3.exe
# Or double-click start.bat — auto-picks the newest Ponitor_*.exe
```

### 3. Open on phone

- Connect the phone to the same WiFi
- Open `http://<YOUR_PC_LAN_IP>:8080` in the browser

`start.bat` launches the monitor silently in the background (no window). A system tray icon appears — right-click it for:

- **Open Webpage** — open the dashboard in the default browser
- **Scan Interval** — 0.5s / 1s / 2s / 5s (applied live, no page reload)
- **Theme** — Matrix Green / Amber / Cyber Blue / Classic Mono (applied live)
- **Quit**

Settings persist to `config.json` and survive restarts.

### Stop

- Close the terminal window
- Or double-click `stop.bat`

### Build for other platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o Ponitor_v0.3_linux_amd64 .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o Ponitor_v0.3_darwin_amd64 .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o Ponitor_v0.3_darwin_arm64 .
```

### Build all platforms via GitHub Actions

Push a `v*` tag (or trigger manually) to build all 6 artifacts — Windows / Linux / macOS × amd64 / arm64 — and attach them to a Release:

```bash
git tag v0.3
git push origin v0.3
```

Naming: `Ponitor_<version>_<os>_<arch>.<ext>`, e.g. `Ponitor_v0.3_windows_amd64.exe`.

## Metrics

| Metric | Source |
|--------|--------|
| CPU usage | `gopsutil/cpu` |
| Memory usage | `gopsutil/mem` |
| GPU utilization / VRAM / temp | `nvidia-smi` |
| Network I/O rate | `gopsutil/net` |

- Configurable poll interval (0.5s / 1s / 2s / 5s) via tray menu, applied live to both backend sampling and frontend refresh
- >70% usage → yellow warning, >85% → red alert
- Landscape rotates to a 2×2 grid layout, optimized for phone landscape mode
- 4 switchable themes: Matrix Green, Amber, Cyber Blue, Classic Mono
- Bundled [Cubic 11](https://github.com/ACh-K/Cubic-11) pixel font (SIL OFL 1.1, redistributed with the binary) — phones get the full pixel aesthetic with no font install needed

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
├── .github/workflows/build.yml  # Cross-platform CI build (triggered by v* tags)
├── Ponitor_*.exe    # Compiled binaries (Ponitor_<version>.exe)
├── start.bat        # Start (Windows)
├── stop.bat         # Stop (Windows)
├── rebuild.bat      # Rebuild + start (Windows)
├── README.md        # Chinese version
└── README-en.md     # This file
```

## Tech Stack

- **Backend**: Go + [gopsutil/v4](https://github.com/shirou/gopsutil) + [getlantern/systray](https://github.com/getlantern/systray)
- **Frontend**: Vanilla HTML/CSS/JS, zero dependencies, CRT terminal aesthetic
- **API**: HTTP JSON — `/api/cpu` `/api/mem` `/api/gpu` `/api/network` `/api/config` `/api/theme` `/api/interval`
- **Cross-platform**: Works on Linux / macOS too (GPU requires NVIDIA GPU + nvidia-smi)
