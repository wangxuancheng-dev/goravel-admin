package config

import (
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"
	redisfacades "github.com/goravel/redis/facades"
	kafka "github.com/wangxuancheng-dev/goravel-kafka"
	nsq "github.com/wangxuancheng-dev/goravel-nsq"
	rabbitmq "github.com/wangxuancheng-dev/goravel-rabbitmq"
	redisstream "github.com/wangxuancheng-dev/goravel-redis-stream"
)

func init() {
	config := facades.Config()
	config.Add("queue", map[string]any{
		// Default Queue Connection Name
		"default": config.Env("QUEUE_CONNECTION", "sync"),
		// Queue Connections
		//
		// Here you may configure the connection information for each server that is used by your application.
		// Drivers: "sync", "database", "custom"
		"connections": map[string]any{
			"sync": map[string]any{
				"driver": "sync",
			},
			"database": map[string]any{
				"driver": "database",
				// 默认跟随 DB_CONNECTION，必要时可通过 QUEUE_DATABASE_CONNECTION 单独指定。
				"connection": config.Env("QUEUE_DATABASE_CONNECTION", config.Env("DB_CONNECTION", "mysql")),
				"queue":      "default",
				"concurrent": 1,
				// "tries": 3,        // 最大重试次数（可选，默认由队列工作进程设置）
				// "retry_after": 90, // 重试延迟时间（秒，可选）
			},
			// "redis1": map[string]any{
			// 	"driver":     "custom",
			// 	"connection": "default",
			// 	"queue":      "default",
			// 	"via": func() (queue.Driver, error) {
			// 		return redisfacades.Queue("redis1") // The `redis` value is the key of `connections`
			// 	},
			// },
			"redis": map[string]any{
				"driver":     "custom",
				"connection": "default",
				"queue":      "default",
				"via": func() (queue.Driver, error) {
					return redisfacades.Queue("redis")
				},
			},
			"redis_stream": map[string]any{
				"driver":         "custom",
				"connection":     "default",
				"queue":          "default",
				"group":          config.Env("QUEUE_REDIS_STREAM_GROUP", "goravel"),     // Redis Stream consumer group 名称
				"block_ms":       config.Env("QUEUE_REDIS_STREAM_BLOCK_MS", 1000),       // XREADGROUP 阻塞等待毫秒数
				"retry_after":    config.Env("QUEUE_REDIS_STREAM_RETRY_AFTER", 90),      // 超过该秒数未 ACK 的消息可被 XAUTOCLAIM 重领
				"claim_count":    config.Env("QUEUE_REDIS_STREAM_CLAIM_COUNT", 10),      // 每轮 XAUTOCLAIM 最多认领消息数
				"delete_on_ack":  config.Env("QUEUE_REDIS_STREAM_DELETE_ON_ACK", false), // true: ACK 后同时 XDEL 删除消息
				"stream_max_len": config.Env("QUEUE_REDIS_STREAM_MAX_LEN", 100000),      // Stream 近似最大长度（0 表示不限制）
				"via": func() (queue.Driver, error) {
					return redisstream.New("redis_stream")
				},
			},
			"rabbitmq": map[string]any{
				"driver":        "custom",
				"queue":         config.Env("QUEUE_RABBITMQ_QUEUE", "default"),
				"url":           config.Env("QUEUE_RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/"),
				"exchange":      config.Env("QUEUE_RABBITMQ_EXCHANGE", "goravel.exchange"),
				"exchange_type": config.Env("QUEUE_RABBITMQ_EXCHANGE_TYPE", "direct"),
				"queue_prefix":  config.Env("QUEUE_RABBITMQ_QUEUE_PREFIX", config.GetString("app.name", "goravel")),
				"routing_key":   config.Env("QUEUE_RABBITMQ_ROUTING_KEY", ""),
				"via": func() (queue.Driver, error) {
					return rabbitmq.New("rabbitmq")
				},
			},
			"nsq": map[string]any{
				"driver":           "custom",
				"queue":            config.Env("QUEUE_NSQ_QUEUE", "default"),
				"nsqd_tcp_address": config.Env("QUEUE_NSQ_NSQD_TCP_ADDRESS", "127.0.0.1:4150"),
				"channel":          config.Env("QUEUE_NSQ_CHANNEL", "goravel"),
				"timeout_ms":       config.Env("QUEUE_NSQ_TIMEOUT_MS", 1000),
				"max_in_flight":    config.Env("QUEUE_NSQ_MAX_IN_FLIGHT", 1),
				"via": func() (queue.Driver, error) {
					return nsq.New("nsq")
				},
			},
			"kafka": map[string]any{
				"driver":           "custom",
				"queue":            config.Env("QUEUE_KAFKA_QUEUE", "default"),
				"brokers":          config.Env("QUEUE_KAFKA_BROKERS", "127.0.0.1:9092"),
				"group_id":         config.Env("QUEUE_KAFKA_GROUP_ID", "goravel"),
				"dial_timeout_ms":  config.Env("QUEUE_KAFKA_DIAL_TIMEOUT_MS", 1000),
				"write_timeout_ms": config.Env("QUEUE_KAFKA_WRITE_TIMEOUT_MS", 1000),
				"read_timeout_ms":  config.Env("QUEUE_KAFKA_READ_TIMEOUT_MS", 1000),
				"max_wait_ms":      config.Env("QUEUE_KAFKA_MAX_WAIT_MS", 1000),
				"via": func() (queue.Driver, error) {
					return kafka.New("kafka")
				},
			},
		},
		// Failed Queue Jobs
		//
		// These options configure the behavior of failed queue job logging so you
		// can control how and where failed jobs are stored.
		"failed": map[string]any{
			"database": config.Env("DB_CONNECTION", "postgres"),
			"table":    "failed_jobs",
		},
		// Retry Configuration
		//
		// 队列工作进程的最大重试次数
		// 可以通过环境变量 QUEUE_TRIES 设置，默认值为 10
		// 注意：这个值是上限，实际重试次数由每个 Job 的 ShouldRetry 方法决定
		"tries": config.Env("QUEUE_TRIES", 10), // 最大重试次数上限（建议设置较大值，如 10）
		// Concurrent Configuration
		//
		// 并发数 = 同一队列上同时执行的任务个数（不是进程数）。以下为经验示例，按机器 CPU、任务是否占满 CPU/IO 再调。
		//
		// QUEUE_CONCURRENT（默认队列 default）示例：
		//   - 本地开发 / 任务很轻：1～2
		//   - 小生产、以 IO 为主（发邮件、调外部 API）：4～8
		//   - CPU 密集（图片处理、大报表）：1～2，避免把机器打满
		//   - 内存占用大的 Job：宁小勿大，例如 1～2
		//
		// QUEUE_LONG_RUNNING_CONCURRENT（long-running 队列）示例：
		//   - 导出、大文件、长耗时：通常 1；允许并行多条时再 2～3，注意数据库与磁盘压力
		"concurrent":              config.Env("QUEUE_CONCURRENT", 3),              // 默认队列；见上注释
		"long_running_concurrent": config.Env("QUEUE_LONG_RUNNING_CONCURRENT", 1), // 耗时队列；见上注释
		// 搜索引擎同步队列（逻辑名见 config search.sync_queue）
		"search_concurrent": config.Env("QUEUE_SEARCH_CONCURRENT", 2),
		// Per-tenant flexible schedule handlers (minute-level collection)
		"schedule_concurrent": config.Env("QUEUE_SCHEDULE_CONCURRENT", 10),
		// "test_concurrent":         config.Env("QUEUE_TEST_CONCURRENT", 1),         // 逻辑队列 test，见 bootstrap.TestQueueRunner
	})
}
