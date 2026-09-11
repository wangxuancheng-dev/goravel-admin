package middleware

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils/errorlog"
	"goravel/app/utils/logger"
)

// Permission 权限验证中间件
func Permission() http.Middleware {
	return newMiddleware("permission", func(ctx http.Context) {
		// 从context中获取admin信息（由JWT中间件设置）
		adminValue := ctx.Value("admin")
		if adminValue == nil {
			response.Abort(ctx, http.StatusUnauthorized, "not_logged_in")
			return
		}

		admin, ok := adminValue.(models.Admin)
		if !ok {
			response.Abort(ctx, http.StatusUnauthorized, "not_logged_in")
			return
		}

		// 加载管理员的角色、权限等关联数据
		adminService := services.NewAdminServiceImpl(ctx)
		if err := adminService.LoadRelationsWithPermissions(&admin); err != nil {
			logger.ErrorfHTTP(ctx, "permission middleware load relations failed: %v", err)
			errorlog.RecordHTTP(ctx, "permission", "Failed to load admin relations with permissions", map[string]any{
				"error":    err.Error(),
				"admin_id": admin.ID,
				"path":     ctx.Request().Path(),
			}, "Load admin relations failed: %v", err)
			response.Abort(ctx, http.StatusInternalServerError, "load_permissions_failed")
			return
		}

		// 检查是否是超级管理员
		// 拥有 super-admin 角色的管理员（包括超级管理员和开发者管理员）都跳过权限“拦截”，但仍然参与权限匹配，用于生成操作标题
		isSuperAdmin := false
		for _, role := range admin.Roles {
			if role.Slug == "super-admin" && role.Status == 1 {
				isSuperAdmin = true
				break
			}
		}

		// 获取当前请求的方法和路径
		method := ctx.Request().Method()
		path := ctx.Request().Path()

		// 收集所有角色的权限（已通过预加载获取）
		var allPermissions []models.Permission
		for _, role := range admin.Roles {
			if role.Status == 1 {
				allPermissions = append(allPermissions, role.Permissions...)
			}
		}

		// 检查是否有权限，并记录匹配的权限标识
		hasPermission := false
		var matchedPermissionSlug string
		var menuDisabled bool
		for _, perm := range allPermissions {
			if perm.Status == 1 {
				// 检查方法匹配
				if perm.Method == "" || perm.Method == method {
					// 检查路径匹配（支持通配符）
					if perm.Path == "" || perm.Path == path || matchPath(perm.Path, path) {
						// 检查关联菜单的状态（如果权限关联了菜单）
						// 如果权限没有关联菜单（MenuID = 0），则允许访问
						// 如果权限关联了菜单，需要检查菜单状态是否为启用（status = 1）
						if perm.MenuID == 0 {
							// 权限没有关联菜单，允许访问
							hasPermission = true
							matchedPermissionSlug = perm.Slug
							break
						} else if perm.Menu.ID > 0 {
							// 权限关联了菜单，检查菜单状态
							if perm.Menu.Status == 1 {
								// 菜单状态为启用，允许访问
								hasPermission = true
								matchedPermissionSlug = perm.Slug
								break
							} else {
								// 菜单状态为关闭，记录但继续查找其他权限
								menuDisabled = true
							}
						}
						// 如果菜单没有加载（perm.Menu.ID == 0），为了安全起见，不允许访问
						// 这种情况应该很少发生，因为我们已经预加载了菜单
					}
				}
			}
		}

		// 非超级管理员且无匹配权限时拦截；超级管理员即使无匹配权限也放行
		if !hasPermission && !isSuperAdmin {
			// 如果是因为菜单状态为关闭而禁止访问，返回更具体的错误信息
			if menuDisabled {
				response.Abort(ctx, http.StatusForbidden, "menu_disabled")
			} else {
				response.Abort(ctx, http.StatusForbidden, "no_permission")
			}
			return
		}

		// 将匹配的权限标识存储到 context 中，供操作日志使用
		if matchedPermissionSlug != "" {
			ctx.WithValue("permission_slug", matchedPermissionSlug)
		}

		ctx.Request().Next()
	})
}

// matchPath 路径匹配，支持通配符
// 支持的模式：
// 1. 精确匹配：/api/admin/roles 匹配 /api/admin/roles
// 2. 末尾通配符：/api/admin/roles/* 匹配 /api/admin/roles/1
// 3. 中间通配符：/api/admin/attachments/*/display-name 匹配 /api/admin/attachments/1/display-name
func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	// 如果模式不包含通配符，直接返回 false
	if !contains(pattern, '*') {
		return false
	}

	// 将模式按 * 分割成多个部分
	parts := splitPattern(pattern)
	if len(parts) == 0 {
		return false
	}

	// 如果模式以 * 开头，需要特殊处理
	if pattern[0] == '*' {
		// 检查路径是否以模式的剩余部分结尾
		if len(parts) > 1 {
			suffix := parts[1]
			return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
		}
		return true
	}

	// 如果模式以 * 结尾
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		if len(path) >= len(prefix) {
			pathPrefix := path[:len(prefix)]
			if pathPrefix == prefix {
				// 如果前缀以 / 结尾，路径必须比前缀长（即后面还有内容）
				if len(prefix) > 0 && prefix[len(prefix)-1] == '/' {
					return len(path) > len(prefix)
				}
				// 如果前缀不以 / 结尾，路径可以等于或长于前缀
				return true
			}
		}
		return false
	}

	// 处理中间有通配符的情况，如 /api/admin/attachments/*/display-name
	// 将模式按 * 分割
	patternParts := splitPattern(pattern)
	if len(patternParts) < 2 {
		return false
	}

	// 过滤掉 "*" 标记，只保留实际的部分
	var actualParts []string
	for _, part := range patternParts {
		if part != "*" {
			actualParts = append(actualParts, part)
		}
	}

	if len(actualParts) < 2 {
		return false
	}

	// 检查路径是否以第一部分开头
	firstPart := actualParts[0]
	if len(path) < len(firstPart) || path[:len(firstPart)] != firstPart {
		return false
	}

	// 检查路径是否以最后一部分结尾
	lastPart := actualParts[len(actualParts)-1]
	if len(path) < len(lastPart) || path[len(path)-len(lastPart):] != lastPart {
		return false
	}

	// 检查中间部分是否存在（通配符匹配任意内容）
	// 路径应该是：firstPart + 任意内容 + lastPart
	remainingPath := path[len(firstPart) : len(path)-len(lastPart)]
	// 确保中间部分不为空（至少有一个字符，通常是数字ID）
	return len(remainingPath) > 0
}

// contains 检查字符串是否包含指定字符
func contains(s string, c byte) bool {
	for i := range s {
		if s[i] == c {
			return true
		}
	}
	return false
}

// splitPattern 按 * 分割模式字符串
func splitPattern(pattern string) []string {
	var parts []string
	var current strings.Builder

	for i := range pattern {
		if pattern[i] == '*' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			parts = append(parts, "*")
		} else {
			current.WriteByte(pattern[i])
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
