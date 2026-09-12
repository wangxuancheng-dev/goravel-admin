package admin

import (
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/apidoc"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/models"
	"goravel/app/services"
)

type AdminController struct{}

// AdminResponse 管理员 JSON 字段（列表项含 2FA/超管；详情/写接口可能不含 is_2fa_bound）
type AdminResponse struct {
	ID           uint             `json:"id" example:"1"`                           // 管理员ID
	Username     string           `json:"username" example:"admin"`                 // 登录用户名
	Nickname     string           `json:"nickname" example:"管理员"`                   // 显示昵称
	Avatar       string           `json:"avatar" example:""`                        // 头像URL
	Email        string           `json:"email" example:"admin@example.com"`        // 联系邮箱
	Phone        string           `json:"phone" example:"13800138000"`              // 联系手机号
	Status       uint8            `json:"status" enums:"0,1" example:"1"`           // 账号状态（1启用，0禁用）
	Is2FABound   bool             `json:"is_2fa_bound,omitempty" example:"true"`    // 是否已绑定2FA
	IsSuperAdmin bool             `json:"is_super_admin,omitempty" example:"false"` // 是否为超级管理员
	DepartmentID uint             `json:"department_id" example:"1"`                // 所属部门ID
	Department   map[string]any   `json:"department"`                               // 部门信息对象
	PositionID   uint             `json:"position_id" example:"1"`                  // 所属岗位ID
	Position     map[string]any   `json:"position"`                                 // 岗位信息对象
	Roles        []map[string]any `json:"roles"`                                    // 角色信息数组
	CreatedAt    string           `json:"created_at" example:"2024-01-01 00:00:00"` // 创建时间
	UpdatedAt    string           `json:"updated_at" example:"2024-01-01 00:00:00"` // 更新时间
}

// AdminListData 列表成功时 data 内层
type AdminListData struct {
	List []AdminResponse `json:"list"`
	apidoc.Pagination
}

// AdminListResponse 管理员分页列表（命名约定：XxxListResponse = apidoc.Success + XxxListData）
type AdminListResponse struct {
	apidoc.Success
	Data AdminListData `json:"data"`
}

// AdminDetailData 详情/创建/更新时 data 内层
type AdminDetailData struct {
	Admin AdminResponse `json:"admin"`
}

// AdminDetailResponse 单条管理员（apidoc.Success + 业务 Data）
type AdminDetailResponse struct {
	apidoc.Success
	Data AdminDetailData `json:"data"`
}

// AdminExportData 导出成功时 data 内层
type AdminExportData struct {
	FilePath string `json:"file_path" example:"exports/admins_20260409_150000.csv"`         // 导出文件存储路径
	FileURL  string `json:"file_url" example:"/storage/exports/admins_20260409_150000.csv"` // 导出文件访问地址
}

// AdminExportResponse 管理员导出响应
type AdminExportResponse struct {
	apidoc.Success
	Data AdminExportData `json:"data"`
}

// UnbindGoogleAuthRequest 解绑管理员2FA请求
type UnbindGoogleAuthRequest struct {
	Code string `json:"code" example:"123456"` // 当前管理员的谷歌验证码（6位动态码）
}

func NewAdminController() *AdminController {
	return &AdminController{}
}

func (c *AdminController) AdminService(ctx http.Context) services.AdminService {
	return services.NewAdminServiceImpl(ctx)
}

func (c *AdminController) googleAuthenticatorService(ctx http.Context) services.GoogleAuthenticatorService {
	return services.NewGoogleAuthenticatorServiceImpl(ctx)
}

func (c *AdminController) buildAdminFilters(ctx http.Context) services.AdminFilters {
	return services.BuildAdminFiltersFromHTTP(ctx)
}

