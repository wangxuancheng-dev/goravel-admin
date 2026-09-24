package admin

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"goravel/app/utils/errorlog"
	wsnotifications "goravel/app/websocket/notifications"
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
	wsLocalAdmins, wsLocalConnections := wsnotifications.Hub().LocalStats()

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
			"online_admins":     wsAdmins,
			"connections":       wsConnections,
			"local_admins":      wsLocalAdmins,
			"local_connections": wsLocalConnections,
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
