package commands

import (
	"context"
	"encoding/json"
	"fmt"
	appfacades "goravel/app/facades"
	"math"
	"os"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/clients"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type QueuePeek struct{}

// Signature The name and signature of the console command.
func (r *QueuePeek) Signature() string {
	return "queue:peek"
}

// ./main artisan queue:peek --connection=redis --queue=long-running --state=all --limit=5 --full
// go run . artisan queue:peek --connection=redis --state=all --limit=5 --full
func (r *QueuePeek) Description() string {
	return "查看队列中前 N 条任务内容（支持 Redis/database），用于排查“导出”等任务到底投递了什么"
}

// Extend The console command extend.
func (r *QueuePeek) Extend() command.Extend {
	return command.Extend{
		Category: "queue",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "queue",
				Aliases: []string{"q"},
				Usage:   "队列名称（可选；导出任务一般在 long-running）",
			},
			&command.StringFlag{
				Name:    "connection",
				Aliases: []string{"c"},
				Usage:   "队列连接名称（可选，默认使用默认连接）",
			},
			&command.StringFlag{
				Name:    "state",
				Aliases: []string{"s"},
				Usage:   "查看队列状态：pending|reserved|delayed|all（默认 pending）",
			},
			&command.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Value:   10,
				Usage:   "最多显示多少条（默认 10）",
			},
			&command.BoolFlag{
				Name:  "raw",
				Usage: "输出原始 payload（不做 JSON 美化/摘要）",
			},
			&command.BoolFlag{
				Name:  "full",
				Usage: "不截断输出（默认会截断到 500 字符）",
			},
		},
	}
}

// Handle Execute the console command.
func (r *QueuePeek) Handle(ctx console.Context) error {
	queueName := ctx.Option("queue")
	connectionName := ctx.Option("connection")
	state := strings.TrimSpace(strings.ToLower(ctx.Option("state")))
	limit := ctx.OptionInt("limit")
	raw := r.hasFlag("--raw")
	full := r.hasFlag("--full")

	if limit <= 0 {
		limit = 10
	}
	if state == "" {
		state = "pending"
	}

	if connectionName == "" {
		connectionName = facades.Config().GetString("queue.default", "sync")
	}

	ctx.Info(fmt.Sprintf("队列连接: %s", connectionName))
	if queueName != "" {
		ctx.Info(fmt.Sprintf("队列名称: %s", queueName))
	}
	ctx.Info(fmt.Sprintf("查看状态: %s, limit=%d, raw=%v, full=%v", state, limit, raw, full))
	ctx.Info("")

	driver := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.driver", connectionName), "")
	isRedis := r.isRedisDriver(connectionName)

	if isRedis {
		defaultQueue := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.queue", connectionName), "default")
		if queueName == "" {
			queueName = defaultQueue
		}

		redisConnectionName := r.getRedisConnectionName(connectionName)
		if redisConnectionName == "" {
			ctx.Warning("无法确定 Redis 连接名称")
			return nil
		}

		redisClient, err := clients.GetRedisClient(redisConnectionName)
		if err != nil {
			errorlog.Record(context.Background(), "queue", "获取 Redis 客户端失败", map[string]any{
				"connection": redisConnectionName,
				"error":      err.Error(),
			}, "获取 Redis 客户端失败: %v", err)
			return fmt.Errorf("获取 Redis 客户端失败: %v", err)
		}

		c := context.Background()

		printPayload := func(prefix, payload string) {
			ctx.Info(prefix)
			ctx.Info(r.renderPayload(payload, raw, full))
			ctx.Info("")
		}

		if state == "pending" || state == "all" {
			key := r.redisQueueKey(connectionName, queueName)
			values, err := redisClient.LRange(c, key, 0, int64(limit-1)).Result()
			if err != nil {
				return fmt.Errorf("读取 pending 队列失败: %v", err)
			}
			ctx.Info("═══════════════════════════════════════")
			ctx.Info(fmt.Sprintf("Redis pending: key=%s, count(shown)=%d", key, len(values)))
			ctx.Info("═══════════════════════════════════════")
			if len(values) == 0 {
				ctx.Info("（空）")
				ctx.Info("")
			}
			for i, v := range values {
				printPayload(fmt.Sprintf("[%d] pending", i+1), v)
			}
		}

		if state == "reserved" || state == "all" {
			key := r.redisReservedKey(connectionName, queueName)
			zs, err := redisClient.ZRangeWithScores(c, key, 0, int64(limit-1)).Result()
			if err != nil {
				return fmt.Errorf("读取 reserved 队列失败: %v", err)
			}
			ctx.Info("═══════════════════════════════════════")
			ctx.Info(fmt.Sprintf("Redis reserved(ZSET): key=%s, count(shown)=%d", key, len(zs)))
			ctx.Info("═══════════════════════════════════════")
			if len(zs) == 0 {
				ctx.Info("（空）")
				ctx.Info("")
			}
			for i, z := range zs {
				member, _ := z.Member.(string)
				scoreInfo := r.formatScoreAsTime(z.Score)
				printPayload(fmt.Sprintf("[%d] reserved score=%v%s", i+1, z.Score, scoreInfo), member)
			}
		}

		if state == "delayed" || state == "all" {
			key := r.redisDelayedKey(connectionName, queueName)
			zs, err := redisClient.ZRangeWithScores(c, key, 0, int64(limit-1)).Result()
			if err != nil {
				return fmt.Errorf("读取 delayed 队列失败: %v", err)
			}
			ctx.Info("═══════════════════════════════════════")
			ctx.Info(fmt.Sprintf("Redis delayed(ZSET): key=%s, count(shown)=%d", key, len(zs)))
			ctx.Info("═══════════════════════════════════════")
			if len(zs) == 0 {
				ctx.Info("（空）")
				ctx.Info("")
			}
			for i, z := range zs {
				member, _ := z.Member.(string)
				scoreInfo := r.formatScoreAsTime(z.Score)
				printPayload(fmt.Sprintf("[%d] delayed score=%v%s", i+1, z.Score, scoreInfo), member)
			}
		}

		return nil
	}

	if driver == "sync" {
		ctx.Info("同步驱动：任务立即执行，无队列数据")
		return nil
	}
	if driver != "database" {
		ctx.Warning(fmt.Sprintf("驱动 %s 暂不支持 peek 查看", driver))
		return nil
	}

	// database driver
	q := appfacades.OrmQuery(ctx).Table("jobs").Select("id", "queue", "payload", "attempts", "reserved_at", "available_at", "created_at")
	now := time.Now()

	switch state {
	case "pending":
		q = q.Where("reserved_at IS NULL").Where("available_at", "<=", now)
	case "reserved":
		q = q.Where("reserved_at IS NOT NULL")
	case "delayed":
		q = q.Where("reserved_at IS NULL").Where("available_at", ">", now)
	case "all":
		// no filter
	default:
		ctx.Warning("state 参数只支持 pending|reserved|delayed|all")
		return nil
	}

	if queueName != "" {
		q = q.Where("queue", "=", queueName)
	}

	var rows []map[string]any
	if err := q.OrderByDesc("id").Limit(limit).Get(&rows); err != nil {
		return fmt.Errorf("查询 jobs 表失败: %v", err)
	}

	ctx.Info("═══════════════════════════════════════")
	ctx.Info(fmt.Sprintf("Database jobs: state=%s, count(shown)=%d", state, len(rows)))
	ctx.Info("═══════════════════════════════════════")
	if len(rows) == 0 {
		ctx.Info("（空）")
		return nil
	}

	for i, row := range rows {
		id := row["id"]
		qn := row["queue"]
		attempts := row["attempts"]
		payload, _ := row["payload"].(string)

		ctx.Info(fmt.Sprintf("[%d] id=%v queue=%v attempts=%v", i+1, id, qn, attempts))
		ctx.Info(r.renderPayload(payload, raw, full))
		ctx.Info("")
	}

	return nil
}

