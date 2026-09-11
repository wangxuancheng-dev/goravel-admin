package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appfacades "goravel/app/facades"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/clients"
	"goravel/app/http/helpers"
)

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
