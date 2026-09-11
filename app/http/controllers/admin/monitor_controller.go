package admin

import (
	"context"
	"encoding/json"
	"fmt"
	appfacades "goravel/app/facades"
	"math"
	nethttp "net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/clients"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sync/singleflight"

	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/utils/errorlog"
	wsnotifications "goravel/app/websocket/notifications"
)

// cloneJSONSafeForSSE 深拷贝监控数据并将 NaN/Inf 转为 0。
// encoding/json 遇到 NaN/Inf 会报错，导致 SSE 静默跳过推送、前端一直无数据。
func cloneJSONSafeForSSE(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return float64(0)
		}
		return x
	case float32:
		xf := float64(x)
		if math.IsNaN(xf) || math.IsInf(xf, 0) {
			return float32(0)
		}
		return x
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = cloneJSONSafeForSSE(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = cloneJSONSafeForSSE(val)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(x))
		for i, val := range x {
			if m, ok := cloneJSONSafeForSSE(val).(map[string]any); ok {
				out[i] = m
			} else {
				out[i] = val
			}
		}
		return out
	default:
		return x
	}
}

// writeMonitorSystemInfoSSE 推送一帧 system_info；返回 false 表示应结束 SSE（写入失败，通常客户端已断开）。
func (r *MonitorController) writeMonitorSystemInfoSSE(ctx http.Context, writer http.StreamWriter) bool {
	defer func() {
		if rec := recover(); rec != nil {
			facades.Log().Debugf("Monitor SSE: panic in write: %v", rec)
		}
	}()

	systemInfo := r.collectSystemInfo(ctx)
	safePayload := cloneJSONSafeForSSE(systemInfo)
	message := map[string]any{
		"type":      "system_info",
		"data":      safePayload,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Failed to marshal system info (SSE)", map[string]any{
			"error": err.Error(),
		}, "Marshal system info SSE error: %v", err)
		return true
	}

	if _, err := writer.WriteString(fmt.Sprintf("data: %s\n\n", string(messageData))); err != nil {
		facades.Log().Debugf("Monitor SSE: write failed, client may have disconnected: %v", err)
		return false
	}
	if err := writer.Flush(); err != nil {
		facades.Log().Debugf("Monitor SSE: flush failed, client may have disconnected: %v", err)
		return false
	}
	return true
}

// 监控数据缓存（短期缓存，减少系统调用）
var (
	monitorCache     map[string]any
	monitorCacheLock sync.RWMutex
	monitorCacheTime time.Time
	cacheDuration    = 1 * time.Second // 缓存1秒，SSE推送间隔2秒时可以减少一半的系统调用
	// singleflight 确保同一时间只有一个 goroutine 执行缓存重建，避免锁竞争
	monitorCacheGroup singleflight.Group
)

// 进程 Top 排行：复用 *process.Process 以使 CPUPercent() 能基于两次采样算出差值（类似 htop）
const processTopLimit = 20

var (
	processTopMu      sync.Mutex
	processTopHandles = make(map[int32]*process.Process)
)

// 网络带宽监控缓存（用于计算速度和峰值）
var (
	lastNetBytesSent  uint64
	lastNetBytesRecv  uint64
	lastNetSampleTime time.Time
	peakSentSpeed     float64 // 峰值发送速度
	peakRecvSpeed     float64 // 峰值接收速度
	peakTotalSpeed    float64 // 峰值总速度
	netSpeedLock      sync.RWMutex
)

// getNetworkSpeed 计算网络速度并记录峰值（需要两次采样）
func getNetworkSpeed(currentBytesSent, currentBytesRecv uint64) (sentSpeed, recvSpeed, totalSpeed, peakSent, peakRecv, peakTotal float64) {
	netSpeedLock.Lock()
	defer netSpeedLock.Unlock()

	now := time.Now()

	// 如果是第一次采样，保存当前值并返回0
	if lastNetSampleTime.IsZero() {
		lastNetBytesSent = currentBytesSent
		lastNetBytesRecv = currentBytesRecv
		lastNetSampleTime = now
		return 0, 0, 0, 0, 0, 0
	}

	// 计算时间差（秒）
	timeDiff := now.Sub(lastNetSampleTime).Seconds()
	if timeDiff <= 0 {
		return sentSpeed, recvSpeed, totalSpeed, peakSentSpeed, peakRecvSpeed, peakTotalSpeed
	}

	// 计算速度（字节/秒）
	var sentSpeedBps, recvSpeedBps float64
	if currentBytesSent >= lastNetBytesSent {
		sentSpeedBps = float64(currentBytesSent-lastNetBytesSent) / timeDiff
	}
	if currentBytesRecv >= lastNetBytesRecv {
		recvSpeedBps = float64(currentBytesRecv-lastNetBytesRecv) / timeDiff
	}

	// 更新缓存
	lastNetBytesSent = currentBytesSent
	lastNetBytesRecv = currentBytesRecv
	lastNetSampleTime = now

	// 转换为Mbps（兆比特每秒）：字节/秒 * 8 / 1024 / 1024
	sentSpeed = sentSpeedBps * 8 / 1024 / 1024
	recvSpeed = recvSpeedBps * 8 / 1024 / 1024
	totalSpeed = sentSpeed + recvSpeed

	// 更新峰值
	if sentSpeed > peakSentSpeed {
		peakSentSpeed = sentSpeed
	}
	if recvSpeed > peakRecvSpeed {
		peakRecvSpeed = recvSpeed
	}
	if totalSpeed > peakTotalSpeed {
		peakTotalSpeed = totalSpeed
	}

	return sentSpeed, recvSpeed, totalSpeed, peakSentSpeed, peakRecvSpeed, peakTotalSpeed
}

type MonitorController struct{}

func NewMonitorController() *MonitorController {
	return &MonitorController{}
}

// convertProcessStatus 将 Linux 进程状态代码转换为友好的状态文本
func convertProcessStatus(statusCode string, processName string) string {
	// Linux 进程状态代码：
	// R = Running (运行中)
	// S = Sleeping (可中断的睡眠)
	// D = Disk sleep (不可中断的睡眠，等待I/O)
	// Z = Zombie (僵尸进程)
	// T = Stopped (停止)
	// I = Idle (空闲)

	statusCode = strings.ToUpper(strings.TrimSpace(statusCode))

	switch statusCode {
	case "R", "RUNNING":
		return "running"
	case "S", "SLEEPING", "SLEEP":
		// 对于服务进程（MySQL、PostgreSQL、Redis、应用），sleep 状态是正常的，显示为 running
		if processName == "mysql" || processName == "postgresql" || processName == "redis" || processName == "app" {
			return "running"
		}
		return "sleep"
	case "D", "DISK SLEEP":
		return "running" // 等待I/O也是运行状态的一种
	case "Z", "ZOMBIE":
		return "zombie"
	case "T", "STOPPED":
		return "stopped"
	case "I", "IDLE":
		return "running" // 空闲也是运行状态
	default:
		// 如果无法识别，对于服务进程默认显示 running
		if processName == "mysql" || processName == "postgresql" || processName == "redis" || processName == "app" {
			return "running"
		}
		return statusCode
	}
}

// getProcessInfo 获取指定进程的 CPU 和内存信息
func getProcessInfo(ctx http.Context, processName string, pid int32) map[string]any {
	result := map[string]any{
		"name":   processName,
		"pid":    pid,
		"cpu":    0.0,
		"memory": 0,
		"status": "not_found",
		"rss":    0, // 物理内存占用
		"vms":    0, // 虚拟内存占用
	}

	if pid <= 0 {
		return result
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		return result
	}

	// 获取 CPU 使用率
	cpuPercent, err := proc.CPUPercent()
	if err == nil {
		result["cpu"] = cpuPercent
	}

	// 获取内存信息
	memInfo, err := proc.MemoryInfo()
	if err == nil {
		result["memory"] = memInfo.RSS // 物理内存占用（字节）
		result["rss"] = memInfo.RSS
		result["vms"] = memInfo.VMS
	}

	// 获取进程状态并转换
	status, err := proc.Status()
	if err == nil && len(status) > 0 {
		// 转换状态代码为友好的文本
		result["status"] = convertProcessStatus(status[0], processName)
	} else {
		// 如果无法获取状态，对于服务进程默认显示 running
		if processName == "mysql" || processName == "postgresql" || processName == "redis" || processName == "app" {
			result["status"] = "running"
		} else {
			result["status"] = "unknown"
		}
	}

	// 获取进程创建时间
	createTime, err := proc.CreateTime()
	if err == nil {
		result["create_time"] = createTime
	}

	// 获取进程名
	name, err := proc.Name()
	if err == nil {
		result["process_name"] = name
	}

	return result
}

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

// findProcessByName 根据进程名查找进程 PID
// 优先匹配更具体的进程名（如 mysqld 优先于 mysql）
func findProcessByName(ctx http.Context, processNames []string) int32 {
	processes, err := process.Processes()
	if err != nil {
		return 0
	}

	// 先尝试精确匹配（更具体的进程名）
	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}
		nameLower := strings.ToLower(name)

		// 优先匹配更具体的进程名
		for _, targetName := range processNames {
			targetLower := strings.ToLower(targetName)
			// 精确匹配或包含匹配
			if nameLower == targetLower || strings.Contains(nameLower, targetLower) {
				// 对于 MySQL，优先选择 mysqld 而不是 mysql
				if targetLower == "mysqld" && nameLower == "mysqld" {
					return proc.Pid
				}
				// 对于 Redis，优先选择 redis-server
				if targetLower == "redis-server" && strings.Contains(nameLower, "redis-server") {
					return proc.Pid
				}
			}
		}
	}

	// 如果精确匹配失败，尝试通过命令行参数匹配
	for _, proc := range processes {
		cmdline, err := proc.Cmdline()
		if err != nil {
			continue
		}
		cmdlineLower := strings.ToLower(cmdline)

		for _, targetName := range processNames {
			targetLower := strings.ToLower(targetName)
			if strings.Contains(cmdlineLower, targetLower) {
				// 排除掉一些明显不是目标进程的情况
				if targetLower == "mysql" && strings.Contains(cmdlineLower, "mysqladmin") {
					continue
				}
				return proc.Pid
			}
		}
	}

	return 0
}

