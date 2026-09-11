package admin

import (
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"time"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type ExportController struct{}

func NewExportController() *ExportController {
	return &ExportController{}
}

func (r *ExportController) exportRecordService(ctx http.Context) services.ExportRecordService {
	return services.NewExportRecordService(ctx)
}

// Index 导出记录列表
func (r *ExportController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})

	filters := services.BuildExportRecordFiltersFromHTTP(ctx)

	svc := r.exportRecordService(ctx)
	exports, total, err := svc.GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, err, nil)
	}

	resultWithURL := make([]map[string]any, len(exports))
	for i := range exports {
		resultWithURL[i] = svc.ExportRecordToJSON(&exports[i])
	}

	return response.Paginate(ctx, resultWithURL, total, page, pageSize)
}

// Destroy 删除导出记录并删除源文件
func (r *ExportController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	if err := r.exportRecordService(ctx).DeleteWithFile(id); err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, err, map[string]any{
			"exportId": id,
		})
	}

	return response.Success(ctx)
}

// Download 下载导出文件
func (r *ExportController) Download(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	export, err := r.exportRecordService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusNotFound, err, map[string]any{
			"exportId": id,
		})
	}

	if export.Path == "" || export.Disk == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrFilePathRequired.Code)
	}

	storage, resp, ok := response.OpenStorageDisk(ctx, "export", export.Disk, map[string]any{
		"path": export.Path,
	})
	if !ok {
		return resp
	}

	content, err := storage.Get(export.Path)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, err, map[string]any{
			"disk": export.Disk,
			"path": export.Path,
		})
	}

	filename := export.Filename
	if filename == "" {
		filename = export.Path
	}

	contentType := "application/octet-stream"
	switch export.Extension {
	case "csv":
		contentType = "text/csv; charset=utf-8"
	case "xlsx", "xls":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}

	httpResp := ctx.Response().
		Header("Content-Type", contentType).
		Header("Content-Length", fmt.Sprintf("%d", len(content))).
		Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename)).
		Header("Cache-Control", "no-cache, no-store, must-revalidate").
		Header("Pragma", "no-cache").
		Header("Expires", "0")

	return httpResp.String(http.StatusOK, content)
}

type ExportBatchDestroyRequest struct {
	IDs []uint `json:"ids"`
}

// BatchDestroy 批量删除导出记录并删除源文件
func (r *ExportController) BatchDestroy(ctx http.Context) http.Response {
	var req ExportBatchDestroyRequest

	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}

	if len(req.IDs) == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDsRequired.Code)
	}

	ids := req.IDs

	if err := r.exportRecordService(ctx).BatchDeleteWithFiles(ids); err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, err, map[string]any{
			"ids": ids,
		})
	}

	return response.Success(ctx)
}

// StreamExportProgress SSE 实时推送导出任务进度
func (r *ExportController) StreamExportProgress(ctx http.Context) http.Response {
	exportID := helpers.GetUintRoute(ctx, "id")
	if exportID == 0 {
		exportID = helpers.GetUintQuery(ctx, "id", 0)
		if exportID == 0 {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
		}
	}

	interval := 1000
	if intervalStr := ctx.Request().Query("interval", ""); intervalStr != "" {
		if parsed, err := time.ParseDuration(intervalStr + "ms"); err == nil {
			interval = max(int(parsed.Milliseconds()), 500)
			if interval > 5000 {
				interval = 5000
			}
		}
	}

	writer := ctx.Response().Writer()
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")

	initMsg := map[string]any{
		"type":      "connected",
		"message":   "SSE连接已建立，开始监控导出任务进度",
		"export_id": exportID,
		"interval":  interval,
	}
	initData, _ := json.Marshal(initMsg)
	fmt.Fprintf(writer, "data: %s\n\n", string(initData))
	if flusher, ok := writer.(nethttp.Flusher); ok {
		flusher.Flush()
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	clientGone := ctx.Request().Origin().Context().Done()

	lastStatus := uint8(255)
	lastPath := ""

	exportRecordSvc := r.exportRecordService(ctx)

	for {
		select {
		case <-clientGone:
			return nil
		case <-ticker.C:
			export, err := exportRecordSvc.GetByID(exportID)
			if err != nil {
				errorMsg := map[string]any{
					"type":    "error",
					"message": "导出任务不存在或已删除",
					"error":   err.Error(),
				}
				errorData, _ := json.Marshal(errorMsg)
				fmt.Fprintf(writer, "data: %s\n\n", string(errorData))
				if flusher, ok := writer.(nethttp.Flusher); ok {
					flusher.Flush()
				}
				continue
			}

			if export.Status == lastStatus && export.Path == lastPath {
				if export.Status == models.ExportStatusSuccess {
					continue
				}
			}

			lastStatus = export.Status
			lastPath = export.Path

			message := map[string]any{
				"type":      "progress",
				"export_id": export.ID,
				"status":    export.Status,
				"timestamp": time.Now().Format(time.RFC3339),
			}

			switch export.Status {
			case models.ExportStatusSuccess:
				message["type"] = "completed"
				message["message"] = "导出任务已完成"
				message["status_text"] = "成功"

				payload := exportRecordSvc.ExportRecordToJSON(export)
				message["file_url"] = payload["file_url"]
				message["filename"] = export.Filename
				message["size"] = export.Size
			case models.ExportStatusFailed:
				message["type"] = "failed"
				message["message"] = "导出任务失败"
				message["status_text"] = "失败"
				if export.ErrorMsg != "" {
					message["error_msg"] = export.ErrorMsg
				}
			default:
				message["message"] = "导出任务处理中"
				message["status_text"] = "处理中"
			}

			if export.Path != "" && export.Status == models.ExportStatusProcessing {
				message["message"] = "正在生成导出文件"
				if export.Disk != "" {
					if storage, err := utils.StorageDisk(export.Disk); err == nil {
						if size, err := storage.Size(export.Path); err == nil {
							message["file_size"] = size
							if export.Size > 0 {
								progress := float64(size) / float64(export.Size) * 100
								if progress > 100 {
									progress = 100
								}
								message["progress"] = progress
							}
						}
					}
				}
			}

			messageData, err := json.Marshal(message)
			if err != nil {
				errorlog.RecordHTTP(ctx, "export", "Failed to marshal progress", map[string]any{
					"error": err.Error(),
				}, "Marshal progress error: %v", err)
				continue
			}

			fmt.Fprintf(writer, "data: %s\n\n", string(messageData))

			if flusher, ok := writer.(nethttp.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
