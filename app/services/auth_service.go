package services

import (
	"context"
	"encoding/json"
	appfacades "goravel/app/facades"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"
	"github.com/spf13/cast"

	"goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/tenancyctx"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
	"goravel/app/utils/logger"
)

type AuthService interface {
	// Login 管理员登录
	Login(ctx http.Context, username, password string) (*models.Admin, string, error)
	// IssueAdminToken 签发管理员 access token（含 UA/IP 与 JWT TTL）
	IssueAdminToken(ctx http.Context, adminID uint) (string, error)
	// GetAdminInfo 获取管理员完整信息（包括权限和菜单）
	GetAdminInfo(ctx http.Context) (*models.Admin, []models.Permission, []models.Menu, error)
	// RecordLoginLog 记录登录日志
	RecordLoginLog(ctx http.Context, adminID uint, username string, status uint8, message string, request string) error
}

type AuthServiceImpl struct {
	ctx          context.Context
	adminService AdminService
	tokenService TokenService
}

func NewAuthServiceImpl(ctx context.Context, adminService AdminService, tokenService TokenService) *AuthServiceImpl {
	return &AuthServiceImpl{
		ctx:          ctx,
		adminService: adminService,
		tokenService: tokenService,
	}
}

// Login 管理员登录
//
// 参数:
//   - ctx: HTTP 上下文
//   - username: 用户名
//   - password: 密码
//
// 返回:
//   - *models.Admin: 管理员对象
//   - string: JWT token
//   - error: 错误信息
func (s *AuthServiceImpl) Login(ctx http.Context, username, password string) (*models.Admin, string, error) {
	// 验证用户名是否存在
	exists, err := appfacades.OrmQuery(ctx).Model(&models.Admin{}).Where("username", username).Exists()
	if err != nil {
		return nil, "", err
	}
	if !exists {
		return nil, "", errors.ErrUsernameOrPasswordErr
	}

	// 获取管理员信息
	var admin models.Admin
	if err := appfacades.OrmQuery(ctx).Where("username", username).First(&admin); err != nil {
		return nil, "", err
	}

	if admin.Status == 0 {
		return nil, "", errors.ErrAccountDisabled
	}

	// 验证密码
	if !facades.Hash().Check(password, admin.Password) {
		// 记录登录失败日志（注意：这个方法可能不再使用，但为了兼容性保留）
		requestData := ""
		if allInputs := ctx.Request().All(); len(allInputs) > 0 {
			if data, err := json.Marshal(allInputs); err == nil {
				requestData = string(data)
			}
		}
		s.RecordLoginLog(ctx, 0, username, 0, "password_error", requestData)
		return nil, "", errors.ErrPasswordError
	}

	// 生成token并存入数据库（类似Laravel Sanctum）
	plainToken, err := s.IssueAdminToken(ctx, admin.ID)
	if err != nil {
		return nil, "", err
	}
	token := plainToken

	// 更新最后登录时间（ORM会自动更新UpdatedAt）
	appfacades.OrmQuery(ctx).Save(&admin)

	// 记录登录成功日志
	requestData := ""
	if allInputs := ctx.Request().All(); len(allInputs) > 0 {
		if data, err := json.Marshal(allInputs); err == nil {
			requestData = string(data)
		}
	}
	s.RecordLoginLog(ctx, admin.ID, username, 1, "login_success", requestData)

	return &admin, token, nil
}

// AdminTokenExpiresAt 按 jwt.ttl 计算过期时间；ttl<=0 表示永不过期。
func AdminTokenExpiresAt() *time.Time {
	ttl := facades.Config().GetInt("jwt.ttl", 60)
	if ttl <= 0 {
		return nil
	}
	exp := time.Now().Add(time.Duration(ttl) * time.Minute)
	return &exp
}

// IssueAdminToken 签发管理员 access token（含 UA/IP 与 JWT TTL）。
func (s *AuthServiceImpl) IssueAdminToken(ctx http.Context, adminID uint) (string, error) {
	browser, osName := helpers.GetBrowserAndOS(ctx)
	ip := helpers.GetRealIP(ctx)
	plainToken, _, err := s.tokenService.CreateToken(
		"admin",
		adminID,
		"admin-token",
		AdminTokenExpiresAt(),
		browser,
		ip,
		osName,
		"",
	)
	return plainToken, err
}