// isLocalHost 检查地址是否为本地地址
func isLocalHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0"
}

// monitorDBConnectionName resolves which database.connections.* entry to read host/port from.
func monitorDBConnectionName(ctx http.Context, driver string) string {
	if conn, ok := helpers.GetTenantConnectionFromContext(ctx); ok && conn != "" {
		return conn
	}
	connectionName := facades.Config().GetString("database.default", "")
	if connectionName == "" {
		if driver == "postgresql" {
			return "postgres"
		}
		if driver == "mysql" {
			return "mysql"
		}
	}
	return connectionName
}

// getMySQLInfoFromDB 通过数据库连接获取MySQL信息
func getMySQLInfoFromDB(ctx http.Context) map[string]any {
	result := map[string]any{
		"name":        "mysql",
		"type":        "remote",
		"status":      "disconnected",
		"version":     "",
		"uptime":      0,
		"threads":     0,
		"queries":     0,
		"connections": 0,
	}

	defer func() {
		if r := recover(); r != nil {
			result["status"] = "error"
		}
	}()

	// 获取数据库连接配置（租户绑定后看租户库）
	driver := strings.ToLower(appfacades.OrmQuery(ctx).Driver())
	connectionName := monitorDBConnectionName(ctx, driver)
	dbHost := facades.Config().GetString(fmt.Sprintf("database.connections.%s.host", connectionName), "127.0.0.1")
	dbPort := facades.Config().GetInt(fmt.Sprintf("database.connections.%s.port", connectionName), 3306)

	// 检查是否为本地数据库
	if !isLocalHost(dbHost) {
		result["host"] = fmt.Sprintf("%s:%d", dbHost, dbPort)
		result["type"] = "remote"
		// 云数据库无法获取进程信息（CPU、内存、PID等），不设置这些字段
	} else {
		result["host"] = fmt.Sprintf("%s:%d", dbHost, dbPort)
		result["type"] = "local"
		// 本地数据库可能会通过进程监控获取CPU、内存等信息，但这里先不设置
	}

	query := appfacades.OrmQuery(ctx)
	if query == nil {
		return result
	}

	// 检查连接类型是否为MySQL
	if driver != "mysql" {
		result["status"] = "not_mysql"
		return result
	}

	// 执行MySQL状态查询
	var uptime, threads, queries, connections int64
	hasData := false

	// 获取MySQL版本
	{
		var versionResult struct {
			Version string `gorm:"column:version"`
		}
		if err := query.Raw("SELECT VERSION() as version").Scan(&versionResult); err == nil && versionResult.Version != "" {
			result["version"] = versionResult.Version
			hasData = true
		}

		// 获取MySQL状态信息
		var statusRows []map[string]any
		if err := query.Raw("SHOW STATUS WHERE Variable_name IN ('Uptime', 'Threads_connected', 'Questions')").Scan(&statusRows); err == nil {
			for _, row := range statusRows {
				if variableName, ok := row["Variable_name"].(string); ok {
					if value, ok := row["Value"].(string); ok {
						intValue, _ := strconv.ParseInt(value, 10, 64)
						switch variableName {
						case "Uptime":
							uptime = intValue
						case "Threads_connected":
							// Threads_connected 是当前连接数，应该用作 connections
							connections = intValue
							threads = intValue // 线程数也使用当前连接数
						case "Questions":
							queries = intValue
						}
					}
				}
			}
			result["uptime"] = uptime
			result["threads"] = threads
			result["queries"] = queries
			result["connections"] = connections // 当前连接数
			if len(statusRows) > 0 {
				hasData = true
			}
		}

		// 获取MySQL变量信息（内存相关）
		var variableRows []map[string]any
		if err := query.Raw("SHOW VARIABLES WHERE Variable_name IN ('max_connections', 'innodb_buffer_pool_size', 'slow_query_log', 'long_query_time')").Scan(&variableRows); err == nil {
			for _, row := range variableRows {
				if variableName, ok := row["Variable_name"].(string); ok {
					if value, ok := row["Value"].(string); ok {
						intValue, _ := strconv.ParseInt(value, 10, 64)
						if variableName == "innodb_buffer_pool_size" {
							result["buffer_pool_size"] = intValue
						} else if variableName == "max_connections" {
							result["max_connections"] = intValue
						} else if variableName == "slow_query_log" {
							result["slow_query_log"] = value
						} else if variableName == "long_query_time" {
							if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
								result["long_query_time"] = floatValue
							}
						}
					}
				}
			}
		}

		// 获取MySQL更多状态信息（慢查询、锁等待等）
		var moreStatusRows []map[string]any
		if err := query.Raw("SHOW STATUS WHERE Variable_name IN ('Slow_queries', 'Table_locks_waited', 'Innodb_row_lock_waits', 'Innodb_row_lock_time_avg', 'Threads_running', 'Threads_created', 'Aborted_connects')").Scan(&moreStatusRows); err == nil {
			for _, row := range moreStatusRows {
				if variableName, ok := row["Variable_name"].(string); ok {
					if value, ok := row["Value"].(string); ok {
						intValue, _ := strconv.ParseInt(value, 10, 64)
						switch variableName {
						case "Slow_queries":
							result["slow_queries"] = intValue
						case "Table_locks_waited":
							result["table_locks_waited"] = intValue
						case "Innodb_row_lock_waits":
							result["innodb_row_lock_waits"] = intValue
						case "Innodb_row_lock_time_avg":
							result["innodb_row_lock_time_avg"] = intValue
						case "Threads_running":
							result["threads_running"] = intValue
						case "Threads_created":
							result["threads_created"] = intValue
						case "Aborted_connects":
							result["aborted_connects"] = intValue
						}
					}
				}
			}
		}
	}

	// 只有当成功获取到数据时才设置为 connected
	if hasData {
		result["status"] = "connected"
	} else {
		result["status"] = "disconnected"
	}
	return result
}

// getPostgreSQLInfoFromDB 通过数据库连接获取PostgreSQL信息
func getPostgreSQLInfoFromDB(ctx http.Context) map[string]any {
	result := map[string]any{
		"name":            "postgresql",
		"type":            "remote",
		"status":          "disconnected",
		"version":         "",
		"uptime":          0,
		"connections":     0,
		"max_connections": 0,
	}

	defer func() {
		if r := recover(); r != nil {
			result["status"] = "error"
		}
	}()

	// 获取数据库连接配置（租户绑定后看租户库）
	driver := strings.ToLower(appfacades.OrmQuery(ctx).Driver())
	connectionName := monitorDBConnectionName(ctx, driver)
	dbHost := facades.Config().GetString(fmt.Sprintf("database.connections.%s.host", connectionName), "127.0.0.1")
	dbPort := facades.Config().GetInt(fmt.Sprintf("database.connections.%s.port", connectionName), 5432)

	// 检查是否为本地数据库
	if !isLocalHost(dbHost) {
		result["host"] = fmt.Sprintf("%s:%d", dbHost, dbPort)
		result["type"] = "remote"
	} else {
		result["host"] = fmt.Sprintf("%s:%d", dbHost, dbPort)
		result["type"] = "local"
	}

	query := appfacades.OrmQuery(ctx)
	if query == nil {
		return result
	}

	// 检查连接类型是否为PostgreSQL
	if driver != "postgresql" {
		result["status"] = "not_postgres"
		return result
	}

	// 执行PostgreSQL查询
	hasData := false
	{
		// 获取PostgreSQL版本
		var versionResult struct {
			Version string `gorm:"column:version"`
		}
		if err := query.Raw("SELECT version() as version").Scan(&versionResult); err == nil && versionResult.Version != "" {
			version := versionResult.Version
			// 提取版本号（例如：PostgreSQL 14.5 on x86_64-pc-linux-gnu）
			if strings.Contains(version, "PostgreSQL") {
				parts := strings.Fields(version)
				if len(parts) >= 2 {
					result["version"] = parts[1]
				} else {
					result["version"] = version
				}
			} else {
				result["version"] = version
			}
			hasData = true
		}

		// 获取PostgreSQL运行时间（秒）
		var uptimeResult struct {
			Uptime int64 `gorm:"column:uptime"`
		}
		if err := query.Raw("SELECT EXTRACT(EPOCH FROM (now() - pg_postmaster_start_time()))::bigint as uptime").Scan(&uptimeResult); err == nil {
			result["uptime"] = uptimeResult.Uptime
			hasData = true
		}

		// 获取当前连接数
		var connectionsResult struct {
			Count int64 `gorm:"column:count"`
		}
		if err := query.Raw("SELECT count(*) as count FROM pg_stat_activity").Scan(&connectionsResult); err == nil {
			result["connections"] = connectionsResult.Count
			hasData = true
		}

		// 获取最大连接数
		var maxConnectionsResult struct {
			Setting int64 `gorm:"column:setting"`
		}
		if err := query.Raw("SELECT setting::bigint as setting FROM pg_settings WHERE name = 'max_connections'").Scan(&maxConnectionsResult); err == nil {
			result["max_connections"] = maxConnectionsResult.Setting
			hasData = true
		}

		// 获取数据库大小
		var dbSizeResult struct {
			PgDatabaseSize *int64 `gorm:"column:pg_database_size"`
		}
		if err := query.Raw("SELECT pg_database_size(current_database()) as pg_database_size").Scan(&dbSizeResult); err == nil && dbSizeResult.PgDatabaseSize != nil {
			result["database_size"] = *dbSizeResult.PgDatabaseSize
		}

		// 获取活跃连接数
		var activeConnectionsResult struct {
			Count int64 `gorm:"column:count"`
		}
		if err := query.Raw("SELECT count(*) as count FROM pg_stat_activity WHERE state = 'active'").Scan(&activeConnectionsResult); err == nil {
			result["active_connections"] = activeConnectionsResult.Count
		}

		// 获取空闲连接数
		var idleConnectionsResult struct {
			Count int64 `gorm:"column:count"`
		}
		if err := query.Raw("SELECT count(*) as count FROM pg_stat_activity WHERE state = 'idle'").Scan(&idleConnectionsResult); err == nil {
			result["idle_connections"] = idleConnectionsResult.Count
		}

		// 获取总查询数（从启动开始）
		var totalQueriesResult struct {
			Sum *int64 `gorm:"column:sum"`
		}
		if err := query.Raw("SELECT COALESCE(sum(xact_commit + xact_rollback), 0)::bigint as sum FROM pg_stat_database WHERE datname = current_database()").Scan(&totalQueriesResult); err == nil && totalQueriesResult.Sum != nil {
			result["queries"] = *totalQueriesResult.Sum
		}
	}

	// 只有当成功获取到数据时才设置为 connected
	if hasData {
		result["status"] = "connected"
	} else {
		result["status"] = "disconnected"
	}
	return result
}

