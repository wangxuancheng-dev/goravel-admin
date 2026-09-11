package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/apidoc"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"
)

type PaymentMethodController struct{}

type PaymentMethodResponse struct {
	ID          uint           `json:"id" example:"1"`                           // 支付方式ID
	Name        string         `json:"name" example:"微信支付"`                      // 名称
	Code        string         `json:"code" example:"wechat"`                    // 代码
	Type        string         `json:"type" example:"wechat"`                    // 类型
	Config      map[string]any `json:"config,omitempty"`                         // 配置（详情接口返回）
	IsActive    bool           `json:"is_active" example:"true"`                 // 是否启用
	Sort        int            `json:"sort" example:"10"`                        // 排序
	Description string         `json:"description" example:"默认支付方式"`             // 描述
	CreatedAt   string         `json:"created_at" example:"2024-01-01 00:00:00"` // 创建时间
	UpdatedAt   string         `json:"updated_at" example:"2024-01-01 00:00:00"` // 更新时间
}

type PaymentMethodListData struct {
	List []PaymentMethodResponse `json:"list"`
	apidoc.Pagination
}

type PaymentMethodListResponse struct {
	apidoc.Success
	Data PaymentMethodListData `json:"data"`
}

type PaymentMethodDetailResponse struct {
	apidoc.Success
	Data PaymentMethodResponse `json:"data"`
}

func NewPaymentMethodController() *PaymentMethodController {
	return &PaymentMethodController{}
}

func (c *PaymentMethodController) PaymentMethodService(ctx http.Context) services.PaymentMethodService {
	return services.NewPaymentMethodService(ctx)
}

func (c *PaymentMethodController) buildPaymentMethodFilters(ctx http.Context) services.PaymentMethodFilters {
	return services.BuildPaymentMethodFiltersFromHTTP(ctx)
}

// Index 支付方式列表
// @Summary      获取支付方式列表
// @Description  分页获取支付方式列表，支持多条件筛选
// @Tags         支付管理
// @Accept       json
// @Produce      json
// @Param        page       query    int     false "页码" default(1)
// @Param        page_size  query    int     false "每页数量" default(10)
// @Param        name       query    string  false "支付方式名称（模糊搜索）"
// @Param        code       query    string  false "支付方式代码"
// @Param        type       query    string  false "支付类型"
// @Param        is_active  query    string  false "是否启用：1-启用，0-禁用"
// @Param        order_by   query    string  false "排序（格式：字段:asc/desc，如：created_at:desc）"
// @Success      200        {object} PaymentMethodListResponse
// @Failure      400        {object} apidoc.Error "参数错误"
// @Failure      500        {object} apidoc.Error "服务器错误"
// @Router       /api/admin/payment-methods [get]
// @Security     BearerAuth
func (c *PaymentMethodController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	filters := c.buildPaymentMethodFilters(ctx)

	paymentMethods, total, err := c.PaymentMethodService(ctx).GetPaymentMethods(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment_method", http.StatusInternalServerError, err, map[string]any{
			"filters": filters,
		})
	}

	svc := c.PaymentMethodService(ctx)
	paymentMethodList := make([]http.Json, len(paymentMethods))
	for i, pm := range paymentMethods {
		paymentMethodList[i] = svc.PaymentMethodListItem(pm)
	}

	return response.Success(ctx, http.Json{
		"list":      paymentMethodList,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Show 支付方式详情
// @Summary      获取支付方式详情
// @Description  根据ID获取支付方式详细信息
// @Tags         支付管理
// @Accept       json
// @Produce      json
// @Param        id         path     int     true  "支付方式ID"
// @Success      200        {object} PaymentMethodDetailResponse
// @Failure      400        {object} apidoc.Error "参数错误"
// @Failure      404        {object} apidoc.Error "支付方式不存在"
// @Failure      500        {object} apidoc.Error "服务器错误"
// @Router       /api/admin/payment-methods/{id} [get]
// @Security     BearerAuth
func (c *PaymentMethodController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	paymentMethod, err := c.PaymentMethodService(ctx).GetPaymentMethodByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment_method", http.StatusNotFound, err, map[string]any{"id": id})
	}

	// Keep flat detail payload for existing Vue/React form loaders.
	return response.Success(ctx, c.PaymentMethodService(ctx).PaymentMethodDetail(paymentMethod))
}

// Store 创建支付方式
// @Summary      创建支付方式
// @Description  创建新的支付方式
// @Tags         支付管理
// @Accept       json
// @Produce      json
// @Param        name        body     string  true  "支付方式名称"
// @Param        code        body     string  true  "支付方式代码"
// @Param        type        body     string  true  "支付类型"
// @Param        config      body     object  true  "支付配置(JSON对象)"
// @Param        is_active   body     bool    false "是否启用"
// @Param        sort        body     int     false "排序"
// @Param        description body     string  false "描述"
// @Success      200      {object} PaymentMethodDetailResponse
// @Failure      400      {object} apidoc.Error "参数错误"
// @Failure      500      {object} apidoc.Error "服务器错误"
// @Router       /api/admin/payment-methods [post]
// @Security     BearerAuth
func (c *PaymentMethodController) Store(ctx http.Context) http.Response {
	var req adminrequests.PaymentMethodCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	paymentMethod, err := c.PaymentMethodService(ctx).CreatePaymentMethodFromRequest(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment_method", http.StatusInternalServerError, err, map[string]any{
			"name": req.Name,
			"code": req.Code,
		})
	}

	return response.Success(ctx, c.PaymentMethodService(ctx).PaymentMethodListItem(*paymentMethod))
}

// Update 更新支付方式
// @Summary      更新支付方式
// @Description  更新支付方式信息
// @Tags         支付管理
// @Accept       json
// @Produce      json
// @Param        id          path     int     true  "支付方式ID"
// @Param        name        body     string  true  "支付方式名称"
// @Param        config      body     object  false "支付配置(JSON对象)"
// @Param        is_active   body     bool    false "是否启用"
// @Param        sort        body     int     false "排序"
// @Param        description body     string  false "描述"
// @Success      200        {object} PaymentMethodDetailResponse
// @Failure      400        {object} apidoc.Error "参数错误"
// @Failure      500        {object} apidoc.Error "服务器错误"
// @Router       /api/admin/payment-methods/{id} [put]
// @Security     BearerAuth
func (c *PaymentMethodController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.PaymentMethodUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	paymentMethod, err := c.PaymentMethodService(ctx).UpdatePaymentMethodByRequest(ctx, id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment_method", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"payment_method": *paymentMethod,
	})
}

// Destroy 删除支付方式
// @Summary      删除支付方式
// @Description  删除支付方式
// @Tags         支付管理
// @Accept       json
// @Produce      json
// @Param        id         path     int     true  "支付方式ID"
// @Success      200        {object} apidoc.Success
// @Failure      400        {object} apidoc.Error "参数错误"
// @Failure      500        {object} apidoc.Error "服务器错误"
// @Router       /api/admin/payment-methods/{id} [delete]
// @Security     BearerAuth
func (c *PaymentMethodController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.PaymentMethodService(ctx).DeletePaymentMethod(id); err != nil {
		return HandleGeneratedServiceError(ctx, "payment_method", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}
