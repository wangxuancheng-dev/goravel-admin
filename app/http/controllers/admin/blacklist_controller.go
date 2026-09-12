package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/apidoc"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"
)

type BlacklistController struct{}

type BlacklistResponse struct {
	ID        uint   `json:"id" example:"1"`
	IP        string `json:"ip" example:"192.168.1.1"`
	Remark    string `json:"remark" example:"测试IP"`
	Status    uint8  `json:"status" enums:"0,1" example:"1"`
	CreatedAt string `json:"created_at" example:"2024-01-01 00:00:00"`
	UpdatedAt string `json:"updated_at" example:"2024-01-01 00:00:00"`
}

type BlacklistListData struct {
	List []BlacklistResponse `json:"list"`
	apidoc.Pagination
}

type BlacklistListResponse struct {
	apidoc.Success
	Data BlacklistListData `json:"data"`
}

type BlacklistDetailData struct {
	Blacklist BlacklistResponse `json:"blacklist"`
}

type BlacklistDetailResponse struct {
	apidoc.Success
	Data BlacklistDetailData `json:"data"`
}

func NewBlacklistController() *BlacklistController {
	return &BlacklistController{}
}

func (c *BlacklistController) buildBlacklistFilters(ctx http.Context) services.BlacklistFilters {
	return services.BuildBlacklistFiltersFromHTTP(ctx)
}

func (c *BlacklistController) BlacklistService(ctx http.Context) services.BlacklistService {
	return services.NewBlacklistService(ctx)
}

// Index 黑名单列表
// @Summary      获取黑名单列表
// @Description  分页获取黑名单，支持按IP、状态、时间区间筛选
// @Tags         风控管理
// @Accept       json
// @Produce      json
// @Param        page        query     int     false  "页码（从1开始）" default(1)
// @Param        page_size   query     int     false  "每页数量（建议 10-100）" default(10)
// @Param        ip          query     string  false  "IP地址或IP段（模糊匹配）"
// @Param        status      query     string  false  "状态（1-启用，0-禁用）" Enums(0,1)
// @Param        start_time  query     string  false  "开始时间（格式：YYYY-MM-DD HH:mm:ss）"
// @Param        end_time    query     string  false  "结束时间（格式：YYYY-MM-DD HH:mm:ss）"
// @Param        order_by    query     string  false  "排序（格式：字段:asc/desc，例如：created_at:desc）"
// @Success      200         {object}  BlacklistListResponse
// @Failure      500         {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/blacklists [get]
// @Security     BearerAuth
func (c *BlacklistController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := c.buildBlacklistFilters(ctx)

	list, total, err := c.BlacklistService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "blacklist", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Show 黑名单详情
// @Summary      获取黑名单详情
// @Description  根据ID获取黑名单详情
// @Tags         风控管理
// @Accept       json
// @Produce      json
// @Param        id    path      int  true  "黑名单ID"
// @Success      200   {object}  BlacklistDetailResponse
// @Failure      404   {object}  apidoc.Error "黑名单不存在"
// @Router       /api/admin/blacklists/{id} [get]
// @Security     BearerAuth
func (c *BlacklistController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	blacklist, err := c.BlacklistService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "blacklist", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"blacklist": blacklist,
	})
}

// Store 创建黑名单
// @Summary      创建黑名单
// @Description  创建黑名单记录，支持单个IP、CIDR、IP范围等格式
// @Tags         风控管理
// @Accept       json
// @Produce      json
// @Param        request  body      adminrequests.BlacklistCreate  true  "创建参数"
// @Success      200      {object}  BlacklistDetailResponse
// @Failure      400      {object}  apidoc.Error "参数错误或IP格式错误"
// @Failure      500      {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/blacklists [post]
// @Security     BearerAuth
func (c *BlacklistController) Store(ctx http.Context) http.Response {
	if resp := RequireSensitiveConfirm(ctx, "blacklist"); resp != nil {
		return resp
	}

	var req adminrequests.BlacklistCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	blacklist, err := c.BlacklistService(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "blacklist", http.StatusInternalServerError, err, map[string]any{"ip": req.IP})
	}

	return response.Success(ctx, http.Json{
		"blacklist": blacklist,
	})
}

// Update 更新黑名单
// @Summary      更新黑名单
// @Description  根据ID更新黑名单信息
// @Tags         风控管理
// @Accept       json
// @Produce      json
// @Param        id       path      int                           true  "黑名单ID"
// @Param        request  body      adminrequests.BlacklistUpdate true  "更新参数"
// @Success      200      {object}  BlacklistDetailResponse
// @Failure      400      {object}  apidoc.Error "参数错误或IP格式错误"
// @Failure      404      {object}  apidoc.Error "黑名单不存在"
// @Failure      500      {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/blacklists/{id} [put]
// @Security     BearerAuth
func (c *BlacklistController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.BlacklistUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	blacklist, err := c.BlacklistService(ctx).Update(id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "blacklist", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"blacklist": blacklist,
	})
}

// Destroy 删除黑名单
// @Summary      删除黑名单
// @Description  根据ID删除黑名单记录
// @Tags         风控管理
// @Accept       json
// @Produce      json
// @Param        id    path      int  true  "黑名单ID"
// @Success      200   {object}  apidoc.Success
// @Failure      404   {object}  apidoc.Error "黑名单不存在"
// @Failure      500   {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/blacklists/{id} [delete]
// @Security     BearerAuth
func (c *BlacklistController) Destroy(ctx http.Context) http.Response {
	if resp := RequireSensitiveConfirm(ctx, "blacklist"); resp != nil {
		return resp
	}

	id := helpers.GetUintRoute(ctx, "id")
	if err := c.BlacklistService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "blacklist", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}
