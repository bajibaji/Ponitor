package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

//go:embed dashboard.html
var dashboard []byte

//go:embed icon.ico
var iconData []byte

// ── snapshot cache ──

type NetIface struct {
	Name     string  `json:"interface_name"`
	RecvRate float64 `json:"bytes_recv_rate_per_sec"`
	SentRate float64 `json:"bytes_sent_rate_per_sec"`
}

type GPUInfo struct {
	Utilization float64 `json:"utilization"`
	MemUsed     int     `json:"mem_used"`
	MemTotal    int     `json:"mem_total"`
	Temperature float64 `json:"temperature"`
	Name        string  `json:"name"`
}

type CPUData struct {
	Total   float64 `json:"total"`
	CPUCore int     `json:"cpucore"`
}

type MemData struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Free      uint64  `json:"free"`
	Available uint64  `json:"available"`
	Percent   float64 `json:"percent"`
}

// ── config ──

type Config struct {
	IntervalMs int    `json:"interval_ms"` // 采集间隔（毫秒）
	Theme      string `json:"theme"`       // 主题名
}

var (
	cfgFile = "config.json"
	cfg     = Config{IntervalMs: 2000, Theme: "green"}
)

func loadConfig() {
	if b, err := os.ReadFile(cfgFile); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
}

func saveConfig() {
	if b, err := json.MarshalIndent(cfg, "", "  "); err == nil {
		_ = os.WriteFile(cfgFile, b, 0644)
	}
}

var (
	mu        sync.RWMutex
	cpuCached CPUData
	memCached MemData
	netCached []NetIface
	gpuCached *GPUInfo
	gpuOK     bool
	prevNet   map[string]net.IOCountersStat
	prevTime  time.Time
	cores     int
)

func poll() {
	cores, _ = cpu.Counts(true)
	cpu.Percent(0, false) // prime：下次调用返回距本次的用量

	for {
		mu.RLock()
		iv := cfg.IntervalMs
		mu.RUnlock()
		if iv <= 0 {
			iv = 2000
		}
		time.Sleep(time.Duration(iv) * time.Millisecond)

		// CPU（距上次调用的用量，即一个扫描间隔）
		pcts, _ := cpu.Percent(0, false)
		if len(pcts) > 0 {
			mu.Lock()
			cpuCached = CPUData{Total: pcts[0], CPUCore: cores}
			mu.Unlock()
		}

		// Memory
		m, _ := mem.VirtualMemory()
		if m != nil {
			mu.Lock()
			memCached = MemData{
				Total:     m.Total,
				Used:      m.Used,
				Free:      m.Free,
				Available: m.Available,
				Percent:   m.UsedPercent,
			}
			mu.Unlock()
		}

		// Network (rate since last poll)
		nics, _ := net.IOCounters(true)
		now := time.Now()
		elapsed := now.Sub(prevTime).Seconds()
		var netList []NetIface
		for _, nic := range nics {
			if prev, ok := prevNet[nic.Name]; ok && elapsed > 0 {
				recvRate := float64(nic.BytesRecv-prev.BytesRecv) / elapsed
				sentRate := float64(nic.BytesSent-prev.BytesSent) / elapsed
				netList = append(netList, NetIface{
					Name: nic.Name, RecvRate: recvRate, SentRate: sentRate,
				})
			}
		}
		if len(netList) > 0 {
			mu.Lock()
			netCached = netList
			mu.Unlock()
		}
		prevNet = make(map[string]net.IOCountersStat)
		for _, nic := range nics {
			prevNet[nic.Name] = nic
		}
		prevTime = now

		// GPU
		cmd := exec.Command("nvidia-smi",
			"--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,name",
			"--format=csv,noheader,nounits",
		)
		hideWindow(cmd)
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			p := strings.Split(strings.TrimSpace(string(out)), ",")
			if len(p) >= 4 {
				u, _ := strconv.ParseFloat(strings.TrimSpace(p[0]), 64)
				muUsed, _ := strconv.Atoi(strings.TrimSpace(p[1]))
				muTotal, _ := strconv.Atoi(strings.TrimSpace(p[2]))
				t, _ := strconv.ParseFloat(strings.TrimSpace(p[3]), 64)
				mu.Lock()
				gpuCached = &GPUInfo{u, muUsed, muTotal, t, strings.TrimSpace(p[4])}
				gpuOK = true
				mu.Unlock()
			}
		} else {
			mu.Lock()
			gpuOK = false
			mu.Unlock()
		}

	}
}

// ── HTTP handlers ──

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(dashboard)
}

func handleCPU(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	d := cpuCached
	mu.RUnlock()
	json.NewEncoder(w).Encode(d)
}

func handleMem(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	d := memCached
	mu.RUnlock()
	json.NewEncoder(w).Encode(d)
}

