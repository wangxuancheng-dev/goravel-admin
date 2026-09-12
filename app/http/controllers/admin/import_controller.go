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

// Index 导入记录列表
func (c *ImportController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.BuildImportRecordFiltersFromHTTP(ctx)

	svc := c.importRecordService(ctx)
	imports, total, err := svc.GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, err, nil)
	}

	result := make([]map[string]any, len(imports))
	for i := range imports {
		payload := svc.ImportRecordToJSON(&imports[i])
		payload["status_text"] = importStatusText(ctx, imports[i].Status)
		result[i] = payload
	}

	return response.Paginate(ctx, result, total, page, pageSize)
}

// Show returns import task status for the current admin (or scoped).
func (c *ImportController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, "id_required")
	}

	record, err := c.importRecordService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusNotFound, err, map[string]any{"id": id})
	}

	payload := c.importRecordService(ctx).ImportRecordToJSON(record)
	payload["status_text"] = importStatusText(ctx, record.Status)
	return response.Success(ctx, payload)
}

// Destroy 删除导入记录及关联文件
func (c *ImportController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	if err := c.importRecordService(ctx).DeleteWithFile(id); err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, err, map[string]any{
			"importId": id,
		})
	}
	return response.Success(ctx)
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
