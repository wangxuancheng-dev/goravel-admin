package admin

import (
	"encoding/json"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/utils"
)

// EnqueueAsyncImportInput 异步导入入队参数。
type EnqueueAsyncImportInput struct {
	LockResource string
	ImportType   string
	Disk         string
	Path         string
	TotalRows    int
	Job          queue.Job
}

// EnqueueAsyncImportResult 异步导入编排结果。
type EnqueueAsyncImportResult struct {
	Unauthorized bool
	Blocked      bool
	ImportID     uint
	Err          error
}

// EnqueueAsyncImport 统一异步导入编排：加锁 → 建记录 → 入队。
func EnqueueAsyncImport(ctx http.Context, in EnqueueAsyncImportInput) EnqueueAsyncImportResult {
	lock := helpers.AcquireExportLock(ctx, in.LockResource)
	if lock.Unauthorized {
		return EnqueueAsyncImportResult{Unauthorized: true}
	}
	if lock.Blocked {
		return EnqueueAsyncImportResult{Blocked: true}
	}

	var tenantID uint
	if tenancy.Enabled() {
		tid, err := helpers.RequireTenant(ctx)
		if err != nil {
			lock.Release()
			return EnqueueAsyncImportResult{Err: apperrors.ErrTenantRequired}
		}
		tenantID = tid
	}

	disk := in.Disk
	if disk == "" {
		disk = helpers.ResolveExportDisk(ctx)
	}

	importRecord := models.Import{
		AdminID:   lock.AdminID,
		Type:      in.ImportType,
		Status:    models.ImportStatusProcessing,
		Disk:      disk,
		Path:      in.Path,
		TotalRows: in.TotalRows,
	}
	if err := appfacades.OrmQuery(ctx).Create(&importRecord); err != nil {
		lock.Release()
		return EnqueueAsyncImportResult{Err: err}
	}

	args := jobs.ImportArgs{
		ImportID: importRecord.ID,
		AdminID:  lock.AdminID,
		TenantID: tenantID,
		Type:     in.ImportType,
		Language: utils.GetCurrentLanguage(ctx),
		Timezone: helpers.GetCurrentTimezone(ctx),
	}
	if conn, ok := helpers.GetTenantConnectionFromContext(ctx); ok {
		args.TenantConnection = conn
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		markAsyncImportFailed(ctx, &importRecord, err)
		return EnqueueAsyncImportResult{ImportID: importRecord.ID, Err: err}
	}

	queueArgs := []queue.Arg{{
		Type:  "string",
		Value: string(argsJSON),
	}}

	if err := facades.Queue().Job(in.Job, queueArgs).OnQueue("long-running").Dispatch(); err != nil {
		lock.Release()
		markAsyncImportFailed(ctx, &importRecord, err)
		return EnqueueAsyncImportResult{ImportID: importRecord.ID, Err: err}
	}

	return EnqueueAsyncImportResult{ImportID: importRecord.ID}
}

func markAsyncImportFailed(ctx http.Context, record *models.Import, err error) {
	if record == nil || err == nil {
		return
	}
	record.Status = models.ImportStatusFailed
	record.ErrorMsg = err.Error()
	_ = appfacades.OrmQuery(ctx).Save(record)
}
