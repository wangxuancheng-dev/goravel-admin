package providers

import (
	"context"
	"time"

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
	services.EnqueueCancelExpiredOrderFn = func(ctx context.Context, orderNo string, expireAt time.Time, tenantID uint) error {
		return facades.Queue().Job(&jobs.CancelExpiredOrder{}, []queue.Arg{
			{Type: "string", Value: orderNo},
			{Type: "int", Value: int(tenantID)},
		}).Delay(expireAt).Dispatch()
	}
	services.EnqueueFlexibleScheduleRunFn = func(scheduleID uint, slot string) error {
		return facades.Queue().Job(&jobs.FlexibleScheduleRun{}, []queue.Arg{
			{Type: "int", Value: int(scheduleID)},
			{Type: "string", Value: slot},
		}).OnQueue(services.FlexibleScheduleQueueName()).Dispatch()
	}
	services.EnqueueTenantOpsFn = func(args services.TenantOpsArgs) error {
		payload, err := services.MarshalTenantOpsArgsJSON(args)
		if err != nil {
			return err
		}
		return facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{
			{Type: "string", Value: payload},
		}).OnQueue("long-running").Dispatch()
	}
	services.EnqueueTenantOpsFleetFn = func(args services.TenantOpsFleetArgs) error {
		payload, err := services.MarshalTenantOpsFleetArgsJSON(args)
		if err != nil {
			return err
		}
		return facades.Queue().Job(&jobs.TenantOpsFleet{}, []queue.Arg{
			{Type: "string", Value: payload},
		}).OnQueue("long-running").Dispatch()
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
		&jobs.TenantOpsFleet{},
		&jobs.ImportArticles{},
		// 搜索引擎同步任务（订单；文章等后续同目录加 sync_*_search.go）
		&jobs.SyncOrderSearch{},
		&jobs.CancelExpiredOrder{},
		&jobs.FlexibleScheduleRun{},
	}
}
