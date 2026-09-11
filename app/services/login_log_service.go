package services

import (
	"context"
	"time"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/constants"
	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
)

type LoginLogService interface {
	GetByID(id uint, withAdmin bool) (*models.LoginLog, error)
	GetList(filters LoginLogFilters, page, pageSize int) ([]models.LoginLog, int64, error)
	Delete(id uint) error
	BatchDelete(ids []uint) error
	Clean(days int) error
}

// LoginLogFilters 登录日志查询过滤器
type LoginLogFilters struct {
	AdminID   string
	Username  string
	IP        string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

func BuildLoginLogFiltersFromHTTP(ctx http.Context) LoginLogFilters {
	return LoginLogFilters{
		AdminID:   ctx.Request().Query("admin_id", ""),
		Username:  ctx.Request().Query("username", ""),
		IP:        ctx.Request().Query("ip", ""),
		Status:    ctx.Request().Query("status", ""),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Query("order_by", ""),
	}
}

type LoginLogServiceImpl struct {
	ctx context.Context
}

func NewLoginLogService(ctx context.Context) LoginLogService {
	return &LoginLogServiceImpl{ctx: ctx}
}

func (s *LoginLogServiceImpl) GetByID(id uint, withAdmin bool) (*models.LoginLog, error) {
	var log models.LoginLog
	query := appfacades.OrmQuery(s.ctx).Where("id", id)
	if withAdmin {
		query = query.With("Admin")
	}
	if err := query.FirstOrFail(&log); err != nil {
		return nil, apperrors.ErrLogNotFound.WithError(err)
	}
	return &log, nil
}

func (s *LoginLogServiceImpl) GetList(filters LoginLogFilters, page, pageSize int) ([]models.LoginLog, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.LoginLog{})

	if filters.AdminID != "" {
		query = query.Where("admin_id", filters.AdminID)
	}
	if filters.Username != "" {
		query = query.Where("username LIKE ?", "%"+filters.Username+"%")
	}
	if filters.IP != "" {
		query = query.Where("ip LIKE ?", "%"+filters.IP+"%")
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "id:desc"
	}
	query = helpers.ApplySort(query, orderBy, "id:desc")

	var logs []models.LoginLog
	var total int64
	if err := query.With("Admin").Paginate(page, pageSize, &logs, &total); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}
	return logs, total, nil
}

func (s *LoginLogServiceImpl) Delete(id uint) error {
	log, err := s.GetByID(id, false)
	if err != nil {
		return err
	}
	if _, err := appfacades.OrmQuery(s.ctx).Delete(log); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *LoginLogServiceImpl) BatchDelete(ids []uint) error {
	if len(ids) == 0 {
		return apperrors.ErrIDsRequired
	}
	idsAny := helpers.ConvertUintSliceToAny(ids)
	if _, err := appfacades.OrmQuery(s.ctx).WhereIn("id", idsAny).Delete(&models.LoginLog{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *LoginLogServiceImpl) Clean(days int) error {
	if days <= 0 {
		days = constants.DefaultCleanLogDays
	}
	cutoffTime := time.Now().AddDate(0, 0, -days)
	if _, err := appfacades.OrmQuery(s.ctx).Model(&models.LoginLog{}).Where("created_at < ?", cutoffTime).Delete(&models.LoginLog{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}
