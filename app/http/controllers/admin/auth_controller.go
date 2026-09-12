package admin

import (
	"encoding/json"
	appfacades "goravel/app/facades"
	"strconv"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/search"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/utils"
)

type AuthController struct{}

func NewAuthController() *AuthController {
	return &AuthController{}
}

func (r *AuthController) adminService(ctx http.Context) services.AdminService {
	return services.NewAdminServiceImpl(ctx)
}

func (r *AuthController) tokenService(ctx http.Context) services.TokenService {
	return services.NewTokenServiceImpl(ctx)
}

func (r *AuthController) authService(ctx http.Context) services.AuthService {
	return services.NewAuthServiceImpl(ctx, r.adminService(ctx), r.tokenService(ctx))
}

func (r *AuthController) captchaService(ctx http.Context) services.CaptchaService {
	return services.NewCaptchaServiceImpl(ctx)
}

func (r *AuthController) googleAuthenticatorService(ctx http.Context) services.GoogleAuthenticatorService {
	return services.NewGoogleAuthenticatorServiceImpl(ctx)
}

func (r *AuthController) treeService(ctx http.Context) services.TreeService {
	return services.NewTreeServiceImpl(ctx)
}

func (r *AuthController) lockoutService(ctx http.Context) services.LoginLockoutService {
	return services.NewLoginLockoutService(ctx)
}

// getLoginRequestData 获取登录请求数据（排除敏感信息）
func (r *AuthController) getLoginRequestData(ctx http.Context) string {
	inputs := make(map[string]any)
	allInputs := ctx.Request().All()
	for key, value := range allInputs {
		// 使用工具函数检查是否是敏感字段
		if utils.IsSensitiveField(key) {
			inputs[key] = "***"
		} else {
			inputs[key] = value
		}
	}
	if data, err := json.Marshal(inputs); err == nil {
		return string(data)
	}
	return ""
}

func parseAdminFromContext(ctx http.Context) (*models.Admin, bool) {
	adminValue := ctx.Value("admin")
	if adminValue == nil {
		return nil, false
	}

	if admin, ok := adminValue.(models.Admin); ok {
		return &admin, true
	}
	if adminPtr, ok := adminValue.(*models.Admin); ok {
		return adminPtr, adminPtr != nil
	}

	return nil, false
}

func isDeveloperAdminByConfig(adminID uint) bool {
	if adminID == 0 {
		return false
	}

	developerIDsStr := facades.Config().GetString("admin.developer_ids", "2")
	parts := str.Of(developerIDsStr).Split(",")
	for _, part := range parts {
		idStr := str.Of(part).Trim().String()
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err == nil && uint(id) == adminID {
			return true
		}
	}

	return false
}

