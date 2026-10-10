package services

import (
	"context"

	"github.com/goravel/framework/contracts/http"

	appfacades "goravel/app/facades"

	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/utils"
)

type PermissionService interface {
	GetByID(id uint) (*models.Permission, error)
	GetList(filters PermissionFilters, page, pageSize int) ([]models.Permission, int64, error)
	Create(req *admin.PermissionCreate) (*models.Permission, error)
	Update(id uint, req *admin.PermissionUpdate) (*models.Permission, error)
	Delete(id uint) error
}

type PermissionFilters struct {
	Name      string
	Slug      string
	Method    string
	Path      string
	Status    string
	MenuID    string
	StartTime string
	EndTime   string
	OrderBy   string
}

// BuildPermissionFiltersFromHTTP reads list filters from query or body.
func BuildPermissionFiltersFromHTTP(ctx http.Context) PermissionFilters {
	return PermissionFilters{
		Name:      ctx.Request().Input("name", ctx.Request().Query("name", "")),
		Slug:      ctx.Request().Input("slug", ctx.Request().Query("slug", "")),
		Method:    ctx.Request().Input("method", ctx.Request().Query("method", "")),
		Path:      ctx.Request().Input("path", ctx.Request().Query("path", "")),
		Status:    ctx.Request().Input("status", ctx.Request().Query("status", "")),
		MenuID:    ctx.Request().Input("menu_id", ctx.Request().Query("menu_id", "")),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}
}

type PermissionServiceImpl struct {
	ctx         context.Context
	treeService TreeService
}

func NewPermissionService(ctx context.Context) PermissionService {
	return &PermissionServiceImpl{
		ctx:         ctx,
		treeService: NewTreeServiceImpl(ctx),
	}
}

func (s *PermissionServiceImpl) GetByID(id uint) (*models.Permission, error) {
	var permission models.Permission
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).With("Menu").FirstOrFail(&permission); err != nil {
		return nil, apperrors.ErrPermissionNotFound.WithError(err)
	}
	return &permission, nil
}

func (s *PermissionServiceImpl) GetList(filters PermissionFilters, page, pageSize int) ([]models.Permission, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Permission{})

	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Slug != "" {
		query = query.Where("slug LIKE ?", "%"+filters.Slug+"%")
	}
	if filters.Path != "" {
		query = query.Where("path LIKE ?", "%"+filters.Path+"%")
	}
	if filters.Method != "" {
		query = query.Where("method", filters.Method)
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}
	if filters.MenuID != "" {
		menuIDUint := cast.ToUint(filters.MenuID)
		if menuIDUint > 0 {
			menuIDs, err := s.treeService.GetMenuChildrenIDs(menuIDUint)
			if err == nil && len(menuIDs) > 0 {
				idsAny := helpers.ConvertUintSliceToAny(menuIDs)
				query = query.WhereIn("menu_id", idsAny)
			} else {
				query = query.Where("1 = 0")
			}
		}
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "sort:asc,id:desc"
	}
	query = helpers.ApplySort(query, orderBy, "sort:asc,id:desc",
		"id", "name", "slug", "method", "path", "status", "sort", "menu_id", "created_at", "updated_at")

	var permissions []models.Permission
	var total int64
	if err := query.With("Menu").Paginate(page, pageSize, &permissions, &total); err != nil {
		return nil, 0, err
	}

	return permissions, total, nil
}

func (s *PermissionServiceImpl) validateUnique(name, slug string, excludeID uint) error {
	if name != "" {
		exists, err := utils.ExistsColumnValue(s.ctx, "permissions", &models.Permission{}, utils.UniqueReuseAllow, "name", name, excludeID)
		if err != nil {
			return apperrors.ErrCreateFailed.WithError(err)
		}
		if exists {
			return apperrors.ErrPermissionNameExists
		}
	}
	if slug != "" {
		exists, err := utils.ExistsColumnValue(s.ctx, "permissions", &models.Permission{}, utils.UniqueReuseAllow, "slug", slug, excludeID)
		if err != nil {
			return apperrors.ErrCreateFailed.WithError(err)
		}
		if exists {
			return apperrors.ErrPermissionSlugExists
		}
	}
	return nil
}

func resolveMethod(method, httpMethod string) string {
	if method != "" {
		return method
	}
	return httpMethod
}

func resolvePath(path, httpPath string) string {
	if path != "" {
		return path
	}
	return httpPath
}

func (s *PermissionServiceImpl) Create(req *admin.PermissionCreate) (*models.Permission, error) {
	method := resolveMethod(req.Method, req.HTTPMethod)
	path := resolvePath(req.Path, req.HTTPPath)

	exists, err := utils.ExistsColumnValueAny(s.ctx, "permissions", &models.Permission{}, utils.UniqueReuseAllow, 0, map[string]any{
		"name": req.Name,
		"slug": req.Slug,
	})
	if err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	if exists {
		return nil, apperrors.ErrPermissionNameOrSlugExists
	}

	permission := &models.Permission{
		Name:        req.Name,
		Slug:        req.Slug,
		Method:      method,
		Path:        path,
		Description: req.Description,
		Status:      req.Status,
		Sort:        req.Sort,
		MenuID:      req.MenuID,
	}

	if err := appfacades.OrmQuery(s.ctx).Create(permission); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	if err := appfacades.OrmQuery(s.ctx).Model(&models.Permission{}).Where("id", permission.ID).With("Menu").First(permission); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return permission, nil
}

func (s *PermissionServiceImpl) Update(id uint, req *admin.PermissionUpdate) (*models.Permission, error) {
	permission, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		permission.Name = *req.Name
	}
	if req.Slug != nil {
		permission.Slug = *req.Slug
	}
	if req.Method != nil {
		permission.Method = *req.Method
	} else if req.HTTPMethod != nil {
		permission.Method = *req.HTTPMethod
	}
	if req.Path != nil {
		permission.Path = *req.Path
	} else if req.HTTPPath != nil {
		permission.Path = *req.HTTPPath
	}
	if req.Description != nil {
		permission.Description = *req.Description
	}
	if req.Status != nil {
		permission.Status = *req.Status
	}
	if req.Sort != nil {
		permission.Sort = *req.Sort
	}
	if req.MenuID != nil {
		permission.MenuID = *req.MenuID
	}

	if err := s.validateUnique(permission.Name, permission.Slug, permission.ID); err != nil {
		return nil, err
	}
	if err := appfacades.OrmQuery(s.ctx).Save(permission); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	if err := appfacades.OrmQuery(s.ctx).Model(&models.Permission{}).Where("id", permission.ID).With("Menu").First(permission); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	return permission, nil
}

func (s *PermissionServiceImpl) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.Permission{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}
