package services

import (
	"context"

	"encoding/csv"
	"fmt"
	"strings"

	"github.com/spf13/cast"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	appfacades "goravel/app/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/rbac"
)

type ArticleService interface {
	GetByID(id uint) (*models.Article, error)
	GetList(filters ArticleFilters, page, pageSize int) ([]models.Article, int64, error)

	GetAllArticleForExport(filters ArticleFilters) ([]models.Article, error)

	ImportFromCSV(csvContent string) (*ImportResult, error)

	Create(req *admin.ArticleCreate) (*models.Article, error)

	Update(id uint, req *admin.ArticleUpdate) (*models.Article, error)

	Delete(id uint) error
}

type ArticleFilters struct {
	AdminId        string
	Title          string
	Content        string
	Status         string
	CreatedAt      string
	CreatedAtStart string
	CreatedAtEnd   string
	UpdatedAt      string
	UpdatedAtStart string
	UpdatedAtEnd   string
}

// BuildArticleFiltersFromHTTP is shared by list/export and reads filters from GET query or POST body.
func BuildArticleFiltersFromHTTP(ctx http.Context) ArticleFilters {
	return ArticleFilters{
		AdminId:        ctx.Request().Input("admin_id", ctx.Request().Query("admin_id", "")),
		Title:          ctx.Request().Input("title", ctx.Request().Query("title", "")),
		Content:        ctx.Request().Input("content", ctx.Request().Query("content", "")),
		Status:         ctx.Request().Input("status", ctx.Request().Query("status", "")),
		CreatedAt:      ctx.Request().Input("created_at", ctx.Request().Query("created_at", "")),
		CreatedAtStart: helpers.GetTimeInputOrQueryParam(ctx, "created_at_start"),
		CreatedAtEnd:   helpers.GetTimeInputOrQueryParam(ctx, "created_at_end"),
		UpdatedAt:      ctx.Request().Input("updated_at", ctx.Request().Query("updated_at", "")),
		UpdatedAtStart: helpers.GetTimeInputOrQueryParam(ctx, "updated_at_start"),
		UpdatedAtEnd:   helpers.GetTimeInputOrQueryParam(ctx, "updated_at_end"),
	}
}

type ArticleServiceImpl struct {
	ctx context.Context
}

func NewArticleService(ctx context.Context) ArticleService {
	return &ArticleServiceImpl{ctx: ctx}
}

func (s *ArticleServiceImpl) withRelations(query orm.Query) orm.Query {
	query = query.With("Admin")
	return query
}