func (r *AuthController) currentAdminFromContextRequired(ctx http.Context) (*models.Admin, http.Response) {
	admin, ok := parseAdminFromContext(ctx)
	if !ok {
		return nil, response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	return admin, nil
}

func (r *AuthController) currentAdminFromContextOptional(ctx http.Context) *models.Admin {
	admin, ok := parseAdminFromContext(ctx)
	if !ok {
		return nil
	}

	return admin
}

func getCurrentTokenIDFromContext(ctx http.Context) uint {
	currentTokenValue := ctx.Value("token")
	if currentTokenValue == nil {
		return 0
	}

	if currentToken, ok := currentTokenValue.(models.PersonalAccessToken); ok {
		return currentToken.ID
	}
	if currentTokenPtr, ok := currentTokenValue.(*models.PersonalAccessToken); ok && currentTokenPtr != nil {
		return currentTokenPtr.ID
	}

	return 0
}

func routeUintID(ctx http.Context, key, requiredErrorCode, invalidErrorCode string) (uint, http.Response) {
	idStr := ctx.Request().Route(key)
	if idStr == "" {
		return 0, response.Error(ctx, http.StatusBadRequest, requiredErrorCode)
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, response.Error(ctx, http.StatusBadRequest, invalidErrorCode)
	}

	return uint(id), nil
}

func requiredInput(ctx http.Context, key, requiredErrorCode string) (string, http.Response) {
	value := ctx.Request().Input(key)
	if value == "" {
		return "", response.Error(ctx, http.StatusBadRequest, requiredErrorCode)
	}

	return value, nil
}

func bearerTokenFromHeader(ctx http.Context) string {
	return str.Of(ctx.Request().Header("Authorization", "")).ChopStart("Bearer ").Trim().String()
}

// Login 管理员登录
func (r *AuthController) Login(ctx http.Context) http.Response {
	var loginRequest admin.Login
	errors, err := ctx.Request().ValidateRequest(&loginRequest)
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if errors != nil {
		return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", errors.All())
	}

	requestData := r.getLoginRequestData(ctx)
	ip := helpers.GetRealIP(ctx)

	// 一户一库：先绑定租户，后续 OrmQuery / 登录日志打到租户库
	if tenancy.Enabled() {
		hint := loginRequest.TenantCode
		if hint == "" {
			hint = loginRequest.TenantID
		}
		if err := services.NewTenantConnectionService().BindHTTP(ctx, hint); err != nil {
			if businessErr, ok := apperrors.GetBusinessError(err); ok {
				switch businessErr.Code {
				case apperrors.ErrTenantRequired.Code:
					return response.Error(ctx, http.StatusBadRequest, businessErr.Code)
				case apperrors.ErrTenantNotFound.Code:
					return response.Error(ctx, http.StatusNotFound, businessErr.Code)
				case apperrors.ErrTenantDisabled.Code, apperrors.ErrTenantNotReady.Code:
					return response.Error(ctx, http.StatusForbidden, businessErr.Code)
				case apperrors.ErrTenantHintConflict.Code:
					return response.Error(ctx, http.StatusBadRequest, businessErr.Code)
				}
			}
			return response.Error(ctx, http.StatusInternalServerError, apperrors.ErrTenantConnectionFailed.Code)
		}
	}

	// ---- 登录失败锁定检查 ----
	if locked, _ := r.lockoutService(ctx).IsLocked(ip, loginRequest.Username); locked {
		lockMinutes := facades.Config().GetInt("login_security.lock_duration_minutes", 15)
		r.authService(ctx).RecordLoginLog(ctx, 0, loginRequest.Username, 0, "login_locked", requestData)
		return response.Error(ctx, http.StatusTooManyRequests,
			apperrors.ErrLoginLocked.WithParams(map[string]any{"minutes": lockMinutes}))
	}

	// 验证用户名是否存在
	exists, err := appfacades.OrmQuery(ctx).Model(&models.Admin{}).Where("username", loginRequest.Username).Exists()
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"username": loginRequest.Username,
		})
	}
	if !exists {
		r.lockoutService(ctx).RecordFailure(ip, loginRequest.Username)
		r.authService(ctx).RecordLoginLog(ctx, 0, loginRequest.Username, 0, "username_not_found", requestData)
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUsernameOrPasswordErr.Code)
	}

	// 获取管理员信息
	var admin models.Admin
	if err := appfacades.OrmQuery(ctx).Where("username", loginRequest.Username).FirstOrFail(&admin); err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"username": loginRequest.Username,
		})
	}

	if admin.Status == 0 {
		r.authService(ctx).RecordLoginLog(ctx, admin.ID, loginRequest.Username, 0, "account_disabled", requestData)
		return response.Error(ctx, http.StatusForbidden, apperrors.ErrAccountDisabled.Code)
	}

	// 验证密码
	if !facades.Hash().Check(loginRequest.Password, admin.Password) {
		r.lockoutService(ctx).RecordFailure(ip, loginRequest.Username)
		r.authService(ctx).RecordLoginLog(ctx, admin.ID, loginRequest.Username, 0, "password_error", requestData)
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUsernameOrPasswordErr.Code)
	}

	// 检查是否绑定了谷歌验证码
	isBound, err := r.googleAuthenticatorService(ctx).IsBound(admin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	// 如果绑定了谷歌验证码，验证谷歌验证码
	if isBound {
		googleCode := loginRequest.GoogleCode
		if googleCode == "" {
			r.authService(ctx).RecordLoginLog(ctx, admin.ID, loginRequest.Username, 0, "google_code_required", requestData)
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeRequired.Code)
		}

		secret, err := r.googleAuthenticatorService(ctx).GetSecret(admin.ID)
		if err != nil {
			return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
				"admin_id": admin.ID,
			})
		}

		if !r.googleAuthenticatorService(ctx).Verify(secret, googleCode) {
			r.lockoutService(ctx).RecordFailure(ip, loginRequest.Username)
			r.authService(ctx).RecordLoginLog(ctx, admin.ID, loginRequest.Username, 0, "google_code_error", requestData)
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeInvalid.Code)
		}
	} else {
		captchaRequired := r.captchaService(ctx).Enabled() ||
			r.lockoutService(ctx).GetFailureCount(ip, loginRequest.Username) >= 2
		if captchaRequired {
			captchaID := ctx.Request().Input("captcha_id")
			captchaAnswer := ctx.Request().Input("captcha_answer")
			if ok, messageKey := r.captchaService(ctx).Verify(captchaID, captchaAnswer); !ok {
				if messageKey == "" {
					messageKey = "captcha_invalid"
				}
				r.authService(ctx).RecordLoginLog(ctx, admin.ID, loginRequest.Username, 0, messageKey, requestData)
				return response.Error(ctx, http.StatusBadRequest, messageKey)
			}
		}
	}

	// 所有验证通过，清除失败计数
	r.lockoutService(ctx).ClearFailures(ip, loginRequest.Username)

	// 登录异常检测（在写入本次成功日志之前对比上次成功 IP）
	if alert, _ := services.NewLoginAnomalyService(ctx).CheckAndAlert(admin, ip); alert != nil && alert.Email != "" {
		_ = services.EnqueueEmailFn(alert.Email, alert.Title, alert.Content)
	}

	// 验证通过，生成token并完成登录
	token, err := r.authService(ctx).IssueAdminToken(ctx, admin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	// 记录登录成功日志
	r.authService(ctx).RecordLoginLog(ctx, admin.ID, loginRequest.Username, 1, "login_success", requestData)

	// 更新最后登录时间（ORM会自动更新UpdatedAt）
	appfacades.OrmQuery(ctx).Save(&admin)

	return response.SuccessWithHeader(ctx, "login_success", "Authorization", "Bearer "+token, http.Json{
		"token": token,
		"admin": http.Json{
			"id":                   admin.ID,
			"username":             admin.Username,
			"nickname":             admin.Nickname,
			"avatar":               admin.Avatar,
			"must_change_password": admin.MustChangePassword == 1,
		},
	})
}

