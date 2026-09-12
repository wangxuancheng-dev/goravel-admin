package services

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/samber/lo"

	"goravel/app/constants"
	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/utils"
)

type OperationLogService interface {
	GetByID(id uint, withAdmin bool) (*models.OperationLog, error)
	GetList(filters OperationLogFilters, page, pageSize int) ([]models.OperationLog, int64, error)
	Delete(id uint) error
	BatchDelete(ids []uint) error
	Clean(days int) error
	Archive(days int) (exportID uint, err error)
	GetTitleOptions() []string
}

// OperationLogFilters 操作日志查询过滤器
type OperationLogFilters struct {
	AdminID   string
	TraceID   string
	Username  string
	Method    string
	Path      string
	Title     string
	IP        string
	Status    string
	Request   string
	StartTime string
	EndTime   string
	OrderBy   string
}

func BuildOperationLogFiltersFromHTTP(ctx http.Context) OperationLogFilters {
	return OperationLogFilters{
		AdminID:   ctx.Request().Query("admin_id", ""),
		TraceID:   ctx.Request().Query("trace_id", ""),
		Username:  ctx.Request().Query("username", ""),
		Method:    ctx.Request().Query("method", ""),
		Path:      ctx.Request().Query("path", ""),
		Title:     ctx.Request().Query("title", ""),
		IP:        ctx.Request().Query("ip", ""),
		Status:    ctx.Request().Query("status", ""),
		Request:   ctx.Request().Query("request", ""),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Query("order_by", ""),
	}
}

type OperationLogServiceImpl struct {
	ctx context.Context
}

func NewOperationLogService(ctx context.Context) OperationLogService {
	return &OperationLogServiceImpl{ctx: ctx}
}

func (s *OperationLogServiceImpl) GetByID(id uint, withAdmin bool) (*models.OperationLog, error) {
	var log models.OperationLog
	query := appfacades.OrmQuery(s.ctx).Where("id", id)
	if withAdmin {
		query = query.With("Admin")
	}
	if err := query.FirstOrFail(&log); err != nil {
		return nil, apperrors.ErrLogNotFound.WithError(err)
	}
	if !CanAccessOwnedBy(s.ctx, log.AdminID) {
		return nil, apperrors.ErrForbidden
	}
	return &log, nil
}

func (s *OperationLogServiceImpl) GetList(filters OperationLogFilters, page, pageSize int) ([]models.OperationLog, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{})

	if filters.AdminID != "" {
		query = query.Where("admin_id", filters.AdminID)
	}
	if filters.TraceID != "" {
		query = query.Where("trace_id LIKE ?", "%"+filters.TraceID+"%")
	}
	if filters.Username != "" {
		var adminIDs []uint
		var admins []models.Admin
		if err := appfacades.OrmQuery(s.ctx).Where("username LIKE ?", "%"+filters.Username+"%").Get(&admins); err == nil {
			for _, admin := range admins {
				adminIDs = append(adminIDs, admin.ID)
			}
			if len(adminIDs) > 0 {
				idsAny := helpers.ConvertUintSliceToAny(adminIDs)
				query = query.WhereIn("admin_id", idsAny)
			} else {
				query = query.Where("admin_id", 0)
			}
		}
	}
	if filters.Method != "" {
		query = query.Where("method = ?", filters.Method)
	}
	if filters.Path != "" {
		query = query.Where("path LIKE ?", "%"+filters.Path+"%")
	}
	if filters.Title != "" {
		query = query.Where("title LIKE ?", "%"+filters.Title+"%")
	}
	if filters.IP != "" {
		query = query.Where("ip LIKE ?", "%"+filters.IP+"%")
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Request != "" {
		query = utils.ApplyFulltextSearch(query, "request", filters.Request)
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	query = ApplyDataScope(s.ctx, query, DataScopeApplyOpts{Mode: DataScopeModeAdmin, AdminColumn: "admin_id"})

	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "id:desc"
	}
	query = helpers.ApplySort(query, orderBy, "id:desc")

	var logs []models.OperationLog
	var total int64
	if err := query.With("Admin").Paginate(page, pageSize, &logs, &total); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}
	return logs, total, nil
}

