package console

import (
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/schedule"

	"goravel/app/console/commands"
)

type Kernel struct {
}

func (kernel *Kernel) Schedule() []schedule.Event {
	// Use ScheduleTracked so automatic runs also write last-run cache for the admin page.
	// Times below are UTC (see comments for Beijing equivalents).
	return []schedule.Event{
		// 北京时间 02:00 = UTC 18:00（前一天）
		ScheduleTracked("app:clear-logs").DailyAt("18:00").OnOneServer(),
		// 北京时间 03:00 = UTC 19:00（前一天）
		ScheduleTracked("app:clear-chunks").DailyAt("19:00").OnOneServer(),
		// 北京时间 03:30 = UTC 19:30
		ScheduleTracked("db:analyze-stats").DailyAt("19:30").OnOneServer(),
		ScheduleTracked("order:create-sharding-tables").Monthly().OnOneServer(),
		ScheduleTracked("payment:create-sharding-tables").Monthly().OnOneServer(),
		ScheduleTracked("search:retry-outbox").Hourly().OnOneServer(),
	}
}

func (kernel *Kernel) Commands() []console.Command {
	return []console.Command{
		&commands.ClearLogs{},
		&commands.ClearChunks{},
		&commands.CreateToken{},
		&commands.QueueStats{},
		&commands.QueueClear{},
		&commands.QueuePeek{},
		&commands.ScheduleTestLog{},
		commands.NewCreateOrderShardingTables(),
		commands.NewCreatePaymentShardingTables(),
		&commands.GenerateTestOrders{},
		&commands.GenerateTestPayments{},
		&commands.AnalyzeStats{},
		&commands.OptimizeTables{},
		&commands.ElasticsearchExample{},
		&commands.InitOrdersSearchIndex{},
		&commands.SyncOrdersSearch{},
		&commands.RetrySearchOutbox{},
		&commands.TenantCreate{},
		&commands.TenantMigrate{},
		&commands.TenantMigrateAll{},
		&commands.TenantSeed{},
		&commands.TenantSeedAll{},
		&commands.TenantList{},
		&commands.TenantDisable{},
		&commands.TenantEnable{},
		&commands.TenantBackup{},
		&commands.TenantRestore{},
		&commands.PlatformAdminCreate{},
	}
}