func handleNet(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	d := netCached
	mu.RUnlock()
	if d == nil {
		d = []NetIface{}
	}
	json.NewEncoder(w).Encode(d)
}

func handleGPU(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	ok := gpuOK
	d := gpuCached
	mu.RUnlock()
	if !ok || d == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"no gpu"}`))
		return
	}
	json.NewEncoder(w).Encode(d)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	json.NewEncoder(w).Encode(cfg)
	mu.RUnlock()
}

func handleSetTheme(w http.ResponseWriter, r *http.Request) {
	theme := r.URL.Query().Get("name")
	if theme == "" {
		http.Error(w, "missing name", 400)
		return
	}
	mu.Lock()
	cfg.Theme = theme
	saveConfig()
	mu.Unlock()
	w.Write([]byte(`{"ok":true}`))
}

func handleSetInterval(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("ms")
	iv, err := strconv.Atoi(s)
	if err != nil || iv <= 0 {
		http.Error(w, "bad interval", 400)
		return
	}
	mu.Lock()
	cfg.IntervalMs = iv
	saveConfig()
	mu.Unlock()
	w.Write([]byte(`{"ok":true}`))
}

// ── systray ──

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("Ponitor")
	systray.SetTooltip("Ponitor - System Monitor")

	// 打开网页（最上方）
	mOpen := systray.AddMenuItem("打开网页", "在浏览器中打开仪表盘")
	systray.AddSeparator()

	// 扫描间隔子菜单
	mInterval := systray.AddMenuItem("扫描间隔", "采集间隔")
	iv05 := mInterval.AddSubMenuItem("0.5s", "0.5 秒")
	iv1 := mInterval.AddSubMenuItem("1s", "1 秒")
	iv2 := mInterval.AddSubMenuItem("2s", "2 秒")
	iv5 := mInterval.AddSubMenuItem("5s", "5 秒")

	// 主题子菜单
	mTheme := systray.AddMenuItem("主题", "切换配色")
	thGreen := mTheme.AddSubMenuItem("黑绿 Matrix", "荧光绿")
	thAmber := mTheme.AddSubMenuItem("琥珀金 Amber", "琥珀金")
	thBlue := mTheme.AddSubMenuItem("赛博蓝 Cyber Blue", "赛博蓝")
	thMono := mTheme.AddSubMenuItem("黑白 Classic Mono", "黑白")

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出 Ponitor")

	// 勾选当前项
	refreshChecks := func() {
		iv05.Uncheck()
		iv1.Uncheck()
		iv2.Uncheck()
		iv5.Uncheck()
		thGreen.Uncheck()
		thAmber.Uncheck()
		thBlue.Uncheck()
		thMono.Uncheck()
		mu.RLock()
		switch cfg.IntervalMs {
		case 500:
			iv05.Check()
		case 1000:
			iv1.Check()
		case 2000:
			iv2.Check()
		case 5000:
			iv5.Check()
		}
		switch cfg.Theme {
		case "green":
			thGreen.Check()
		case "amber":
			thAmber.Check()
		case "blue":
			thBlue.Check()
		case "mono":
			thMono.Check()
		}
		mu.RUnlock()
	}
	refreshChecks()

	setInterval := func(ms int, item *systray.MenuItem) {
		mu.Lock()
		cfg.IntervalMs = ms
		saveConfig()
		mu.Unlock()
		refreshChecks()
	}
	setTheme := func(name string, item *systray.MenuItem) {
		mu.Lock()
		cfg.Theme = name
		saveConfig()
		mu.Unlock()
		refreshChecks()
	}

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser("http://" + lanIP() + ":8080")
			case <-iv05.ClickedCh:
				setInterval(500, iv05)
			case <-iv1.ClickedCh:
				setInterval(1000, iv1)
			case <-iv2.ClickedCh:
				setInterval(2000, iv2)
			case <-iv5.ClickedCh:
				setInterval(5000, iv5)
			case <-thGreen.ClickedCh:
				setTheme("green", thGreen)
			case <-thAmber.ClickedCh:
				setTheme("amber", thAmber)
			case <-thBlue.ClickedCh:
				setTheme("blue", thBlue)
			case <-thMono.ClickedCh:
				setTheme("mono", thMono)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	os.Exit(0)
}

func main() {
	loadConfig()
	go poll()
	http.HandleFunc("/", serveDashboard)
	http.HandleFunc("/api/cpu", handleCPU)
	http.HandleFunc("/api/mem", handleMem)
	http.HandleFunc("/api/network", handleNet)
	http.HandleFunc("/api/gpu", handleGPU)
	http.HandleFunc("/api/config", handleConfig)
	http.HandleFunc("/api/theme", handleSetTheme)
	http.HandleFunc("/api/interval", handleSetInterval)
	go http.ListenAndServe("0.0.0.0:8080", nil)
	systray.Run(onReady, onExit)
}