// getRedisInfoFromConnection 通过Redis连接获取Redis信息
func getRedisInfoFromConnection(ctx http.Context) map[string]any {
	result := map[string]any{
		"name":                     "redis",
		"type":                     "remote",
		"status":                   "disconnected",
		"version":                  "",
		"used_memory":              0,
		"used_memory_human":        "",
		"connected_clients":        0,
		"total_commands_processed": 0,
		"keyspace_hits":            0,
		"keyspace_misses":          0,
	}

	defer func() {
		if r := recover(); r != nil {
			result["status"] = "error"
		}
	}()

	// 获取Redis连接配置（用于显示主机信息）
	redisHost := facades.Config().GetString("database.redis.default.host", "")
	redisPort := facades.Config().GetInt("database.redis.default.port", 6379)

	// 检查是否为本地Redis
	if !isLocalHost(redisHost) {
		result["host"] = fmt.Sprintf("%s:%d", redisHost, redisPort)
		result["type"] = "remote"
		// 云数据库无法获取进程信息（CPU、PID等），不设置这些字段
		// 但可以通过INFO命令获取内存使用情况
	} else {
		result["host"] = fmt.Sprintf("%s:%d", redisHost, redisPort)
		result["type"] = "local"
		// 本地Redis可能会通过进程监控获取CPU等信息，但这里先不设置
	}

	// 使用公共 Redis 客户端（用于执行INFO命令）
	redisClient, err := clients.GetRedisClient("default")
	if err != nil {
		result["status"] = "disconnected"
		return result
	}
	// 注意：使用公共 Redis 客户端池，不需要手动关闭

	// 测试连接（GetRedisClient 已经测试过连接，这里可以省略，但保留用于保险）
	redisCtx := context.Background()
	if err := redisClient.Ping(redisCtx).Err(); err != nil {
		result["status"] = "disconnected"
		return result
	}

	result["status"] = "connected"

	// 执行INFO命令获取详细信息
	infoStr, err := redisClient.Info(redisCtx).Result()
	if err == nil && infoStr != "" {
		// 解析Redis INFO命令返回的信息
		parseRedisInfo(infoStr, result)
		// 将 used_memory 也设置到 memory 字段（用于兼容）
		if usedMemory, ok := result["used_memory"].(int64); ok {
			result["memory"] = usedMemory
		}
	}

	return result
}

// parseRedisInfo 解析Redis INFO命令返回的字符串
func parseRedisInfo(infoStr string, result map[string]any) {
	lines := strings.Split(infoStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "redis_version":
			result["version"] = value
		case "used_memory":
			if mem, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["used_memory"] = mem
			}
		case "used_memory_human":
			result["used_memory_human"] = value
		case "used_memory_peak":
			if mem, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["used_memory_peak"] = mem
			}
		case "used_memory_rss":
			if mem, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["used_memory_rss"] = mem
			}
		case "connected_clients":
			if clients, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["connected_clients"] = clients
			}
		case "total_commands_processed":
			if cmds, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["total_commands_processed"] = cmds
			}
		case "instantaneous_ops_per_sec":
			if ops, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["instantaneous_ops_per_sec"] = ops
			}
		case "keyspace_hits":
			if hits, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["keyspace_hits"] = hits
			}
		case "keyspace_misses":
			if misses, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["keyspace_misses"] = misses
			}
		case "expired_keys":
			if expired, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["expired_keys"] = expired
			}
		case "evicted_keys":
			if evicted, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["evicted_keys"] = evicted
			}
		case "uptime_in_seconds":
			if uptime, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["uptime"] = uptime
			}
		case "role":
			result["role"] = value
		case "connected_slaves":
			if slaves, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["connected_slaves"] = slaves
			}
		case "rdb_last_save_time":
			if saveTime, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["rdb_last_save_time"] = saveTime
			}
		case "aof_enabled":
			if enabled, err := strconv.ParseInt(value, 10, 64); err == nil {
				result["aof_enabled"] = enabled == 1
			}
		}
	}

	// 计算命中率
	if hits, ok := result["keyspace_hits"].(int64); ok {
		if misses, ok2 := result["keyspace_misses"].(int64); ok2 {
			total := hits + misses
			if total > 0 {
				hitRate := float64(hits) / float64(total) * 100
				result["keyspace_hit_rate"] = hitRate
			}
		}
	}
}

// getProcessesInfo 获取 MySQL、PostgreSQL、Redis 和当前应用进程的信息
func (r *MonitorController) getProcessesInfo(ctx http.Context) map[string]any {
	result := map[string]any{
		"mysql": map[string]any{
			"name":   "mysql",
			"pid":    0,
			"cpu":    0.0,
			"memory": 0,
			"status": "not_found",
			"rss":    0,
			"vms":    0,
		},
		"postgresql": map[string]any{
			"name":   "postgresql",
			"pid":    0,
			"cpu":    0.0,
			"memory": 0,
			"status": "not_found",
			"rss":    0,
			"vms":    0,
		},
		"redis": map[string]any{
			"name":   "redis",
			"pid":    0,
			"cpu":    0.0,
			"memory": 0,
			"status": "not_found",
			"rss":    0,
			"vms":    0,
		},
		"app": map[string]any{
			"name":   "app",
			"pid":    0,
			"cpu":    0.0,
			"memory": 0,
			"status": "not_found",
			"rss":    0,
			"vms":    0,
		},
	}

	// 使用 defer recover 确保即使进程查找出错也不影响整体功能
	defer func() {
		if r := recover(); r != nil {
			// 静默处理错误，返回默认值
		}
	}()

	// 获取数据库和Redis连接配置
	var driver string
	var connectionName string
	if facades.Orm() != nil {
		driver = strings.ToLower(appfacades.OrmQuery(ctx).Driver())
		connectionName = monitorDBConnectionName(ctx, driver)
	} else {
		driver = ""
		connectionName = ""
	}
	// 使用连接名获取配置，而不是驱动名
	dbHost := facades.Config().GetString(fmt.Sprintf("database.connections.%s.host", connectionName), "127.0.0.1")
	redisHost := facades.Config().GetString("database.redis.default.host", "")

	// PostgreSQL处理：检查是否为PostgreSQL数据库
	if driver == "postgresql" {
		// 无论本地还是远程，都先通过数据库连接获取统计信息
		postgresDBInfo := getPostgreSQLInfoFromDB(ctx)

		if isLocalHost(dbHost) {
			// 本地PostgreSQL，尝试查找进程获取 CPU、内存等信息
			postgresNames := []string{"postgres", "postmaster", "postgresql"}
			if runtime.GOOS == "windows" {
				postgresNames = []string{"postgres", "postgres.exe", "postmaster.exe"}
			}
			postgresPid := findProcessByName(ctx, postgresNames)
			if postgresPid > 0 {
				// 获取进程信息（CPU、内存等）
				processInfo := getProcessInfo(ctx, "postgresql", postgresPid)
				if processInfo != nil {
					// 合并进程信息和数据库统计信息
					for k, v := range postgresDBInfo {
						if _, exists := processInfo[k]; !exists {
							// 如果进程信息中没有这个字段，使用数据库信息
							processInfo[k] = v
						}
					}
					// 确保类型和状态正确
					processInfo["type"] = "local"
					if postgresDBInfo["status"] == "connected" {
						processInfo["status"] = "connected"
					}
					result["postgresql"] = processInfo
				} else {
					// 如果获取进程信息失败，使用数据库信息
					result["postgresql"] = postgresDBInfo
				}
			} else {
				// 找不到进程，使用数据库信息
				result["postgresql"] = postgresDBInfo
			}
		} else {
			// 远程PostgreSQL，直接使用数据库信息
			result["postgresql"] = postgresDBInfo
		}
	}

	// MySQL处理：检查是否为本地数据库
	if driver == "mysql" {
		// 无论本地还是远程，都先通过数据库连接获取统计信息（连接数、线程数等）
		mysqlDBInfo := getMySQLInfoFromDB(ctx)

		if isLocalHost(dbHost) {
			// 本地MySQL，尝试查找进程获取 CPU、内存等信息
			mysqlNames := []string{"mysqld", "mysql", "mariadb"}
			if runtime.GOOS == "windows" {
				mysqlNames = []string{"mysqld", "mysql", "mysqld-nt"}
			}
			mysqlPid := findProcessByName(ctx, mysqlNames)
			if mysqlPid > 0 {
				// 获取进程信息（CPU、内存等）
				processInfo := getProcessInfo(ctx, "mysql", mysqlPid)
				if processInfo != nil {
					// 合并进程信息和数据库统计信息
					// 进程信息覆盖基础字段，数据库信息提供统计字段
					for k, v := range mysqlDBInfo {
						if _, exists := processInfo[k]; !exists {
							// 如果进程信息中没有这个字段，使用数据库信息
							processInfo[k] = v
						}
					}
					// 确保类型和状态正确
					processInfo["type"] = "local"
					if mysqlDBInfo["status"] == "connected" {
						processInfo["status"] = "connected"
					}
					result["mysql"] = processInfo
				} else {
					// 如果获取进程信息失败，使用数据库信息
					result["mysql"] = mysqlDBInfo
				}
			} else {
				// 找不到进程，使用数据库信息
				result["mysql"] = mysqlDBInfo
			}
		} else {
			// 远程MySQL，直接使用数据库信息
			result["mysql"] = mysqlDBInfo
		}
	}

	// Redis处理：检查是否为本地Redis
	if redisHost == "" {
		// Redis配置为空，尝试查找本地进程
		redisNames := []string{"redis-server", "redis"}
		if runtime.GOOS == "windows" {
			redisNames = []string{"redis-server", "redis-server.exe", "redis"}
		}
		redisPid := findProcessByName(ctx, redisNames)
		if redisPid > 0 {
			redisInfo := getProcessInfo(ctx, "redis", redisPid)
			if redisInfo != nil {
				redisInfo["type"] = "local"
				result["redis"] = redisInfo
			}
		}
		// 如果找不到进程，保持默认的 not_found 状态
	} else if isLocalHost(redisHost) {
		// 本地Redis，尝试查找进程
		redisNames := []string{"redis-server", "redis"}
		if runtime.GOOS == "windows" {
			redisNames = []string{"redis-server", "redis-server.exe", "redis"}
		}
		redisPid := findProcessByName(ctx, redisNames)
		if redisPid > 0 {
			redisInfo := getProcessInfo(ctx, "redis", redisPid)
			if redisInfo != nil {
				redisInfo["type"] = "local"
				result["redis"] = redisInfo
			}
		} else {
			// 本地但找不到进程，尝试通过连接获取信息
			result["redis"] = getRedisInfoFromConnection(ctx)
		}
	} else {
		// 远程Redis，通过连接获取信息
		result["redis"] = getRedisInfoFromConnection(ctx)
	}

	// 获取当前应用进程信息（总是尝试获取，因为这是当前进程）
	currentPid := int32(os.Getpid())
	if currentPid > 0 {
		appInfo := getProcessInfo(ctx, "app", currentPid)
		if appInfo != nil {
			appInfo["type"] = "local"
			// 确保应用进程总是有状态信息
			if appInfo["status"] == "not_found" {
				appInfo["status"] = "running" // 当前进程应该总是运行中
			}
			result["app"] = appInfo
		} else {
			// 如果 getProcessInfo 返回 nil，使用默认值
			result["app"] = map[string]any{
				"name":   "app",
				"pid":    currentPid,
				"cpu":    0.0,
				"memory": 0,
				"status": "running",
				"type":   "local",
			}
		}
	} else {
		// 如果无法获取 PID，至少返回基本信息
		result["app"] = map[string]any{
			"name":   "app",
			"pid":    0,
			"cpu":    0.0,
			"memory": 0,
			"status": "unknown",
			"type":   "local",
		}
	}

	return result
}

