package services

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/rbac"
	"goravel/app/tenancyctx"
	"goravel/app/utils"
	wsnotifications "goravel/app/websocket/notifications"
)

type NotificationService interface {
	Create(title, content, notifType string, senderID *uint, receiverID *uint) (*models.Notification, error)
	CreateAnnouncement(ctx http.Context, title, content, notifType string, senderID uint, receiverID *uint) (*models.Notification, error)
	List(adminID uint, page int, pageSize int, notifType string, isRead string) ([]models.Notification, int64, error)
	ListRecent(adminID uint, limit int) ([]models.Notification, error)
	MarkRead(adminID uint, notificationID uint) error
	MarkAllRead(adminID uint) error
	UnreadCount(adminID uint) (int64, error)
}

type NotificationServiceImpl struct {
	ctx context.Context
}

func NewNotificationServiceImpl(ctx context.Context) NotificationService {
	return &NotificationServiceImpl{ctx: ctx}
}

// PrepareAnnouncementContent normalizes rich-text image URLs to relative paths and sanitizes content.
func PrepareAnnouncementContent(ctx http.Context, raw string) string {
	content := raw

	re := regexp.MustCompile(`https?://[^/]+(/api/admin/public/images/)`)
	content = re.ReplaceAllString(content, "$1")

	appURL := facades.Config().GetString("app.url")
	if appURL != "" {
		appURL = strings.TrimSuffix(appURL, "/")
		content = strings.ReplaceAll(content, appURL, "")
	}

	if ctx != nil {
		host := ctx.Request().Header("Host", "")
		if host != "" {
			scheme := "http"
			if ctx.Request().Header("X-Forwarded-Proto", "") == "https" {
				scheme = "https"
			}
			currentBaseURL := scheme + "://" + host + "/"
			content = strings.ReplaceAll(content, currentBaseURL, "/")
			currentBaseURLNoSlash := scheme + "://" + host
			content = strings.ReplaceAll(content, currentBaseURLNoSlash, "")
		}
	}

	return utils.SanitizeRichTextContent(content)
}

func (s *NotificationServiceImpl) CreateAnnouncement(ctx http.Context, title, content, notifType string, senderID uint, receiverID *uint) (*models.Notification, error) {
	if title == "" {
		return nil, apperrors.ErrParamsRequired
	}
	content = PrepareAnnouncementContent(ctx, content)
	if content == "" {
		return nil, apperrors.ErrParamsRequired
	}
	if notifType == "" {
		notifType = "announcement"
	}
	return s.Create(title, content, notifType, &senderID, receiverID)
}

func (s *NotificationServiceImpl) Create(title, content, notifType string, senderID *uint, receiverID *uint) (*models.Notification, error) {
	if receiverID == nil {
		var admins []models.Admin
		if err := appfacades.OrmQuery(s.ctx).Find(&admins); err != nil {
			return nil, apperrors.ErrQueryFailed.WithError(err)
		}

		if len(admins) == 0 {
			return nil, apperrors.ErrRecordNotFound.WithMessage("no admins found")
		}

		var first *models.Notification
		var notifications []*models.Notification
		var createdIDs []uint

		for _, admin := range admins {
			rid := admin.ID
			notification := &models.Notification{
				Title:      title,
				Content:    content,
				Type:       notifType,
				SenderID:   senderID,
				ReceiverID: &rid,
			}
			if err := appfacades.OrmQuery(s.ctx).Create(notification); err != nil {
				for _, id := range createdIDs {
					_, _ = appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.Notification{})
				}
				return nil, apperrors.ErrCreateFailed.WithError(err)
			}
			if first == nil {
				first = notification
			}
			notifications = append(notifications, notification)
			createdIDs = append(createdIDs, notification.ID)
		}

		for _, notification := range notifications {
			wsnotifications.Hub().Broadcast(s.wsTenantID(), notification)
		}

		DispatchNotificationChannels(s.ctx, notifications...)

		return first, nil
	}

	notification := &models.Notification{
		Title:      title,
		Content:    content,
		Type:       notifType,
		SenderID:   senderID,
		ReceiverID: receiverID,
	}
	if err := appfacades.OrmQuery(s.ctx).Create(notification); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	wsnotifications.Hub().Broadcast(s.wsTenantID(), notification)
	DispatchNotificationChannels(s.ctx, notification)

	return notification, nil
}

