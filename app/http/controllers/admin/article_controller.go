package admin

import (
	"goravel/app/jobs"

	"goravel/app/utils"

	"strings"

	"goravel/app/http/trans"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"

	"github.com/goravel/framework/contracts/http"
)

type ArticleController struct{}

func (c *ArticleController) buildArticleFilters(ctx http.Context) services.ArticleFilters {
	return services.BuildArticleFiltersFromHTTP(ctx)
}

func NewArticleController() *ArticleController {
	return &ArticleController{}
}

func (c *ArticleController) ArticleService(ctx http.Context) services.ArticleService {
	return services.NewArticleService(ctx)
}

// Index lists Article records.
func (c *ArticleController) Index(ctx http.Context) http.Response {

	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)

	filters := c.buildArticleFilters(ctx)

	list, total, err := c.ArticleService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "article", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})

}

// Show returns Article details.
func (c *ArticleController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	item, err := c.ArticleService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "article", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"article": item,
	})
}

// Store creates a new Article.
func (c *ArticleController) Store(ctx http.Context) http.Response {
	var req adminrequests.ArticleCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	item, err := c.ArticleService(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "article", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"article": item,
	})
}

// Update modifies an existing Article.
func (c *ArticleController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.ArticleUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	item, err := c.ArticleService(ctx).Update(id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "article", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"article": item,
	})
}

// Destroy deletes a Article.
func (c *ArticleController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.ArticleService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "article", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}

// Export exports Article records.
func (c *ArticleController) Export(ctx http.Context) http.Response {
	filters := c.buildArticleFilters(ctx)
	filtersMap := utils.ExportFiltersToMap(filters)
	result := EnqueueAsyncExport(ctx, EnqueueAsyncExportInput{
		LockResource: "articles",
		ExportType:   "articles",
		Filters:      filtersMap,
		Job:          &jobs.ExportArticles{},
	})
	if result.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUnauthorized.Code)
	}
	if result.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, apperrors.ErrGetLockFailed.Code)
	}
	if result.Err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, result.Err, nil)
	}

	return response.Success(ctx, http.Json{
		"export_id": result.ExportID,
		"message":   "queued",
	})
}

// Import imports Article records from CSV.
func (c *ArticleController) Import(ctx http.Context) http.Response {
	adminID, err := helpers.GetAdminIDFromContext(ctx)
	if err != nil {
		return response.Error(ctx, http.StatusUnauthorized, "unauthorized")
	}

	file, err := ctx.Request().File("file")
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, "file_required")
	}

	asyncFlag := strings.TrimSpace(ctx.Request().Query("async", ctx.Request().Input("async", "")))
	forceAsync := asyncFlag == "1" || strings.EqualFold(asyncFlag, "true")

	disk, path, filename, dataRows, saveErr := services.SaveUploadedCSVForAsync(ctx, file)
	if saveErr != nil {
		attrs := map[string]any{"filename": filename, "admin_id": adminID}
		fallback := http.StatusInternalServerError
		if _, ok := apperrors.GetBusinessError(saveErr); ok {
			fallback = http.StatusBadRequest
		}
		return HandleGeneratedServiceError(ctx, "import", fallback, saveErr, attrs)
	}
	if forceAsync || dataRows >= services.AsyncImportRowThreshold {
		return c.enqueueArticleImport(ctx, disk, path, dataRows)
	}

	storage, storErr := utils.StorageDisk(disk)
	if storErr != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, storErr, nil)
	}
	csvContent, getErr := storage.Get(path)
	if getErr != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, getErr, nil)
	}
	defer func() { _ = storage.Delete(path) }()

	result, err := c.ArticleService(ctx).ImportFromCSV(csvContent)
	if err != nil {
		attrs := map[string]any{"filename": filename, "admin_id": adminID}
		fallback := http.StatusInternalServerError
		if _, ok := apperrors.GetBusinessError(err); ok {
			fallback = http.StatusBadRequest
		}
		return HandleGeneratedServiceError(ctx, "import", fallback, err, attrs)
	}

	return response.Success(ctx, http.Json{
		"async":         false,
		"total_rows":    result.TotalRows,
		"success_count": result.SuccessCount,
		"failed_count":  result.FailedCount,
		"errors":        result.Errors,
		"message":       trans.Get(ctx, "import_success"),
	})
}
func (c *ArticleController) enqueueArticleImport(ctx http.Context, disk, path string, totalRows int) http.Response {
	result := EnqueueAsyncImport(ctx, EnqueueAsyncImportInput{
		LockResource: "articles_import",
		ImportType:   "articles",
		Disk:         disk,
		Path:         path,
		TotalRows:    totalRows,
		Job:          &jobs.ImportArticles{},
	})
	if result.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, "unauthorized")
	}
	if result.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, "too_many_requests")
	}
	if result.Err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, result.Err, map[string]any{
			"import_id": result.ImportID,
		})
	}
	return response.Success(ctx, http.Json{
		"async":     true,
		"import_id": result.ImportID,
		"message":   trans.Get(ctx, "queued"),
	})
}