// GetSystemInfo 获取系统监控信息
func (r *MonitorController) GetSystemInfo(ctx http.Context) http.Response {
	// CPU信息 - 使用0秒采样，避免阻塞（gopsutil会使用上次采样的差值）
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get CPU percent error", map[string]any{
			"error": err.Error(),
		}, "Get CPU percent error: %v", err)
		cpuPercent = []float64{0}
	}
	cpuInfo, err := cpu.Info()
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get CPU info error", map[string]any{
			"error": err.Error(),
		}, "Get CPU info error: %v", err)
		cpuInfo = []cpu.InfoStat{}
	}

	// 内存信息
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get memory info error", map[string]any{
			"error": err.Error(),
		}, "Get memory info error: %v", err)
		memInfo = &mem.VirtualMemoryStat{}
	}

	// 磁盘信息（根据操作系统选择路径）
	var diskPath string
	if runtime.GOOS == "windows" {
		// Windows 系统使用当前工作目录的驱动器
		wd, _ := os.Getwd()
		if len(wd) > 0 {
			diskPath = wd[:1] + ":\\"
		} else {
			diskPath = "C:\\"
		}
	} else {
		diskPath = "/"
	}
	diskInfo, err := disk.Usage(diskPath)
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get disk info error", map[string]any{
			"error": err.Error(),
			"path":  diskPath,
		}, "Get disk info error: %v", err)
		diskInfo = &disk.UsageStat{}
	}

	// 网络信息 - 获取所有网卡的详细信息
	netIO, err := net.IOCounters(true) // true 表示获取每个网卡的详细信息
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get network info error", map[string]any{
			"error": err.Error(),
		}, "Get network info error: %v", err)
		netIO = []net.IOCountersStat{}
	}

	// 汇总所有网卡的统计信息
	var totalBytesSent, totalBytesRecv, totalPacketsSent, totalPacketsRecv uint64
	var totalErrin, totalErrout, totalDropin, totalDropout uint64

	// 每个网卡的详细信息
	var interfaces []map[string]any
	for _, io := range netIO {
		// 跳过回环接口（通常以 lo 或 Loopback 开头）
		if io.Name == "lo" || io.Name == "Loopback" || io.Name == "lo0" {
			continue
		}

		totalBytesSent += io.BytesSent
		totalBytesRecv += io.BytesRecv
		totalPacketsSent += io.PacketsSent
		totalPacketsRecv += io.PacketsRecv
		totalErrin += io.Errin
		totalErrout += io.Errout
		totalDropin += io.Dropin
		totalDropout += io.Dropout

		interfaces = append(interfaces, map[string]any{
			"name":         io.Name,
			"bytes_sent":   io.BytesSent,
			"bytes_recv":   io.BytesRecv,
			"packets_sent": io.PacketsSent,
			"packets_recv": io.PacketsRecv,
			"errin":        io.Errin,
			"errout":       io.Errout,
			"dropin":       io.Dropin,
			"dropout":      io.Dropout,
		})
	}

	// 计算网络速度（Mbps）并获取峰值
	sentSpeed, recvSpeed, totalSpeed, peakSent, peakRecv, peakTotal := getNetworkSpeed(totalBytesSent, totalBytesRecv)

	// 汇总统计
	netStats := map[string]any{
		"bytes_sent":       totalBytesSent,
		"bytes_recv":       totalBytesRecv,
		"packets_sent":     totalPacketsSent,
		"packets_recv":     totalPacketsRecv,
		"errin":            totalErrin,
		"errout":           totalErrout,
		"dropin":           totalDropin,
		"dropout":          totalDropout,
		"interfaces":       interfaces, // 所有网卡的详细信息
		"speed_sent_mbps":  sentSpeed,  // 当前发送速度（Mbps）
		"speed_recv_mbps":  recvSpeed,  // 当前接收速度（Mbps）
		"speed_total_mbps": totalSpeed, // 当前总速度（Mbps）
		"peak_sent_mbps":   peakSent,   // 峰值发送速度（Mbps）
		"peak_recv_mbps":   peakRecv,   // 峰值接收速度（Mbps）
		"peak_total_mbps":  peakTotal,  // 峰值总速度（Mbps）
	}

	var cpuModel string
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	// 负载信息（仅Linux/Unix系统）
	var loadAvg map[string]any
	if runtime.GOOS != "windows" {
		avg, err := load.Avg()
		if err != nil {
			errorlog.RecordHTTP(ctx, "monitor", "Get load average error", map[string]any{
				"error": err.Error(),
			}, "Get load average error: %v", err)
			loadAvg = map[string]any{
				"load1":  0.0,
				"load5":  0.0,
				"load15": 0.0,
			}
		} else {
			// 计算负载百分比（相对于CPU核心数）
			cores := float64(len(cpuInfo))
			if cores == 0 {
				cores = 1
			}
			loadPercent1 := (avg.Load1 / cores) * 100
			loadPercent5 := (avg.Load5 / cores) * 100
			loadPercent15 := (avg.Load15 / cores) * 100

			loadAvg = map[string]any{
				"load1":          avg.Load1,
				"load5":          avg.Load5,
				"load15":         avg.Load15,
				"load1_percent":  loadPercent1,
				"load5_percent":  loadPercent5,
				"load15_percent": loadPercent15,
			}
		}
	} else {
		// Windows系统不支持负载
		loadAvg = map[string]any{
			"load1":          0.0,
			"load5":          0.0,
			"load15":         0.0,
			"load1_percent":  0.0,
			"load5_percent":  0.0,
			"load15_percent": 0.0,
		}
	}

	// 文件描述符信息（仅Linux/Unix系统，获取系统全局的）
	var fileDescriptors map[string]any
	if runtime.GOOS != "windows" {
		// 读取系统全局文件描述符信息 /proc/sys/fs/file-nr
		// 格式：已分配 已使用但未释放 最大数量
		used := uint64(0)
		max := uint64(0)

		if data, err := os.ReadFile("/proc/sys/fs/file-nr"); err == nil {
			// 清理数据：去除换行符和空白字符
			dataStr := strings.TrimSpace(string(data))
			// 解析文件内容：例如 "1024 512 65536"
			// 格式：已分配的文件描述符数 已分配但未使用的文件描述符数 系统最大文件描述符数
			var allocated, unused, tempMax uint64
			n, err := fmt.Sscanf(dataStr, "%d %d %d", &allocated, &unused, &tempMax)
			if err == nil && n == 3 {
				// 验证值的合理性：最大文件描述符数不应该超过 10^9 (1 billion)
				if tempMax > 0 && tempMax < 1000000000 {
					max = tempMax
				}
				// 已使用 = 已分配（第一个数字是已分配的文件描述符数，代表系统已使用的）
				if allocated > 0 && allocated < 1000000000 {
					used = allocated
				}
			}
			// 解析失败或值不合理时静默处理，后续会使用默认值
		}
		// 读取失败时静默处理，后续会尝试读取 file-max 或使用默认值

		// 如果无法读取file-nr中的max，尝试单独读取最大限制
		if max == 0 {
			if data, err := os.ReadFile("/proc/sys/fs/file-max"); err == nil {
				// 清理数据：去除换行符和空白字符
				dataStr := strings.TrimSpace(string(data))
				var tempMax uint64
				n, err := fmt.Sscanf(dataStr, "%d", &tempMax)
				if err == nil && n == 1 {
					// 验证值的合理性：最大文件描述符数不应该超过 10^9 (1 billion)
					// 正常的系统值通常在 65536 到几百万之间
					if tempMax > 0 && tempMax < 1000000000 {
						max = tempMax
					}
					// 值异常时静默处理，后续会使用默认值
				}
				// 解析失败或读取失败时静默处理，后续会使用默认值
			}
		}

		// 验证 max 值的合理性，如果异常则重置为0，后续会使用默认值
		if max > 1000000000 {
			max = 0
		}

		// 如果还是无法获取或值异常，使用默认值
		if max == 0 {
			max = 65536 // Linux常见默认值
		}

		// 计算剩余文件描述符，确保不会溢出
		free := uint64(0)
		if max > used {
			free = max - used
		}

		percent := float64(0)
		if max > 0 {
			percent = (float64(used) / float64(max)) * 100
		}

		fileDescriptors = map[string]any{
			"max":     max,
			"used":    used,
			"free":    free,
			"percent": percent,
		}
	} else {
		// Windows系统不支持文件描述符限制
		fileDescriptors = map[string]any{
			"max":     0,
			"used":    0,
			"free":    0,
			"percent": 0.0,
		}
	}

	// 确保 cpuPercent 不为空
	if len(cpuPercent) == 0 {
		cpuPercent = []float64{0}
	}

	// 获取磁盘IO统计（仅Linux/Unix系统）
	var diskIO map[string]any
	if runtime.GOOS != "windows" {
		ioCounters, err := disk.IOCounters()
		if err == nil && len(ioCounters) > 0 {
			// 汇总所有磁盘的IO统计
			var totalReadBytes, totalWriteBytes, totalReadCount, totalWriteCount uint64
			var diskIOCounters []map[string]any
			for name, io := range ioCounters {
				totalReadBytes += io.ReadBytes
				totalWriteBytes += io.WriteBytes
				totalReadCount += io.ReadCount
				totalWriteCount += io.WriteCount
				diskIOCounters = append(diskIOCounters, map[string]any{
					"name":        name,
					"read_bytes":  io.ReadBytes,
					"write_bytes": io.WriteBytes,
					"read_count":  io.ReadCount,
					"write_count": io.WriteCount,
					"read_time":   io.ReadTime,
					"write_time":  io.WriteTime,
				})
			}
			diskIO = map[string]any{
				"total_read_bytes":  totalReadBytes,
				"total_write_bytes": totalWriteBytes,
				"total_read_count":  totalReadCount,
				"total_write_count": totalWriteCount,
				"disks":             diskIOCounters,
			}
		} else {
			diskIO = map[string]any{
				"total_read_bytes":  0,
				"total_write_bytes": 0,
				"total_read_count":  0,
				"total_write_count": 0,
				"disks":             []map[string]any{},
			}
		}
	} else {
		// Windows系统不支持磁盘IO统计
		diskIO = map[string]any{
			"total_read_bytes":  0,
			"total_write_bytes": 0,
			"total_read_count":  0,
			"total_write_count": 0,
			"disks":             []map[string]any{},
		}
	}

	// 获取TCP连接统计（带超时和采样优化，避免高并发下性能损耗）
	var tcpConnections map[string]any
	tcpConnections = r.getTCPConnectionsWithTimeout(ctx, 2*time.Second)

	// 获取所有磁盘分区信息（带超时和并发控制，避免网络磁盘阻塞）
	var diskPartitions []map[string]any
	diskPartitions = r.getDiskPartitionsWithTimeout(ctx, 3*time.Second)

	// 生成系统告警提示（根据当前语言动态生成）
	alerts := r.generateAlerts(ctx, memInfo, diskInfo, cpuPercent, fileDescriptors)
	healthStatus := "ok"
	if len(alerts) > 0 {
		healthStatus = "warning"
	}
	wsAdmins, wsConnections := wsnotifications.Hub().Stats()

	physicalCores := 0
	for _, info := range cpuInfo {
		if info.Cores > 0 {
			physicalCores += int(info.Cores)
		}
	}

	return response.Success(ctx, http.Json{
		"os": runtime.GOOS, // 操作系统类型
		"cpu": map[string]any{
			"percent":        cpuPercent[0],
			"model":          cpuModel,
			"cores":          runtime.NumCPU(), // 逻辑核心数（与 runtime.num_cpu 一致）
			"physical_cores": physicalCores,
		},
		"memory": map[string]any{
			"total":     memInfo.Total,
			"available": memInfo.Available,
			"used":      memInfo.Used,
			"free":      memInfo.Free,
			"percent":   memInfo.UsedPercent,
			"cached":    memInfo.Cached,
			"buffers":   memInfo.Buffers,
		},
		"disk": map[string]any{
			"total":   diskInfo.Total,
			"free":    diskInfo.Free,
			"used":    diskInfo.Used,
			"percent": diskInfo.UsedPercent,
			"fstype":  diskInfo.Fstype,
			"path":    diskInfo.Path,
		},
		"disk_partitions":  diskPartitions,
		"disk_io":          diskIO,
		"net":              netStats,
		"tcp_connections":  tcpConnections,
		"load":             loadAvg,
		"file_descriptors": fileDescriptors,
		"runtime": map[string]any{
			"goroutines": runtime.NumGoroutine(),
			"num_cpu":    runtime.NumCPU(),
			"gomaxprocs": runtime.GOMAXPROCS(0),
			"total_processes": func() int {
				processes, err := process.Processes()
				if err != nil {
					errorlog.RecordHTTP(ctx, "monitor", "Get processes error", map[string]any{
						"error": err.Error(),
					}, "Get processes error: %v", err)
					return 0
				}
				return len(processes)
			}(),
			"memory": func() map[string]any {
				memStats := runtime.MemStats{}
				runtime.ReadMemStats(&memStats)
				return map[string]any{
					"alloc":          memStats.Alloc,
					"total_alloc":    memStats.TotalAlloc,
					"sys":            memStats.Sys,
					"lookups":        memStats.Lookups,
					"mallocs":        memStats.Mallocs,
					"frees":          memStats.Frees,
					"heap_alloc":     memStats.HeapAlloc,
					"heap_sys":       memStats.HeapSys,
					"heap_idle":      memStats.HeapIdle,
					"heap_inuse":     memStats.HeapInuse,
					"heap_objects":   memStats.HeapObjects,
					"stack_inuse":    memStats.StackInuse,
					"stack_sys":      memStats.StackSys,
					"num_gc":         memStats.NumGC,
					"pause_total_ns": memStats.PauseTotalNs,
					"last_gc":        memStats.LastGC,
				}
			}(),
		},
		"app": map[string]any{
			"env":              facades.Config().GetString("app.env", "production"),
			"debug":            facades.Config().GetBool("app.debug", false),
			"timezone":         facades.Config().GetString("app.timezone", "UTC"),
			"queue_connection": facades.Config().GetString("queue.default", "sync"),
			"cache_store":      facades.Config().GetString("cache.default", "file"),
		},
		"websocket": map[string]any{
			"online_admins": wsAdmins,
			"connections":   wsConnections,
		},
		"health": map[string]any{
			"status":      healthStatus,
			"alert_count": len(alerts),
		},
		"system": map[string]any{
			"hostname": func() string {
				hostname, err := os.Hostname()
				if err != nil {
					errorlog.RecordHTTP(ctx, "monitor", "Get hostname error", map[string]any{
						"error": err.Error(),
					}, "Get hostname error: %v", err)
					return "unknown"
				}
				return hostname
			}(),
			"arch":       runtime.GOARCH,
			"os":         runtime.GOOS,
			"go_version": runtime.Version(),
		},
		"processes":   r.getProcessesInfo(ctx),
		"process_top": r.getProcessTopRankings(ctx, memInfo.Total),
		"alerts":      alerts,
	})
}

