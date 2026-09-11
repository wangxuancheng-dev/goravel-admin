package services

import (
	"context"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"
	"github.com/spf13/cast"

	"goravel/app/constants"
	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
)

type OnlineAdminService interface {
	List(filters OnlineAdminFilters, page, pageSize int) ([]map[string]any, int64, error)
	KickOut(tokenID uint) error
	BatchKickOut(ids []uint) (int, error)
}

// OnlineAdminFilters 在线管理员查询过滤器
type OnlineAdminFilters struct {
	IP       string
	Browser  string
	OS       string
	Username string
	OrderBy  string
}

func BuildOnlineAdminFiltersFromHTTP(ctx http.Context) OnlineAdminFilters {
	return OnlineAdminFilters{
		IP:       ctx.Request().Query("ip", ""),
		Browser:  ctx.Request().Query("browser", ""),
		OS:       ctx.Request().Query("os", ""),
		Username: ctx.Request().Query("username", ""),
		OrderBy:  ctx.Request().Query("order_by", ""),
	}
}

type OnlineAdminServiceImpl struct {
	ctx context.Context
}

func NewOnlineAdminService(ctx context.Context) OnlineAdminService {
	return &OnlineAdminServiceImpl{ctx: ctx}
}

func (s *OnlineAdminServiceImpl) buildTokenQuery(filters OnlineAdminFilters) orm.Query {
	onlineThreshold := time.Now().Add(-constants.OnlineAdminThreshold)
	query := appfacades.OrmQuery(s.ctx).Model(&models.PersonalAccessToken{}).
		Where("tokenable_type", "admin").
		Where("last_used_at IS NOT NULL").
		Where("last_used_at >= ?", onlineThreshold)

	if filters.IP != "" {
		query = query.Where("ip LIKE ?", "%"+filters.IP+"%")
	}
	if filters.Browser != "" {
		query = query.Where("browser LIKE ?", "%"+filters.Browser+"%")
	}
	if filters.OS != "" {
		query = query.Where("os LIKE ?", "%"+filters.OS+"%")
	}

	query = helpers.ApplySort(query, filters.OrderBy, "last_used_at:desc")
	return query
}

func (s *OnlineAdminServiceImpl) List(filters OnlineAdminFilters, page, pageSize int) ([]map[string]any, int64, error) {
	query := s.buildTokenQuery(filters)

	var tokens []models.PersonalAccessToken
	if err := query.Get(&tokens); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}

	var adminIDs []uint
	adminIDMap := make(map[uint]bool)
	for _, token := range tokens {
		if !adminIDMap[token.TokenableID] {
			adminIDs = append(adminIDs, token.TokenableID)
			adminIDMap[token.TokenableID] = true
		}
	}

	adminMap := make(map[uint]models.Admin)
	if len(adminIDs) > 0 {
		developerIDsStr := facades.Config().GetString("admin.developer_ids", "2")
		developerIDs := s.parseProtectedIDs(developerIDsStr)

		adminQuery := appfacades.OrmQuery(s.ctx).Where("id IN ?", adminIDs)
		if len(developerIDs) > 0 {
			adminQuery = adminQuery.Where("id NOT IN ?", developerIDs)
		}

		var admins []models.Admin
		if err := adminQuery.Find(&admins); err != nil {
			return nil, 0, apperrors.ErrQueryFailed.WithError(err)
		}

		for _, admin := range admins {
			adminMap[admin.ID] = admin
		}
	}

	var onlineAdmins []map[string]any
	for _, token := range tokens {
		admin, ok := adminMap[token.TokenableID]
		if !ok {
			continue
		}

		if filters.Username != "" && !strings.Contains(strings.ToLower(admin.Username), strings.ToLower(filters.Username)) {
			continue
		}

		onlineAdmins = append(onlineAdmins, map[string]any{
			"id":          token.ID,
			"admin_id":    admin.ID,
			"username":    admin.Username,
			"nickname":    admin.Nickname,
			"avatar":      admin.Avatar,
			"browser":     token.Browser,
			"ip":          token.IP,
			"os":          token.OS,
			"session_id":  token.SessionID,
			"last_active": token.LastUsedAt,
			"created_at":  token.CreatedAt,
		})
	}

	paginated, total := helpers.PaginateSlice(onlineAdmins, page, pageSize)
	return paginated, total, nil
}

func (s *OnlineAdminServiceImpl) KickOut(tokenID uint) error {
	if tokenID == 0 {
		return apperrors.ErrTokenIDRequired
	}

	var token models.PersonalAccessToken
	if err := appfacades.OrmQuery(s.ctx).Where("id", tokenID).FirstOrFail(&token); err != nil {
		return apperrors.ErrTokenNotFound.WithError(err)
	}

	if _, err := appfacades.OrmQuery(s.ctx).Delete(&token); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *OnlineAdminServiceImpl) BatchKickOut(ids []uint) (int, error) {
	if len(ids) == 0 {
		return 0, apperrors.ErrInvalidTokenIDs
	}

	idsAny := helpers.ConvertUintSliceToAny(ids)
	if _, err := appfacades.OrmQuery(s.ctx).WhereIn("id", idsAny).Delete(&models.PersonalAccessToken{}); err != nil {
		return 0, apperrors.ErrDeleteFailed.WithError(err)
	}
	return len(ids), nil
}

func (s *OnlineAdminServiceImpl) parseProtectedIDs(idsStr string) []uint {
	var ids []uint
	if idsStr == "" {
		return ids
	}

	parts := str.Of(idsStr).Split(",")
	for _, part := range parts {
		part = str.Of(part).Trim().String()
		if !str.Of(part).IsEmpty() {
			if id := cast.ToUint(part); id > 0 {
				ids = append(ids, id)
			}
		}
	}
	return ids
}
