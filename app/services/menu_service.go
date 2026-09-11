package services

import (
	"context"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	admin "goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/utils"
)

type MenuService interface {
	GetByID(id uint) (*models.Menu, error)
	// GetTree returns the full menu tree for menu management (no module/monitor filters).
	GetTree() ([]utils.MenuTreeItem, error)
	// GetTreeForAdmin returns a filtered tree for role forms / dropdowns (module + monitor).
	GetTreeForAdmin(adminID uint) ([]utils.MenuTreeItem, error)
	ValidateSlugUnique(slug string, excludeID uint) error
	Create(req *admin.MenuCreate) (*models.Menu, error)
	Update(httpCtx http.Context, id uint, req *admin.MenuUpdate) (*models.Menu, error)
	Delete(id uint) error
}

type MenuServiceImpl struct {
	ctx context.Context
}

func NewMenuService(ctx context.Context) MenuService {
	return &MenuServiceImpl{ctx: ctx}
}

func (s *MenuServiceImpl) treeService() TreeService {
	return NewTreeServiceImpl(s.ctx)
}

func (s *MenuServiceImpl) GetByID(id uint) (*models.Menu, error) {
	var menu models.Menu
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&menu); err != nil {
		return nil, apperrors.ErrMenuNotFound.WithError(err)
	}
	return &menu, nil
}

func (s *MenuServiceImpl) GetTree() ([]utils.MenuTreeItem, error) {
	menus, err := s.treeService().BuildMenuTree(0)
	if err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}
	return utils.ConvertMenuTree(menus), nil
}

func (s *MenuServiceImpl) GetTreeForAdmin(adminID uint) ([]utils.MenuTreeItem, error) {
	menus, err := s.treeService().BuildMenuTree(0)
	if err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}
	menus = s.applyMenuTreeFilters(adminID, menus)
	return utils.ConvertMenuTree(menus), nil
}

func (s *MenuServiceImpl) ValidateSlugUnique(slug string, excludeID uint) error {
	if slug == "" {
		return nil
	}
	exists, err := utils.ExistsColumnValue(s.ctx, "menus", &models.Menu{}, utils.UniqueReuseAllow, "slug", slug, excludeID)
	if err != nil {
		return apperrors.ErrCreateFailed.WithError(err)
	}
	if exists {
		return apperrors.ErrMenuSlugExists
	}
	return nil
}

func (s *MenuServiceImpl) Create(req *admin.MenuCreate) (*models.Menu, error) {
	if err := s.ValidateSlugUnique(req.Slug, 0); err != nil {
		return nil, err
	}

	linkType := req.LinkType
	openType := req.OpenType
	if linkType == 0 {
		linkType = 1
	}
	if openType == 0 {
		openType = 1
	}

	menu := &models.Menu{
		ParentID:   req.ParentID,
		Title:      req.Title,
		Slug:       req.Slug,
		Icon:       req.Icon,
		Path:       req.Path,
		Component:  req.Component,
		Permission: req.Permission,
		Type:       req.Type,
		Status:     req.Status,
		Sort:       req.Sort,
		IsHidden:   req.IsHidden,
		LinkType:   linkType,
		OpenType:   openType,
		NoCache:    req.NoCache,
	}

	if err := appfacades.OrmQuery(s.ctx).Create(menu); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return menu, nil
}

func (s *MenuServiceImpl) Update(httpCtx http.Context, id uint, req *admin.MenuUpdate) (*models.Menu, error) {
	menu, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	allInputs := httpCtx.Request().All()

	if req.Title != nil {
		menu.Title = *req.Title
	}
	if req.Slug != nil {
		menu.Slug = *req.Slug
	}
	if req.ParentID != nil {
		menu.ParentID = *req.ParentID
	} else if val, exists := allInputs["parent_id"]; exists && val == nil {
		// Vue form sends null for top-level parent.
		menu.ParentID = 0
	}
	if req.Icon != nil {
		menu.Icon = *req.Icon
	}
	if req.Path != nil {
		menu.Path = *req.Path
	}
	if req.Component != nil {
		menu.Component = *req.Component
	}
	if req.Permission != nil {
		menu.Permission = *req.Permission
	}
	if req.Type != nil {
		menu.Type = *req.Type
	}
	if req.Status != nil {
		menu.Status = *req.Status
	}
	if req.Sort != nil {
		menu.Sort = *req.Sort
	}
	if req.IsHidden != nil {
		menu.IsHidden = *req.IsHidden
	}
	if req.LinkType != nil {
		menu.LinkType = *req.LinkType
	}
	if req.OpenType != nil {
		menu.OpenType = *req.OpenType
	}
	if req.NoCache != nil {
		menu.NoCache = *req.NoCache
	}

	if err := s.ValidateSlugUnique(menu.Slug, menu.ID); err != nil {
		return nil, err
	}
	if err := appfacades.OrmQuery(s.ctx).Save(menu); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	return menu, nil
}

func (s *MenuServiceImpl) Delete(id uint) error {
	menu, err := s.GetByID(id)
	if err != nil {
		return err
	}

	hasChildren, err := s.treeService().HasMenuChildren(id)
	if err != nil {
		return apperrors.ErrQueryFailed.WithError(err)
	}
	if hasChildren {
		return apperrors.ErrMenuHasChildren
	}

	if _, err := appfacades.OrmQuery(s.ctx).Delete(menu); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *MenuServiceImpl) applyMenuTreeFilters(adminID uint, menus []models.Menu) []models.Menu {
	developerIDsStr := facades.Config().GetString("admin.developer_ids", "2")
	isDeveloper := isDeveloperAdmin(adminID, developerIDsStr)

	monitorHidden := facades.Config().GetString("admin.monitor_hidden", "")
	if monitorHidden != "" && monitorHidden != "0" && !isDeveloper {
		menus = filterMonitorMenu(menus)
	}
	return utils.FilterTreeMenusByModule(menus)
}

func isDeveloperAdmin(adminID uint, developerIDsStr string) bool {
	if developerIDsStr == "" {
		return false
	}
	parts := str.Of(developerIDsStr).Split(",")
	for _, part := range parts {
		part = str.Of(part).Trim().String()
		if !str.Of(part).IsEmpty() {
			if id := cast.ToUint(part); id > 0 && id == adminID {
				return true
			}
		}
	}
	return false
}

func filterMonitorMenu(menus []models.Menu) []models.Menu {
	var filteredMenus []models.Menu
	for _, menu := range menus {
		if menu.Slug != "monitor" && menu.Slug != "monitor-center" {
			if len(menu.Children) > 0 {
				menu.Children = filterMonitorMenu(menu.Children)
			}
			filteredMenus = append(filteredMenus, menu)
		}
	}
	return filteredMenus
}