// Captcha 获取登录验证码
// Query check=1：仅返回是否开启，不生成图片（避免登录页探测时产生无用 captcha）
func (r *AuthController) Captcha(ctx http.Context) http.Response {
	enabled := r.captchaService(ctx).Enabled()
	ip := helpers.GetRealIP(ctx)
	username := ctx.Request().Query("username", "")
	if username == "" {
		username = ctx.Request().Input("username", "")
	}
	adaptiveRequired := username != "" && r.lockoutService(ctx).GetFailureCount(ip, username) >= 2
	required := enabled || adaptiveRequired

	captchaData := http.Json{
		"enabled":  enabled,
		"required": required,
	}

	checkOnly := ctx.Request().Query("check", "") == "1"
	if required && !checkOnly {
		captchaID, image, err := r.captchaService(ctx).Generate()
		if err != nil {
			return HandleGeneratedServiceError(ctx, "captcha", http.StatusInternalServerError, err, nil)
		}
		captchaData["captcha_id"] = captchaID
		captchaData["captcha_image"] = image
	}

	return response.Success(ctx, http.Json{
		"captcha": captchaData,
	})
}

// Info 获取当前登录管理员信息
func (r *AuthController) Info(ctx http.Context) http.Response {
	admin, permissions, menus, err := r.authService(ctx).GetAdminInfo(ctx)
	if err != nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	// 获取配置：是否显示无权限的按钮
	showButtonsWithoutPermission := facades.Config().GetBool("admin.show_buttons_without_permission", false)
	pprofEnabled := facades.Config().GetBool("pprof.enabled", false) || facades.Config().GetBool("app.debug", false)
	pprofTokenRequired := facades.Config().GetString("pprof.token", "") != ""
	isDeveloperAdmin := isDeveloperAdminByConfig(admin.ID)

	// 检查是否是超级管理员
	const SuperAdminRoleSlug = "super-admin"
	isSuperAdmin := false
	for _, role := range admin.Roles {
		if role.Slug == SuperAdminRoleSlug && role.Status == 1 {
			isSuperAdmin = true
			break
		}
	}

	// 将扁平菜单数组构建为树形结构，然后转换为前端格式
	// 可见菜单 = 角色勾选的菜单 + 权限关联的菜单（只要勾选了某目录下任一权限，该目录就显示）+ 这些菜单的所有祖先
	var menuTree []models.Menu
	menuIDSet := make(map[uint]bool)
	for _, menu := range menus {
		menuIDSet[menu.ID] = true
	}
	for _, perm := range permissions {
		if perm.MenuID > 0 {
			menuIDSet[perm.MenuID] = true
		}
	}
	menuIDs := make([]uint, 0, len(menuIDSet))
	for id := range menuIDSet {
		menuIDs = append(menuIDs, id)
	}
	if len(menuIDs) > 0 {
		expandedIDs, err := r.treeService(ctx).GetMenuIDsWithAncestors(menuIDs)
		if err == nil && len(expandedIDs) > 0 {
			menuIDSet = make(map[uint]bool)
			for _, id := range expandedIDs {
				menuIDSet[id] = true
			}
		}
		// 先构建完整树形结构
		menuTree, _ = r.treeService(ctx).BuildMenuTree(0)
		// 递归过滤树形结构，只保留有权限的菜单（含因权限而显示的目录）
		var filterMenuTree func([]models.Menu) []models.Menu
		filterMenuTree = func(menuList []models.Menu) []models.Menu {
			var result []models.Menu
			for _, menu := range menuList {
				if menuIDSet[menu.ID] {
					menu.Children = filterMenuTree(menu.Children)
					result = append(result, menu)
				}
			}
			return result
		}
		menuTree = filterMenuTree(menuTree)
	}

	// 按模块开关再过滤（避免权限 MenuID 把已关闭模块菜单加回来）
	menuTree = utils.FilterTreeMenusByModule(menuTree)

	// 转换为前端格式
	menuTreeData := utils.ConvertMenuTree(menuTree)

	return response.Success(ctx, http.Json{
		"admin": http.Json{
			"id":                   admin.ID,
			"username":             admin.Username,
			"nickname":             admin.Nickname,
			"avatar":               admin.Avatar,
			"email":                admin.Email,
			"phone":                admin.Phone,
			"department_id":        admin.DepartmentID,
			"department":           admin.Department,
			"position_id":          admin.PositionID,
			"position":             admin.Position,
			"roles":                admin.Roles,
			"permissions":          permissions,
			"menus":                menuTreeData, // 返回树形结构
			"is_super_admin":       isSuperAdmin,
			"must_change_password": admin.MustChangePassword == 1,
		},
		"config": http.Json{
			"show_buttons_without_permission": showButtonsWithoutPermission,
			"monitor_hidden":                  facades.Config().GetString("admin.monitor_hidden", ""),
			"is_developer_admin":              isDeveloperAdmin,
			"pprof_enabled":                   pprofEnabled,
			"pprof_token_required":            pprofTokenRequired,
			"ai_enabled":                      utils.AIEnabled(),
			"orders_enabled":                  utils.OrdersEnabled(),
			"payments_enabled":                utils.PaymentsEnabled(),
			"payment_gateways":                services.EnabledPaymentGateways(),
			"dev_tools_enabled":               utils.DevToolsEnabled(),
			"code_generator_enabled":          utils.CodeGeneratorEnabled(),
			"code_generator_frontends":        utils.CodeGeneratorFrontends(),
			"search_enabled":                  utils.SearchEnabled(),
			"search_driver":                   search.Driver(),
			"otel_enabled":                    utils.OTELEnabled(),
		},
	})
}

