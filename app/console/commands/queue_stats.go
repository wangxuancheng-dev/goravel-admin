package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/services"
)

type QueueStats struct {
}

func (r *QueueStats) Signature() string {
	return "queue:stats"
}

func (r *QueueStats) Description() string {
	return "查询队列统计信息，显示待执行、正在执行和失败任务数量"
}

func (r *QueueStats) Extend() command.Extend {
	return command.Extend{
		Category: "queue",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "queue",
				Aliases: []string{"q"},
				Usage:   "队列名称（可选，用于筛选特定队列）",
			},
			&command.StringFlag{
				Name:    "connection",
				Aliases: []string{"c"},
				Usage:   "队列连接名称（可选，默认使用默认连接）",
			},
		},
	}
}

func (r *QueueStats) Handle(ctx console.Context) error {
	queueName := ctx.Option("queue")
	connectionName := ctx.Option("connection")
	reader := services.NewQueueStatsReader(context.Background())

	if connectionName == "" {
		connectionName = facades.Config().GetString("queue.default", "sync")
	}

	ctx.Info(fmt.Sprintf("队列连接: %s", connectionName))
	if queueName != "" {
		ctx.Info(fmt.Sprintf("队列名称: %s", queueName))
	}
	ctx.Info("")

	driver := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.driver", connectionName), "")
	ctx.Info(fmt.Sprintf("驱动类型: %s", driver))

	if reader.IsRedisDriver(connectionName) {
		defaultQueue := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.queue", connectionName), "default")
		originalQueueName := queueName
		queueNameForStats := queueName
		if queueNameForStats == "" {
			queueNameForStats = defaultQueue
		}

		redisConnectionName := reader.GetRedisConnectionName(connectionName)
		if redisConnectionName == "" {
			ctx.Warning("无法确定 Redis 连接名称")
			return nil
		}

		stats, err := reader.GetRedisQueueStats(redisConnectionName, connectionName, queueNameForStats)
		if err != nil {
			ctx.Error(fmt.Sprintf("查询 Redis 队列统计失败: %v", err))
			ctx.Info("提示：请确保 Redis 连接配置正确且 Redis 服务正在运行")
			return err
		}

		totalCount := stats.Pending + stats.Reserved

		ctx.Info("═══════════════════════════════════════")
		ctx.Info("队列统计信息 (Redis)")
		ctx.Info("═══════════════════════════════════════")
		ctx.Table(
			[]string{"指标", "数量"},
			[][]string{
				{"队列名称", queueNameForStats},
				{"待执行任务", fmt.Sprintf("%d", stats.Pending)},
				{"正在执行任务", fmt.Sprintf("%d", stats.Reserved)},
				{"延迟任务", fmt.Sprintf("%d", stats.Delayed)},
				{"失败任务", fmt.Sprintf("%d", stats.Failed)},
				{"总任务数", fmt.Sprintf("%d", totalCount)},
			},
		)

		if stats.Pending > 0 {
			ctx.Info("")
			ctx.Warning(fmt.Sprintf("提示：队列中有 %d 个待执行任务", stats.Pending))
			ctx.Info("")
			ctx.Info("如果需要处理这些任务：")
			ctx.Info("  1. 启动主程序（main.go 中会自动启动队列 Worker）")
			ctx.Info("     go run .")
			ctx.Info("  2. 确保 Worker 监听正确的队列名称和连接")
			ctx.Info("  3. 确保任务已正确注册到 QueueServiceProvider")
			ctx.Info("")
			ctx.Info("如果不需要这些任务，可以清理队列：")
			ctx.Info("  使用 Redis 客户端执行以下命令清理队列：")
			appName := facades.Config().GetString("app.name", "goravel")
			baseKey := reader.RedisQueueKey(connectionName, queueNameForStats)
			ctx.Info(fmt.Sprintf("    # app.name=%s, queue.connection=%s, queue=%s", appName, connectionName, queueNameForStats))
			ctx.Info(fmt.Sprintf("    redis-cli DEL %s", baseKey))
			ctx.Info(fmt.Sprintf("    redis-cli DEL %s:reserved", baseKey))
			ctx.Info(fmt.Sprintf("    redis-cli DEL %s:delayed", baseKey))
			ctx.Info("  或者使用命令：go run . artisan queue:clear --queue=" + queueNameForStats)
		}

		if originalQueueName == "" {
			ctx.Info("")
			ctx.Info("按队列分组统计:")
			byQueue, err := reader.GetRedisStatsByQueue(redisConnectionName, connectionName)
			if err != nil {
				ctx.Warning(fmt.Sprintf("获取按队列分组统计失败: %v", err))
			} else {
				if len(byQueue) == 0 {
					ctx.Info("  暂无队列数据")
				} else {
					for qName, qStats := range byQueue {
						ctx.Info(fmt.Sprintf("  队列 [%s]:", qName))
						ctx.Info(fmt.Sprintf("    待执行: %d, 正在执行: %d, 延迟: %d, 失败: %d, 总计: %d",
							qStats.Pending, qStats.Reserved, qStats.Delayed, qStats.Failed, qStats.Total))
					}
				}
			}
		}

		return nil
	}

	if driver == "sync" {
		ctx.Info("同步驱动：任务立即执行，无队列数据")
		return nil
	}

	if driver != "database" {
		ctx.Warning(fmt.Sprintf("驱动 %s 暂不支持统计查询", driver))
		return nil
	}

	var pendingCount, reservedCount int64
	var err error

	pendingQuery := appfacades.OrmQuery(ctx).Table("jobs").
		Where("available_at <= ?", time.Now()).
		Where("reserved_at IS NULL")

	reservedQuery := appfacades.OrmQuery(ctx).Table("jobs").
		Where("reserved_at IS NOT NULL")

	if queueName != "" {
		pendingQuery = pendingQuery.Where("queue = ?", queueName)
		reservedQuery = reservedQuery.Where("queue = ?", queueName)
	}

	pendingCount, err = pendingQuery.Count()
	if err != nil {
		ctx.Error(fmt.Sprintf("查询待执行任务数失败: %v", err))
		return err
	}

	reservedCount, err = reservedQuery.Count()
	if err != nil {
		ctx.Error(fmt.Sprintf("查询正在执行任务数失败: %v", err))
		return err
	}

	failedQuery := appfacades.OrmQuery(ctx).Table("failed_jobs")
	if queueName != "" {
		failedQuery = failedQuery.Where("queue = ?", queueName)
	}
	failedCount, err := failedQuery.Count()
	if err != nil {
		ctx.Error(fmt.Sprintf("查询失败任务数失败: %v", err))
		return err
	}

	totalCount := pendingCount + reservedCount

	ctx.Info("═══════════════════════════════════════")
	ctx.Info("队列统计信息")
	ctx.Info("═══════════════════════════════════════")
	ctx.Table(
		[]string{"指标", "数量"},
		[][]string{
			{"待执行任务", fmt.Sprintf("%d", pendingCount)},
			{"正在执行任务", fmt.Sprintf("%d", reservedCount)},
			{"失败任务", fmt.Sprintf("%d", failedCount)},
			{"总任务数", fmt.Sprintf("%d", totalCount)},
		},
	)
	ctx.Info("═══════════════════════════════════════")

	if queueName == "" {
		ctx.Info("")
		ctx.Info("按队列分组统计:")
		byQueue, err := reader.GetStatsByQueue()
		if err != nil {
			ctx.Warning(fmt.Sprintf("获取按队列分组统计失败: %v", err))
		} else {
			if len(byQueue) == 0 {
				ctx.Info("  暂无队列数据")
			} else {
				for qName, stats := range byQueue {
					ctx.Info(fmt.Sprintf("  队列 [%s]:", qName))
					ctx.Info(fmt.Sprintf("    待执行: %d, 正在执行: %d, 失败: %d, 总计: %d",
						stats.Pending, stats.Reserved, stats.Failed, stats.Total))
				}
			}
		}
	}

	return nil
}
