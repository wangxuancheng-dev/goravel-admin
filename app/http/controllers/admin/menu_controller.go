package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
)

type MenuController struct{}

func NewMenuController() *MenuController {
	return &MenuController{}
}

func (c *MenuController) MenuService(ctx http.Context) services.MenuService {
	return services.NewMenuService(ctx)
}

func (c *MenuController) extractAdminID(ctx http.Context) uint {
	adminValue := ctx.Value("admin")
	if adminValue == nil {
		return 0
	}
	if admin, ok := adminValue.(models.Admin); ok {
		return admin.ID
	}
	if adminPtr, ok := adminValue.(*models.Admin); ok && adminPtr != nil {
		return adminPtr.ID
	}
	return 0
}

func (c *MenuController) menusSuccess(ctx http.Context, treeData any) http.Response {
	return response.Success(ctx, http.Json{
		"menus": treeData,
		"list":  treeData,
	})
}

// Tree 菜单树（仅登录即可访问；按模块/监控开关过滤，用于角色/权限表单等）
func (c *MenuController) Tree(ctx http.Context) http.Response {
	treeData, err := c.MenuService(ctx).GetTreeForAdmin(c.extractAdminID(ctx))
	if err != nil {
		return HandleGeneratedServiceError(ctx, "menu", http.StatusInternalServerError, err, nil)
	}
	return c.menusSuccess(ctx, treeData)
}

// Index 菜单管理列表：返回完整树（含禁用、模块未开启的菜单），便于管理
func (c *MenuController) Index(ctx http.Context) http.Response {
	treeData, err := c.MenuService(ctx).GetTree()
	if err != nil {
		return HandleGeneratedServiceError(ctx, "menu", http.StatusInternalServerError, err, nil)
	}
	return c.menusSuccess(ctx, treeData)
}

func (c *MenuController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	menu, err := c.MenuService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "menu", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{
		"menu": *menu,
	})
}

func (c *MenuController) Store(ctx http.Context) http.Response {
	var req adminrequests.MenuCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	menu, err := c.MenuService(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "menu", http.StatusInternalServerError, err, map[string]any{
			"title": req.Title,
			"slug":  req.Slug,
		})
	}

	return response.Success(ctx, http.Json{
		"menu": *menu,
	})
}

func (c *MenuController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.MenuUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	menu, err := c.MenuService(ctx).Update(ctx, id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "menu", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"menu": *menu,
	})
}

func (c *MenuController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.MenuService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "menu", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}