// UpdateProfile 更新个人信息
func (r *AuthController) UpdateProfile(ctx http.Context) http.Response {
	currentAdmin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}
	admin := *currentAdmin

	// 重新查询admin以确保获取最新数据
	if err := appfacades.OrmQuery(ctx).Where("id", admin.ID).FirstOrFail(&admin); err != nil {
		return response.Error(ctx, http.StatusNotFound, apperrors.ErrAdminNotFound.Code)
	}

	nickname := ctx.Request().Input("nickname")
	email := ctx.Request().Input("email")
	phone := ctx.Request().Input("phone")
	avatar := ctx.Request().Input("avatar")

	if nickname != "" {
		admin.Nickname = nickname
	}
	if email != "" {
		admin.Email = email
	}
	if phone != "" {
		admin.Phone = phone
	}
	if avatar != "" {
		admin.Avatar = avatar
	}

	if err := appfacades.OrmQuery(ctx).Save(&admin); err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	// 重新加载关联数据（确保部门和角色被正确加载）
	var adminWithRelations models.Admin
	if err := appfacades.OrmQuery(ctx).With("Department").With("Position").With("Roles").Where("id", admin.ID).FirstOrFail(&adminWithRelations); err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}
	admin = adminWithRelations

	return response.Success(ctx, http.Json{
		"admin": http.Json{
			"id":            admin.ID,
			"username":      admin.Username,
			"nickname":      admin.Nickname,
			"avatar":        admin.Avatar,
			"email":         admin.Email,
			"phone":         admin.Phone,
			"department_id": admin.DepartmentID,
			"department":    admin.Department,
			"position_id":   admin.PositionID,
			"position":      admin.Position,
			"roles":         admin.Roles,
		},
	})
}