// Index 管理员列表
// @Summary      获取管理员列表
// @Description  分页获取管理员列表
// @Tags         管理员管理
// @Accept       json
// @Produce      json
// @Param        page          query     int     false  "页码（从1开始）" default(1)
// @Param        page_size     query     int     false  "每页数量（建议 10-100）" default(10)
// @Param        username      query     string  false  "登录用户名（模糊匹配）"
// @Param        status        query     string  false  "账号状态（1-启用，0-禁用）" Enums(0,1)
// @Param        role_id       query     string  false  "角色ID（精确匹配）"
// @Param        department_id query     string  false  "部门ID（精确匹配）"
// @Param        position_id   query     string  false  "岗位ID（精确匹配）"
// @Param        is_2fa_bound  query     string  false  "是否已绑定2FA（1-已绑定，0-未绑定）" Enums(0,1)
// @Param        start_time    query     string  false  "创建时间开始（格式：YYYY-MM-DD HH:mm:ss）"
// @Param        end_time      query     string  false  "创建时间结束（格式：YYYY-MM-DD HH:mm:ss）"
// @Param        order_by      query     string  false  "排序字段（格式：字段:asc/desc，例如：created_at:desc）"
// @Success      200           {object}  AdminListResponse
// @Failure      500           {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/admins [get]
// @Security     BearerAuth
func (c *AdminController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	filters := c.buildAdminFilters(ctx)

	admins, total, err := c.AdminService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"action": "list_admins",
		})
	}

	svc := c.AdminService(ctx)
	adminList := make([]http.Json, len(admins))
	for i, admin := range admins {
		adminList[i] = svc.ToListItem(admin)
	}

	return response.Success(ctx, http.Json{
		"list":      adminList,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Show 管理员详情
// @Summary      获取管理员详情
// @Description  根据ID获取管理员详细信息
// @Tags         管理员管理
// @Accept       json
// @Produce      json
// @Param        id   path     int  true  "管理员ID"
// @Success      200  {object} AdminDetailResponse
// @Failure      404  {object} apidoc.Error "管理员不存在"
// @Router       /api/admin/admins/{id} [get]
// @Security     BearerAuth
func (c *AdminController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	admin, err := c.AdminService(ctx).GetByID(id, true, true)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"admin": c.AdminService(ctx).ToDetail(admin),
	})
}

// Store 创建管理员
// @Summary      创建管理员
// @Description  创建新的管理员账号（status：1-启用，0-禁用）
// @Description  字段说明：username-登录用户名（必填）；password-登录密码（必填）；nickname-显示昵称；email-联系邮箱；phone-联系手机号；department_id-所属部门ID；position_id-所属岗位ID；role_ids-角色ID数组；status-账号状态（1启用/0禁用）
// @Tags         管理员管理
// @Accept       json
// @Produce      json
// @Param        request       body     adminrequests.AdminCreate  true  "创建参数（必填：username、password；可选：nickname、email、phone、department_id、position_id、status、role_ids）"
// @Success      200           {object} AdminDetailResponse
// @Failure      500           {object} apidoc.Error "服务器错误"
// @Router       /api/admin/admins [post]
// @Security     BearerAuth
func (c *AdminController) Store(ctx http.Context) http.Response {
	var req adminrequests.AdminCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	admin, err := c.AdminService(ctx).CreateAdmin(services.CreateAdminInput{
		Username:     req.Username,
		Password:     req.Password,
		Nickname:     req.Nickname,
		Email:        req.Email,
		Phone:        req.Phone,
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		Status:       req.Status,
		RoleIDs:      req.RoleIDs,
	})
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"action":   "create_admin",
			"username": req.Username,
		})
	}

	return response.Success(ctx, http.Json{
		"admin": admin,
	})
}

// Update 更新管理员
// @Summary      更新管理员信息
// @Description  更新管理员的基本信息（status：1-启用，0-禁用）
// @Description  字段说明：nickname-显示昵称；email-联系邮箱；phone-联系手机号；password-登录密码；department_id-所属部门ID；position_id-所属岗位ID；role_ids-角色ID数组；status-账号状态（1启用/0禁用）
// @Tags         管理员管理
// @Accept       json
// @Produce      json
// @Param        id            path     int                       true  "管理员ID"
// @Param        request       body     adminrequests.AdminUpdate true  "更新参数（可按需提交任意字段）"
// @Success      200           {object} AdminDetailResponse
// @Failure      403           {object} apidoc.Error "无权限或受保护管理员不能禁用"
// @Failure      404           {object} apidoc.Error "管理员不存在"
// @Router       /api/admin/admins/{id} [put]
// @Security     BearerAuth
func (c *AdminController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.AdminUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	admin, err := c.AdminService(ctx).UpdateByRequest(ctx, id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"admin": *admin,
	})
}

