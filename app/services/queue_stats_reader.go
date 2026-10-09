package services

import (
	"context"
	"fmt"
	appfacades "goravel/app/facades"
	"sort"
	"strings"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/samber/lo"

	"goravel/app/clients"
	"goravel/app/utils/errorlog"
)

// QueueStatsReader shares key/stats logic with artisan queue:stats for console and HTTP dashboards.
type QueueStatsReader struct {
	ctx context.Context
}

func NewQueueStatsReader(ctx context.Context) *QueueStatsReader {
	return &QueueStatsReader{
		ctx: ctx,
	}
}

// QueueStatsInfo aggregates database-queue counts by logical queue name.
type QueueStatsInfo struct {
	Pending  int64 `json:"pending"`
	Reserved int64 `json:"reserved"`
	Delayed  int64 `json:"delayed"`
	Failed   int64 `json:"failed"`
	Total    int64 `json:"total"`
}

// RedisQueueStatsInfo is Redis list queue stats.
type RedisQueueStatsInfo struct {
	Pending  int64 `json:"pending"`
	Reserved int64 `json:"reserved"`
	Delayed  int64 `json:"delayed"`
	Failed   int64 `json:"failed"`
	Total    int64 `json:"total"`
}

// IsRedisDriver reports whether the queue connection name is a Redis-backed queue.
func (s *QueueStatsReader) IsRedisDriver(connectionName string) bool {
	lower := strings.ToLower(connectionName)
	return strings.Contains(lower, "redis")
}

// GetRedisConnectionName resolves database.redis.* client name for a queue connection.
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

// RedisQueueKey is Goravel Redis list key: {app}_queues:{queueConnection}_{queue}
func (s *QueueStatsReader) RedisQueueKey(queueConnectionName, queueName string) string {
	appName := facades.Config().GetString("app.name", "goravel")
	return fmt.Sprintf("%s_queues:%s_%s", appName, queueConnectionName, queueName)
}

// RedisReservedKey is the reserved ZSET key.
func (s *QueueStatsReader) RedisReservedKey(queueConnectionName, queueName string) string {
	return fmt.Sprintf("%s:reserved", s.RedisQueueKey(queueConnectionName, queueName))
}

// RedisDelayedKey is the delayed ZSET key.
func (s *QueueStatsReader) RedisDelayedKey(queueConnectionName, queueName string) string {
	return fmt.Sprintf("%s:delayed", s.RedisQueueKey(queueConnectionName, queueName))
}

// GetStatsByQueue aggregates jobs / failed_jobs for the database driver.
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

// GetRedisQueueStats returns Redis list stats for one logical queue.
func (s *QueueStatsReader) GetRedisQueueStats(redisConnectionName, queueConnectionName, queueName string) (*RedisQueueStatsInfo, error) {
	redisClient, err := clients.GetRedisClient(redisConnectionName)
	if err != nil {
		errorlog.Record(s.ctx, "queue", "get redis client failed", map[string]any{
			"connection": redisConnectionName,
			"error":      err.Error(),
		}, "get redis client failed: %v", err)
		return nil, fmt.Errorf("get redis client failed: %v", err)
	}
	ctx := context.Background()
	stats := &RedisQueueStatsInfo{}
	baseKey := s.RedisQueueKey(queueConnectionName, queueName)
	pendingLen, err := redisClient.LLen(ctx, baseKey).Result()
	if err != nil {
		errorlog.Record(s.ctx, "queue", "query pending queue failed", map[string]any{
			"queue_name": queueName,
			"key":        baseKey,
			"error":      err.Error(),
		}, "query pending queue failed: %v", err)
		return nil, fmt.Errorf("query pending queue failed: %v", err)
	}
	stats.Pending = pendingLen
	reservedKey := s.RedisReservedKey(queueConnectionName, queueName)
	reservedLen, err := redisClient.ZCard(ctx, reservedKey).Result()
	if err != nil {
		errorlog.Record(s.ctx, "queue", "query reserved queue failed", map[string]any{
			"queue_name": queueName,
			"key":        reservedKey,
			"error":      err.Error(),
		}, "query reserved queue failed: %v", err)
		return nil, fmt.Errorf("query reserved queue failed: %v", err)
	}
	stats.Reserved = reservedLen
	delayedKey := s.RedisDelayedKey(queueConnectionName, queueName)
	delayedLen, err := redisClient.ZCard(ctx, delayedKey).Result()
	if err != nil {
		errorlog.Record(s.ctx, "queue", "query delayed queue failed", map[string]any{
			"queue_name": queueName,
			"key":        delayedKey,
			"error":      err.Error(),
		}, "query delayed queue failed: %v", err)
		return nil, fmt.Errorf("query delayed queue failed: %v", err)
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

// GetRedisStatsByQueue scans Redis keys and aggregates stats per logical queue.
func (s *QueueStatsReader) GetRedisStatsByQueue(redisConnectionName, queueConnectionName string) (map[string]*RedisQueueStatsInfo, error) {
	redisClient, err := clients.GetRedisClient(redisConnectionName)
	if err != nil {
		errorlog.Record(s.ctx, "queue", "get redis client failed", map[string]any{
			"connection": redisConnectionName,
			"error":      err.Error(),
		}, "get redis client failed: %v", err)
		return nil, fmt.Errorf("get redis client failed: %v", err)
	}
	ctx := context.Background()
	result := make(map[string]*RedisQueueStatsInfo)
	appName := facades.Config().GetString("app.name", "goravel")
	prefix := fmt.Sprintf("%s_queues:%s_", appName, queueConnectionName)
	pattern := prefix + "*"
	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		errorlog.Record(s.ctx, "queue", "scan queue keys failed", map[string]any{
			"pattern": pattern,
			"error":   err.Error(),
		}, "scan queue keys failed: %v", err)
		return nil, fmt.Errorf("scan queue keys failed: %v", err)
	}
	queueNames := lo.FilterMap(keys, func(key string, _ int) (string, bool) {
		if !strings.HasPrefix(key, prefix) {
			return "", false
		}
		after := strings.TrimPrefix(key, prefix)
		if after == "" {
			return "", false
		}
		// Ignore leftover :stream keys from removed drivers.
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
			validQueues := lo.Filter(failedQueues, func(q string, _ int) bool {
				return q != "" && !strings.HasPrefix(q, "stream_")
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

// ListQueueConnectionNames returns sorted queue.connections names.
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

// QueueDashboardConnection is one connection row for the light dashboard.
type QueueDashboardConnection struct {
	Name         string `json:"connection"`
	DriverRaw    string `json:"driver_raw"`
	Kind         string `json:"kind"`
	Supported    bool   `json:"supported"`
	IsDefault    bool   `json:"is_default"`
	RedisClient  string `json:"redis_client,omitempty"`
	DefaultQueue string `json:"default_queue,omitempty"`
	Queues       any    `json:"queues,omitempty"`
	MessageKey   string `json:"message_key,omitempty"`
	FetchError   string `json:"fetch_error,omitempty"`
}

// BuildQueueDashboard aggregates configured connections; database and redis_list are supported.
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
			row.Kind = "redis_list"
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
			continue
		}
		out = append(out, row)
	}
	return out, defaultConn
}
