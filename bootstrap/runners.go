package bootstrap

import (
	"context"
	"sync"

	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/contracts/queue"

	"goravel/app/facades"
	"goravel/app/search"
	"goravel/app/services"
)

// DefaultQueueRunner 默认队列工作进程 runner
type DefaultQueueRunner struct {
	worker queue.Worker
	mu     sync.Mutex
}

func (r *DefaultQueueRunner) Signature() string {
	return "queue-default"
}

func (r *DefaultQueueRunner) ShouldRun() bool {
	return shouldRunQueueRunner(r.Signature())
}

func (r *DefaultQueueRunner) Run() error {
	tries := facades.Config().GetInt("queue.tries", 5)
	concurrent := facades.Config().GetInt("queue.concurrent", 1)

	r.mu.Lock()
	r.worker = facades.Queue().Worker(queue.Args{
		Connection: "",
		Queue:      "",
		Concurrent: concurrent,
		Tries:      tries,
	})
	r.mu.Unlock()

	facades.Log().Infof("默认队列工作进程启动 - 队列: default, 并发数: %d, 最大重试次数: %d", concurrent, tries)
	systemLogService := services.NewSystemLogService(context.Background())
	_ = systemLogService.Record(context.Background(), "info", "queue", "默认队列工作进程启动", map[string]any{
		"queue":      "default",
		"concurrent": concurrent,
		"tries":      tries,
	})

	return r.worker.Run()
}

func (r *DefaultQueueRunner) Shutdown() error {
	r.mu.Lock()
	worker := r.worker
	r.mu.Unlock()
	if worker != nil {
		return worker.Shutdown()
	}
	return nil
}

// LongRunningQueueRunner 耗时任务队列工作进程 runner
type LongRunningQueueRunner struct {
	worker queue.Worker
	mu     sync.Mutex
}

func (r *LongRunningQueueRunner) Signature() string {
	return "queue-long-running"
}

func (r *LongRunningQueueRunner) ShouldRun() bool {
	return shouldRunQueueRunner(r.Signature())
}

func (r *LongRunningQueueRunner) Run() error {
	tries := facades.Config().GetInt("queue.tries", 5)
	longRunningConcurrent := facades.Config().GetInt("queue.long_running_concurrent", 1)

	r.mu.Lock()
	r.worker = facades.Queue().Worker(queue.Args{
		Connection: "",
		Queue:      "long-running",
		Concurrent: longRunningConcurrent,
		Tries:      tries,
	})
	r.mu.Unlock()

	facades.Log().Infof("耗时任务队列工作进程启动 - 队列: long-running, 并发数: %d, 最大重试次数: %d", longRunningConcurrent, tries)
	systemLogService := services.NewSystemLogService(context.Background())
	_ = systemLogService.Record(context.Background(), "info", "queue", "耗时任务队列工作进程启动", map[string]any{
		"queue":      "long-running",
		"concurrent": longRunningConcurrent,
		"tries":      tries,
	})

	return r.worker.Run()
}

func (r *LongRunningQueueRunner) Shutdown() error {
	r.mu.Lock()
	worker := r.worker
	r.mu.Unlock()
	if worker != nil {
		return worker.Shutdown()
	}
	return nil
}

// SearchQueueRunner 搜索引擎同步任务专用逻辑队列（与默认队列隔离）。
type SearchQueueRunner struct {
	worker queue.Worker
	mu     sync.Mutex
}

func (r *SearchQueueRunner) Signature() string {
	return "queue-search"
}

func (r *SearchQueueRunner) ShouldRun() bool {
	if !shouldRunQueueRunner(r.Signature()) {
		return false
	}
	return search.ShouldRunQueueWorker()
}

func (r *SearchQueueRunner) Run() error {
	tries := facades.Config().GetInt("queue.tries", 5)
	concurrent := facades.Config().GetInt("queue.search_concurrent", 2)
	queueName := search.SyncQueue()

	r.mu.Lock()
	r.worker = facades.Queue().Worker(queue.Args{
		Connection: "",
		Queue:      queueName,
		Concurrent: concurrent,
		Tries:      tries,
	})
	r.mu.Unlock()

	facades.Log().Infof("搜索同步队列启动 - driver=%s 队列=%s 并发=%d 最大重试=%d", search.Driver(), queueName, concurrent, tries)
	systemLogService := services.NewSystemLogService(context.Background())
	_ = systemLogService.Record(context.Background(), "info", "queue", "搜索同步队列启动", map[string]any{
		"driver":     search.Driver(),
		"queue":      queueName,
		"concurrent": concurrent,
		"tries":      tries,
	})

	return r.worker.Run()
}

func (r *SearchQueueRunner) Shutdown() error {
	r.mu.Lock()
	worker := r.worker
	r.mu.Unlock()
	if worker != nil {
		return worker.Shutdown()
	}
	return nil
}

// QueueRunners 返回队列相关的 runners
func QueueRunners() []foundation.Runner {
	return []foundation.Runner{
		&DefaultQueueRunner{},
		&LongRunningQueueRunner{},
		&SearchQueueRunner{},
		// &TestQueueRunner{}, // 需要时再取消下面整块注释并取消本行注释；config 见 queue.test_concurrent
	}
}

/*
TestQueueRunner：逻辑队列名「test」，与 tests/feature/queue_test.go 中 OnQueue("test")、queue.test_concurrent 对应。
入队：facades.Queue().Job(..., args).OnQueue("test").Dispatch()

type TestQueueRunner struct {
	worker queue.Worker
	mu     sync.Mutex
}

func (r *TestQueueRunner) Signature() string {
	return "queue-test"
}

func (r *TestQueueRunner) ShouldRun() bool {
	connection := facades.Config().GetString("queue.default")
	driver := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.driver", connection))
	return connection != "" && driver != "sync" && shouldRunQueueRunner(r.Signature())
}

func (r *TestQueueRunner) Run() error {
	tries := facades.Config().GetInt("queue.tries", 5)
	testConcurrent := facades.Config().GetInt("queue.test_concurrent", 1)

	r.mu.Lock()
	r.worker = facades.Queue().Worker(queue.Args{
		Connection: "",
		Queue:      "test",
		Concurrent: testConcurrent,
		Tries:      tries,
	})
	r.mu.Unlock()

	facades.Log().Infof("测试队列工作进程启动 - 队列: test, 并发数: %d, 最大重试次数: %d", testConcurrent, tries)
	systemLogService := services.NewSystemLogService(context.Background())
	_ = systemLogService.Record(context.Background(), "info", "queue", "测试队列工作进程启动", map[string]any{
		"queue":      "test",
		"concurrent": testConcurrent,
		"tries":      tries,
	})

	return r.worker.Run()
}

func (r *TestQueueRunner) Shutdown() error {
	r.mu.Lock()
	worker := r.worker
	r.mu.Unlock()
	if worker != nil {
		return worker.Shutdown()
	}
	return nil
}
*/
