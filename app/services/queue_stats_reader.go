package services

import (
	"context"
	"fmt"
	appfacades "goravel/app/facades"
	"sort"
	"strings"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"

	"goravel/app/clients"
	"goravel/app/utils/errorlog"
)

// QueueStatsReader 与 artisan queue:stats 使用同一套键与统计逻辑，供控制台与 HTTP 看板复用。
type QueueStatsReader struct {
	ctx context.Context
}

func NewQueueStatsReader(ctx context.Context) *QueueStatsReader {
	return &QueueStatsReader{
		ctx: ctx}
}

// QueueStatsInfo 数据库队列按逻辑队列名聚合的统计。
type QueueStatsInfo struct {
	Pending  int64 `json:"pending"`
	Reserved int64 `json:"reserved"`
	Delayed  int64 `json:"delayed"`
	Failed   int64 `json:"failed"`
	Total    int64 `json:"total"`
}

// RedisQueueStatsInfo Redis 列表 / Stream 队列统计。
type RedisQueueStatsInfo struct {
	Pending  int64 `json:"pending"`
	Reserved int64 `json:"reserved"`
	Delayed  int64 `json:"delayed"`
	Failed   int64 `json:"failed"`
	Total    int64 `json:"total"`
	// StreamTotal 仅用于 redis_stream，表示 stream 当前总长度（含历史已ACK未删除消息）。
	StreamTotal int64 `json:"stream_total,omitempty"`
}

// IsRedisDriver 判断是否为 Redis 列表或 Stream 等基于 Redis 的队列连接。
func (s *QueueStatsReader) IsRedisDriver(connectionName string) bool {
	// 仅把明确的 Redis 连接名判定为 Redis 队列，避免将 kafka/nsq/rabbitmq 的 custom via 误判为 redis。
	lower := strings.ToLower(connectionName)
	return strings.Contains(lower, "redis")
}

// IsRedisStreamDriver 是否为 Redis Stream 驱动连接。
func (s *QueueStatsReader) IsRedisStreamDriver(connectionName string) bool {
	if strings.Contains(strings.ToLower(connectionName), "stream") {
		return true
	}
	return facades.Config().Get(fmt.Sprintf("queue.connections.%s.stream_max_len", connectionName)) != nil
}

// GetRedisConnectionName 解析队列配置中的 Redis 客户端连接名（database.redis.*）。
func (s *QueueStatsReader) GetRedisConnectionName(queueConnectionName string) string {
	connection := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.connection", queueConnectionName), "default")
	if strings.Contains(queueConnectionName, "redis") {
		redisHost := facades.Config().GetString(fmt.Sprintf("database.redis.%s.host", queueConnectionName), "")
		if redisHost != "" {
			return queueConnectionName
		}
	}
	return connection
}

// RedisQueueKey Goravel Redis 队列键：{app}_queues:{queueConnection}_{queue}
func (s *QueueStatsReader) RedisQueueKey(queueConnectionName, queueName string) string {
	appName := facades.Config().GetString("app.name", "goravel")
	return fmt.Sprintf("%s_queues:%s_%s", appName, queueConnectionName, queueName)
}

// RedisReservedKey 正在执行（ZSET）。
func (s *QueueStatsReader) RedisReservedKey(queueConnectionName, queueName string) string {
	return fmt.Sprintf("%s:reserved", s.RedisQueueKey(queueConnectionName, queueName))
}

// RedisDelayedKey 延迟队列（ZSET）。
func (s *QueueStatsReader) RedisDelayedKey(queueConnectionName, queueName string) string {
	return fmt.Sprintf("%s:delayed", s.RedisQueueKey(queueConnectionName, queueName))
}

// RedisStreamKey Stream 键。
func (s *QueueStatsReader) RedisStreamKey(queueConnectionName, queueName string) string {
	return fmt.Sprintf("%s:stream", s.RedisQueueKey(queueConnectionName, queueName))
}

