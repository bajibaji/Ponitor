package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
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
	Lang       string `json:"lang"`        // 语言：auto/zh/en（auto=跟随系统）
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
	w.Header().Set("Content-Type", "application/json")
	mu.RLock()
	d := cpuCached
	mu.RUnlock()
	json.NewEncoder(w).Encode(d)
}

func handleMem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mu.RLock()
	d := memCached
	mu.RUnlock()
	json.NewEncoder(w).Encode(d)
}

func handleNet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mu.RLock()
	d := netCached
	mu.RUnlock()
	if d == nil {
		d = []NetIface{}
	}
	json.NewEncoder(w).Encode(d)
}

func handleGPU(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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
	w.Header().Set("Content-Type", "application/json")
	mu.RLock()
	json.NewEncoder(w).Encode(cfg)
	mu.RUnlock()
}

func handleSetTheme(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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
	w.Header().Set("Content-Type", "application/json")
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
	w.Header().Set("Content-Type", "application/json")
	s := r.URL.Query().Get("h")
	h, err := strconv.Atoi(s)
	if err != nil || h < 0 || h > 400 { // 0 = 自适应
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
	// 保存成功后重定向回首页，避免停留在纯文字成功页
	http.Redirect(w, r, "/", http.StatusFound)
}

// systemUserName 返回当前系统用户名（Windows: %USERNAME%），失败返回空
func systemUserName() string {
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	return ""
}

// systemLang 返回系统语言（zh 或 en），Windows 用注册表 UI 语言
func systemLang() string {
	lang := strings.ToLower(os.Getenv("LANG"))
	if strings.HasPrefix(lang, "zh") {
		return "zh"
	}
	return "en"
}

// menuLang 返回菜单实际语言：config.lang 优先，auto 跟随系统
func menuLang() string {
	mu.RLock()
	l := cfg.Lang
	mu.RUnlock()
	if l == "zh" || l == "en" {
		return l
	}
	return systemLang()
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

var (
	mOpen, mInterval, mHeight, mTheme, mLang, mUser, mAutoStart, mQuit *systray.MenuItem
	iv05, iv1, iv2, iv5                                   *systray.MenuItem
	thGreen, thAmber, thBlue, thMono                      *systray.MenuItem
	chAuto, ch180, ch110, ch70, chHp, chHm                *systray.MenuItem
	uReset, uSet                                          *systray.MenuItem
	langAuto, langZh, langEn                              *systray.MenuItem
)

// menuText 返回指定语言的菜单文案
func menuText(lang string) map[string]string {
	if lang == "zh" {
		return map[string]string{
			"open": "打开网页", "interval": "扫描间隔", "theme": "主题配色",
			"height": "卡片高度", "user": "显示用户名", "lang": "界面语言",
			"autoStart": "开机自启", "quit": "退出程序",
			"iv05": "0.5 秒", "iv1": "1 秒", "iv2": "2 秒", "iv5": "5 秒",
			"thGreen": "黑绿 Matrix", "thAmber": "琥珀金 Amber",
			"thBlue": "赛博蓝 Cyber Blue", "thMono": "黑白 Classic Mono",
			"hAuto": "自适应 (Auto)", "h180": "默认 (180)", "h110": "紧凑 (110)", "h70": "迷你 (70)",
			"hp": "+10", "hm": "-10",
			"uSet": "自定义用户名…", "uReset": "恢复系统用户名",
			"langAuto": "自动（跟随系统）", "langZh": "中文", "langEn": "English",
		}
	}
	return map[string]string{
		"open": "Open Web", "interval": "Interval", "theme": "Theme",
		"height": "Card Height", "user": "Username", "lang": "Language",
		"autoStart": "Run at Startup", "quit": "Quit",
		"iv05": "0.5s", "iv1": "1s", "iv2": "2s", "iv5": "5s",
		"thGreen": "Matrix Green", "thAmber": "Amber", "thBlue": "Cyber Blue",
		"thMono": "Classic Mono",
		"hAuto": "Auto (Adaptive)", "h180": "Default (180)", "h110": "Compact (110)", "h70": "Mini (70)",
		"hp": "+10", "hm": "-10",
		"uSet": "Custom…", "uReset": "Reset Username",
		"langAuto": "Auto (System)", "langZh": "中文", "langEn": "English",
	}
}

// applyMenuLang 按当前语言刷新所有菜单文案（SetTitle 动态更新）
func applyMenuLang() {
	t := menuText(menuLang())
	mOpen.SetTitle(t["open"])
	mInterval.SetTitle(t["interval"])
	mHeight.SetTitle(t["height"])
	mTheme.SetTitle(t["theme"])
	mLang.SetTitle(t["lang"])
	mUser.SetTitle(t["user"])
	mAutoStart.SetTitle(t["autoStart"])
	mQuit.SetTitle(t["quit"])
	iv05.SetTitle(t["iv05"])
	iv1.SetTitle(t["iv1"])
	iv2.SetTitle(t["iv2"])
	iv5.SetTitle(t["iv5"])
	thGreen.SetTitle(t["thGreen"])
	thAmber.SetTitle(t["thAmber"])
	thBlue.SetTitle(t["thBlue"])
	thMono.SetTitle(t["thMono"])
	chAuto.SetTitle(t["hAuto"])
	ch180.SetTitle(t["h180"])
	ch110.SetTitle(t["h110"])
	ch70.SetTitle(t["h70"])
	chHp.SetTitle(t["hp"])
	chHm.SetTitle(t["hm"])
	uReset.SetTitle(t["uReset"])
	uSet.SetTitle(t["uSet"])
	langAuto.SetTitle(t["langAuto"])
	langZh.SetTitle(t["langZh"])
	langEn.SetTitle(t["langEn"])
}

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("Ponitor")
	systray.SetTooltip("Ponitor - System Monitor")

	// 打开网页（最上方）
	mOpen = systray.AddMenuItem("", "")
	systray.AddSeparator()

	// 扫描间隔子菜单
	mInterval = systray.AddMenuItem("", "")
	iv05 = mInterval.AddSubMenuItem("", "")
	iv1 = mInterval.AddSubMenuItem("", "")
	iv2 = mInterval.AddSubMenuItem("", "")
	iv5 = mInterval.AddSubMenuItem("", "")

	// 卡片高度子菜单
	mHeight = systray.AddMenuItem("", "")
	chAuto = mHeight.AddSubMenuItem("", "")
	ch180 = mHeight.AddSubMenuItem("", "")
	ch110 = mHeight.AddSubMenuItem("", "")
	ch70 = mHeight.AddSubMenuItem("", "")
	chHp = mHeight.AddSubMenuItem("", "")
	chHm = mHeight.AddSubMenuItem("", "")

	// 主题子菜单
	mTheme = systray.AddMenuItem("", "")
	thGreen = mTheme.AddSubMenuItem("", "")
	thAmber = mTheme.AddSubMenuItem("", "")
	thBlue = mTheme.AddSubMenuItem("", "")
	thMono = mTheme.AddSubMenuItem("", "")

	// 语言子菜单
	mLang = systray.AddMenuItem("", "")
	langAuto = mLang.AddSubMenuItem("", "")
	langZh = mLang.AddSubMenuItem("", "")
	langEn = mLang.AddSubMenuItem("", "")

	// 用户名子菜单（自定义为主操作，在前）
	mUser = systray.AddMenuItem("", "")
	uSet = mUser.AddSubMenuItem("", "")
	uReset = mUser.AddSubMenuItem("", "")

	// 开机自启动开关
	mAutoStart = systray.AddMenuItem("", "")

	systray.AddSeparator()
	mQuit = systray.AddMenuItem("", "")

	applyMenuLang()

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
		chAuto.Uncheck()
		ch180.Uncheck()
		ch110.Uncheck()
		ch70.Uncheck()
		langAuto.Uncheck()
		langZh.Uncheck()
		langEn.Uncheck()
		mAutoStart.Uncheck()
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
		case 0:
			chAuto.Check()
		case 180:
			ch180.Check()
		case 110:
			ch110.Check()
		case 70:
			ch70.Check()
		}
		switch cfg.Lang {
		case "zh":
			langZh.Check()
		case "en":
			langEn.Check()
		default:
			langAuto.Check()
		}
		mu.RUnlock()
		if autoStartEnabled() {
			mAutoStart.Check()
		}
	}
	refreshChecks()

	setInterval := func(ms int) {
		mu.Lock()
		cfg.IntervalMs = ms
		saveConfig()
		mu.Unlock()
		refreshChecks()
	}
	setTheme := func(name string) {
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
	setLang := func(l string) {
		mu.Lock()
		cfg.Lang = l
		saveConfig()
		mu.Unlock()
		refreshChecks()
		applyMenuLang()
	}

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser("http://" + lanIP() + ":8080")
			case <-iv05.ClickedCh:
				setInterval(500)
			case <-iv1.ClickedCh:
				setInterval(1000)
			case <-iv2.ClickedCh:
				setInterval(2000)
			case <-iv5.ClickedCh:
				setInterval(5000)
			case <-thGreen.ClickedCh:
				setTheme("green")
			case <-thAmber.ClickedCh:
				setTheme("amber")
			case <-thBlue.ClickedCh:
				setTheme("blue")
			case <-thMono.ClickedCh:
				setTheme("mono")
			case <-chAuto.ClickedCh:
				setCardHeight(0)
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
			case <-langAuto.ClickedCh:
				setLang("")
			case <-langZh.ClickedCh:
				setLang("zh")
			case <-langEn.ClickedCh:
				setLang("en")
			case <-mAutoStart.ClickedCh:
				if autoStartEnabled() {
					setAutoStart(false)
					mAutoStart.Uncheck()
				} else {
					setAutoStart(true)
					mAutoStart.Check()
				}
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
	if ensureSingleInstance() {
		fmt.Println("Ponitor 已在运行")
		os.Exit(0)
	}
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
