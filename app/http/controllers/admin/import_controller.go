package admin

import (
	"fmt"
	"path"
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/models"
	"goravel/app/services"
)

type ImportController struct{}

func NewImportController() *ImportController {
	return &ImportController{}
}

func (c *ImportController) importRecordService(ctx http.Context) services.ImportRecordService {
	return services.NewImportRecordService(ctx)
}

// Show returns import task status for the current admin (or super).
func (c *ImportController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, "id_required")
	}

	record, err := c.importRecordService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if resp := ForbidUnlessOwnerOrSuper(ctx, record.AdminID); resp != nil {
		return resp
	}

	errorFileURL := ""
	if record.ErrorFilePath != "" {
		errorFileURL = fmt.Sprintf("/api/admin/imports/%d/error-file", record.ID)
	}

	return response.Success(ctx, http.Json{
		"id":             record.ID,
		"type":           record.Type,
		"status":         record.Status,
		"status_text":    importStatusText(ctx, record.Status),
		"total_rows":     record.TotalRows,
		"success_rows":   record.SuccessRows,
		"failed_rows":    record.FailedRows,
		"error_file_url": errorFileURL,
		"error_msg":      record.ErrorMsg,
	})
}

// DownloadErrorFile downloads the failed-rows CSV for an import.
func (c *ImportController) DownloadErrorFile(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, "id_required")
	}

	record, err := c.importRecordService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if resp := ForbidUnlessOwnerOrSuper(ctx, record.AdminID); resp != nil {
		return resp
	}
	if strings.TrimSpace(record.ErrorFilePath) == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrFilePathRequired.Code)
	}

	disk := record.Disk
	if disk == "" {
		disk = "local"
	}
	storage, resp, ok := response.OpenStorageDisk(ctx, "import", disk, map[string]any{
		"path": record.ErrorFilePath,
	})
	if !ok {
		return resp
	}
	content, err := storage.Get(record.ErrorFilePath)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, err, map[string]any{
			"path": record.ErrorFilePath,
		})
	}

	filename := "import_errors.csv"
	if base := path.Base(record.ErrorFilePath); base != "" && base != "." {
		filename = base
	}

	return ctx.Response().
		Header("Content-Type", "text/csv; charset=utf-8").
		Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename)).
		String(http.StatusOK, content)
}

func importStatusText(ctx http.Context, status uint8) string {
	switch status {
	case models.ImportStatusProcessing:
		return trans.Get(ctx, "processing")
	case models.ImportStatusSuccess:
		return trans.Get(ctx, "success")
	case models.ImportStatusFailed:
		return trans.Get(ctx, "failed")
	default:
		return trans.Get(ctx, "unknown")
	}
}