// GetStatsByQueue 数据库驱动：按 queue 字段聚合 jobs / failed_jobs。
func (s *QueueStatsReader) GetStatsByQueue() (map[string]QueueStatsInfo, error) {
	var queues []string
	err := appfacades.PlatformOrmQuery(s.ctx).Table("jobs").
		Select("DISTINCT queue").
		Pluck("queue", &queues)
	if err != nil {
		return nil, err
	}
	var failedQueues []string
	err = appfacades.PlatformOrmQuery(s.ctx).Table("failed_jobs").
		Select("DISTINCT queue").
		Pluck("queue", &failedQueues)
	if err != nil {
		return nil, err
	}
	queueMap := make(map[string]bool)
	for _, q := range queues {
		queueMap[q] = true
	}
	for _, q := range failedQueues {
		queueMap[q] = true
	}
	result := make(map[string]QueueStatsInfo)
	now := time.Now()
	for qName := range queueMap {
		pendingCount, _ := appfacades.PlatformOrmQuery(s.ctx).Table("jobs").
			Where("queue = ?", qName).
			Where("available_at <= ?", now).
			Where("reserved_at IS NULL").
			Count()
		delayedCount, _ := appfacades.PlatformOrmQuery(s.ctx).Table("jobs").
			Where("queue = ?", qName).
			Where("available_at > ?", now).
			Where("reserved_at IS NULL").
			Count()
		reservedCount, _ := appfacades.PlatformOrmQuery(s.ctx).Table("jobs").
			Where("queue = ?", qName).
			Where("reserved_at IS NOT NULL").
			Count()
		failedCount, _ := appfacades.PlatformOrmQuery(s.ctx).Table("failed_jobs").
			Where("queue = ?", qName).
			Count()
		result[qName] = QueueStatsInfo{
			Pending:  pendingCount,
			Reserved: reservedCount,
			Delayed:  delayedCount,
			Failed:   failedCount,
			Total:    pendingCount + reservedCount,
		}
	}
	return result, nil
}

// GetRedisQueueStats 单个逻辑队列在指定队列连接上的 Redis 统计。
func (s *QueueStatsReader) GetRedisQueueStats(redisConnectionName, queueConnectionName, queueName string) (*RedisQueueStatsInfo, error) {
	redisClient, err := clients.GetRedisClient(redisConnectionName)
	if err != nil {
		errorlog.Record(s.ctx, "queue", "获取 Redis 客户端失败", map[string]any{
			"connection": redisConnectionName,
			"error":      err.Error(),
		}, "获取 Redis 客户端失败: %v", err)
		return nil, fmt.Errorf("获取 Redis 客户端失败: %v", err)
	}
	ctx := context.Background()
	stats := &RedisQueueStatsInfo{}
	if s.IsRedisStreamDriver(queueConnectionName) {
		streamKey := s.RedisStreamKey(queueConnectionName, queueName)
		group := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.group", queueConnectionName), "goravel")
		streamLen, err := redisClient.XLen(ctx, streamKey).Result()
		if err != nil {
			errorlog.Record(s.ctx, "queue", "查询 stream 长度失败", map[string]any{
				"queue_name": queueName,
				"key":        streamKey,
				"error":      err.Error(),
			}, "查询 stream 长度失败: %v", err)
			return nil, fmt.Errorf("查询 stream 长度失败: %v", err)
		}
		pendingInfo, err := redisClient.XPending(ctx, streamKey, group).Result()
		if err != nil {
			pendingInfo = &redis.XPending{}
		}
		stats.StreamTotal = streamLen
		stats.Reserved = pendingInfo.Count
		// Stream 的“待执行”采用消费组 lag 口径，避免把已 ACK 未删除历史误算为积压。
		lag, lagErr := s.getStreamLag(ctx, redisClient, streamKey, group)
		if lagErr == nil && lag >= 0 {
			stats.Pending = lag
		} else if streamLen >= stats.Reserved {
			// 回退口径：当 Redis 不支持 lag 或 group 信息缺失时，保底沿用旧逻辑。
			stats.Pending = streamLen - stats.Reserved
		}
		delayedKey := s.RedisDelayedKey(queueConnectionName, queueName)
		delayedLen, err := redisClient.ZCard(ctx, delayedKey).Result()
		if err != nil {
			errorlog.Record(s.ctx, "queue", "查询延迟队列失败", map[string]any{
				"queue_name": queueName,
				"key":        delayedKey,
				"error":      err.Error(),
			}, "查询延迟队列失败: %v", err)
			return nil, fmt.Errorf("查询延迟队列失败: %v", err)
		}
		stats.Delayed = delayedLen
		var failedCount int64
		if queueName != "" {
			failedCount, err = appfacades.PlatformOrmQuery(s.ctx).Table("failed_jobs").
				Where("queue = ?", queueName).
				Count()
		} else {
			failedCount, err = appfacades.PlatformOrmQuery(s.ctx).Table("failed_jobs").Count()
		}
		if err == nil {
			stats.Failed = failedCount
		}
		stats.Total = stats.Pending + stats.Reserved
		return stats, nil
	}
	baseKey := s.RedisQueueKey(queueConnectionName, queueName)
	pendingLen, err := redisClient.LLen(ctx, baseKey).Result()
	if err != nil {
		errorlog.Record(s.ctx, "queue", "查询待执行队列失败", map[string]any{
			"queue_name": queueName,
			"key":        baseKey,
			"error":      err.Error(),
		}, "查询待执行队列失败: %v", err)
		return nil, fmt.Errorf("查询待执行队列失败: %v", err)
	}
	stats.Pending = pendingLen
	reservedKey := s.RedisReservedKey(queueConnectionName, queueName)
	reservedLen, err := redisClient.ZCard(ctx, reservedKey).Result()
	if err != nil {
		errorlog.Record(s.ctx, "queue", "查询正在执行队列失败", map[string]any{
			"queue_name": queueName,
			"key":        reservedKey,
			"error":      err.Error(),
		}, "查询正在执行队列失败: %v", err)
		return nil, fmt.Errorf("查询正在执行队列失败: %v", err)
	}
	stats.Reserved = reservedLen
	delayedKey := s.RedisDelayedKey(queueConnectionName, queueName)
	delayedLen, err := redisClient.ZCard(ctx, delayedKey).Result()
	if err != nil {
		errorlog.Record(s.ctx, "queue", "查询延迟队列失败", map[string]any{
			"queue_name": queueName,
			"key":        delayedKey,
			"error":      err.Error(),
		}, "查询延迟队列失败: %v", err)
		return nil, fmt.Errorf("查询延迟队列失败: %v", err)
	}
	stats.Delayed = delayedLen
	var failedCount int64
	if queueName != "" {
		failedCount, err = appfacades.PlatformOrmQuery(s.ctx).Table("failed_jobs").
			Where("queue = ?", queueName).
			Count()
	} else {
		failedCount, err = appfacades.PlatformOrmQuery(s.ctx).Table("failed_jobs").Count()
	}
	if err != nil {
		stats.Failed = 0
	} else {
		stats.Failed = failedCount
	}
	stats.Total = stats.Pending + stats.Reserved
	return stats, nil
}

