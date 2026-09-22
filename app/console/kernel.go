package console

import (
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/schedule"
	"github.com/goravel/framework/facades"

	"goravel/app/console/commands"
)

type Kernel struct {
}

func (kernel *Kernel) Schedule() []schedule.Event {
	// Use ScheduleTracked so automatic runs also write last-run cache for the admin page.
	// Times below are UTC (see comments for Beijing equivalents).
	return []schedule.Event{
		// 北京时间 01:00 = UTC 17:00（前一天）：先归档再清理
		ScheduleTracked("operation_log:archive").DailyAt("17:00").OnOneServer(),
		// 北京时间 02:00 = UTC 18:00（前一天）
		ScheduleTracked("app:clear-logs").DailyAt("18:00").OnOneServer(),
		// 北京时间 03:00 = UTC 19:00（前一天）
		ScheduleTracked("app:clear-chunks").DailyAt("19:00").OnOneServer(),
		// 北京时间 03:30 = UTC 19:30
		ScheduleTracked("db:analyze-stats").DailyAt("19:30").OnOneServer(),
		ScheduleTracked("order:create-sharding-tables").Monthly().OnOneServer(),
		ScheduleTracked("payment:create-sharding-tables").Monthly().OnOneServer(),
		ScheduleTracked("search:retry-outbox").Hourly().OnOneServer(),
		ScheduleTracked("queue:alert-backlog").Hourly().OnOneServer(),
		// Beijing 04:00 = UTC 20:00: hard-delete expired soft-deleted tenants
		ScheduleTracked("tenant:cleanup-deleted").DailyAt("20:00").OnOneServer(),
		// Landlord backup: env gate inside command; time from TENANT_BACKUP_SCHEDULE_AT (UTC)
		ScheduleTracked("tenant:backup-scheduled").DailyAt(tenantBackupScheduleAtUTC()).OnOneServer(),
		// Tenant health: ping / schema / quota / migrate fail → webhook/email
		ScheduleTracked("tenant:health-inspect").Hourly().OnOneServer(),
		// Open-source demos: activity windows + unpaid order expire safety net
		ScheduleTracked("activity:sync-status").EveryTenSeconds().OnOneServer(),
		ScheduleTracked("order:cancel-expired").EveryMinute().OnOneServer(),
		// Whitelist handler + DB cron rows (see flexible_schedules)
		ScheduleTracked("flexible-schedule:tick").EveryMinute().OnOneServer().SkipIfStillRunning(),
	}
}

func tenantBackupScheduleAtUTC() string {
	at := strings.TrimSpace(facades.Config().GetString("tenancy.backup_schedule_at", "20:00"))
	if at == "" {
		return "20:00"
	}
	parts := strings.Split(at, ":")
	if len(parts) != 2 {
		return "20:00"
	}
	return at
}

func (kernel *Kernel) Commands() []console.Command {
	return []console.Command{
		&commands.ClearLogs{},
		&commands.ArchiveOperationLogs{},
		&commands.ClearChunks{},
		&commands.CreateToken{},
		&commands.QueueStats{},
		&commands.QueueClear{},
		&commands.QueuePeek{},
		&commands.QueueAlertBacklog{},
		&commands.ScheduleTestLog{},
		&commands.FlexibleScheduleTick{},
		&commands.SyncDemoActivities{},
		&commands.CancelExpiredOrders{},
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
		&commands.TenantBackupAll{},
		&commands.TenantBackupScheduled{},
		&commands.TenantRestore{},
		&commands.TenantCleanupDeleted{},
		&commands.TenantHealthInspect{},
		&commands.PlatformAdminCreate{},
		&commands.PlatformInstall{},
	}
}