// Destroy 删除管理员
// @Summary      删除管理员
// @Description  删除指定的管理员账号
// @Tags         管理员管理
// @Accept       json
// @Produce      json
// @Param        id   path     int  true  "管理员ID"
// @Success      200  {object} apidoc.Success "删除成功"
// @Failure      403  {object} apidoc.Error "无权限、受保护管理员不能删除或不能删除自己"
// @Failure      404  {object} apidoc.Error "管理员不存在"
// @Router       /api/admin/admins/{id} [delete]
// @Security     BearerAuth
func (c *AdminController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	actorID := uint(0)
	currentAdmin, resp := c.currentAdminFromContext(ctx)
	if resp != nil {
		return resp
	}
	if currentAdmin != nil {
		actorID = currentAdmin.ID
	}

	if err := c.AdminService(ctx).Delete(id, actorID); err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}

// UnbindGoogleAuthenticator 管理员解绑其他管理员的谷歌验证码
// @Summary      解绑管理员的谷歌验证码
// @Description  管理员解绑其他管理员的谷歌验证码
// @Tags         管理员管理
// @Accept       json
// @Produce      json
// @Param        id       path     int                     true  "要解绑的管理员ID"
// @Param        request  body     UnbindGoogleAuthRequest true  "验证码确认参数"
// @Success      200      {object} apidoc.Success "解绑成功"
// @Failure      400      {object} apidoc.Error "参数错误、验证码错误或目标管理员未绑定2FA"
// @Failure      401      {object} apidoc.Error "未登录"
// @Failure      403      {object} apidoc.Error "当前管理员未绑定2FA，禁止执行"
// @Failure      404      {object} apidoc.Error "管理员不存在"
// @Failure      500      {object} apidoc.Error "服务器错误"
// @Router       /api/admin/admins/{id}/unbind-google-auth [post]
// @Security     BearerAuth
func (c *AdminController) UnbindGoogleAuthenticator(ctx http.Context) http.Response {
	targetAdminID := helpers.GetUintRoute(ctx, "id")
	if targetAdminID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	if _, err := c.AdminService(ctx).GetByID(targetAdminID, false, false); err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusNotFound, err, map[string]any{"id": targetAdminID})
	}

	currentAdmin, resp := c.currentAdminFromContext(ctx)
	if resp != nil {
		return resp
	}
	if currentAdmin == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	isBound, err := c.googleAuthenticatorService(ctx).IsBound(currentAdmin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"admin_id": currentAdmin.ID,
		})
	}
	if !isBound {
		return response.Error(ctx, http.StatusForbidden, apperrors.ErrGoogleAuthenticatorNotBound.Code)
	}

	code := ctx.Request().Input("code")
	if code == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrCodeRequired.Code)
	}

	secret, err := c.googleAuthenticatorService(ctx).GetSecret(currentAdmin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"admin_id": currentAdmin.ID,
		})
	}
	if secret == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleAuthenticatorNotBound.Code)
	}
	if !c.googleAuthenticatorService(ctx).Verify(secret, code) {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeInvalid.Code)
	}

	targetIsBound, err := c.googleAuthenticatorService(ctx).IsBound(targetAdminID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"target_admin_id": targetAdminID,
		})
	}
	if !targetIsBound {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleAuthenticatorNotBound.Code)
	}

	if err := c.googleAuthenticatorService(ctx).Unbind(targetAdminID); err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"target_admin_id":  targetAdminID,
			"current_admin_id": currentAdmin.ID,
		})
	}

	return response.Success(ctx, "unbind_success")
}

