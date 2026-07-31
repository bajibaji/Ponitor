package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
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

//go:embed Cubic_11.woff2
var fontData []byte

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
	CardHeight int    `json:"card_height"` // 卡片高度（px）
	UserName   string `json:"user_name"`   // 仪表盘用户名（空=系统用户名）
}

var (
	cfgFile = "config.json"
	cfg     = Config{IntervalMs: 2000, Theme: "green", CardHeight: 180}
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
	cpu.Percent(0, false)                 // prime
	time.Sleep(100 * time.Millisecond)     // 计数器稳定
	cpu.Percent(0, false)                 // 重置基线，后续读数窗口一致

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

		// GPU（N 卡 nvidia-smi，其他走 PDH+WMI；失败返回 nil）
		if g := pollGPU(); g != nil {
			mu.Lock()
			gpuCached = g
			gpuOK = true
			mu.Unlock()
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

func serveFont(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "font/woff2")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(fontData)
}

// serveSetName 用户名自定义页面（简单内联表单，无 JS）
func serveSetName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="zh"><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Ponitor - 自定义用户名</title><style>body{background:#0c0d0e;color:#00ff66;font-family:monospace;display:flex;align-items:center;justify-content:center;min-height:100dvh;margin:0}
form{display:flex;flex-direction:column;gap:12px;width:280px}</style>
<body><form method="GET" action="/api/username">
<label>自定义用户名（留空恢复系统用户名）：</label>
<input name="name" maxlength="32" placeholder="%s" autofocus style="padding:8px;background:#000;color:#00ff66;border:1px solid #006622;font-family:monospace">
<button type="submit" style="padding:8px;background:#00ff66;color:#000;border:none;font-family:monospace;cursor:pointer">保存</button>
</form></body></html>`, html.EscapeString(displayUserName()))
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
	mu.Unlock()
	saveConfig() // 放锁后写文件，避免持锁 I/O 阻塞读取
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
	mu.Unlock()
	saveConfig()
	w.Write([]byte(`{"ok":true}`))
}

func handleSetCardHeight(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("h")
	h, err := strconv.Atoi(s)
	if err != nil || h < 40 || h > 400 {
		http.Error(w, "bad height", 400)
		return
	}
	mu.Lock()
	cfg.CardHeight = h
	mu.Unlock()
	saveConfig()
	w.Write([]byte(`{"ok":true}`))
}

func handleSetUserName(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if len(name) > 32 {
		http.Error(w, "name too long", 400)
		return
	}
	mu.Lock()
	cfg.UserName = name
	mu.Unlock()
	saveConfig()
	w.Write([]byte(`{"ok":true}`))
}

// systemUserName 返回当前系统用户名（Windows: %USERNAME%），失败返回空
func systemUserName() string {
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	return ""
}

// displayUserName 返回仪表盘显示的用户名：自定义优先，否则系统用户名
func displayUserName() string {
	mu.RLock()
	un := cfg.UserName
	mu.RUnlock()
	if un != "" {
		return un
	}
	return systemUserName()
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

	// 卡片高度子菜单（适配手机横竖屏：默认 180 / 紧凑 110 / 迷你 70 / ±10 微调）
	mHeight := systray.AddMenuItem("卡片高度", "调整卡片高度适配不同屏幕")
	ch180 := mHeight.AddSubMenuItem("默认 (180)", "iPhone 7P 横屏比例")
	ch110 := mHeight.AddSubMenuItem("紧凑 (110)", "小屏横屏")
	ch70 := mHeight.AddSubMenuItem("迷你 (70)", "极致紧凑")
	chHp := mHeight.AddSubMenuItem("+10", "卡片增高")
	chHm := mHeight.AddSubMenuItem("-10", "卡片降低")

	// 用户名子菜单
	mUser := systray.AddMenuItem("用户名", "设置仪表盘显示的用户名")
	uReset := mUser.AddSubMenuItem("恢复系统用户名", "清空自定义，显示电脑账户名")
	uSet := mUser.AddSubMenuItem("自定义…", "打开设置页输入自定义用户名")

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
		ch180.Uncheck()
		ch110.Uncheck()
		ch70.Uncheck()
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
		switch cfg.CardHeight {
		case 180:
			ch180.Check()
		case 110:
			ch110.Check()
		case 70:
			ch70.Check()
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
	setCardHeight := func(h int) {
		mu.Lock()
		cfg.CardHeight = h
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
			case <-ch180.ClickedCh:
				setCardHeight(180)
			case <-ch110.ClickedCh:
				setCardHeight(110)
			case <-ch70.ClickedCh:
				setCardHeight(70)
			case <-chHp.ClickedCh:
				mu.RLock()
				h := cfg.CardHeight + 10
				mu.RUnlock()
				setCardHeight(h)
			case <-chHm.ClickedCh:
				mu.RLock()
				h := cfg.CardHeight - 10
				mu.RUnlock()
				setCardHeight(h)
			case <-uReset.ClickedCh:
				mu.Lock()
				cfg.UserName = ""
				mu.Unlock()
				saveConfig()
			case <-uSet.ClickedCh:
				openBrowser("http://" + lanIP() + ":8080/setname")
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
	http.HandleFunc("/font/Cubic_11.woff2", serveFont)
	http.HandleFunc("/setname", serveSetName)
	http.HandleFunc("/api/cpu", handleCPU)
	http.HandleFunc("/api/mem", handleMem)
	http.HandleFunc("/api/network", handleNet)
	http.HandleFunc("/api/gpu", handleGPU)
	http.HandleFunc("/api/config", handleConfig)
	http.HandleFunc("/api/theme", handleSetTheme)
	http.HandleFunc("/api/interval", handleSetInterval)
	http.HandleFunc("/api/cardheight", handleSetCardHeight)
	http.HandleFunc("/api/username", handleSetUserName)
	go func() {
		srv := &http.Server{
			Addr:         "0.0.0.0:8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		_ = srv.ListenAndServe()
	}()
	systray.Run(onReady, onExit)
}
