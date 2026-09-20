package utils

import (
	"context"
	"strings"

	appfacades "goravel/app/facades"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/support/str"

	"goravel/app/models"
)

// GetOperationTitleFromContext 从 context 中获取操作标题
// 优先使用权限标识（permission_slug），如果没有权限标识，则根据路径和方法生成默认标题
func GetOperationTitleFromContext(ctx http.Context) string {
	if ctx == nil {
		return "operation.unknown"
	}

	// 优先从 context 中获取权限标识（由权限中间件设置）
	permissionSlugValue := ctx.Value("permission_slug")
	if permissionSlugValue != nil {
		if permissionSlug, ok := permissionSlugValue.(string); ok && permissionSlug != "" {
			// 直接返回权限标识，前端多语言文件中已有对应翻译
			return permissionSlug
		}
	}

	// 如果没有权限标识，尝试从权限表中查询匹配的权限
	method := ctx.Request().Method()
	path := ctx.Request().Path()

	// 先从权限表中查询匹配的权限
	permissionSlug := findPermissionSlugFromDB(ctx, method, path)
	if permissionSlug != "" {
		return permissionSlug
	}

	// 如果权限表中没有找到，根据路径和方法生成默认标题
	defaultTitle := generateDefaultTitle(method, path)
	if defaultTitle != "" {
		return defaultTitle
	}

	// 无法生成标题时，返回未知操作
	return "operation.unknown"
}

// generateDefaultTitle 根据方法和路径生成默认操作标题
func generateDefaultTitle(method, path string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	pathStr := str.Of(path)

	// 分片上传相关（与权限配置中的 slug 保持一致）
	if pathStr.Contains("/attachments/chunk") {
		if method == "POST" || method == "GET" {
			// 权限配置中的 slug 是 attachment.chunk
			return "attachment.chunk"
		}
	}

	// 附件上传
	if pathStr.Contains("/attachments/upload") && method == "POST" {
		return "attachment.upload"
	}

	// 附件删除
	if pathStr.Contains("/attachments/") && pathStr.EndsWith("/batch-delete") && method == "POST" {
		return "attachment.batch_delete"
	}
	if pathStr.Contains("/attachments/") && method == "DELETE" {
		return "attachment.destroy"
	}

	// 附件更新显示名称
	if pathStr.Contains("/attachments/") && pathStr.EndsWith("/display-name") && method == "PUT" {
		return "attachment.update_display_name"
	}

	// 导出下载
	if pathStr.Contains("/exports/") && pathStr.EndsWith("/download") && method == "GET" {
		return "export.download"
	}

	// pprof 相关操作（观测采样）
	if pathStr.Contains("/observability/pprof/status") && method == "GET" {
		return "observability.pprof_status"
	}
	if pathStr.Contains("/observability/pprof/verify") && method == "POST" {
		return "observability.pprof_verify"
	}
	if pathStr.Contains("/observability/pprof/cpu-hotspots") && method == "POST" {
		return "observability.pprof_cpu_hotspots"
	}
	if pathStr.Contains("/observability/pprof/memory-hotspots") && method == "POST" {
		return "observability.pprof_memory_hotspots"
	}

	// 订单导入
	if pathStr.Contains("/orders/import") && method == "POST" {
		return "order.import"
	}

	// 订单导出
	if pathStr.Contains("/orders/export") && method == "POST" {
		return "order.export"
	}

	// Own Google Authenticator bind/unbind
	if pathStr.Contains("/google-authenticator/") {
		return "google_authenticator.manage"
	}

	// 管理员解绑谷歌验证码
	if pathStr.Contains("/admins/") && pathStr.EndsWith("/unbind-google-auth") && method == "POST" {
		return "admin.unbind_google_auth"
	}

	// 管理员重置谷歌验证码
	if pathStr.Contains("/admins/") && pathStr.EndsWith("/reset-google-auth") && method == "POST" {
		return "admin.reset_google_auth"
	}

	// 管理员重置密码
	if pathStr.Contains("/admins/") && pathStr.EndsWith("/password") && (method == "PUT" || method == "PATCH") {
		return "admin.password"
	}

	// 用户重置密码
	if pathStr.Contains("/users/") && pathStr.EndsWith("/password") && (method == "PUT" || method == "PATCH") {
		return "user.password"
	}

	// 更新个人资料
	if pathStr.EndsWith("/profile") && (method == "PUT" || method == "PATCH") {
		return "profile.update"
	}

	// 修改密码
	if pathStr.EndsWith("/password") && (method == "PUT" || method == "PATCH") {
		return "password.update"
	}

	// 批量删除（通用模式）
	if pathStr.EndsWith("/batch-delete") && method == "POST" {
		parts := pathStr.ChopStart("/api/admin/").Split("/")
		if len(parts) > 0 {
			module := str.Of(parts[0]).Replace("-", "_").String()
			return str.Of(module).Append(".batch_delete").String()
		}
	}

	// 清理操作（通用模式）
	if pathStr.EndsWith("/clean") && method == "POST" {
		parts := pathStr.ChopStart("/api/admin/").Split("/")
		if len(parts) > 0 {
			module := str.Of(parts[0]).Replace("-", "_").String()
			return str.Of(module).Append(".clean").String()
		}
	}

	// 标准 CRUD 操作（通用模式）
	parts := pathStr.ChopStart("/api/admin/").Split("/")
	if len(parts) >= 1 {
		module := str.Of(parts[0]).Replace("-", "_").String()
		switch method {
		case "POST":
			// 创建操作
			if len(parts) == 1 || (len(parts) == 2 && parts[1] != "batch-delete" && parts[1] != "clean") {
				return str.Of(module).Append(".store").String()
			}
		case "PUT", "PATCH":
			// 更新操作
			if len(parts) >= 2 {
				return str.Of(module).Append(".update").String()
			}
		case "DELETE":
			// 删除操作
			if len(parts) >= 2 {
				return str.Of(module).Append(".destroy").String()
			}
		}
	}

	return ""
}