// Refresh 刷新Token
// 注意：此接口需要在JWT中间件之前调用，或者使用特殊的中间件处理
// 因为Refresh方法需要token过期但仍在刷新窗口内才能工作
func (r *AuthController) Refresh(ctx http.Context) http.Response {
	// 从请求头获取token
	token := bearerTokenFromHeader(ctx)
	if token == "" {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUnauthorized.Code)
	}

	// 先尝试解析token，如果token有效，直接重新生成（滑动过期）
	if _, err := facades.Auth(ctx).Guard("admin").Parse(token); err == nil {
		// Token有效，重新生成新token（延长过期时间）
		if userID, err := facades.Auth(ctx).Guard("admin").ID(); err == nil {
			if newToken, err := facades.Auth(ctx).Guard("admin").LoginUsingID(userID); err == nil {
				return response.SuccessWithHeader(ctx, "token_refresh_success", "Authorization", "Bearer "+newToken, http.Json{
					"token": newToken,
				})
			}
		}
	}

	// 如果token已过期，尝试刷新（需要在刷新窗口内）
	newToken, err := facades.Auth(ctx).Guard("admin").Refresh()
	if err != nil {
		// 刷新失败，返回错误
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrTokenRefreshFailed.Code)
	}

	// 刷新成功，返回新token
	return response.SuccessWithHeader(ctx, "token_refresh_success", "Authorization", "Bearer "+newToken, http.Json{
		"token": newToken,
	})
}

// Heartbeat 心跳接口，用于更新用户的最后活跃时间
// JWT中间件会自动更新 last_used_at，这个接口只是确保用户在线状态
func (r *AuthController) Heartbeat(ctx http.Context) http.Response {
	// JWT中间件已经更新了 last_used_at，这里只需要返回成功即可
	return response.Success(ctx, "heartbeat_success")
}

// Logout 退出登录
func (r *AuthController) Logout(ctx http.Context) http.Response {
	if admin := r.currentAdminFromContextOptional(ctx); admin != nil && admin.ID > 0 {
		// 获取token
		token := bearerTokenFromHeader(ctx)

		if token != "" {
			// 删除token
			_ = r.tokenService(ctx).DeleteToken(token)
		}

		// 记录退出日志
		logoutRequestData := r.getLoginRequestData(ctx)
		r.authService(ctx).RecordLoginLog(ctx, admin.ID, admin.Username, 1, "logout_success", logoutRequestData)
	}

	return response.Success(ctx, "logout_success")
}

// Tokens 获取当前用户的所有token列表
func (r *AuthController) Tokens(ctx http.Context) http.Response {
	admin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}

	// 获取用户的所有token
	tokens, err := r.tokenService(ctx).GetTokensByUser("admin", admin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	// 获取当前使用的token
	currentTokenID := getCurrentTokenIDFromContext(ctx)

	// 格式化token列表
	tokenList := make([]http.Json, 0, len(tokens))
	for _, token := range tokens {
		tokenList = append(tokenList, http.Json{
			"id":           token.ID,
			"name":         token.Name,
			"last_used_at": token.LastUsedAt,
			"expires_at":   token.ExpiresAt,
			"created_at":   token.CreatedAt,
			"is_current":   token.ID == currentTokenID,
		})
	}

	return response.Success(ctx, http.Json{
		"tokens": tokenList,
	})
}

// RevokeToken 删除指定的token（踢出指定设备）
func (r *AuthController) RevokeToken(ctx http.Context) http.Response {
	admin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}

	tokenID, resp := routeUintID(ctx, "id", apperrors.ErrTokenIDRequired.Code, apperrors.ErrInvalidTokenID.Code)
	if resp != nil {
		return resp
	}

	// 查询token是否存在且属于当前用户
	var token models.PersonalAccessToken
	if err := appfacades.OrmQuery(ctx).
		Where("id", tokenID).
		Where("tokenable_type", "admin").
		Where("tokenable_id", admin.ID).
		First(&token); err != nil {
		return response.Error(ctx, http.StatusNotFound, apperrors.ErrTokenNotFound.Code)
	}

	// 删除token（直接通过ID删除，因为数据库中存储的是hash值，无法获取原始token）
	if _, err := appfacades.OrmQuery(ctx).Delete(&token); err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"token_id": token.ID,
			"admin_id": admin.ID,
		})
	}

	return response.Success(ctx, "revoke_success")
}