// StreamSystemInfo SSE 实时推送系统监控信息
// 每 2-3 秒推送一次系统监控数据
func (r *MonitorController) StreamSystemInfo(ctx http.Context) http.Response {
	// 获取推送间隔（秒），默认 2 秒
	interval := 2
	if intervalStr := ctx.Request().Query("interval", ""); intervalStr != "" {
		if parsed, err := time.ParseDuration(intervalStr + "s"); err == nil {
			interval = min(max(int(parsed.Seconds()), 1), 10)
		}
	}

	// 须使用 Response().Stream()，直接写 Writer() 会被 Goravel/Gin 缓冲，客户端收不到 SSE 数据
	return ctx.Response().
		Header("Content-Type", "text/event-stream").
		Header("Cache-Control", "no-cache").
		Header("Connection", "keep-alive").
		Header("X-Accel-Buffering", "no").
		Stream(nethttp.StatusOK, func(writer http.StreamWriter) error {
			initMsg := map[string]any{
				"type":        "connected",
				"message_key": "monitor_sse_connected",
				"interval":    interval,
			}
			initData, _ := json.Marshal(initMsg)
			if _, err := writer.WriteString(fmt.Sprintf("data: %s\n\n", string(initData))); err != nil {
				return err
			}
			if err := writer.Flush(); err != nil {
				return err
			}

			clientGone := ctx.Request().Origin().Context().Done()
			appDone := facades.App().Context().Done()

			// 立即推送首帧：time.NewTicker 首次在整段 interval 后才触发，否则首屏长时间无业务数据
			select {
			case <-clientGone:
				return nil
			case <-appDone:
				return nil
			default:
				if !r.writeMonitorSystemInfoSSE(ctx, writer) {
					return nil
				}
			}

			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-clientGone:
					return nil
				case <-appDone:
					return nil
				case <-ticker.C:
					if !r.writeMonitorSystemInfoSSE(ctx, writer) {
						return nil
					}
				}
			}
		})
}