// findPermissionSlugFromDB 从权限表中查询匹配的权限标识
// 优先匹配精确路径，然后匹配通配符路径
func findPermissionSlugFromDB(ctx context.Context, method, path string) string {
	var permissions []models.Permission

	query := appfacades.OrmQuery(ctx).Model(&models.Permission{}).
		Where("status", 1).
		Where("(method = ? OR method = '')", method)

	if err := query.Find(&permissions); err != nil {
		// 查询失败时返回空，使用默认逻辑
		return ""
	}

	// 优先匹配精确路径
	for _, perm := range permissions {
		if perm.Path == path {
			return perm.Slug
		}
	}

	// 然后匹配通配符路径
	bestSlug := ""
	bestScore := -1
	for _, perm := range permissions {
		if perm.Path != "" && perm.Path != path {
			if matchPermissionPath(perm.Path, path) {
				score := pathSpecificityScore(perm.Path)
				if score > bestScore {
					bestScore = score
					bestSlug = perm.Slug
				}
			}
		}
	}

	if bestSlug != "" {
		return bestSlug
	}

	return ""
}

// pathSpecificityScore 计算权限路径的具体程度分数。
// 分数越高表示路径越具体（越应优先匹配）。
func pathSpecificityScore(pattern string) int {
	if pattern == "" {
		return 0
	}
	// 越长越具体，同时惩罚通配符数量。
	score := len(pattern)
	for i := range pattern {
		if pattern[i] == '*' {
			score -= 10
		}
	}
	return score
}

// matchPermissionPath 路径匹配，支持通配符（与权限中间件中的 matchPath 逻辑一致）
func matchPermissionPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	// 如果模式不包含通配符，直接返回 false
	if !containsChar(pattern, '*') {
		return false
	}

	// 将模式按 * 分割成多个部分
	parts := splitPatternString(pattern)
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
	// 过滤掉 "*" 标记，只保留实际的部分
	var actualParts []string
	for _, part := range parts {
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
	remainingPath := path[len(firstPart) : len(path)-len(lastPart)]
	// 确保中间部分不为空（至少有一个字符，通常是数字ID）
	return len(remainingPath) > 0
}

// containsChar 检查字符串是否包含指定字符
func containsChar(s string, c byte) bool {
	for i := range s {
		if s[i] == c {
			return true
		}
	}
	return false
}

// splitPatternString 按 * 分割模式字符串
func splitPatternString(pattern string) []string {
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