func (r *QueuePeek) isRedisDriver(connectionName string) bool {
	via := facades.Config().Get(fmt.Sprintf("queue.connections.%s.via", connectionName))
	return via != nil || strings.Contains(connectionName, "redis")
}

func (r *QueuePeek) getRedisConnectionName(queueConnectionName string) string {
	connection := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.connection", queueConnectionName), "default")

	if strings.Contains(queueConnectionName, "redis") {
		redisHost := facades.Config().GetString(fmt.Sprintf("database.redis.%s.host", queueConnectionName), "")
		if redisHost != "" {
			return queueConnectionName
		}
	}

	return connection
}

// redisQueueKey Goravel Redis queue key format:
// {appName}_queues:{queueConnection}_{queue}
func (r *QueuePeek) redisQueueKey(queueConnectionName, queueName string) string {
	appName := facades.Config().GetString("app.name", "goravel")
	return fmt.Sprintf("%s_queues:%s_%s", appName, queueConnectionName, queueName)
}

func (r *QueuePeek) redisReservedKey(queueConnectionName, queueName string) string {
	return fmt.Sprintf("%s:reserved", r.redisQueueKey(queueConnectionName, queueName))
}

func (r *QueuePeek) redisDelayedKey(queueConnectionName, queueName string) string {
	return fmt.Sprintf("%s:delayed", r.redisQueueKey(queueConnectionName, queueName))
}

func (r *QueuePeek) renderPayload(payload string, raw, full bool) string {
	if raw {
		return r.maybeTruncate(payload, full)
	}

	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return "（空 payload）"
	}

	// best-effort JSON pretty + add a tiny summary if we can detect job/signature fields
	var obj any
	if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
		summary := r.summarizePayload(obj)
		pretty, _ := json.MarshalIndent(obj, "", "  ")
		out := string(pretty)
		if summary != "" {
			out = summary + "\n" + out
		}
		return r.maybeTruncate(out, full)
	}

	// not JSON, fallback to raw
	return r.maybeTruncate(payload, full)
}

func (r *QueuePeek) summarizePayload(obj any) string {
	m, ok := obj.(map[string]any)
	if !ok {
		return ""
	}

	// Goravel/Laravel-like payloads often include these keys (best-effort)
	candidates := []string{"job", "name", "signature", "displayName", "command"}
	for _, k := range candidates {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return fmt.Sprintf("摘要: %s=%s", k, s)
			}
		}
	}

	// sometimes job info nested
	if data, ok := m["data"].(map[string]any); ok {
		for _, k := range candidates {
			if v, ok := data[k]; ok {
				if s, ok := v.(string); ok && s != "" {
					return fmt.Sprintf("摘要: data.%s=%s", k, s)
				}
			}
		}
	}

	return ""
}

func (r *QueuePeek) maybeTruncate(s string, full bool) string {
	if full {
		return s
	}
	const max = 500
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...(truncated, use --full to show all)"
}

func (r *QueuePeek) hasFlag(flag string) bool {
	for _, arg := range os.Args {
		if arg == flag {
			return true
		}
	}
	return false
}

func (r *QueuePeek) formatScoreAsTime(score float64) string {
	// Redis ZSET score for delayed jobs is commonly a unix timestamp (seconds).
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return ""
	}
	sec := int64(score)
	// heuristic: 10-digit seconds timestamp range
	if sec < 1000000000 || sec > 5000000000 {
		return ""
	}
	t := time.Unix(sec, 0).Local()
	return fmt.Sprintf(" (%s)", t.Format(utils.DateTimeFormat))
}