// generateAlerts 生成系统告警提示（返回原始数据，由前端根据当前语言翻译）
// 返回的告警数据包含：type, level, metric, value, message_key
// 前端可以根据 message_key 和 value 自行翻译显示
func (r *MonitorController) generateAlerts(ctx http.Context, memInfo *mem.VirtualMemoryStat, diskInfo *disk.UsageStat, cpuPercent []float64, fileDescriptors map[string]any) []map[string]any {
	alerts := []map[string]any{}
	if memInfo.UsedPercent > 90 {
		alerts = append(alerts, map[string]any{
			"type":        "warning",
			"level":       "high",
			"metric":      "memory",
			"value":       memInfo.UsedPercent,
			"message_key": "monitor_memory_usage_high",
		})
	} else if memInfo.UsedPercent > 80 {
		alerts = append(alerts, map[string]any{
			"type":        "warning",
			"level":       "medium",
			"metric":      "memory",
			"value":       memInfo.UsedPercent,
			"message_key": "monitor_memory_usage_medium",
		})
	}
	if diskInfo.UsedPercent > 90 {
		alerts = append(alerts, map[string]any{
			"type":        "warning",
			"level":       "high",
			"metric":      "disk",
			"value":       diskInfo.UsedPercent,
			"message_key": "monitor_disk_usage_high",
		})
	} else if diskInfo.UsedPercent > 80 {
		alerts = append(alerts, map[string]any{
			"type":        "warning",
			"level":       "medium",
			"metric":      "disk",
			"value":       diskInfo.UsedPercent,
			"message_key": "monitor_disk_usage_medium",
		})
	}
	if len(cpuPercent) > 0 && cpuPercent[0] > 90 {
		alerts = append(alerts, map[string]any{
			"type":        "warning",
			"level":       "high",
			"metric":      "cpu",
			"value":       cpuPercent[0],
			"message_key": "monitor_cpu_usage_high",
		})
	} else if len(cpuPercent) > 0 && cpuPercent[0] > 80 {
		alerts = append(alerts, map[string]any{
			"type":        "warning",
			"level":       "medium",
			"metric":      "cpu",
			"value":       cpuPercent[0],
			"message_key": "monitor_cpu_usage_medium",
		})
	}
	if runtime.GOOS != "windows" {
		if percent, ok := fileDescriptors["percent"].(float64); ok && percent > 90 {
			alerts = append(alerts, map[string]any{
				"type":        "warning",
				"level":       "high",
				"metric":      "file_descriptors",
				"value":       percent,
				"message_key": "monitor_file_descriptors_usage_high",
			})
		}
	}
	return alerts
}

// collectSystemInfo 收集系统监控信息（从 GetSystemInfo 提取的逻辑）
func (r *MonitorController) collectSystemInfo(ctx http.Context) map[string]any {
	// 检查缓存（仅在SSE流中使用缓存，减少系统调用）
	monitorCacheLock.RLock()
	cacheValid := monitorCache != nil && time.Since(monitorCacheTime) < cacheDuration
	if cacheValid {
		cached := monitorCache
		monitorCacheLock.RUnlock()
		// 从缓存中提取数据并动态生成告警消息
		result := make(map[string]any)
		for k, v := range cached {
			result[k] = v
		}
		// 从缓存的数据中提取监控指标，动态生成告警消息
		if memInfoMap, ok := result["memory"].(map[string]any); ok {
			if diskInfoMap, ok2 := result["disk"].(map[string]any); ok2 {
				if cpuInfoMap, ok3 := result["cpu"].(map[string]any); ok3 {
					if fileDesc, ok4 := result["file_descriptors"].(map[string]any); ok4 {
						// 构造临时对象用于生成告警
						memInfo := &mem.VirtualMemoryStat{
							UsedPercent: getFloat64(memInfoMap["percent"]),
						}
						diskInfo := &disk.UsageStat{
							UsedPercent: getFloat64(diskInfoMap["percent"]),
						}
						var cpuPercent []float64
						if cpuPercentVal, ok5 := cpuInfoMap["percent"].(float64); ok5 {
							cpuPercent = []float64{cpuPercentVal}
						}
						alerts := r.generateAlerts(ctx, memInfo, diskInfo, cpuPercent, fileDesc)
						result["alerts"] = alerts
						result["health"] = map[string]any{
							"status":      map[bool]string{true: "warning", false: "ok"}[len(alerts) > 0],
							"alert_count": len(alerts),
						}
					}
				}
			}
		}
		return result
	}
	monitorCacheLock.RUnlock()

	// 使用 singleflight 确保同一时间只有一个 goroutine 重建缓存
	// 其他 goroutine 等待结果，避免重复计算和锁竞争
	result, _, _ := monitorCacheGroup.Do("collectSystemInfo", func() (any, error) {
		// 再次检查缓存（double-check），可能在等待期间其他 goroutine 已更新缓存
		monitorCacheLock.RLock()
		if monitorCache != nil && time.Since(monitorCacheTime) < cacheDuration {
			cached := monitorCache
			monitorCacheLock.RUnlock()
			return cached, nil
		}
		monitorCacheLock.RUnlock()

		// 执行实际的数据收集
		return r.doCollectSystemInfo(ctx), nil
	})

	if data, ok := result.(map[string]any); ok {
		// 从缓存的数据中提取监控指标，动态生成告警消息
		if memInfoMap, ok1 := data["memory"].(map[string]any); ok1 {
			if diskInfoMap, ok2 := data["disk"].(map[string]any); ok2 {
				if cpuInfoMap, ok3 := data["cpu"].(map[string]any); ok3 {
					if fileDesc, ok4 := data["file_descriptors"].(map[string]any); ok4 {
						// 构造临时对象用于生成告警
						memInfo := &mem.VirtualMemoryStat{
							UsedPercent: getFloat64(memInfoMap["percent"]),
						}
						diskInfo := &disk.UsageStat{
							UsedPercent: getFloat64(diskInfoMap["percent"]),
						}
						var cpuPercent []float64
						if cpuPercentVal, ok5 := cpuInfoMap["percent"].(float64); ok5 {
							cpuPercent = []float64{cpuPercentVal}
						}
						alerts := r.generateAlerts(ctx, memInfo, diskInfo, cpuPercent, fileDesc)
						data["alerts"] = alerts
						data["health"] = map[string]any{
							"status":      map[bool]string{true: "warning", false: "ok"}[len(alerts) > 0],
							"alert_count": len(alerts),
						}
					}
				}
			}
		}
		return data
	}
	// 如果类型转换失败，执行一次实际收集（兜底）
	return r.doCollectSystemInfo(ctx)
}

// getFloat64 从 any 安全地获取 float64 值
func getFloat64(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	if i, ok := v.(int); ok {
		return float64(i)
	}
	if i, ok := v.(int64); ok {
		return float64(i)
	}
	return 0
}

