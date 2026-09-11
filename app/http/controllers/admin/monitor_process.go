package admin

import (
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/shirou/gopsutil/v3/process"
)

// 进程 Top 排行：复用 *process.Process 以使 CPUPercent() 能基于两次采样算出差值（类似 htop）
const processTopLimit = 20

var (
	processTopMu      sync.Mutex
	processTopHandles = make(map[int32]*process.Process)
)

// processTopSample 进程采样数据
type processTopSample struct {
	pid           int32
	name          string
	user          string
	cpuPercent    float64
	memoryBytes   uint64
	memoryPercent float64
}

// processTopResult 进程排行缓存结果
var (
	processTopCacheMu    sync.RWMutex
	processTopCacheData  map[string]any
	processTopCacheTime  time.Time
	processTopCacheTTL   = 3 * time.Second
	processTopCollecting int32 // 0=空闲，1=正在采集（atomic CAS）
)

// getProcessTopRankings 按 CPU、内存分别取 Top N 进程。
// 采集可能很慢（Windows 上 CPUPercent/Username 等系统调用开销大），
// 因此使用 "后台采集 + 读缓存" 策略：SSE 主循环永远不会被阻塞。
func (r *MonitorController) getProcessTopRankings(ctx http.Context, memTotal uint64) map[string]any {
	empty := map[string]any{
		"by_cpu":    []map[string]any{},
		"by_memory": []map[string]any{},
		"limit":     processTopLimit,
	}

	// 1. 尝试返回缓存
	processTopCacheMu.RLock()
	if processTopCacheData != nil && time.Since(processTopCacheTime) < processTopCacheTTL {
		cached := processTopCacheData
		processTopCacheMu.RUnlock()
		return cached
	}
	processTopCacheMu.RUnlock()

	// 2. 如果已有 goroutine 在采集，直接返回上一次缓存（或空）
	if !swapProcessTopCollecting(0, 1) {
		processTopCacheMu.RLock()
		if processTopCacheData != nil {
			cached := processTopCacheData
			processTopCacheMu.RUnlock()
			return cached
		}
		processTopCacheMu.RUnlock()
		return empty
	}

	// 3. 在后台 goroutine 中采集，带整体 8 秒超时
	go func() {
		defer swapProcessTopCollecting(1, 0)
		result := r.doCollectProcessTop(memTotal)
		processTopCacheMu.Lock()
		processTopCacheData = result
		processTopCacheTime = time.Now()
		processTopCacheMu.Unlock()
	}()

	// 4. 本次返回旧缓存或空
	processTopCacheMu.RLock()
	if processTopCacheData != nil {
		cached := processTopCacheData
		processTopCacheMu.RUnlock()
		return cached
	}
	processTopCacheMu.RUnlock()
	return empty
}

// swapProcessTopCollecting 原子 CAS
func swapProcessTopCollecting(old, new int32) bool {
	return atomic.CompareAndSwapInt32(&processTopCollecting, old, new)
}

// doCollectProcessTop 实际采集逻辑（可能耗时较长，只在后台 goroutine 调用）
func (r *MonitorController) doCollectProcessTop(memTotal uint64) map[string]any {
	empty := map[string]any{
		"by_cpu":    []map[string]any{},
		"by_memory": []map[string]any{},
		"limit":     processTopLimit,
	}

	// 整体超时 8 秒
	done := make(chan map[string]any, 1)
	go func() {
		done <- r.collectProcessTopSamples(memTotal)
	}()
	select {
	case result := <-done:
		return result
	case <-time.After(8 * time.Second):
		return empty
	}
}

