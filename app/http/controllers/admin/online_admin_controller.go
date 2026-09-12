package admin

import (
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

type OnlineAdminController struct{}

func NewOnlineAdminController() *OnlineAdminController {
	return &OnlineAdminController{}
}

func (c *OnlineAdminController) OnlineAdminService(ctx http.Context) services.OnlineAdminService {
	return services.NewOnlineAdminService(ctx)
}

// Index 获取在线管理员列表（最近 OnlineAdminThreshold 内有活动）
func (c *OnlineAdminController) Index(ctx http.Context) http.Response {
	filters := services.BuildOnlineAdminFiltersFromHTTP(ctx)
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})

	list, total, err := c.OnlineAdminService(ctx).List(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "online_admin", http.StatusInternalServerError, err, map[string]any{
			"filters": filters,
		})
	}

	return response.Paginate(ctx, list, total, page, pageSize)
}

// KickOut 踢下线（删除 token）
func (c *OnlineAdminController) KickOut(ctx http.Context) http.Response {
	if resp := RequireSensitiveConfirm(ctx, "online_admin"); resp != nil {
		return resp
	}

	tokenID := helpers.GetUintRoute(ctx, "id")
	if err := c.OnlineAdminService(ctx).KickOut(tokenID); err != nil {
		return HandleGeneratedServiceError(ctx, "online_admin", http.StatusInternalServerError, err, map[string]any{
			"token_id": tokenID,
		})
	}
	return response.Success(ctx, "kick_out_success")
}

// BatchKickOut 批量踢下线
func (c *OnlineAdminController) BatchKickOut(ctx http.Context) http.Response {
	if resp := RequireSensitiveConfirm(ctx, "online_admin"); resp != nil {
		return resp
	}

	tokenIDs := ctx.Request().Input("token_ids")
	if tokenIDs == "" {
		return HandleGeneratedServiceError(ctx, "online_admin", http.StatusBadRequest, apperrors.ErrTokenIDsRequired, nil)
	}

	ids := helpers.ParseIDsFromString(tokenIDs)
	count, err := c.OnlineAdminService(ctx).BatchKickOut(ids)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "online_admin", http.StatusInternalServerError, err, map[string]any{
			"token_ids": ids,
		})
	}

	return response.Success(ctx, "batch_kick_out_success", http.Json{
		"count": count,
	})
}
