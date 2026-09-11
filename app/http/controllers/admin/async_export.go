package admin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/carbon"

	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

// EnqueueAsyncExportInput 异步导出入队参数（控制器/生成器共用）。
type EnqueueAsyncExportInput struct {
	// LockResource 导出锁资源名，如 users / orders / payments / articles
	LockResource string
	// ExportType 写入 exports.type，并作为 job args.Type
	ExportType string
	// Filters 传给导出任务的筛选条件
	Filters map[string]any
	// Job 具体导出任务实例，如 &jobs.ExportUsers{}
	Job queue.Job
}

// EnqueueAsyncExportResult 异步导出编排结果。
type EnqueueAsyncExportResult struct {
	Unauthorized bool
	Blocked      bool
	ExportID     uint
	Err          error
}

// EnqueueAsyncExport 统一异步导出编排：加锁 → 建记录 → 序列化参数 → 入队。
// 入队失败会释放锁并将导出记录标为失败。
func EnqueueAsyncExport(ctx http.Context, in EnqueueAsyncExportInput) EnqueueAsyncExportResult {
	lock := helpers.AcquireExportLock(ctx, in.LockResource)
	if lock.Unauthorized {
		return EnqueueAsyncExportResult{Unauthorized: true}
	}
	if lock.Blocked {
		return EnqueueAsyncExportResult{Blocked: true}
	}

	exportRecord := models.Export{
		AdminID: lock.AdminID,
		Type:    in.ExportType,
		Status:  models.ExportStatusProcessing,
		Disk:    helpers.ResolveExportDisk(ctx),
		Path:    "",
	}
	if err := appfacades.OrmQuery(ctx).Create(&exportRecord); err != nil {
		return EnqueueAsyncExportResult{Err: err}
	}

	args := jobs.ExportArgs{
		ExportID: exportRecord.ID,
		AdminID:  lock.AdminID,
		Filters:  in.Filters,
		Type:     in.ExportType,
		Language: utils.GetCurrentLanguage(ctx),
		Timezone: helpers.GetCurrentTimezone(ctx),
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		markAsyncExportFailed(ctx, &exportRecord, err)
		return EnqueueAsyncExportResult{ExportID: exportRecord.ID, Err: err}
	}

	queueArgs := []queue.Arg{{
		Type:  "string",
		Value: string(argsJSON),
	}}

	if err := facades.Queue().Job(in.Job, queueArgs).OnQueue("long-running").Dispatch(); err != nil {
		lock.Release()
		markAsyncExportFailed(ctx, &exportRecord, err)
		return EnqueueAsyncExportResult{ExportID: exportRecord.ID, Err: err}
	}

	return EnqueueAsyncExportResult{ExportID: exportRecord.ID}
}

func markAsyncExportFailed(ctx http.Context, record *models.Export, err error) {
	if record == nil || err == nil {
		return
	}
	record.Status = models.ExportStatusFailed
	record.ErrorMsg = err.Error()
	_ = appfacades.OrmQuery(ctx).Save(record)
}

// OwnedExportStatusResponse 返回当前管理员自己的导出任务状态（订单/支付等共用）。
func OwnedExportStatusResponse(ctx http.Context) http.Response {
	exportID := helpers.GetUintRoute(ctx, "id")
	if exportID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "id_required")
	}

	exportRecord, err := services.NewExportRecordService(ctx).GetByID(exportID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, err, map[string]any{
			"export_id": exportID,
		})
	}

	adminID, err := helpers.GetAdminIDFromContext(ctx)
	if err != nil {
		return response.Error(ctx, http.StatusUnauthorized, "unauthorized")
	}
	if exportRecord.AdminID != adminID {
		return response.Error(ctx, http.StatusForbidden, "forbidden")
	}

	fileURL := ""
	if exportRecord.Path != "" && exportRecord.Status == models.ExportStatusSuccess {
		if exportRecord.Disk == "local" || exportRecord.Disk == "public" {
			fileURL = fmt.Sprintf("/api/admin/exports/%d/download", exportRecord.ID)
		} else {
			fileURL = services.NewExportService(ctx).GetExportURL(exportRecord.Path)
		}
	}

	return response.Success(ctx, http.Json{
		"id":          exportRecord.ID,
		"status":      exportRecord.Status,
		"status_text": exportStatusText(ctx, exportRecord.Status),
		"file_url":    fileURL,
		"filename":    exportRecord.Filename,
		"size":        exportRecord.Size,
		"error_msg":   exportRecord.ErrorMsg,
		"created_at":  formatExportTimestamp(exportRecord.CreatedAt),
		"updated_at":  formatExportTimestamp(exportRecord.UpdatedAt),
	})
}

func exportStatusText(ctx http.Context, status uint8) string {
	switch status {
	case models.ExportStatusProcessing:
		return trans.Get(ctx, "processing")
	case models.ExportStatusSuccess:
		return trans.Get(ctx, "success")
	case models.ExportStatusFailed:
		return trans.Get(ctx, "failed")
	default:
		return trans.Get(ctx, "unknown")
	}
}

func formatExportTimestamp(t any) string {
	if t == nil {
		return ""
	}
	switch v := t.(type) {
	case time.Time:
		return utils.FormatDateTime(v)
	case *time.Time:
		return utils.FormatDateTimePtr(v)
	case carbon.DateTime:
		if v.IsZero() {
			return ""
		}
		return v.ToDateTimeString()
	case *carbon.DateTime:
		if v == nil || v.IsZero() {
			return ""
		}
		return v.ToDateTimeString()
	default:
		if str := fmt.Sprintf("%v", t); str != "" && str != "<nil>" {
			return str
		}
		return ""
	}
}