// GetAdminInfo 获取管理员完整信息（包括权限和菜单）
//
// 参数:
//   - ctx: HTTP 上下文
//
// 返回:
//   - *models.Admin: 管理员对象
//   - []models.Permission: 权限列表
//   - []models.Menu: 菜单列表
//   - error: 错误信息
func (s *AuthServiceImpl) GetAdminInfo(ctx http.Context) (*models.Admin, []models.Permission, []models.Menu, error) {
	// 从context中获取admin信息（由JWT中间件设置）
	adminValue := ctx.Value("admin")
	if adminValue == nil {
		logger.ErrorfHTTP(ctx, "GetAdminInfo: admin value is nil in context")
		return nil, nil, nil, errors.ErrNotLoggedIn
	}

	var admin models.Admin
	// 尝试值类型
	if adminVal, ok := adminValue.(models.Admin); ok {
		admin = adminVal
	} else if adminPtr, ok := adminValue.(*models.Admin); ok {
		// 尝试指针类型
		if adminPtr == nil {
			logger.ErrorfHTTP(ctx, "GetAdminInfo: admin pointer is nil")
			return nil, nil, nil, errors.ErrNotLoggedIn
		}
		admin = *adminPtr
	} else {
		logger.ErrorfHTTP(ctx, "GetAdminInfo: admin value type assertion failed, type: %T, value: %+v", adminValue, adminValue)
		return nil, nil, nil, errors.ErrNotLoggedIn
	}

	// facades.Log().Debugf("GetAdminInfo: admin found, ID: %d, Username: %s", admin.ID, admin.Username)

	// 重新查询admin并加载关联（避免使用已存在的admin对象，可能导致关联加载问题）
	var adminWithRelations models.Admin
	if err := appfacades.OrmQuery(ctx).With("Department").With("Position").With("Roles").Where("id", admin.ID).First(&adminWithRelations); err != nil {
		errorlog.RecordHTTP(ctx, "auth", "Failed to load admin with relations", map[string]any{
			"error":    err.Error(),
			"admin_id": admin.ID,
		}, "GetAdminInfo: failed to load admin with relations, error: %v", err)
		return nil, nil, nil, err
	}
	admin = adminWithRelations

	// 批量加载所有角色的权限和菜单，避免 N+1 查询
	if len(admin.Roles) > 0 {
		var roleIDs []uint
		for _, role := range admin.Roles {
			roleIDs = append(roleIDs, role.ID)
		}

		// 批量加载权限
		type RolePermission struct {
			RoleID       uint `gorm:"column:role_id"`
			PermissionID uint `gorm:"column:permission_id"`
		}
		var rolePermissions []RolePermission
		if err := appfacades.OrmQuery(ctx).Table("role_permission").Where("role_id IN ?", roleIDs).Find(&rolePermissions); err == nil {
			var permissionIDs []uint
			rolePermissionMap := make(map[uint][]uint)
			for _, rp := range rolePermissions {
				rolePermissionMap[rp.RoleID] = append(rolePermissionMap[rp.RoleID], rp.PermissionID)
				permissionIDs = append(permissionIDs, rp.PermissionID)
			}
			if len(permissionIDs) > 0 {
				var permissions []models.Permission
				if err := appfacades.OrmQuery(ctx).Where("id IN ?", permissionIDs).Find(&permissions); err == nil {
					permissionMap := make(map[uint]models.Permission)
					for _, perm := range permissions {
						permissionMap[perm.ID] = perm
					}
					for i := range admin.Roles {
						if permIDs, ok := rolePermissionMap[admin.Roles[i].ID]; ok {
							for _, permID := range permIDs {
								if perm, ok := permissionMap[permID]; ok {
									admin.Roles[i].Permissions = append(admin.Roles[i].Permissions, perm)
								}
							}
						}
					}
				}
			}
		}

		// 批量加载菜单
		type RoleMenu struct {
			RoleID uint `gorm:"column:role_id"`
			MenuID uint `gorm:"column:menu_id"`
		}
		var roleMenus []RoleMenu
		if err := appfacades.OrmQuery(ctx).Table("role_menu").Where("role_id IN ?", roleIDs).Find(&roleMenus); err == nil {
			var menuIDs []uint
			roleMenuMap := make(map[uint][]uint)
			for _, rm := range roleMenus {
				roleMenuMap[rm.RoleID] = append(roleMenuMap[rm.RoleID], rm.MenuID)
				menuIDs = append(menuIDs, rm.MenuID)
			}
			if len(menuIDs) > 0 {
				var menus []models.Menu
				if err := appfacades.OrmQuery(ctx).Where("id IN ?", menuIDs).Find(&menus); err == nil {
					menuMap := make(map[uint]models.Menu)
					for _, menu := range menus {
						menuMap[menu.ID] = menu
					}
					for i := range admin.Roles {
						if mIDs, ok := roleMenuMap[admin.Roles[i].ID]; ok {
							for _, menuID := range mIDs {
								if menu, ok := menuMap[menuID]; ok {
									admin.Roles[i].Menus = append(admin.Roles[i].Menus, menu)
								}
							}
						}
					}
				}
			}
		}
	}

	// 检查是否是超级管理员
	const SuperAdminRoleSlug = "super-admin"
	isSuperAdmin := false
	for _, role := range admin.Roles {
		if role.Slug == SuperAdminRoleSlug && role.Status == 1 {
			isSuperAdmin = true
			break
		}
	}

	// 收集所有角色的权限和菜单（去重）
	permissionMap := make(map[uint]models.Permission)
	menuMap := make(map[uint]models.Menu)

	for _, role := range admin.Roles {
		for _, perm := range role.Permissions {
			permissionMap[perm.ID] = perm
		}
		for _, menu := range role.Menus {
			menuMap[menu.ID] = menu
		}
	}

	// 如果是超级管理员，返回所有菜单（用于前端显示）
	// 但不需要返回所有权限，因为权限检查在中间件中会跳过
	if isSuperAdmin {
		var allMenus []models.Menu
		if err := appfacades.OrmQuery(ctx).Where("status", 1).Order("sort ASC").Find(&allMenus); err == nil {
			for _, menu := range allMenus {
				menuMap[menu.ID] = menu
			}
		}
	}

	// 转换为切片
	var permissions []models.Permission
	var menus []models.Menu
	for _, perm := range permissionMap {
		permissions = append(permissions, perm)
	}
	for _, menu := range menuMap {
		menus = append(menus, menu)
	}

	// 检查是否需要隐藏服务监控菜单
	// 只有当配置值不为空且不等于 "0" 时才隐藏（"0" 表示不隐藏）
	monitorHidden := facades.Config().GetString("admin.monitor_hidden", "")
	if monitorHidden != "" && monitorHidden != "0" {
		// 检查是否是开发者管理员
		developerIDsStr := facades.Config().GetString("admin.developer_ids", "2")
		isDeveloperAdmin := s.isDeveloperAdmin(admin.ID, developerIDsStr)

		// 如果不是开发者管理员，则过滤掉监控中心及其子菜单
		if !isDeveloperAdmin {
			hiddenIDs := make(map[uint]bool)
			for _, menu := range menus {
				if menu.Slug == "monitor-center" || menu.Slug == "monitor" {
					hiddenIDs[menu.ID] = true
				}
			}
			changed := true
			for changed {
				changed = false
				for _, menu := range menus {
					if hiddenIDs[menu.ParentID] && !hiddenIDs[menu.ID] {
						hiddenIDs[menu.ID] = true
						changed = true
					}
				}
			}

			var filteredMenus []models.Menu
			for _, menu := range menus {
				if !hiddenIDs[menu.ID] {
					filteredMenus = append(filteredMenus, menu)
				}
			}
			menus = filteredMenus
		}
	}

	menus = utils.FilterFlatMenusByModule(menus)

	return &admin, permissions, menus, nil
}

