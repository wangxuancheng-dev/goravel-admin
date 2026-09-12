package services

import (
	"context"
	"strconv"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"
	"github.com/spf13/cast"

	appfacades "goravel/app/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/utils"
)

type RoleService interface {
	GetByID(id uint) (*models.Role, error)
	GetDetail(id uint) (*models.Role, error)
	GetList(filters RoleFilters, page, pageSize int) ([]models.Role, int64, error)
	Create(httpCtx http.Context, req *admin.RoleCreate) (*models.Role, error)
	Update(httpCtx http.Context, id uint, req *admin.RoleUpdate) (*models.Role, error)
	Delete(id uint) error
}

type RoleFilters struct {
	Name      string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

// BuildRoleFiltersFromHTTP reads list filters from query or body.
func BuildRoleFiltersFromHTTP(ctx http.Context) RoleFilters {
	return RoleFilters{
		Name:      ctx.Request().Input("name", ctx.Request().Query("name", "")),
		Status:    ctx.Request().Input("status", ctx.Request().Query("status", "")),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}
}

type RoleServiceImpl struct {
	ctx context.Context
}

func NewRoleService(ctx context.Context) RoleService {
	return &RoleServiceImpl{ctx: ctx}
}

func (s *RoleServiceImpl) GetByID(id uint) (*models.Role, error) {
	var role models.Role
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&role); err != nil {
		return nil, apperrors.ErrRoleNotFound.WithError(err)
	}
	return &role, nil
}

func (s *RoleServiceImpl) GetDetail(id uint) (*models.Role, error) {
	var role models.Role
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).With("Permissions").With("Menus").With("Departments").FirstOrFail(&role); err != nil {
		return nil, apperrors.ErrRoleNotFound.WithError(err)
	}
	return &role, nil
}

func (s *RoleServiceImpl) GetList(filters RoleFilters, page, pageSize int) ([]models.Role, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Role{})

	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "sort:asc,created_at:desc"
	}
	query = helpers.ApplySort(query, orderBy, "sort:asc,created_at:desc")

	var roles []models.Role
	var total int64
	if err := query.Paginate(page, pageSize, &roles, &total); err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

func (s *RoleServiceImpl) Create(httpCtx http.Context, req *admin.RoleCreate) (*models.Role, error) {
	if err := s.validateUnique(req.Name, req.Slug, 0); err != nil {
		return nil, err
	}

	dataScope := req.DataScope
	if dataScope == 0 {
		dataScope = models.DataScopeAll
	}

	role := &models.Role{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Status:      req.Status,
		Sort:        req.Sort,
		DataScope:   dataScope,
	}

	if err := appfacades.OrmQuery(s.ctx).Create(role); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	permissionIDs := s.parseIDsFromRequest(httpCtx, "permission_ids")
	if len(permissionIDs) > 0 {
		if err := s.syncPermissions(role, permissionIDs); err != nil {
			return nil, apperrors.ErrUpdateFailed.WithError(err)
		}
	}

	menuIDs := s.parseIDsFromRequest(httpCtx, "menu_ids")
	if len(menuIDs) > 0 {
		if err := s.syncMenus(role, menuIDs); err != nil {
			return nil, apperrors.ErrUpdateFailed.WithError(err)
		}
	}

	departmentIDs := s.parseIDsFromRequest(httpCtx, "department_ids")
	if role.DataScope == models.DataScopeCustom {
		if err := s.syncDepartments(role, departmentIDs); err != nil {
			return nil, apperrors.ErrUpdateFailed.WithError(err)
		}
	} else {
		_ = s.syncDepartments(role, nil)
	}

	return role, nil
}

func (s *RoleServiceImpl) Update(httpCtx http.Context, id uint, req *admin.RoleUpdate) (*models.Role, error) {
	role, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	isProtected := s.isProtectedRole(role.Slug)
	allInputs := httpCtx.Request().All()

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Slug != nil {
		if *req.Slug != role.Slug {
			if isProtected {
				return nil, apperrors.ErrRoleProtectedCannotModifySlug
			}
			role.Slug = *req.Slug
		}
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Status != nil {
		if isProtected && *req.Status == 0 {
			return nil, apperrors.ErrRoleProtectedCannotDisable
		}
		role.Status = *req.Status
	}
	if req.Sort != nil {
		role.Sort = *req.Sort
	}
	if isProtected {
		role.DataScope = models.DataScopeAll
	} else if req.DataScope != nil {
		role.DataScope = *req.DataScope
		if role.DataScope == 0 {
			role.DataScope = models.DataScopeAll
		}
	}

	if err := s.validateUnique(role.Name, role.Slug, role.ID); err != nil {
		return nil, err
	}
	if err := appfacades.OrmQuery(s.ctx).Save(role); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}

	if !isProtected {
		if _, exists := allInputs["permission_ids"]; exists {
			permissionIDs := s.parseIDsFromRequest(httpCtx, "permission_ids")
			if err := s.syncPermissions(role, permissionIDs); err != nil {
				return nil, apperrors.ErrUpdateFailed.WithError(err)
			}
		}
		if _, exists := allInputs["menu_ids"]; exists {
			menuIDs := s.parseIDsFromRequest(httpCtx, "menu_ids")
			if err := s.syncMenus(role, menuIDs); err != nil {
				return nil, apperrors.ErrUpdateFailed.WithError(err)
			}
		}
		if _, exists := allInputs["department_ids"]; exists || req.DataScope != nil {
			departmentIDs := s.parseIDsFromRequest(httpCtx, "department_ids")
			if role.DataScope == models.DataScopeCustom {
				if err := s.syncDepartments(role, departmentIDs); err != nil {
					return nil, apperrors.ErrUpdateFailed.WithError(err)
				}
			} else {
				_ = s.syncDepartments(role, nil)
			}
		}
	} else {
		_ = s.syncDepartments(role, nil)
	}

	return role, nil
}