// doCollectSystemInfo 实际执行系统信息收集（从 collectSystemInfo 提取）
func (r *MonitorController) doCollectSystemInfo(ctx http.Context) map[string]any {

	// CPU信息 - 使用0秒采样，避免阻塞（gopsutil会使用上次采样的差值）
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get CPU percent error", map[string]any{
			"error": err.Error(),
		}, "Get CPU percent error: %v", err)
		cpuPercent = []float64{0}
	}
	cpuInfo, err := cpu.Info()
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get CPU info error", map[string]any{
			"error": err.Error(),
		}, "Get CPU info error: %v", err)
		cpuInfo = []cpu.InfoStat{}
	}

	// 内存信息
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get memory info error", map[string]any{
			"error": err.Error(),
		}, "Get memory info error: %v", err)
		memInfo = &mem.VirtualMemoryStat{}
	}

	// 磁盘信息
	var diskPath string
	if runtime.GOOS == "windows" {
		wd, _ := os.Getwd()
		if len(wd) > 0 {
			diskPath = wd[:1] + ":\\"
		} else {
			diskPath = "C:\\"
		}
	} else {
		diskPath = "/"
	}
	diskInfo, err := disk.Usage(diskPath)
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get disk info error", map[string]any{
			"error": err.Error(),
			"path":  diskPath,
		}, "Get disk info error: %v", err)
		diskInfo = &disk.UsageStat{}
	}

	// 网络信息
	netIO, err := net.IOCounters(true)
	if err != nil {
		errorlog.RecordHTTP(ctx, "monitor", "Get network info error", map[string]any{
			"error": err.Error(),
		}, "Get network info error: %v", err)
		netIO = []net.IOCountersStat{}
	}

	// 汇总网络统计
	var totalBytesSent, totalBytesRecv, totalPacketsSent, totalPacketsRecv uint64
	var totalErrin, totalErrout, totalDropin, totalDropout uint64
	var interfaces []map[string]any

	for _, io := range netIO {
		if io.Name == "lo" || io.Name == "Loopback" || io.Name == "lo0" {
			continue
		}
		totalBytesSent += io.BytesSent
		totalBytesRecv += io.BytesRecv
		totalPacketsSent += io.PacketsSent
		totalPacketsRecv += io.PacketsRecv
		totalErrin += io.Errin
		totalErrout += io.Errout
		totalDropin += io.Dropin
		totalDropout += io.Dropout

		interfaces = append(interfaces, map[string]any{
			"name":         io.Name,
			"bytes_sent":   io.BytesSent,
			"bytes_recv":   io.BytesRecv,
			"packets_sent": io.PacketsSent,
			"packets_recv": io.PacketsRecv,
			"errin":        io.Errin,
			"errout":       io.Errout,
			"dropin":       io.Dropin,
			"dropout":      io.Dropout,
		})
	}

	// 计算网络速度（Mbps）并获取峰值
	sentSpeed, recvSpeed, totalSpeed, peakSent, peakRecv, peakTotal := getNetworkSpeed(totalBytesSent, totalBytesRecv)

	netStats := map[string]any{
		"bytes_sent":       totalBytesSent,
		"bytes_recv":       totalBytesRecv,
		"packets_sent":     totalPacketsSent,
		"packets_recv":     totalPacketsRecv,
		"errin":            totalErrin,
		"errout":           totalErrout,
		"dropin":           totalDropin,
		"dropout":          totalDropout,
		"interfaces":       interfaces,
		"speed_sent_mbps":  sentSpeed,  // 当前发送速度（Mbps）
		"speed_recv_mbps":  recvSpeed,  // 当前接收速度（Mbps）
		"speed_total_mbps": totalSpeed, // 当前总速度（Mbps）
		"peak_sent_mbps":   peakSent,   // 峰值发送速度（Mbps）
		"peak_recv_mbps":   peakRecv,   // 峰值接收速度（Mbps）
		"peak_total_mbps":  peakTotal,  // 峰值总速度（Mbps）
	}

	var cpuModel string
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	// 负载信息
	var loadAvg map[string]any
	if runtime.GOOS != "windows" {
		avg, err := load.Avg()
		if err != nil {
			loadAvg = map[string]any{
				"load1":  0.0,
				"load5":  0.0,
				"load15": 0.0,
			}
		} else {
			cores := float64(len(cpuInfo))
			if cores == 0 {
				cores = 1
			}
			loadPercent1 := (avg.Load1 / cores) * 100
			loadPercent5 := (avg.Load5 / cores) * 100
			loadPercent15 := (avg.Load15 / cores) * 100

			loadAvg = map[string]any{
				"load1":          avg.Load1,
				"load5":          avg.Load5,
				"load15":         avg.Load15,
				"load1_percent":  loadPercent1,
				"load5_percent":  loadPercent5,
				"load15_percent": loadPercent15,
			}
		}
	} else {
		loadAvg = map[string]any{
			"load1":          0.0,
			"load5":          0.0,
			"load15":         0.0,
			"load1_percent":  0.0,
			"load5_percent":  0.0,
			"load15_percent": 0.0,
		}
	}

	// 文件描述符信息（简化版，避免重复代码）
	var fileDescriptors map[string]any
	if runtime.GOOS != "windows" {
		used := uint64(0)
		max := uint64(0)

		if data, err := os.ReadFile("/proc/sys/fs/file-nr"); err == nil {
			dataStr := strings.TrimSpace(string(data))
			var allocated, unused, tempMax uint64
			if n, err := fmt.Sscanf(dataStr, "%d %d %d", &allocated, &unused, &tempMax); err == nil && n == 3 {
				// 验证值的合理性
				if tempMax > 0 && tempMax < 1000000000 {
					max = tempMax
				}
				if allocated > 0 && allocated < 1000000000 {
					used = allocated
				}
			}
			// 解析失败或值不合理时静默处理，后续会使用默认值
		}

		if max == 0 {
			if data, err := os.ReadFile("/proc/sys/fs/file-max"); err == nil {
				dataStr := strings.TrimSpace(string(data))
				var tempMax uint64
				if n, err := fmt.Sscanf(dataStr, "%d", &tempMax); err == nil && n == 1 {
					// 验证值的合理性
					if tempMax > 0 && tempMax < 1000000000 {
						max = tempMax
					}
					// 值异常时静默处理，后续会使用默认值
				}
			}
		}

		if max == 0 {
			max = 65536
		}

		free := uint64(0)
		if max > used {
			free = max - used
		}

		percent := float64(0)
		if max > 0 {
			percent = (float64(used) / float64(max)) * 100
		}

		fileDescriptors = map[string]any{
			"max":     max,
			"used":    used,
			"free":    free,
			"percent": percent,
		}
	} else {
		fileDescriptors = map[string]any{
			"max":     0,
			"used":    0,
			"free":    0,
			"percent": 0.0,
		}
	}

	// 确保 cpuPercent 不为空
	if len(cpuPercent) == 0 {
		cpuPercent = []float64{0}
	}

	// 获取磁盘IO统计（仅Linux/Unix系统）
	var diskIO map[string]any
	if runtime.GOOS != "windows" {
		ioCounters, err := disk.IOCounters()
		if err == nil && len(ioCounters) > 0 {
			// 汇总所有磁盘的IO统计
			var totalReadBytes, totalWriteBytes, totalReadCount, totalWriteCount uint64
			var diskIOCounters []map[string]any
			for name, io := range ioCounters {
				totalReadBytes += io.ReadBytes
				totalWriteBytes += io.WriteBytes
				totalReadCount += io.ReadCount
				totalWriteCount += io.WriteCount
				diskIOCounters = append(diskIOCounters, map[string]any{
					"name":        name,
					"read_bytes":  io.ReadBytes,
					"write_bytes": io.WriteBytes,
					"read_count":  io.ReadCount,
					"write_count": io.WriteCount,
					"read_time":   io.ReadTime,
					"write_time":  io.WriteTime,
				})
			}
			diskIO = map[string]any{
				"total_read_bytes":  totalReadBytes,
				"total_write_bytes": totalWriteBytes,
				"total_read_count":  totalReadCount,
				"total_write_count": totalWriteCount,
				"disks":             diskIOCounters,
			}
		} else {
			diskIO = map[string]any{
				"total_read_bytes":  0,
				"total_write_bytes": 0,
				"total_read_count":  0,
				"total_write_count": 0,
				"disks":             []map[string]any{},
			}
		}
	} else {
		// Windows系统不支持磁盘IO统计
		diskIO = map[string]any{
			"total_read_bytes":  0,
			"total_write_bytes": 0,
			"total_read_count":  0,
			"total_write_count": 0,
			"disks":             []map[string]any{},
		}
	}

	// 获取TCP连接统计（带超时和采样优化，避免高并发下性能损耗）
	var tcpConnections map[string]any
	tcpConnections = r.getTCPConnectionsWithTimeout(ctx, 2*time.Second)

	// 获取所有磁盘分区信息（带超时和并发控制，避免网络磁盘阻塞）
	var diskPartitions []map[string]any
	diskPartitions = r.getDiskPartitionsWithTimeout(ctx, 3*time.Second)

	// 注意：告警消息不在 doCollectSystemInfo 中生成，而是在 collectSystemInfo 返回时根据当前语言动态生成

	physicalCores := 0
	for _, info := range cpuInfo {
		if info.Cores > 0 {
			physicalCores += int(info.Cores)
		}
	}
	wsAdmins, wsConnections := wsnotifications.Hub().Stats()

	result := map[string]any{
		"os": runtime.GOOS,
		"cpu": map[string]any{
			"percent":        cpuPercent[0],
			"model":          cpuModel,
			"cores":          runtime.NumCPU(), // 逻辑核心数（与 runtime.num_cpu 一致）
			"physical_cores": physicalCores,
		},
		"memory": map[string]any{
			"total":     memInfo.Total,
			"available": memInfo.Available,
			"used":      memInfo.Used,
			"free":      memInfo.Free,
			"percent":   memInfo.UsedPercent,
			"cached":    memInfo.Cached,
			"buffers":   memInfo.Buffers,
		},
		"disk": map[string]any{
			"total":   diskInfo.Total,
			"free":    diskInfo.Free,
			"used":    diskInfo.Used,
			"percent": diskInfo.UsedPercent,
			"fstype":  diskInfo.Fstype,
			"path":    diskInfo.Path,
		},
		"disk_partitions":  diskPartitions,
		"disk_io":          diskIO,
		"net":              netStats,
		"tcp_connections":  tcpConnections,
		"load":             loadAvg,
		"file_descriptors": fileDescriptors,
		"runtime": func() map[string]any {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			return map[string]any{
				"goroutines": runtime.NumGoroutine(),
				"num_cpu":    runtime.NumCPU(),
				"gomaxprocs": runtime.GOMAXPROCS(0), // 0表示获取当前值，不修改
				"total_processes": func() int {
					processes, err := process.Processes()
					if err != nil {
						return 0
					}
					return len(processes)
				}(),
				"memory": map[string]any{
					"alloc":          memStats.Alloc,        // 当前分配的内存
					"total_alloc":    memStats.TotalAlloc,   // 累计分配的内存
					"sys":            memStats.Sys,          // 系统内存
					"lookups":        memStats.Lookups,      // 指针查找次数
					"mallocs":        memStats.Mallocs,      // 分配次数
					"frees":          memStats.Frees,        // 释放次数
					"heap_alloc":     memStats.HeapAlloc,    // 堆内存分配
					"heap_sys":       memStats.HeapSys,      // 堆内存系统
					"heap_idle":      memStats.HeapIdle,     // 堆内存空闲
					"heap_inuse":     memStats.HeapInuse,    // 堆内存使用
					"heap_objects":   memStats.HeapObjects,  // 堆对象数
					"stack_inuse":    memStats.StackInuse,   // 栈内存使用
					"stack_sys":      memStats.StackSys,     // 栈内存系统
					"num_gc":         memStats.NumGC,        // GC次数
					"pause_total_ns": memStats.PauseTotalNs, // GC总暂停时间（纳秒）
					"last_gc":        memStats.LastGC,       // 上次GC时间
				},
			}
		}(),
		"app": map[string]any{
			"env":              facades.Config().GetString("app.env", "production"),
			"debug":            facades.Config().GetBool("app.debug", false),
			"timezone":         facades.Config().GetString("app.timezone", "UTC"),
			"queue_connection": facades.Config().GetString("queue.default", "sync"),
			"cache_store":      facades.Config().GetString("cache.default", "file"),
		},
		"websocket": map[string]any{
			"online_admins": wsAdmins,
			"connections":   wsConnections,
		},
		"system": map[string]any{
			"hostname": func() string {
				hostname, err := os.Hostname()
				if err != nil {
					return "unknown"
				}
				return hostname
			}(),
			"arch":       runtime.GOARCH,
			"os":         runtime.GOOS,
			"go_version": runtime.Version(),
		},
		"processes":   r.getProcessesInfo(ctx),
		"process_top": r.getProcessTopRankings(ctx, memInfo.Total),
		// 注意：alerts 不包含在缓存中，会在返回时根据当前语言动态生成
	}

	// 更新缓存（仅在collectSystemInfo中缓存，用于SSE流）
	// 注意：缓存中不包含 alerts，因为告警消息需要根据当前语言动态生成
	monitorCacheLock.Lock()
	monitorCache = result
	monitorCacheTime = time.Now()
	monitorCacheLock.Unlock()

	// 在返回前，根据当前语言动态生成告警消息
	alerts := r.generateAlerts(ctx, memInfo, diskInfo, cpuPercent, fileDescriptors)
	result["alerts"] = alerts
	result["health"] = map[string]any{
		"status":      map[bool]string{true: "warning", false: "ok"}[len(alerts) > 0],
		"alert_count": len(alerts),
	}

	return result
}

