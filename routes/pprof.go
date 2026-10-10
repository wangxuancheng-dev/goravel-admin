package routes

import (
	"net"
	"net/http/pprof"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/http/helpers"
)

// pprof 失败尝试记录（用于防止暴力破解）
type pprofAttemptRecord struct {
	Count        int       // 失败次数
	LastFailed   time.Time // 最后失败时间
	BlockedUntil time.Time // 封禁到期时间
}

var (
	pprofAttempts     = make(map[string]*pprofAttemptRecord)
	pprofAttemptsLock sync.RWMutex
)

// Pprof 注册 pprof 性能分析路由
// 可通过环境变量控制：
//   - PPROF_ENABLED: 是否启用 pprof（默认：仅在 APP_DEBUG=true 时启用）
//   - PPROF_ALLOWED_IPS: 允许访问的 IP 地址，逗号分隔（例如：127.0.0.1,192.168.1.100）
//   - PPROF_TOKEN: 访问 token（可选，如果设置则需要在请求头或查询参数中提供 X-Pprof-Token）
//   - PPROF_MAX_ATTEMPTS: 最大失败尝试次数（默认：5）
//   - PPROF_BLOCK_DURATION: 封禁时长（秒，默认：300，即5分钟）
func Pprof() {
	// 检查是否启用 pprof
	pprofEnabled := facades.Config().GetBool("pprof.enabled", false)
	// 如果没有显式设置 PPROF_ENABLED，则默认在 debug 模式下启用
	if !pprofEnabled {
		pprofEnabled = facades.Config().GetBool("app.debug", false)
	}

	if !pprofEnabled {
		return
	}

	// 获取允许的 IP 地址列表
	allowedIPsStr := facades.Config().GetString("pprof.allowed_ips", "")
	var allowedIPs []string
	if allowedIPsStr != "" {
		ips := strings.Split(allowedIPsStr, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				allowedIPs = append(allowedIPs, ip)
			}
		}
	}

	// 获取访问 token
	pprofToken := facades.Config().GetString("pprof.token", "")

	// Refuse to expose pprof without at least one control (token and/or IP allowlist).
	if strings.TrimSpace(pprofToken) == "" && len(allowedIPs) == 0 {
		facades.Log().Warning("pprof: enabled but PPROF_TOKEN and PPROF_ALLOWED_IPS are both empty — routes not registered")
		return
	}

	// 获取速率限制配置
	maxAttempts := facades.Config().GetInt("pprof.max_attempts", 5)       // 默认5次失败后封禁
	blockDuration := facades.Config().GetInt("pprof.block_duration", 300) // 默认封禁5分钟
	resetDuration := facades.Config().GetInt("pprof.reset_duration", 600) // 默认10分钟后重置计数

	// 创建 pprof 中间件（IP 白名单、token 验证和暴力破解防护）
	// 返回 true 表示验证通过，false 表示验证失败（已发送响应）
	pprofMiddleware := func(ctx http.Context) bool {
		realIP := helpers.GetRealIP(ctx)

		// 检查是否被封禁
		pprofAttemptsLock.RLock()
		record, exists := pprofAttempts[realIP]
		pprofAttemptsLock.RUnlock()

		if exists && record != nil {
			// 检查是否仍在封禁期内
			if time.Now().Before(record.BlockedUntil) {
				remaining := int(time.Until(record.BlockedUntil).Seconds())
				_ = ctx.Response().Json(http.StatusTooManyRequests, http.Json{
					"code":        http.StatusTooManyRequests,
					"message":     "Too many failed attempts. IP blocked. Please try again later.",
					"retry_after": remaining,
				}).Abort()
				return false
			}

			// 检查是否需要重置计数（超过重置时间）
			if time.Since(record.LastFailed) > time.Duration(resetDuration)*time.Second {
				pprofAttemptsLock.Lock()
				delete(pprofAttempts, realIP)
				pprofAttemptsLock.Unlock()
			}
		}

		// 检查 IP 白名单
		if len(allowedIPs) > 0 {
			allowed := false
			for _, allowedIP := range allowedIPs {
				// 支持 CIDR 格式（如 192.168.1.0/24）
				if isIPAllowed(realIP, allowedIP) {
					allowed = true
					break
				}
			}
			if !allowed {
				_ = ctx.Response().Json(http.StatusForbidden, http.Json{
					"code":    http.StatusForbidden,
					"message": "Access denied: IP not allowed",
				}).Abort()
				return false
			}
		}

		// 检查 token（如果设置了）
		if pprofToken != "" {
			// 从请求头或查询参数获取 token
			token := ctx.Request().Header("X-Pprof-Token", "")
			if token == "" {
				token = ctx.Request().Query("token", "")
			}
			if token != pprofToken {
				// Token 验证失败，记录失败尝试
				pprofAttemptsLock.Lock()
				record, exists := pprofAttempts[realIP]
				if !exists || record == nil {
					record = &pprofAttemptRecord{
						Count:      0,
						LastFailed: time.Now(),
					}
					pprofAttempts[realIP] = record
				}
				record.Count++
				record.LastFailed = time.Now()

				// 如果失败次数超过阈值，封禁 IP
				if record.Count >= maxAttempts {
					record.BlockedUntil = time.Now().Add(time.Duration(blockDuration) * time.Second)
					pprofAttemptsLock.Unlock()

					// 记录安全日志
					facades.Log().Warningf("pprof: IP %s blocked after %d failed token attempts", realIP, record.Count)

					_ = ctx.Response().Json(http.StatusTooManyRequests, http.Json{
						"code":        http.StatusTooManyRequests,
						"message":     "Too many failed attempts. IP temporarily blocked.",
						"retry_after": blockDuration,
					}).Abort()
					return false
				}
				pprofAttemptsLock.Unlock()

				// 记录失败尝试日志
				facades.Log().Warningf("pprof: Invalid token attempt from IP %s (attempt %d/%d)", realIP, record.Count, maxAttempts)

				_ = ctx.Response().Json(http.StatusUnauthorized, http.Json{
					"code":    http.StatusUnauthorized,
					"message": "Access denied: invalid token",
				}).Abort()
				return false
			}

			// Token 验证成功，清除失败记录
			pprofAttemptsLock.Lock()
			delete(pprofAttempts, realIP)
			pprofAttemptsLock.Unlock()
		}

		return true
	}

	// 获取底层 HTTP 服务器（Gin 或 Fiber）
	// 由于 Goravel 框架封装，我们需要通过反射或直接访问底层路由
	// 这里我们创建一个包装器来处理 pprof 路由

	// 包装处理函数，应用中间件
	wrapHandler := func(handler func(ctx http.Context) http.Response) func(ctx http.Context) http.Response {
		return func(ctx http.Context) http.Response {
			// 先执行中间件检查
			if !pprofMiddleware(ctx) {
				// 中间件验证失败，已经发送了响应，返回 nil
				return nil
			}

			// 中间件验证通过，继续执行处理函数
			return handler(ctx)
		}
	}

	// CPU 性能分析主页
	facades.Route().Get("/debug/pprof/", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Index(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// CPU 性能分析（30秒采样）
	facades.Route().Get("/debug/pprof/profile", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Profile(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// CPU 性能分析（自定义采样时间，通过 seconds 参数）
	facades.Route().Get("/debug/pprof/profile/{seconds}", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Profile(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 堆内存分析
	facades.Route().Get("/debug/pprof/heap", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Handler("heap").ServeHTTP(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 协程分析
	facades.Route().Get("/debug/pprof/goroutine", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Handler("goroutine").ServeHTTP(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 阻塞分析
	facades.Route().Get("/debug/pprof/block", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Handler("block").ServeHTTP(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 互斥锁分析
	facades.Route().Get("/debug/pprof/mutex", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Handler("mutex").ServeHTTP(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 线程创建分析
	facades.Route().Get("/debug/pprof/threadcreate", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Handler("threadcreate").ServeHTTP(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 内存分配分析
	facades.Route().Get("/debug/pprof/allocs", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Handler("allocs").ServeHTTP(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 命令行工具
	facades.Route().Get("/debug/pprof/cmdline", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Cmdline(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 符号表
	facades.Route().Get("/debug/pprof/symbol", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Symbol(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// Trace 追踪
	facades.Route().Get("/debug/pprof/trace", wrapHandler(func(ctx http.Context) http.Response {
		pprof.Trace(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	}))

	// 运行时统计信息
	facades.Route().Get("/debug/pprof/runtime", wrapHandler(func(ctx http.Context) http.Response {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return ctx.Response().Json(http.StatusOK, http.Json{
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]any{
				"alloc":       m.Alloc,
				"total_alloc": m.TotalAlloc,
				"sys":         m.Sys,
				"lookups":     m.Lookups,
				"mallocs":     m.Mallocs,
				"frees":       m.Frees,
			},
			"gc": map[string]any{
				"num_gc":      m.NumGC,
				"pause_total": m.PauseTotalNs,
			},
		})
	}))
}

// isIPAllowed 检查 IP 是否在允许列表中
// 支持 CIDR 格式（如 192.168.1.0/24）和单个 IP
func isIPAllowed(clientIP, allowedIP string) bool {
	// 精确匹配
	if clientIP == allowedIP {
		return true
	}

	// 检查是否为 CIDR 格式
	if strings.Contains(allowedIP, "/") {
		_, ipNet, err := net.ParseCIDR(allowedIP)
		if err != nil {
			return false
		}
		clientIPAddr := net.ParseIP(clientIP)
		if clientIPAddr == nil {
			return false
		}
		return ipNet.Contains(clientIPAddr)
	}

	// 单个 IP 匹配
	return clientIP == allowedIP
}
