package services

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/constants"
	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/rbac"
	"goravel/app/utils"
)

type LoginLogService interface {
	GetByID(id uint, withAdmin bool) (*models.LoginLog, error)
	GetList(filters LoginLogFilters, page, pageSize int) ([]models.LoginLog, int64, error)
	Delete(id uint) error
	BatchDelete(ids []uint) error
	Clean(days int) error
	Archive(days int) (uint, error)
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
	if !rbac.CanAccessOwnedBy(s.ctx, log.AdminID) {
		return nil, apperrors.ErrForbidden
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

	query = rbac.ApplyDataScope(s.ctx, query, rbac.DataScopeApplyOpts{Mode: rbac.DataScopeModeAdmin, AdminColumn: "admin_id"})

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

// Archive exports login logs older than N days to CSV then deletes those rows.
func (s *LoginLogServiceImpl) Archive(days int) (uint, error) {
	if days <= 0 {
		days = constants.DefaultCleanLogDays
	}
	cutoffTime := time.Now().AddDate(0, 0, -days)

	var logs []models.LoginLog
	if err := appfacades.OrmQuery(s.ctx).Model(&models.LoginLog{}).
		Where("created_at < ?", cutoffTime).
		Order("id asc").
		Find(&logs); err != nil {
		return 0, apperrors.ErrQueryFailed.WithError(err)
	}

	adminID := uint(0)
	if admin := rbac.AdminFromContext(s.ctx); admin != nil {
		adminID = admin.ID
	}

	disk := utils.ResolveFileDisk(s.ctx)
	exportRecord := models.Export{
		AdminID: adminID,
		Type:    models.ExportTypeLoginLogsArchive,
		Status:  models.ExportStatusProcessing,
		Disk:    disk,
	}
	if err := appfacades.OrmQuery(s.ctx).Create(&exportRecord); err != nil {
		return 0, apperrors.ErrCreateFailed.WithError(err)
	}

	headers := []string{
		"id", "admin_id", "username", "ip", "user_agent", "location", "status", "message", "request", "created_at",
	}
	data := make([][]string, 0, len(logs))
	for _, log := range logs {
		createdAt := ""
		if log.CreatedAt != nil && !log.CreatedAt.IsZero() {
			createdAt = log.CreatedAt.ToDateTimeString()
		}
		data = append(data, []string{
			fmt.Sprintf("%d", log.ID),
			fmt.Sprintf("%d", log.AdminID),
			log.Username,
			log.IP,
			log.UserAgent,
			log.Location,
			fmt.Sprintf("%d", log.Status),
			log.Message,
			log.Request,
			createdAt,
		})
	}

	filename := fmt.Sprintf("login_logs_archive_%d_%s.csv", exportRecord.ID, time.Now().Format("20060102_150405"))
	exportService := &ExportServiceImpl{
		disk:   disk,
		path:   "exports",
		format: "csv",
	}
	filePath, err := exportService.ExportToCSV(headers, data, filename, true)
	if err != nil {
		exportRecord.Status = models.ExportStatusFailed
		exportRecord.ErrorMsg = err.Error()
		_ = appfacades.OrmQuery(s.ctx).Save(&exportRecord)
		return exportRecord.ID, err
	}

	exportRecord.Path = filePath
	exportRecord.Filename = path.Base(filePath)
	exportRecord.Extension = "csv"
	exportRecord.Status = models.ExportStatusSuccess
	if storage, storErr := utils.StorageDisk(exportRecord.Disk); storErr == nil {
		if size, sizeErr := storage.Size(filePath); sizeErr == nil {
			exportRecord.Size = size
		}
	}
	if err := appfacades.OrmQuery(s.ctx).Save(&exportRecord); err != nil {
		return exportRecord.ID, apperrors.ErrUpdateFailed.WithError(err)
	}

	if err := s.Clean(days); err != nil {
		return exportRecord.ID, err
	}
	return exportRecord.ID, nil
}