// collectProcessTopSamples 枚举进程并采样
func (r *MonitorController) collectProcessTopSamples(memTotal uint64) map[string]any {
	empty := map[string]any{
		"by_cpu":    []map[string]any{},
		"by_memory": []map[string]any{},
		"limit":     processTopLimit,
	}

	procs, err := process.Processes()
	if err != nil {
		return empty
	}

	type sampleResult struct {
		s  processTopSample
		ok bool
	}

	// 并发采样，限制并发数
	concurrency := 16
	if runtime.GOOS == "windows" {
		concurrency = 8
	}
	sem := make(chan struct{}, concurrency)
	results := make(chan sampleResult, len(procs))

	var wg sync.WaitGroup

	processTopMu.Lock()
	current := make(map[int32]struct{}, len(procs))
	for _, p := range procs {
		if p.Pid <= 0 {
			continue
		}
		current[p.Pid] = struct{}{}
		if _, ok := processTopHandles[p.Pid]; !ok {
			processTopHandles[p.Pid] = p
		}
	}
	// 清理已退出的进程句柄
	for pid := range processTopHandles {
		if _, ok := current[pid]; !ok {
			delete(processTopHandles, pid)
		}
	}
	// 拷贝出需要采样的句柄（避免长时间持锁）
	handlesCopy := make(map[int32]*process.Process, len(processTopHandles))
	for pid, proc := range processTopHandles {
		handlesCopy[pid] = proc
	}
	processTopMu.Unlock()

	for pid, proc := range handlesCopy {
		wg.Add(1)
		go func(pid int32, proc *process.Process) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			s := processTopSample{pid: pid}

			name, errName := proc.Name()
			if errName != nil || name == "" {
				name = "?"
			}
			s.name = name

			cpuPct, errCPU := proc.CPUPercent()
			if errCPU != nil || math.IsNaN(cpuPct) || math.IsInf(cpuPct, 0) {
				cpuPct = 0
			}
			s.cpuPercent = cpuPct

			if mi, errMem := proc.MemoryInfo(); errMem == nil && mi != nil {
				s.memoryBytes = mi.RSS
			}

			if memTotal > 0 {
				s.memoryPercent = float64(s.memoryBytes) / float64(memTotal) * 100
			}

			// Username 在 Windows 上极慢，跳过
			if runtime.GOOS != "windows" {
				if u, errU := proc.Username(); errU == nil {
					s.user = u
				}
			}

			results <- sampleResult{s: s, ok: true}
		}(pid, proc)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	samples := make([]processTopSample, 0, len(handlesCopy))
	for r := range results {
		if r.ok {
			samples = append(samples, r.s)
		}
	}

	if len(samples) == 0 {
		return empty
	}

	byCPU := make([]processTopSample, len(samples))
	copy(byCPU, samples)
	sort.Slice(byCPU, func(i, j int) bool {
		if byCPU[i].cpuPercent == byCPU[j].cpuPercent {
			return byCPU[i].memoryBytes > byCPU[j].memoryBytes
		}
		return byCPU[i].cpuPercent > byCPU[j].cpuPercent
	})
	if len(byCPU) > processTopLimit {
		byCPU = byCPU[:processTopLimit]
	}

	byMem := make([]processTopSample, len(samples))
	copy(byMem, samples)
	sort.Slice(byMem, func(i, j int) bool {
		if byMem[i].memoryBytes == byMem[j].memoryBytes {
			return byMem[i].cpuPercent > byMem[j].cpuPercent
		}
		return byMem[i].memoryBytes > byMem[j].memoryBytes
	})
	if len(byMem) > processTopLimit {
		byMem = byMem[:processTopLimit]
	}

	toMaps := func(list []processTopSample) []map[string]any {
		out := make([]map[string]any, len(list))
		for i, s := range list {
			m := map[string]any{
				"pid":            s.pid,
				"name":           s.name,
				"cpu_percent":    s.cpuPercent,
				"memory_bytes":   s.memoryBytes,
				"memory_percent": s.memoryPercent,
			}
			if s.user != "" {
				m["user"] = s.user
			}
			out[i] = m
		}
		return out
	}

	return map[string]any{
		"by_cpu":    toMaps(byCPU),
		"by_memory": toMaps(byMem),
		"limit":     processTopLimit,
	}
}