// getTCPConnectionsWithTimeout 获取 TCP 连接统计（带超时和采样优化）
// 优化策略：
//   - 添加超时控制（默认 2 秒）
//   - 连接数超过阈值时使用采样策略（每 N 个连接采样 1 个）
//   - 限制最大处理数量（10000）
func (r *MonitorController) getTCPConnectionsWithTimeout(ctx http.Context, timeout time.Duration) map[string]any {
	// 创建带超时的 context（使用 Background，因为这是异步操作）
	connCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 在 goroutine 中执行，支持超时取消
	connectionsChan := make(chan []net.ConnectionStat, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in net.Connections: %v", r)
			}
		}()
		connections, err := net.Connections("tcp")
		if err != nil {
			errChan <- err
			return
		}
		connectionsChan <- connections
	}()

	// 等待结果或超时
	select {
	case connections := <-connectionsChan:
		return r.processTCPConnections(connections)
	case err := <-errChan:
		errorlog.RecordHTTP(ctx, "monitor", "Get TCP connections error", map[string]any{
			"error": err.Error(),
		}, "Get TCP connections error: %v", err)
		return r.getEmptyTCPConnections()
	case <-connCtx.Done():
		// 超时，返回空数据
		errorlog.RecordHTTP(ctx, "monitor", "Get TCP connections timeout", map[string]any{
			"timeout": timeout.String(),
		}, "Get TCP connections timeout after %v", timeout)
		return r.getEmptyTCPConnections()
	}
}

// processTCPConnections 处理 TCP 连接数据（采样优化）
func (r *MonitorController) processTCPConnections(connections []net.ConnectionStat) map[string]any {
	var established, listen, timeWait, closeWait int
	var listeningPorts []int
	portMap := make(map[int]bool)

	maxConnections := 10000
	totalConnections := len(connections)

	// 如果连接数超过阈值，使用采样策略
	sampleRate := 1
	if totalConnections > maxConnections {
		// 采样率：每 N 个连接采样 1 个，确保处理数量不超过 maxConnections
		sampleRate = totalConnections / maxConnections
		if sampleRate == 0 {
			sampleRate = 1
		}
	}

	processed := 0
	for i, conn := range connections {
		// 采样：只处理满足采样条件的连接
		if i%sampleRate != 0 {
			continue
		}

		if processed >= maxConnections {
			break
		}
		processed++

		switch conn.Status {
		case "ESTABLISHED":
			established++
		case "LISTEN":
			listen++
			port := int(conn.Laddr.Port)
			if port > 0 && !portMap[port] {
				listeningPorts = append(listeningPorts, port)
				portMap[port] = true
			}
		case "TIME_WAIT":
			timeWait++
		case "CLOSE_WAIT":
			closeWait++
		}
	}

	// 如果使用了采样，需要按比例估算总数
	if sampleRate > 1 {
		established = established * sampleRate
		listen = listen * sampleRate
		timeWait = timeWait * sampleRate
		closeWait = closeWait * sampleRate
	}

	return map[string]any{
		"total":           totalConnections,
		"established":     established,
		"listen":          listen,
		"time_wait":       timeWait,
		"close_wait":      closeWait,
		"listening_ports": listeningPorts,
		"sampled":         sampleRate > 1, // 标记是否使用了采样
		"sample_rate":     sampleRate,
	}
}

// getEmptyTCPConnections 返回空的 TCP 连接统计
func (r *MonitorController) getEmptyTCPConnections() map[string]any {
	return map[string]any{
		"total":           0,
		"established":     0,
		"listen":          0,
		"time_wait":       0,
		"close_wait":      0,
		"listening_ports": []int{},
		"sampled":         false,
		"sample_rate":     1,
	}
}

// getDiskPartitionsWithTimeout 获取磁盘分区信息（带超时和并发控制）
// 优化策略：
//   - 添加超时控制（默认 3 秒）
//   - 并发获取分区信息（限制并发数）
//   - 单个分区超时保护（1 秒）
func (r *MonitorController) getDiskPartitionsWithTimeout(ctx http.Context, timeout time.Duration) []map[string]any {
	// 创建带超时的 context（使用 Background，因为这是异步操作）
	partCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 在 goroutine 中获取分区列表
	partitionsChan := make(chan []disk.PartitionStat, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in disk.Partitions: %v", r)
			}
		}()
		partitions, err := disk.Partitions(false)
		if err != nil {
			errChan <- err
			return
		}
		partitionsChan <- partitions
	}()

	var partitions []disk.PartitionStat
	select {
	case partitions = <-partitionsChan:
		// 成功获取分区列表
	case err := <-errChan:
		errorlog.RecordHTTP(ctx, "monitor", "Get disk partitions error", map[string]any{
			"error": err.Error(),
		}, "Get disk partitions error: %v", err)
		return []map[string]any{}
	case <-partCtx.Done():
		// 超时
		errorlog.RecordHTTP(ctx, "monitor", "Get disk partitions timeout", map[string]any{
			"timeout": timeout.String(),
		}, "Get disk partitions timeout after %v", timeout)
		return []map[string]any{}
	}

	// 限制最多处理的分区数量
	maxPartitions := 20
	if len(partitions) > maxPartitions {
		partitions = partitions[:maxPartitions]
	}

	// 并发获取分区使用情况（限制并发数）
	maxConcurrency := 5 // 最多 5 个并发
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var diskPartitions []map[string]any

	// 为每个分区创建带超时的 context
	partitionTimeout := 1 * time.Second
	if partitionTimeout > timeout {
		partitionTimeout = timeout / 2 // 确保不超过总超时时间
	}

	for _, part := range partitions {
		wg.Add(1)
		go func(partition disk.PartitionStat) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 创建单个分区的超时 context
			partitionCtx, partitionCancel := context.WithTimeout(partCtx, partitionTimeout)
			defer partitionCancel()

			// 在 goroutine 中获取分区使用情况
			usageChan := make(chan *disk.UsageStat, 1)
			errChan := make(chan error, 1)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						errChan <- fmt.Errorf("panic in disk.Usage: %v", r)
					}
				}()
				usage, err := disk.Usage(partition.Mountpoint)
				if err != nil {
					errChan <- err
					return
				}
				usageChan <- usage
			}()

			// 等待结果或超时
			select {
			case usage := <-usageChan:
				mu.Lock()
				diskPartitions = append(diskPartitions, map[string]any{
					"device":     partition.Device,
					"mountpoint": partition.Mountpoint,
					"fstype":     partition.Fstype,
					"total":      usage.Total,
					"free":       usage.Free,
					"used":       usage.Used,
					"percent":    usage.UsedPercent,
				})
				mu.Unlock()
			case err := <-errChan:
				// 单个分区获取失败，记录但不影响其他分区
				errorlog.RecordHTTP(ctx, "monitor", "Get disk usage error", map[string]any{
					"mountpoint": partition.Mountpoint,
					"error":      err.Error(),
				}, "Get disk usage error for %s: %v", partition.Mountpoint, err)
			case <-partitionCtx.Done():
				// 单个分区超时，跳过该分区
				errorlog.RecordHTTP(ctx, "monitor", "Get disk usage timeout", map[string]any{
					"mountpoint": partition.Mountpoint,
					"timeout":    partitionTimeout.String(),
				}, "Get disk usage timeout for %s after %v", partition.Mountpoint, partitionTimeout)
			}
		}(part)
	}

	// 等待所有 goroutine 完成或总超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有分区处理完成
	case <-partCtx.Done():
		// 总超时，返回已收集的数据
		errorlog.RecordHTTP(ctx, "monitor", "Get disk partitions total timeout", map[string]any{
			"timeout": timeout.String(),
		}, "Get disk partitions total timeout after %v", timeout)
	}

	return diskPartitions
}