// RevokeAllTokens 删除当前用户的所有token（踢出所有设备）
func (r *AuthController) RevokeAllTokens(ctx http.Context) http.Response {
	admin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}

	// 删除用户的所有token
	if err := r.tokenService(ctx).DeleteTokensByUser("admin", admin.ID); err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	return response.Success(ctx, "revoke_all_success")
}

// KickOutUser 踢出指定用户的所有token（管理员操作）
func (r *AuthController) KickOutUser(ctx http.Context) http.Response {
	admin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}

	userID, resp := routeUintID(ctx, "id", apperrors.ErrUserIDRequired.Code, apperrors.ErrInvalidUserID.Code)
	if resp != nil {
		return resp
	}

	// 查询用户是否存在
	var targetAdmin models.Admin
	if err := appfacades.OrmQuery(ctx).Where("id", userID).FirstOrFail(&targetAdmin); err != nil {
		return response.Error(ctx, http.StatusNotFound, apperrors.ErrUserNotFound.Code)
	}

	// 删除用户的所有token
	if err := r.tokenService(ctx).DeleteTokensByUser("admin", targetAdmin.ID); err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"target_user_id": targetAdmin.ID,
			"operator_id":    admin.ID,
		})
	}

	return response.Success(ctx, "kick_out_success")
}

// GetGoogleAuthenticatorQRCode 获取谷歌验证码二维码（用于绑定）
func (r *AuthController) GetGoogleAuthenticatorQRCode(ctx http.Context) http.Response {
	admin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}

	// 检查是否已经绑定
	isBound, err := r.googleAuthenticatorService(ctx).IsBound(admin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	if isBound {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleAuthenticatorAlreadyBound.Code)
	}

	// 生成密钥和二维码
	accountName := admin.Username
	if admin.Email != "" {
		accountName = admin.Email
	}
	secret, qrCodeURL, err := r.googleAuthenticatorService(ctx).GenerateSecret(accountName)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	// 生成二维码图片
	qrCodeImage, err := r.googleAuthenticatorService(ctx).GenerateQRCodeImage(accountName, secret)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	return response.Success(ctx, http.Json{
		"secret":        secret,
		"qr_code_url":   qrCodeURL,
		"qr_code_image": qrCodeImage,
	})
}

// BindGoogleAuthenticator 绑定谷歌验证码
func (r *AuthController) BindGoogleAuthenticator(ctx http.Context) http.Response {
	admin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}

	secret, resp := requiredInput(ctx, "secret", apperrors.ErrSecretAndCodeRequired.Code)
	if resp != nil {
		return resp
	}
	code, resp := requiredInput(ctx, "code", apperrors.ErrSecretAndCodeRequired.Code)
	if resp != nil {
		return resp
	}

	// 绑定谷歌验证码
	if err := r.googleAuthenticatorService(ctx).Bind(admin.ID, secret, code); err != nil {
		if err.Error() == "invalid_code" {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeInvalid.Code)
		}
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	return response.Success(ctx, "bind_success")
}

// UnbindGoogleAuthenticator 解绑谷歌验证码
func (r *AuthController) UnbindGoogleAuthenticator(ctx http.Context) http.Response {
	admin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}

	// 需要验证码确认
	code, resp := requiredInput(ctx, "code", apperrors.ErrCodeRequired.Code)
	if resp != nil {
		return resp
	}

	// 获取管理员的密钥
	secret, err := r.googleAuthenticatorService(ctx).GetSecret(admin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	if secret == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleAuthenticatorNotBound.Code)
	}

	// 验证验证码
	if !r.googleAuthenticatorService(ctx).Verify(secret, code) {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeInvalid.Code)
	}

	// 解绑谷歌验证码
	if err := r.googleAuthenticatorService(ctx).Unbind(admin.ID); err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	return response.Success(ctx, "unbind_success")
}

// GetGoogleAuthenticatorStatus 获取谷歌验证码绑定状态
func (r *AuthController) GetGoogleAuthenticatorStatus(ctx http.Context) http.Response {
	admin, resp := r.currentAdminFromContextRequired(ctx)
	if resp != nil {
		return resp
	}

	// 检查是否绑定
	isBound, err := r.googleAuthenticatorService(ctx).IsBound(admin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	return response.Success(ctx, http.Json{
		"is_bound": isBound,
	})
}