func (s *RoleServiceImpl) Delete(id uint) error {
	role, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if s.isProtectedRole(role.Slug) {
		return apperrors.ErrRoleProtectedCannotDelete
	}
	if _, err := appfacades.OrmQuery(s.ctx).Delete(role); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *RoleServiceImpl) validateUnique(name, slug string, excludeID uint) error {
	if name != "" {
		exists, err := utils.ExistsColumnValue(s.ctx, "roles", &models.Role{}, utils.UniqueReuseAllow, "name", name, excludeID)
		if err != nil {
			return apperrors.ErrCreateFailed.WithError(err)
		}
		if exists {
			return apperrors.ErrRoleNameExists
		}
	}
	if slug != "" {
		exists, err := utils.ExistsColumnValue(s.ctx, "roles", &models.Role{}, utils.UniqueReuseAllow, "slug", slug, excludeID)
		if err != nil {
			return apperrors.ErrCreateFailed.WithError(err)
		}
		if exists {
			return apperrors.ErrRoleSlugExists
		}
	}
	return nil
}

func (s *RoleServiceImpl) syncPermissions(role *models.Role, permissionIDs []uint) error {
	var permissions []models.Permission
	if len(permissionIDs) > 0 {
		if err := appfacades.OrmQuery(s.ctx).Where("id IN ?", permissionIDs).Find(&permissions); err != nil {
			return err
		}
	}
	return appfacades.OrmQuery(s.ctx).Model(role).Association("Permissions").Replace(permissions)
}

func (s *RoleServiceImpl) syncMenus(role *models.Role, menuIDs []uint) error {
	var menus []models.Menu
	if len(menuIDs) > 0 {
		if err := appfacades.OrmQuery(s.ctx).Where("id IN ?", menuIDs).Find(&menus); err != nil {
			return err
		}
	}
	return appfacades.OrmQuery(s.ctx).Model(role).Association("Menus").Replace(menus)
}

func (s *RoleServiceImpl) syncDepartments(role *models.Role, departmentIDs []uint) error {
	var departments []models.Department
	if len(departmentIDs) > 0 {
		if err := appfacades.OrmQuery(s.ctx).Where("id IN ?", departmentIDs).Find(&departments); err != nil {
			return err
		}
	}
	return appfacades.OrmQuery(s.ctx).Model(role).Association("Departments").Replace(departments)
}

func (s *RoleServiceImpl) parseIDsFromRequest(ctx http.Context, key string) []uint {
	if ctx == nil {
		return nil
	}
	all := ctx.Request().All()
	if raw, ok := all[key]; ok && raw != nil {
		switch v := raw.(type) {
		case []any:
			ids := make([]uint, 0, len(v))
			for _, item := range v {
				if id := cast.ToUint(item); id > 0 {
					ids = append(ids, id)
				}
			}
			return ids
		case []uint:
			return v
		case []int:
			ids := make([]uint, 0, len(v))
			for _, item := range v {
				if item > 0 {
					ids = append(ids, uint(item))
				}
			}
			return ids
		}
	}

	var ids []uint
	if idsStr := ctx.Request().Input(key); idsStr != "" {
		for _, idStr := range ctx.Request().InputArray(key) {
			if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
				ids = append(ids, uint(id))
			}
		}
	}
	return ids
}

func (s *RoleServiceImpl) isProtectedRole(roleSlug string) bool {
	protectedSlugsStr := facades.Config().GetString("role.protected_slugs", "super-admin")
	for _, protectedSlug := range parseProtectedRoleSlugs(protectedSlugsStr) {
		if roleSlug == protectedSlug {
			return true
		}
	}
	return false
}

func parseProtectedRoleSlugs(slugsStr string) []string {
	var slugs []string
	if slugsStr == "" {
		return slugs
	}

	parts := str.Of(slugsStr).Split(",")
	for _, part := range parts {
		part = str.Of(part).Trim().String()
		if !str.Of(part).IsEmpty() {
			slugs = append(slugs, part)
		}
	}

	return slugs
}
