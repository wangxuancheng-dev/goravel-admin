package admin

import (
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/apidoc"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"
)

type DictionaryController struct{}

type DictionaryResponse struct {
	ID             uint   `json:"id" example:"1"`
	Type           string `json:"type" example:"order_status"`
	Label          string `json:"label" example:"已支付"`
	Value          string `json:"value" example:"paid"`
	TranslationKey string `json:"translation_key" example:"order.status.paid"`
	Description    string `json:"description" example:"订单支付成功状态"`
	Status         uint8  `json:"status" enums:"0,1" example:"1"`
	IsSystem       uint8  `json:"is_system" enums:"0,1" example:"0"`
	Sort           int    `json:"sort" example:"10"`
	Remark         string `json:"remark" example:"系统默认值"`
	CreatedAt      string `json:"created_at" example:"2024-01-01 00:00:00"`
	UpdatedAt      string `json:"updated_at" example:"2024-01-01 00:00:00"`
}

type DictionaryListData struct {
	List []DictionaryResponse `json:"list"`
	apidoc.Pagination
}

type DictionaryListResponse struct {
	apidoc.Success
	Data DictionaryListData `json:"data"`
}

type DictionaryDetailData struct {
	Dictionary DictionaryResponse `json:"dictionary"`
}

type DictionaryDetailResponse struct {
	apidoc.Success
	Data DictionaryDetailData `json:"data"`
}

type DictionaryByTypeData struct {
	Dictionaries []DictionaryResponse `json:"dictionaries"`
}

type DictionaryByTypeResponse struct {
	apidoc.Success
	Data DictionaryByTypeData `json:"data"`
}

type DictionaryTypesData struct {
	Types []string `json:"types"`
}

type DictionaryTypesResponse struct {
	apidoc.Success
	Data DictionaryTypesData `json:"data"`
}

func NewDictionaryController() *DictionaryController {
	return &DictionaryController{}
}

func (c *DictionaryController) buildDictionaryFilters(ctx http.Context) services.DictionaryFilters {
	return services.BuildDictionaryFiltersFromHTTP(ctx)
}

func (c *DictionaryController) DictionaryService(ctx http.Context) services.DictionaryService {
	return services.NewDictionaryService(ctx)
}

// Index 字典列表
// @Summary      获取字典列表
// @Description  分页获取字典列表，支持按类型、状态、时间范围筛选
// @Tags         字典管理
// @Accept       json
// @Produce      json
// @Param        page        query     int     false  "页码（从1开始）" default(1)
// @Param        page_size   query     int     false  "每页数量（建议 10-100）" default(10)
// @Param        type        query     string  false  "字典类型（精确匹配）"
// @Param        status      query     string  false  "状态（1-启用，0-禁用）" Enums(0,1)
// @Param        start_time  query     string  false  "开始时间（格式：YYYY-MM-DD HH:mm:ss）"
// @Param        end_time    query     string  false  "结束时间（格式：YYYY-MM-DD HH:mm:ss）"
// @Param        order_by    query     string  false  "排序（格式：字段:asc/desc，例如：created_at:desc）"
// @Success      200         {object}  DictionaryListResponse
// @Failure      500         {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/dictionaries [get]
// @Security     BearerAuth
func (c *DictionaryController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	filters := c.buildDictionaryFilters(ctx)

	list, total, err := c.DictionaryService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "dictionary", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Show 字典详情
// @Summary      获取字典详情
// @Description  根据ID获取字典详情
// @Tags         字典管理
// @Accept       json
// @Produce      json
// @Param        id      path      int  true  "字典ID"
// @Success      200     {object}  DictionaryDetailResponse
// @Failure      404     {object}  apidoc.Error "字典不存在"
// @Router       /api/admin/dictionaries/{id} [get]
// @Security     BearerAuth
func (c *DictionaryController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	dictionary, err := c.DictionaryService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "dictionary", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"dictionary": dictionary,
	})
}

// Store 创建字典
// @Summary      创建字典
// @Description  创建新的字典项
// @Tags         字典管理
// @Accept       json
// @Produce      json
// @Param        request  body      adminrequests.DictionaryCreate  true  "创建参数"
// @Success      200      {object}  DictionaryDetailResponse
// @Failure      400      {object}  apidoc.Error "参数错误"
// @Failure      500      {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/dictionaries [post]
// @Security     BearerAuth
func (c *DictionaryController) Store(ctx http.Context) http.Response {
	var req adminrequests.DictionaryCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	dictionary, err := c.DictionaryService(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "dictionary", http.StatusInternalServerError, err, map[string]any{
			"type": req.Type, "label": req.Label,
		})
	}

	return response.Success(ctx, http.Json{
		"dictionary": dictionary,
	})
}

// Update 更新字典
// @Summary      更新字典
// @Description  根据ID更新字典信息
// @Tags         字典管理
// @Accept       json
// @Produce      json
// @Param        id       path      int                              true  "字典ID"
// @Param        request  body      adminrequests.DictionaryUpdate   true  "更新参数"
// @Success      200      {object}  DictionaryDetailResponse
// @Failure      400      {object}  apidoc.Error "参数错误"
// @Failure      404      {object}  apidoc.Error "字典不存在"
// @Failure      500      {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/dictionaries/{id} [put]
// @Security     BearerAuth
func (c *DictionaryController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.DictionaryUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	dictionary, err := c.DictionaryService(ctx).Update(id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "dictionary", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"dictionary": dictionary,
	})
}

// Destroy 删除字典
// @Summary      删除字典
// @Description  根据ID删除字典项
// @Tags         字典管理
// @Accept       json
// @Produce      json
// @Param        id    path      int  true  "字典ID"
// @Success      200   {object}  apidoc.Success
// @Failure      404   {object}  apidoc.Error "字典不存在"
// @Failure      500   {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/dictionaries/{id} [delete]
// @Security     BearerAuth
func (c *DictionaryController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.DictionaryService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "dictionary", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}

// GetByType 按类型获取字典项
// @Summary      按类型获取字典项
// @Description  根据字典类型获取启用状态的字典列表
// @Tags         字典管理
// @Accept       json
// @Produce      json
// @Param        type   path      string  true  "字典类型"
// @Success      200    {object}  DictionaryByTypeResponse
// @Failure      400    {object}  apidoc.Error "字典类型不能为空"
// @Failure      500    {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/dictionaries/type/{type} [get]
// @Security     BearerAuth
func (c *DictionaryController) GetByType(ctx http.Context) http.Response {
	dictType := ctx.Request().Route("type")
	if dictType == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrDictionaryTypeRequired.Code)
	}

	dictionaries, err := c.DictionaryService(ctx).GetByType(dictType)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "dictionary", http.StatusInternalServerError, err, map[string]any{"type": dictType})
	}

	return response.Success(ctx, http.Json{
		"dictionaries": dictionaries,
	})
}

// GetAllTypes 获取全部字典类型
// @Summary      获取全部字典类型
// @Description  获取系统中已配置的全部字典类型列表
// @Tags         字典管理
// @Accept       json
// @Produce      json
// @Success      200    {object}  DictionaryTypesResponse
// @Failure      500    {object}  apidoc.Error "服务器错误"
// @Router       /api/admin/dictionaries/types [get]
// @Security     BearerAuth
func (c *DictionaryController) GetAllTypes(ctx http.Context) http.Response {
	types, err := c.DictionaryService(ctx).GetAllTypes()
	if err != nil {
		return HandleGeneratedServiceError(ctx, "dictionary", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"types": types,
	})
}
