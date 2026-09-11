package api

import (
	"context"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/helpers"
	apirequests "goravel/app/http/requests/api"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/search"
	"goravel/app/services"
)

type OrderController struct {}


func NewOrderController() *OrderController {
	return &OrderController{}
}

func (c *OrderController) orderService(ctx http.Context) services.OrderService {
	return services.NewOrderService(ctx)
}


// SearchMyOrders GET：当前登录用户检索自己的订单。开启搜索引擎时走索引（可多字段含商品名）；否则走分表数据库。
func (c *OrderController) SearchMyOrders(ctx http.Context) http.Response {
	var req apirequests.OrderSearch
	errors, err := ctx.Request().ValidateRequest(&req)
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if errors != nil {
		return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", errors.All())
	}

	userID, err := helpers.GetUserIDFromContext(ctx)
	if err != nil || userID == 0 {
		return response.Error(ctx, http.StatusUnauthorized, "not_logged_in")
	}

	page, pageSize := helpers.ValidatePaginationEx(req.Page, req.PageSize, helpers.PaginationLimits{})

	timeRange, errField, errMsgKey := helpers.ParseOrderSearchCreatedRange(req.CreatedFrom, req.CreatedTo)
	if errField != "" {
		return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", map[string]map[string]string{
			errField: {"time": trans.Get(ctx, errMsgKey)},
		})
	}

	searchCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	list, total, err := c.orderService(ctx).SearchMyOrdersForUser(searchCtx, userID, req.Q, page, pageSize, timeRange)
	if err != nil {
		if resp := timeRangeErrorResponse(ctx, err); resp != nil {
			return resp
		}
		if search.Enabled() {
			facades.Log().Errorf("order search: %v", err)
			return response.Error(ctx, http.StatusBadGateway, "search_failed")
		}
		facades.Log().Errorf("order DB search: %v", err)
		return response.Error(ctx, http.StatusInternalServerError, "query_failed")
	}

	return response.Paginate(ctx, "success", list, total, page, pageSize)
}