// RecordLoginLog 记录登录日志
func (s *AuthServiceImpl) RecordLoginLog(ctx http.Context, adminID uint, username string, status uint8, message string, request string) error {
	ip := helpers.GetRealIP(ctx)

	// 先创建登录日志记录（Location 字段先为空，避免阻塞登录流程）
	loginLog := models.LoginLog{
		AdminID:   adminID,
		Username:  username,
		IP:        ip,
		UserAgent: ctx.Request().Header("User-Agent", ""),
		Location:  "", // 先为空，异步更新
		Status:    status,
		Message:   message,
		Request:   request,
	}

	if err := appfacades.OrmQuery(ctx).Create(&loginLog); err != nil {
		return err
	}

	DispatchAuditWebhook(tenancyctx.Detach(ctx), "login_log", map[string]any{
		"id":       loginLog.ID,
		"admin_id": adminID,
		"username": username,
		"status":   status,
		"message":  message,
		"ip":       ip,
	})

	// 异步查询 IP 地理位置信息并更新日志记录
	// 这样不会阻塞登录流程
	persistCtx := tenancyctx.Detach(ctx)
	go func() {
		// 添加 panic 恢复机制
		defer func() {
			if r := recover(); r != nil {
				facades.Log().Errorf("Recovered from panic in IP location update: %v", r)
			}
		}()

		// 添加上下文超时控制（5秒超时）；Detach 保留租户连接，避免写到平台库
		bg, cancel := context.WithTimeout(persistCtx, 5*time.Second)
		defer cancel()

		location := utils.GetIPLocation(ip)
		if location != "" {
			// 更新登录日志的 Location 字段
			if _, err := appfacades.OrmQuery(bg).
				Model(&models.LoginLog{}).
				Where("id", loginLog.ID).
				Update("location", location); err != nil {
				facades.Log().Errorf("Failed to update login log location: %v", err)
			}
		}

		// 检查上下文是否超时
		select {
		case <-bg.Done():
			if bg.Err() == context.DeadlineExceeded {
				facades.Log().Errorf("IP location update timeout for login log ID: %d", loginLog.ID)
			}
		default:
		}
	}()

	return nil
}

// isDeveloperAdmin 检查是否是开发者管理员
func (s *AuthServiceImpl) isDeveloperAdmin(adminID uint, developerIDsStr string) bool {
	if developerIDsStr == "" {
		return false
	}

	// 解析开发者ID列表
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