// buildNotificationQuery 构建通知查询条件（消除代码重复）
func (s *NotificationServiceImpl) buildNotificationQuery(adminID uint, notifType, isRead string) orm.Query {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Notification{})

	if notifType == "message" {
		query = query.Where("(receiver_id = ? OR sender_id = ?) AND type = ?", adminID, adminID, "message")
	} else if notifType != "" {
		query = query.Where("receiver_id = ? AND type = ?", adminID, notifType)
	} else {
		query = query.Where("receiver_id = ? OR (sender_id = ? AND type = ?)", adminID, adminID, "message")
	}

	if isRead == "true" {
		query = query.Where("is_read = ?", true)
	} else if isRead == "false" {
		query = query.Where("is_read = ?", false)
	}

	// Scope by receiver_id for announcement-style lists. Skip for message threads so
	// "sent by me" rows remain visible when the receiver is outside the actor's dept scope.
	if notifType != "message" {
		query = rbac.ApplyDataScope(s.ctx, query, rbac.DataScopeApplyOpts{Mode: rbac.DataScopeModeAdmin, AdminColumn: "receiver_id"})
	}
	return query
}

func (s *NotificationServiceImpl) List(adminID uint, page int, pageSize int, notifType string, isRead string) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := s.buildNotificationQuery(adminID, notifType, isRead).With("Sender").With("Receiver").Order("created_at desc")
	if err := query.Paginate(page, pageSize, &notifications, &total); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}

	return notifications, total, nil
}

func (s *NotificationServiceImpl) ListRecent(adminID uint, limit int) ([]models.Notification, error) {
	var notifications []models.Notification
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	if err := appfacades.OrmQuery(s.ctx).Model(&models.Notification{}).With("Sender").With("Receiver").
		Where("(receiver_id = ? OR (sender_id = ? AND type = ?))", adminID, adminID, "message").
		Order("created_at desc").
		Limit(limit).
		Find(&notifications); err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}
	return notifications, nil
}

func (s *NotificationServiceImpl) MarkRead(adminID uint, notificationID uint) error {
	var notification models.Notification
	if err := appfacades.OrmQuery(s.ctx).Where("id = ?", notificationID).
		Where("receiver_id = ?", adminID).
		First(&notification); err != nil {
		return apperrors.ErrRecordNotFound.WithMessage("notification not found")
	}

	if notification.IsRead {
		return nil
	}

	now := time.Now()

	_, err := appfacades.OrmQuery(s.ctx).
		Model(&models.Notification{}).
		Where("id = ?", notificationID).
		Update(map[string]any{
			"is_read": true,
			"read_at": now,
		})
	if err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}

	notification.IsRead = true
	notification.ReadAt = &now
	wsnotifications.Hub().Broadcast(s.wsTenantID(), &notification)

	return nil
}

func (s *NotificationServiceImpl) MarkAllRead(adminID uint) error {
	now := time.Now()
	_, err := appfacades.OrmQuery(s.ctx).
		Table("notifications").
		Where("receiver_id = ?", adminID).
		Where("is_read = ?", false).
		Update(map[string]any{
			"is_read": true,
			"read_at": now,
		})
	if err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}

	wsnotifications.Hub().SendToAdmin(s.wsTenantID(), adminID, map[string]any{
		"type":        "read_all",
		"receiver_id": adminID,
		"read_at":     now.Format(time.RFC3339),
	})

	return nil
}

func (s *NotificationServiceImpl) UnreadCount(adminID uint) (int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Notification{}).
		Where("receiver_id = ?", adminID).
		Where("is_read = ?", false)

	count, err := query.Count()
	if err != nil {
		return 0, apperrors.ErrQueryFailed.WithError(err)
	}
	return count, nil
}

func (s *NotificationServiceImpl) wsTenantID() uint {
	if tid, ok := tenancyctx.IDFrom(s.ctx); ok {
		return tid
	}
	return 0
}
