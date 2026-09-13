package providers

import (
	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/contracts/queue"

	"goravel/app/facades"
	"goravel/app/jobs"
	"goravel/app/services"
)

type QueueServiceProvider struct {
}

func (receiver *QueueServiceProvider) Register(app foundation.Application) {
	facades.Queue().Register(receiver.Jobs())
}

func (receiver *QueueServiceProvider) Boot(app foundation.Application) {
	// Break services↔jobs import cycle: services call this hook instead of importing jobs.
	services.EnqueueEmailFn = func(to, subject, content string) error {
		return facades.Queue().Job(&jobs.SendEmail{}, []queue.Arg{
			{Type: "string", Value: to},
			{Type: "string", Value: subject},
			{Type: "string", Value: content},
		}).Dispatch()
	}
}

func (receiver *QueueServiceProvider) Jobs() []queue.Job {
	return []queue.Job{
		&jobs.Test{},
		&jobs.TestErr{},
		&jobs.TestClaim{},
		&jobs.TestBackoff{},
		// 实际场景的Job
		&jobs.SendEmail{},
		&jobs.ProcessImage{},
		&jobs.GenerateReport{},
		// 导出任务
		&jobs.ExportOrders{},
		&jobs.ExportPayments{},
		&jobs.ExportUsers{},
		&jobs.ExportArticles{},
		&jobs.ImportOrders{},
		&jobs.TenantOps{},
		// 搜索引擎同步任务（订单；文章等后续同目录加 sync_*_search.go）
		&jobs.SyncOrderSearch{},
	}
}
