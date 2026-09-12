package providers

import (
	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/contracts/queue"

	"goravel/app/facades"
	"goravel/app/jobs"
	"goravel/app/queuejobs"
)

type QueueServiceProvider struct {
}

func (receiver *QueueServiceProvider) Register(app foundation.Application) {
	facades.Queue().Register(receiver.Jobs())
}

func (receiver *QueueServiceProvider) Boot(app foundation.Application) {

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
		&jobs.TenantOps{},
		// 搜索引擎同步任务（可切换 driver）
		&queuejobs.SyncOrderSearch{},
	}
}