// ResetGoogleAuthenticator 重置管理员的谷歌验证码（强制清除绑定，无需验证码，用于管理员丢失手机等场景）
// @Summary      重置管理员的谷歌验证码
// @Description  强制清除指定管理员的谷歌验证码绑定
// @Tags         管理员管理
// @Accept       json
// @Produce      json
// @Param        id   path     int true "要重置的管理员ID"
// @Success      200  {object} apidoc.Success "重置成功"
// @Failure      400  {object} apidoc.Error "参数错误或目标管理员未绑定2FA"
// @Failure      401  {object} apidoc.Error "未登录"
// @Failure      403  {object} apidoc.Error "无权限或不可操作受保护管理员"
// @Failure      404  {object} apidoc.Error "管理员不存在"
// @Failure      500  {object} apidoc.Error "服务器错误"
// @Router       /api/admin/admins/{id}/reset-google-auth [post]
// @Security     BearerAuth
func (c *AdminController) ResetGoogleAuthenticator(ctx http.Context) http.Response {
	targetAdminID := helpers.GetUintRoute(ctx, "id")
	if targetAdminID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	if _, err := c.AdminService(ctx).GetByID(targetAdminID, false, false); err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusNotFound, err, map[string]any{"id": targetAdminID})
	}

	if c.AdminService(ctx).IsProtectedAdmin(targetAdminID) {
		return response.Error(ctx, http.StatusForbidden, apperrors.ErrProtectedAdmin.Code)
	}

	currentAdmin, resp := c.currentAdminFromContext(ctx)
	if resp != nil {
		return resp
	}
	if currentAdmin == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	confirmCode := SensitiveConfirmCodeFromRequest(ctx)
	if err := services.VerifySensitiveConfirm(ctx, currentAdmin.ID, confirmCode); err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusBadRequest, err, map[string]any{
			"target_admin_id":  targetAdminID,
			"current_admin_id": currentAdmin.ID,
		})
	}

	targetIsBound, err := c.googleAuthenticatorService(ctx).IsBound(targetAdminID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"target_admin_id": targetAdminID,
		})
	}
	if !targetIsBound {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleAuthenticatorNotBound.Code)
	}

	if err := c.googleAuthenticatorService(ctx).Unbind(targetAdminID); err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"target_admin_id": targetAdminID,
		})
	}

	return response.Success(ctx, "reset_success")
}

func (c *AdminController) currentAdminFromContext(ctx http.Context) (*models.Admin, http.Response) {
	adminValue := ctx.Value("admin")
	if adminValue == nil {
		return nil, nil
	}

	if admin, ok := adminValue.(models.Admin); ok {
		return &admin, nil
	}
	if adminPtr, ok := adminValue.(*models.Admin); ok {
		return adminPtr, nil
	}

	return nil, response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
}

// Export 导出管理员列表
// @Summary      导出管理员列表
// @Description  根据筛选条件导出管理员列表为CSV文件
// @Tags         管理员管理
// @Accept       json
// @Produce      json
// @Param        request body     adminrequests.AdminListFilter false "筛选（字段同列表 GET query）"
// @Success      200     {object} AdminExportResponse "导出成功，返回文件下载信息"
// @Failure      500     {object} apidoc.Error "服务器错误"
// @Router       /api/admin/admins/export [post]
// @Security     BearerAuth
func (c *AdminController) Export(ctx http.Context) http.Response {
	lock := helpers.AcquireExportLock(ctx, "admins")
	if lock.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUnauthorized.Code)
	}
	if lock.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, apperrors.ErrGetLockFailed.Code)
	}
	adminID := lock.AdminID

	filters := c.buildAdminFilters(ctx)

	admins, err := c.AdminService(ctx).GetAllAdminsForExport(filters)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "admin", http.StatusInternalServerError, err, map[string]any{
			"action":   "export_admins",
			"admin_id": adminID,
		})
	}

	headers := []string{
		"id",
		"username",
		"nickname",
		"email",
		"phone",
		"status",
		"department",
		"position",
		"roles",
		"created_at",
		"updated_at",
	}

	timezone := helpers.GetCurrentTimezone(ctx)
	var data [][]string
	for _, admin := range admins {
		statusText := trans.Get(ctx, "disabled")
		if admin.Status == 1 {
			statusText = trans.Get(ctx, "enabled")
		}

		departmentName := ""
		if admin.Department.ID > 0 {
			departmentName = admin.Department.Name
		}

		positionName := ""
		if admin.Position.ID > 0 {
			positionName = admin.Position.Name
		}

		roleNameParts := make([]string, 0, len(admin.Roles))
		for _, role := range admin.Roles {
			roleNameParts = append(roleNameParts, role.Name)
		}
		roleNames := strings.Join(roleNameParts, ", ")

		createdAt := helpers.FormatCarbonWithTimezone(admin.CreatedAt, timezone)
		updatedAt := helpers.FormatCarbonWithTimezone(admin.UpdatedAt, timezone)

		row := []string{
			cast.ToString(admin.ID),
			admin.Username,
			admin.Nickname,
			admin.Email,
			admin.Phone,
			statusText,
			departmentName,
			positionName,
			roleNames,
			createdAt,
			updatedAt,
		}
		data = append(data, row)
	}

	ctx.WithValue("export_type", models.ExportTypeAdmins)

	return response.Export(ctx, "exported", headers, data, "admins")
}