func (s *OperationLogServiceImpl) Delete(id uint) error {
	log, err := s.GetByID(id, false)
	if err != nil {
		return err
	}
	if _, err := appfacades.OrmQuery(s.ctx).Delete(log); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *OperationLogServiceImpl) BatchDelete(ids []uint) error {
	if len(ids) == 0 {
		return apperrors.ErrIDsRequired
	}
	idsAny := helpers.ConvertUintSliceToAny(ids)
	if _, err := appfacades.OrmQuery(s.ctx).WhereIn("id", idsAny).Delete(&models.OperationLog{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *OperationLogServiceImpl) Clean(days int) error {
	if days <= 0 {
		days = constants.DefaultCleanLogDays
	}
	cutoffTime := time.Now().AddDate(0, 0, -days)
	if _, err := appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{}).Where("created_at < ?", cutoffTime).Delete(&models.OperationLog{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

// Archive exports operation logs older than N days to CSV (sync), then deletes those rows.
func (s *OperationLogServiceImpl) Archive(days int) (uint, error) {
	if days <= 0 {
		days = constants.DefaultCleanLogDays
	}
	cutoffTime := time.Now().AddDate(0, 0, -days)

	var logs []models.OperationLog
	if err := appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{}).
		Where("created_at < ?", cutoffTime).
		Order("id asc").
		Find(&logs); err != nil {
		return 0, apperrors.ErrQueryFailed.WithError(err)
	}

	adminID := uint(0)
	if admin := adminFromContext(s.ctx); admin != nil {
		adminID = admin.ID
	}

	disk := "local"
	if v := utils.GetConfigValue(s.ctx, "storage", "file_disk", ""); v != "" {
		disk = v
	} else if v := utils.GetConfigValue(s.ctx, "storage", "export_disk", ""); v != "" {
		disk = v
	}

	exportRecord := models.Export{
		AdminID: adminID,
		Type:    models.ExportTypeOperationLogsArchive,
		Status:  models.ExportStatusProcessing,
		Disk:    disk,
	}
	if err := appfacades.OrmQuery(s.ctx).Create(&exportRecord); err != nil {
		return 0, apperrors.ErrCreateFailed.WithError(err)
	}

	headers := []string{
		"id", "admin_id", "trace_id", "method", "path", "title", "ip",
		"user_agent", "request", "response", "changes", "status", "error_msg", "duration", "created_at",
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
			log.TraceID,
			log.Method,
			log.Path,
			log.Title,
			log.IP,
			log.UserAgent,
			log.Request,
			log.Response,
			log.Changes,
			fmt.Sprintf("%d", log.Status),
			log.ErrorMsg,
			fmt.Sprintf("%d", log.Duration),
			createdAt,
		})
	}

	filename := fmt.Sprintf("operation_logs_archive_%d_%s.csv", exportRecord.ID, time.Now().Format("20060102_150405"))
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

func (s *OperationLogServiceImpl) GetTitleOptions() []string {
	var dbTitles []string
	_ = appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{}).
		Select("DISTINCT title").
		Where("title IS NOT NULL AND title != ''").
		Order("title ASC").
		Pluck("title", &dbTitles)

	var permissionSlugs []string
	_ = appfacades.OrmQuery(s.ctx).Model(&models.Permission{}).
		Select("slug").
		Where("status = 1").
		Order("slug ASC").
		Pluck("slug", &permissionSlugs)

	result := lo.Uniq(lo.Filter(append(dbTitles, permissionSlugs...), func(title string, _ int) bool {
		return title != "" && title != "operation.unknown" && !strings.HasPrefix(title, "operation.")
	}))
	sort.Strings(result)
	return result
}
