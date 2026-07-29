# Ponitor

> [中文](./README-zh.md)

A single-binary Go system monitor that exposes CPU, memory, network and GPU metrics over LAN via a terminal-style web dashboard. No runtime, <5MB binary, ~14MB RAM.

**Turn your old phone into a dedicated performance monitor.** Run the binary on your PC, open the browser on an old phone over the same LAN — you get a real-time "second screen" showing CPU, memory, GPU, and network traffic.

<p align="center"><img src="./screenshot.png" alt="screenshot"></p>

## Performance

| Metric | Value |
|--------|-------|
| Binary size | <5 MB (compiled with `-ldflags="-s -w"`) |
| Runtime memory | ~14 MB |
| CPU overhead | Near zero (polls every 2s, sleeps the rest) |
| Dependencies | None — no Go, Node, or any runtime needed |
| Frontend | Static HTML — no app install on the phone, just a browser |

## Quick Start

### 1. Build

```bash
# Requires Go 1.26+
go build -ldflags="-s -w" -o monitor.exe .
# Or double-click rebuild.bat (Windows)
```

### 2. Run

```bash
monitor.exe
# Or double-click start.bat (Windows)
```

### 3. Open on phone

- Connect the phone to the same WiFi
- Open `http://<YOUR_PC_LAN_IP>:8080` in the browser

`start.bat` prints the full URL automatically on launch.

### Stop

- Close the terminal window
- Or double-click `stop.bat`

### Build for other platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o monitor .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o monitor .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o monitor .
```

## Metrics

| Metric | Source |
|--------|--------|
| CPU usage | `gopsutil/cpu` |
| Memory usage | `gopsutil/mem` |
| GPU utilization / VRAM / temp | `nvidia-smi` |
| Network I/O rate | `gopsutil/net` |

- Polls every 2 seconds
- >70% usage → yellow warning, >85% → red alert
- Landscape rotates to a 2×2 grid layout, optimized for phone landscape mode

## Project Structure

```
monitor/
├── main.go          # Backend: data collection + HTTP API
├── dashboard.html   # Frontend: terminal-style dashboard (embedded in binary)
├── go.mod / go.sum  # Go module dependencies
├── monitor.exe      # Compiled binary
├── start.bat        # Start (Windows)
├── stop.bat         # Stop (Windows)
├── rebuild.bat      # Rebuild + start (Windows)
├── README.md        # This file
└── README-zh.md     # Chinese version
```

## Tech Stack

- **Backend**: Go + [gopsutil/v4](https://github.com/shirou/gopsutil)
- **Frontend**: Vanilla HTML/CSS/JS, zero dependencies, CRT terminal aesthetic
- **API**: HTTP JSON — `/api/cpu` `/api/mem` `/api/gpu` `/api/network`
- **Cross-platform**: Works on Linux / macOS too (GPU requires NVIDIA GPU + nvidia-smi)
