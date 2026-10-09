package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/clients"
	"goravel/app/services"
)

type QueueClear struct {
}

// Signature The name and signature of the console command.
func (r *QueueClear) Signature() string {
	return "queue:clear"
}

// Description The console command description.
// go run . artisan queue:clear --connection=redis --queue=default --force
func (r *QueueClear) Description() string {
	return "清理队列中的任务（仅支持 Redis 驱动）"
}

// Extend The console command extend.
func (r *QueueClear) Extend() command.Extend {
	return command.Extend{
		Category: "queue",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "queue",
				Aliases: []string{"q"},
				Usage:   "队列名称（可选，默认清理默认队列）",
			},
			&command.StringFlag{
				Name:    "connection",
				Aliases: []string{"c"},
				Usage:   "队列连接名称（可选，默认使用默认连接）",
			},
			&command.BoolFlag{
				Name:  "force",
				Usage: "强制清理，不提示确认",
			},
		},
	}
}

// Handle Execute the console command.
func (r *QueueClear) Handle(ctx console.Context) error {
	reader := services.NewQueueStatsReader(context.Background())
	queueName := ctx.Option("queue")
	connectionName := ctx.Option("connection")
	// 布尔标志：检查命令行参数中是否包含 --force
	// 在 Goravel 框架中，BoolFlag 存在时 ctx.Option 返回空字符串，所以需要检查命令行参数
	force := r.hasForceFlag()

	if connectionName == "" {
		connectionName = facades.Config().GetString("queue.default", "sync")
	}

	// 判断队列驱动类型
	driver := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.driver", connectionName), "")

	// 检查是否是 Redis 驱动
	isRedis := reader.IsRedisDriver(connectionName)

	if !isRedis {
		if driver == "sync" {
			ctx.Info("同步驱动：任务立即执行，无需清理")
			return nil
		}
		if driver == "database" {
			ctx.Warning("数据库驱动暂不支持清理命令，请直接操作数据库 jobs 表")
			return nil
		}
		ctx.Warning(fmt.Sprintf("驱动 %s 暂不支持清理命令", driver))
		return nil
	}

	// Redis 驱动：清理队列
	defaultQueue := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.queue", connectionName), "default")
	if queueName == "" {
		queueName = defaultQueue
	}

	// 获取 Redis 连接名称
	redisConnectionName := reader.GetRedisConnectionName(connectionName)
	if redisConnectionName == "" {
		ctx.Warning("无法确定 Redis 连接名称")
		return nil
	}

	// 查询当前队列统计
	stats, err := reader.GetRedisQueueStats(redisConnectionName, connectionName, queueName)
	if err != nil {
		ctx.Error(fmt.Sprintf("查询队列统计失败: %v", err))
		return err
	}

	totalCount := stats.Pending + stats.Reserved + stats.Delayed
	if totalCount == 0 {
		ctx.Info(fmt.Sprintf("队列 '%s' 中没有任何任务", queueName))
		return nil
	}

	// 显示当前统计
	ctx.Info(fmt.Sprintf("队列 '%s' 当前状态：", queueName))
	ctx.Info(fmt.Sprintf("  待执行任务: %d", stats.Pending))
	ctx.Info(fmt.Sprintf("  正在执行任务: %d", stats.Reserved))
	ctx.Info(fmt.Sprintf("  延迟任务: %d", stats.Delayed))
	ctx.Info(fmt.Sprintf("  总计: %d", totalCount))
	ctx.Info("")

	// 确认清理
	if !force {
		ctx.Warning(fmt.Sprintf("警告：此操作将删除队列 '%s' 中的所有任务（共 %d 个）", queueName, totalCount))
		ctx.Info("如果确定要继续，请使用 --force 参数")
		ctx.Info(fmt.Sprintf("   go run . artisan queue:clear --queue=%s --connection=%s --force", queueName, connectionName))
		return nil
	}

	// 执行清理
	ctx.Info("开始清理队列...")
	redisClient, err := clients.GetRedisClient(redisConnectionName)
	if err != nil {
		ctx.Error(fmt.Sprintf("获取 Redis 客户端失败: %v", err))
		return err
	}
	// 注意：使用公共 Redis 客户端池，不需要手动关闭

	ctxRedis := context.Background()
	clearedCount := int64(0)

	// Clear pending list
	pendingKey := reader.RedisQueueKey(connectionName, queueName)
	pendingLen, _ := redisClient.LLen(ctxRedis, pendingKey).Result()
	if pendingLen > 0 {
		if err := redisClient.Del(ctxRedis, pendingKey).Err(); err != nil {
			ctx.Error(fmt.Sprintf("清理待执行队列失败: %v", err))
		} else {
			clearedCount += pendingLen
			ctx.Info(fmt.Sprintf("已清理待执行队列: %d 个任务", pendingLen))
		}
	}

	// 清理正在执行队列
	reservedKey := reader.RedisReservedKey(connectionName, queueName)
	reservedLen, _ := redisClient.ZCard(ctxRedis, reservedKey).Result()
	if reservedLen > 0 {
		if err := redisClient.Del(ctxRedis, reservedKey).Err(); err != nil {
			ctx.Error(fmt.Sprintf("清理正在执行队列失败: %v", err))
		} else {
			clearedCount += reservedLen
			ctx.Info(fmt.Sprintf("已清理正在执行队列: %d 个任务", reservedLen))
		}
	}

	// 清理延迟队列
	delayedKey := reader.RedisDelayedKey(connectionName, queueName)
	delayedLen, _ := redisClient.ZCard(ctxRedis, delayedKey).Result()
	if delayedLen > 0 {
		if err := redisClient.Del(ctxRedis, delayedKey).Err(); err != nil {
			ctx.Error(fmt.Sprintf("清理延迟队列失败: %v", err))
		} else {
			clearedCount += delayedLen
			ctx.Info(fmt.Sprintf("已清理延迟队列: %d 个任务", delayedLen))
		}
	}

	ctx.Info(fmt.Sprintf("队列清理完成！共清理 %d 个任务", clearedCount))
	return nil
}

// hasForceFlag 检查命令行参数中是否包含 --force 标志
func (r *QueueClear) hasForceFlag() bool {
	for _, arg := range os.Args {
		if arg == "--force" || arg == "-force" {
			return true
		}
	}
	return false
}
