package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

//go:embed dashboard.html
var dashboard []byte

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

	for {
		// CPU
		pcts, _ := cpu.Percent(2*time.Second, false)
		if len(pcts) > 0 {
			cp := CPUData{Total: pcts[0], CPUCore: cores}
			pcts, _ = cpu.Percent(0, false) // warm next
			_ = pcts
			mu.Lock()
			cpuCached = cp
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
		out, err := exec.Command("nvidia-smi",
			"--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,name",
			"--format=csv,noheader,nounits",
		).Output()
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

		time.Sleep(2 * time.Second)
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

func main() {
	go poll()
	http.HandleFunc("/", serveDashboard)
	http.HandleFunc("/api/cpu", handleCPU)
	http.HandleFunc("/api/mem", handleMem)
	http.HandleFunc("/api/network", handleNet)
	http.HandleFunc("/api/gpu", handleGPU)
	http.ListenAndServe("0.0.0.0:8080", nil)
}