func (s *QueueStatsReader) getStreamLag(ctx context.Context, redisClient redis.UniversalClient, streamKey, group string) (int64, error) {
	groups, err := redisClient.XInfoGroups(ctx, streamKey).Result()
	if err != nil {
		return -1, err
	}
	for _, item := range groups {
		if item.Name == group {
			return item.Lag, nil
		}
	}
	return -1, fmt.Errorf("group %s not found", group)
}

// GetRedisStatsByQueue 扫描 Redis 键前缀，汇总该队列连接下各逻辑队列统计。
func (s *QueueStatsReader) GetRedisStatsByQueue(redisConnectionName, queueConnectionName string) (map[string]*RedisQueueStatsInfo, error) {
	redisClient, err := clients.GetRedisClient(redisConnectionName)
	if err != nil {
		errorlog.Record(s.ctx, "queue", "获取 Redis 客户端失败", map[string]any{
			"connection": redisConnectionName,
			"error":      err.Error(),
		}, "获取 Redis 客户端失败: %v", err)
		return nil, fmt.Errorf("获取 Redis 客户端失败: %v", err)
	}
	ctx := context.Background()
	result := make(map[string]*RedisQueueStatsInfo)
	appName := facades.Config().GetString("app.name", "goravel")
	prefix := fmt.Sprintf("%s_queues:%s_", appName, queueConnectionName)
	pattern := prefix + "*"
	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		errorlog.Record(s.ctx, "queue", "查找队列键失败", map[string]any{
			"pattern": pattern,
			"error":   err.Error(),
		}, "查找队列键失败: %v", err)
		return nil, fmt.Errorf("查找队列键失败: %v", err)
	}
	queueNames := lo.FilterMap(keys, func(key string, _ int) (string, bool) {
		isStreamConn := s.IsRedisStreamDriver(queueConnectionName)
		if !strings.HasPrefix(key, prefix) {
			return "", false
		}
		after := strings.TrimPrefix(key, prefix)
		if after == "" {
			return "", false
		}
		// 反推逻辑队列名：
		// - redis_list: base/list、:reserved、:delayed 都应映射到同一队列名
		// - redis_stream: :stream、:delayed 映射到同一队列名
		if isStreamConn {
			if strings.HasSuffix(after, ":stream") {
				name := strings.TrimSuffix(after, ":stream")
				return name, name != ""
			}
			if strings.HasSuffix(after, ":delayed") {
				name := strings.TrimSuffix(after, ":delayed")
				return name, name != ""
			}
			return "", false
		}
		// Redis List 连接：避免把 redis_stream 的 :stream 键误识别成 stream_* 队列名。
		if strings.HasSuffix(after, ":stream") {
			return "", false
		}
		if strings.HasSuffix(after, ":reserved") {
			name := strings.TrimSuffix(after, ":reserved")
			return name, name != ""
		}
		if strings.HasSuffix(after, ":delayed") {
			name := strings.TrimSuffix(after, ":delayed")
			return name, name != ""
		}
		return after, true
	})
	queueMap := lo.SliceToMap(lo.Uniq(queueNames), func(queueName string) (string, bool) {
		return queueName, true
	})
	if len(queueMap) == 0 {
		var failedQueues []string
		err = appfacades.PlatformOrmQuery(s.ctx).Table("failed_jobs").
			Select("DISTINCT queue").
			Pluck("queue", &failedQueues)
		if err == nil {
			isStreamConn := s.IsRedisStreamDriver(queueConnectionName)
			validQueues := lo.Filter(failedQueues, func(q string, _ int) bool {
				if q == "" {
					return false
				}
				// Redis List 连接回退到 failed_jobs 时，过滤掉历史 stream_* 队列名，避免看板误导。
				if !isStreamConn && strings.HasPrefix(q, "stream_") {
					return false
				}
				return true
			})
			queueMap = lo.SliceToMap(validQueues, func(queueName string) (string, bool) {
				return queueName, true
			})
		}
	}
	for queueName := range queueMap {
		stats, err := s.GetRedisQueueStats(redisConnectionName, queueConnectionName, queueName)
		if err != nil {
			continue
		}
		result[queueName] = stats
	}
	return result, nil
}

