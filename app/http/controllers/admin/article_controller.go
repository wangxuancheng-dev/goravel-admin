package admin

import (
	"goravel/app/jobs"
	"goravel/app/utils"

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
