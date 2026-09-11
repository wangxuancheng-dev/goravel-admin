package admin

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
)

type NotificationController struct{}

func NewNotificationController() *NotificationController {
	return &NotificationController{}
}

func (r *NotificationController) service(ctx http.Context) services.NotificationService {
	return services.NewNotificationServiceImpl(ctx)
}

func (r *NotificationController) Index(ctx http.Context) http.Response {
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	notifType := ctx.Request().Query("type", "")
	isRead := ctx.Request().Query("is_read", "")
	notifications, total, err := r.service(ctx).List(admin.ID, page, pageSize, notifType, isRead)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "notification", http.StatusInternalServerError, err, nil)
	}
	count, _ := r.service(ctx).UnreadCount(admin.ID)

	return response.Success(ctx, http.Json{
		"notifications": notifications,
		"unread_count":  count,
		"pagination": http.Json{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (r *NotificationController) UnreadCount(ctx http.Context) http.Response {
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	count, err := r.service(ctx).UnreadCount(admin.ID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "notification", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"count": count,
	})
}

func (r *NotificationController) Recent(ctx http.Context) http.Response {
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	limit := helpers.GetIntQuery(ctx, "limit", 5)
	notifications, err := r.service(ctx).ListRecent(admin.ID, limit)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "notification", http.StatusInternalServerError, err, nil)
	}

	count, _ := r.service(ctx).UnreadCount(admin.ID)

	return response.Success(ctx, http.Json{
		"notifications": notifications,
		"unread_count":  count,
	})
}

func (r *NotificationController) MarkRead(ctx http.Context) http.Response {
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsRequired.Code)
	}

	if err := r.service(ctx).MarkRead(admin.ID, id); err != nil {
		return HandleGeneratedServiceError(ctx, "notification", http.StatusInternalServerError, err, map[string]any{
			"id": id,
		})
	}

	return response.Success(ctx, http.Json{
		"id": id,
	})
}

func (r *NotificationController) MarkAllRead(ctx http.Context) http.Response {
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	if err := r.service(ctx).MarkAllRead(admin.ID); err != nil {
		return HandleGeneratedServiceError(ctx, "notification", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx)
}

func (r *NotificationController) Store(ctx http.Context) http.Response {
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	title := ctx.Request().Input("title")
	content := ctx.Request().Input("content")
	notificationType := ctx.Request().Input("type", "announcement")
	if title == "" || content == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsRequired.Code)
	}

	var receiverID *uint
	receiverVal := ctx.Request().Input("receiver_id")
	if receiverVal != "" {
		id := cast.ToUint(receiverVal)
		if id > 0 {
			receiverID = &id
		}
	}

	notification, err := r.service(ctx).CreateAnnouncement(ctx, title, content, notificationType, admin.ID, receiverID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "notification", http.StatusInternalServerError, err, nil)
	}

	if notification == nil {
		return response.Success(ctx)
	}

	return response.Success(ctx, http.Json{
		"notification": notification,
	})
}

func (r *NotificationController) currentAdmin(ctx http.Context) *models.Admin {
	if adminValue := ctx.Value("admin"); adminValue != nil {
		if admin, ok := adminValue.(models.Admin); ok {
			return &admin
		}
		if adminPtr, ok := adminValue.(*models.Admin); ok {
			return adminPtr
		}
	}
	return nil
}