// BuildArticleQuery builds the Article query shared by list/export.
func BuildArticleQuery(ctx context.Context, filters ArticleFilters) orm.Query {
	query := appfacades.OrmQuery(ctx).Model(&models.Article{})
	if filters.AdminId != "" {
		query = query.Where("admin_id = ?", filters.AdminId)
	}
	if filters.Title != "" {
		query = query.Where("title LIKE ?", "%"+filters.Title+"%")
	}
	if filters.Content != "" {
		query = query.Where("content LIKE ?", "%"+filters.Content+"%")
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.CreatedAtStart != "" {
		query = query.Where("created_at >= ?", filters.CreatedAtStart)
	}
	if filters.CreatedAtEnd != "" {
		query = query.Where("created_at <= ?", filters.CreatedAtEnd)
	}
	if filters.CreatedAt != "" {
		query = query.Where("created_at = ?", filters.CreatedAt)
	}
	if filters.UpdatedAtStart != "" {
		query = query.Where("updated_at >= ?", filters.UpdatedAtStart)
	}
	if filters.UpdatedAtEnd != "" {
		query = query.Where("updated_at <= ?", filters.UpdatedAtEnd)
	}
	if filters.UpdatedAt != "" {
		query = query.Where("updated_at = ?", filters.UpdatedAt)
	}

	query = rbac.ApplyDataScope(ctx, query, rbac.DataScopeApplyOpts{Mode: rbac.DataScopeModeAdmin, AdminColumn: "admin_id"})
	return query
}

func (s *ArticleServiceImpl) GetByID(id uint) (*models.Article, error) {
	var item models.Article
	query := s.withRelations(appfacades.OrmQuery(s.ctx).Model(&models.Article{})).Where("id", id)
	if err := query.FirstOrFail(&item); err != nil {
		return nil, apperrors.ErrRecordNotFound.WithError(err)
	}
	if !rbac.CanAccessOwnedBy(s.ctx, item.AdminId) {
		return nil, apperrors.ErrForbidden
	}
	return &item, nil
}

func (s *ArticleServiceImpl) GetList(filters ArticleFilters, page, pageSize int) ([]models.Article, int64, error) {
	query := s.withRelations(BuildArticleQuery(s.ctx, filters))

	var list []models.Article
	var total int64
	if err := query.Order("id desc").Paginate(page, pageSize, &list, &total); err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (s *ArticleServiceImpl) GetAllArticleForExport(filters ArticleFilters) ([]models.Article, error) {
	query := s.withRelations(BuildArticleQuery(s.ctx, filters))

	var list []models.Article
	if err := query.Order("id desc").Find(&list); err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}

	return list, nil
}

// ImportFromCSV imports Article rows from CSV content.
// Header names should match field names (case-insensitive). Customize as needed.
func (s *ArticleServiceImpl) ImportFromCSV(csvContent string) (*ImportResult, error) {
	reader := csv.NewReader(strings.NewReader(csvContent))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, apperrors.ErrInvalidCSVFormat.WithError(err)
	}
	if len(records) < 2 {
		return nil, apperrors.ErrInvalidCSVFormat.WithMessage("CSV文件至少需要表头和数据行")
	}

	headerMap := make(map[string]int)
	for i, header := range records[0] {
		headerMap[strings.TrimSpace(strings.ToLower(header))] = i
	}

	result := &ImportResult{
		TotalRows: len(records) - 1,
		Errors:    []string{},
	}

	for rowIndex, row := range records[1:] {
		lineNo := rowIndex + 2
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			result.TotalRows--
			continue
		}

		item := &models.Article{}
		if idx, ok := headerMap["admin_id"]; ok && idx < len(row) {
			val := strings.TrimSpace(row[idx])
			item.AdminId = cast.ToUint(val)
		}
		if idx, ok := headerMap["title"]; ok && idx < len(row) {
			val := strings.TrimSpace(row[idx])
			item.Title = val
		}
		if idx, ok := headerMap["content"]; ok && idx < len(row) {
			val := strings.TrimSpace(row[idx])
			item.Content = val
		}
		if idx, ok := headerMap["status"]; ok && idx < len(row) {
			val := strings.TrimSpace(row[idx])
			item.Status = uint8(cast.ToUint(val))
		}

		if err := appfacades.OrmQuery(s.ctx).Create(item); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行：%v", lineNo, err))
			continue
		}
		result.SuccessCount++
	}

	return result, nil
}

func (s *ArticleServiceImpl) Create(req *admin.ArticleCreate) (*models.Article, error) {
	item := &models.Article{
		AdminId: uint(req.AdminId),
		Title:   req.Title,
		Content: req.Content,
		Status:  req.Status,
	}

	if err := appfacades.OrmQuery(s.ctx).Create(item); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return item, nil
}

func (s *ArticleServiceImpl) Update(id uint, req *admin.ArticleUpdate) (*models.Article, error) {
	item, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if req.AdminId != nil {
		item.AdminId = uint(*req.AdminId)
	}
	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.Content != nil {
		item.Content = *req.Content
	}
	if req.Status != nil {
		item.Status = *req.Status
	}

	if err := appfacades.OrmQuery(s.ctx).Save(item); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}

	return item, nil
}

func (s *ArticleServiceImpl) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}

	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.Article{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}