func queueConnectionsMap(raw any) map[string]any {
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return nil
	}
	return m
}

// ListQueueConnectionNames 返回 queue.connections 下的连接名（排序后）。
func (s *QueueStatsReader) ListQueueConnectionNames() []string {
	raw := facades.Config().Get("queue.connections")
	m := queueConnectionsMap(raw)
	if m == nil {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// QueueDashboardConnection 轻量看板中单条连接的可序列化视图。
type QueueDashboardConnection struct {
	Name          string `json:"connection"`
	DriverRaw     string `json:"driver_raw"`
	Kind          string `json:"kind"`
	Supported     bool   `json:"supported"`
	IsDefault     bool   `json:"is_default"`
	RedisClient   string `json:"redis_client,omitempty"`
	DefaultQueue  string `json:"default_queue,omitempty"`
	ConsumerGroup string `json:"consumer_group,omitempty"`
	Queues        any    `json:"queues,omitempty"`
	MessageKey    string `json:"message_key,omitempty"`
	FetchError    string `json:"fetch_error,omitempty"`
}

// BuildQueueDashboard 汇总所有已配置队列连接；仅 database / redis_list / redis_stream 为 supported。
func (s *QueueStatsReader) BuildQueueDashboard() ([]QueueDashboardConnection, string) {
	defaultConn := facades.Config().GetString("queue.default", "sync")
	names := s.ListQueueConnectionNames()
	if len(names) == 0 {
		return nil, defaultConn
	}
	out := make([]QueueDashboardConnection, 0, len(names))
	for _, name := range names {
		driver := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.driver", name), "")
		row := QueueDashboardConnection{
			Name:      name,
			DriverRaw: driver,
			IsDefault: name == defaultConn,
		}
		switch {
		case driver == "database":
			row.Kind = "database"
			row.Supported = true
			row.DefaultQueue = facades.Config().GetString(fmt.Sprintf("queue.connections.%s.queue", name), "default")
			by, err := s.GetStatsByQueue()
			if err != nil {
				row.FetchError = err.Error()
			} else {
				row.Queues = by
			}
		case s.IsRedisDriver(name):
			if s.IsRedisStreamDriver(name) {
				row.Kind = "redis_stream"
				row.ConsumerGroup = facades.Config().GetString(fmt.Sprintf("queue.connections.%s.group", name), "goravel")
			} else {
				row.Kind = "redis_list"
			}
			row.Supported = true
			redisConn := s.GetRedisConnectionName(name)
			row.RedisClient = redisConn
			row.DefaultQueue = facades.Config().GetString(fmt.Sprintf("queue.connections.%s.queue", name), "default")
			by, err := s.GetRedisStatsByQueue(redisConn, name)
			if err != nil {
				row.FetchError = err.Error()
				if def := row.DefaultQueue; def != "" {
					if single, e2 := s.GetRedisQueueStats(redisConn, name, def); e2 == nil {
						row.Queues = map[string]*RedisQueueStatsInfo{def: single}
					}
				}
			} else {
				row.Queues = by
			}
		default:
			// 轻量看板仅展示 database / redis_list / redis_stream 三类连接。
			continue
		}
		out = append(out, row)
	}
	return out, defaultConn
}
