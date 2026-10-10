package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"

	appfacades "goravel/app/facades"
	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
)

type AttachmentCategoryService interface {
	GetByID(id uint) (*models.AttachmentCategory, error)
	GetUncategorized() (*models.AttachmentCategory, error)
	GetUncategorizedID() (uint, error)
	GetList(filters AttachmentCategoryFilters, page, pageSize int) ([]models.AttachmentCategory, int64, error)
	Create(req *admin.AttachmentCategoryCreate) (*models.AttachmentCategory, error)
	Update(id uint, req *admin.AttachmentCategoryUpdate) (*models.AttachmentCategory, error)
	Delete(id uint) error
}

type AttachmentCategoryFilters struct {
	Name    string
	Status  string
	OrderBy string
}

// BuildAttachmentCategoryFiltersFromHTTP reads list filters from query or body.
func BuildAttachmentCategoryFiltersFromHTTP(ctx http.Context) AttachmentCategoryFilters {
	return AttachmentCategoryFilters{
		Name:    ctx.Request().Input("name", ctx.Request().Query("name", "")),
		Status:  ctx.Request().Input("status", ctx.Request().Query("status", "")),
		OrderBy: ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}
}

type AttachmentCategoryServiceImpl struct {
	ctx context.Context
}

func NewAttachmentCategoryService(ctx context.Context) AttachmentCategoryService {
	return &AttachmentCategoryServiceImpl{ctx: ctx}
}

func (s *AttachmentCategoryServiceImpl) GetByID(id uint) (*models.AttachmentCategory, error) {
	var category models.AttachmentCategory
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&category); err != nil {
		return nil, apperrors.ErrAttachmentCategoryNotFound.WithError(err)
	}
	return &category, nil
}

func (s *AttachmentCategoryServiceImpl) GetUncategorized() (*models.AttachmentCategory, error) {
	var category models.AttachmentCategory
	if err := appfacades.OrmQuery(s.ctx).Where("slug", models.AttachmentCategorySlugUncategorized).FirstOrFail(&category); err != nil {
		return nil, apperrors.ErrAttachmentCategoryNotFound.WithError(err)
	}
	return &category, nil
}

func (s *AttachmentCategoryServiceImpl) GetUncategorizedID() (uint, error) {
	category, err := s.GetUncategorized()
	if err != nil {
		return 0, err
	}
	return category.ID, nil
}

func (s *AttachmentCategoryServiceImpl) GetList(filters AttachmentCategoryFilters, page, pageSize int) ([]models.AttachmentCategory, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.AttachmentCategory{})
	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}
	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "is_system:desc,sort:asc,id:asc"
	}
	query = helpers.ApplySort(query, orderBy, "is_system:desc,sort:asc,id:asc",
		"id", "name", "parent_id", "status", "sort", "is_system", "created_at", "updated_at")
	var list []models.AttachmentCategory
	var total int64
	if err := query.Paginate(page, pageSize, &list, &total); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *AttachmentCategoryServiceImpl) Create(req *admin.AttachmentCategoryCreate) (*models.AttachmentCategory, error) {
	name := strings.TrimSpace(req.Name)
	slug := fmt.Sprintf("custom_%d", time.Now().UnixNano())
	category := &models.AttachmentCategory{
		Name:     name,
		Slug:     slug,
		Status:   req.Status,
		IsSystem: 0,
		Sort:     req.Sort,
		Remark:   req.Remark,
	}
	if err := appfacades.OrmQuery(s.ctx).Create(category); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	return category, nil
}

func (s *AttachmentCategoryServiceImpl) Update(id uint, req *admin.AttachmentCategoryUpdate) (*models.AttachmentCategory, error) {
	category, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		category.Name = *req.Name
	}
	if req.Status != nil {
		category.Status = *req.Status
	}
	if req.Sort != nil {
		category.Sort = *req.Sort
	}
	if req.Remark != nil {
		category.Remark = *req.Remark
	}
	if category.IsSystem == 1 {
		category.Slug = models.AttachmentCategorySlugUncategorized
		category.Status = 1
		category.IsSystem = 1
	}
	if err := appfacades.OrmQuery(s.ctx).Save(category); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	return category, nil
}

func (s *AttachmentCategoryServiceImpl) Delete(id uint) error {
	category, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if category.IsSystem == 1 {
		return apperrors.ErrAttachmentCategoryProtectedCannotDelete
	}
	uncategorizedID, err := s.GetUncategorizedID()
	if err != nil {
		return err
	}
	if _, err := appfacades.OrmQuery(s.ctx).Model(&models.Attachment{}).
		Where("category_id", category.ID).
		Update("category_id", uncategorizedID); err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}
	if _, err := appfacades.OrmQuery(s.ctx).Delete(category); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}
