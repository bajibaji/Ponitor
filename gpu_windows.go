//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows"
)

// ── PDH 性能计数器（任务管理器同款 GPU Engine）──

var (
	modpdh = windows.NewLazyDLL("pdh.dll")

	procPdhOpenQuery          = modpdh.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounter  = modpdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData   = modpdh.NewProc("PdhCollectQueryData")
	procPdhGetFmtCounterArray = modpdh.NewProc("PdhGetFormattedCounterArrayW")
	procPdhRemoveCounter      = modpdh.NewProc("PdhRemoveCounter")
	procPdhCloseQuery         = modpdh.NewProc("PdhCloseQuery")
)

type gpuQuery struct {
	query   uintptr
	counter uintptr
}

func (g *gpuQuery) open() bool {
	hQuery := uintptr(0)
	procPdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&hQuery)))
	if hQuery == 0 {
		return false
	}
	path, _ := windows.UTF16PtrFromString(`\GPU Engine(*)\Utilization Percentage`)
	hCounter := uintptr(0)
	procPdhAddEnglishCounter.Call(hQuery, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&hCounter)))
	if hCounter == 0 {
		procPdhCloseQuery.Call(hQuery)
		return false
	}
	g.query, g.counter = hQuery, hCounter
	return true
}

func (g *gpuQuery) close() {
	if g.counter != 0 {
		procPdhRemoveCounter.Call(g.counter)
	}
	if g.query != 0 {
		procPdhCloseQuery.Call(g.query)
	}
}

// collect 返回所有 GPU 引擎利用率之和（%）。PDH 每次 collect 需两次采样
// （首次只建基线），跨轮只需一次即可拿到"上一轮~本轮"的均值。
// 为简单起见每次调用采集两次，保证数据新鲜；失败返回 -1
func (g *gpuQuery) collect() float64 {
	procPdhCollectQueryData.Call(g.query)
	time.Sleep(200 * time.Millisecond) // 两次采样之间留间隔
	procPdhCollectQueryData.Call(g.query)

	var bufSize, itemCount uint32
	const pdhFmtDouble = 0x00000200
	procPdhGetFmtCounterArray.Call(g.counter, pdhFmtDouble, uintptr(unsafe.Pointer(&bufSize)), uintptr(unsafe.Pointer(&itemCount)), 0)
	if bufSize == 0 {
		return -1
	}
	buf := make([]byte, bufSize)
	procPdhGetFmtCounterArray.Call(g.counter, pdhFmtDouble, uintptr(unsafe.Pointer(&bufSize)),
		uintptr(unsafe.Pointer(&itemCount)), uintptr(unsafe.Pointer(&buf[0])))
	if itemCount == 0 {
		return -1
	}
	// PDH_FMT_COUNTERVALUE_ITEM_W: {WCHAR* szName(8); PDH_FMT_COUNTERVALUE(16)} = 24B
	// PDH_FMT_COUNTERVALUE: CStatus(4)+pad(4)+double(8) → 数值在偏移 16
	const itemSize, valOffset = 24, 16
	var total float64
	base := uintptr(unsafe.Pointer(&buf[0]))
	for i := uint32(0); i < itemCount; i++ {
		v := *(*float64)(unsafe.Pointer(base + uintptr(i)*itemSize + valOffset))
		if v > 0 {
			total += v
		}
	}
	if total <= 0 {
		return -1
	}
	return total
}

// ── WMI 显卡名 ──

func gpuNameWMI() string {
	type videoController struct {
		Name string
	}
	var vcs []videoController
	if err := wmi.Query("SELECT Name FROM Win32_VideoController", &vcs); err != nil || len(vcs) == 0 {
		return ""
	}
	return vcs[0].Name
}

// ── 统一入口：N 卡走 nvidia-smi，其他走 PDH+WMI ──

var fallbackQuery gpuQuery // ponytail: 单例，多 GPU 只做聚合展示

// pollGPU 返回最新 GPU 数据；无 GPU 或采集失败返回 nil
func pollGPU() *GPUInfo {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,name",
		"--format=csv,noheader,nounits",
	)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err == nil {
		p := strings.Split(strings.TrimSpace(string(out)), ",")
		if len(p) >= 4 {
			u, _ := strconv.ParseFloat(strings.TrimSpace(p[0]), 64)
			muUsed, _ := strconv.Atoi(strings.TrimSpace(p[1]))
			muTotal, _ := strconv.Atoi(strings.TrimSpace(p[2]))
			t, _ := strconv.ParseFloat(strings.TrimSpace(p[3]), 64)
			return &GPUInfo{u, muUsed, muTotal, t, strings.TrimSpace(p[4])}
		}
	}

	// 非 N 卡：PDH 聚合利用率 + WMI 名称；显存/温度不可得（置 0，前端显示 N/A）
	if fallbackQuery.query == 0 && !fallbackQuery.open() {
		return nil
	}
	pct := fallbackQuery.collect()
	if pct < 0 {
		return nil
	}
	return &GPUInfo{pct, 0, 0, 0, gpuNameWMI()}
}
