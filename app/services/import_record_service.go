package services

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/rbac"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type ImportRecordService interface {
	GetByID(id uint) (*models.Import, error)
	GetList(filters ImportRecordFilters, page, pageSize int) ([]models.Import, int64, error)
	Delete(id uint) error
	DeleteWithFile(id uint) error
	ImportRecordToJSON(record *models.Import) map[string]any
}

// ImportRecordFilters 导入记录查询过滤器
type ImportRecordFilters struct {
	AdminID   string
	Type      string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

func BuildImportRecordFiltersFromHTTP(ctx http.Context) ImportRecordFilters {
	return ImportRecordFilters{
		AdminID:   ctx.Request().Query("admin_id", ""),
		Type:      ctx.Request().Query("type", ""),
		Status:    ctx.Request().Query("status", ""),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Query("order_by", ""),
	}
}

type ImportRecordServiceImpl struct {
	ctx context.Context
}

func NewImportRecordService(ctx context.Context) ImportRecordService {
	return &ImportRecordServiceImpl{ctx: ctx}
}

func (s *ImportRecordServiceImpl) GetByID(id uint) (*models.Import, error) {
	var item models.Import
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&item); err != nil {
		return nil, apperrors.ErrRecordNotFound.WithError(err)
	}
	if !rbac.CanAccessOwnedBy(s.ctx, item.AdminID) {
		return nil, apperrors.ErrForbidden
	}
	return &item, nil
}

func (s *ImportRecordServiceImpl) GetList(filters ImportRecordFilters, page, pageSize int) ([]models.Import, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Import{})

	if filters.AdminID != "" {
		query = query.Where("admin_id", filters.AdminID)
	}
	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
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

	var imports []models.Import
	var total int64
	if err := query.With("Admin").Paginate(page, pageSize, &imports, &total); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}
	return imports, total, nil
}

func (s *ImportRecordServiceImpl) Delete(id uint) error {
	item, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if _, err := appfacades.OrmQuery(s.ctx).Delete(item); err != nil {
		errorlog.Record(s.ctx, "import-record", "删除导入记录失败", map[string]any{
			"import_id": id,
			"error":     err.Error(),
		}, "删除导入记录失败: %v", err)
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *ImportRecordServiceImpl) DeleteWithFile(id uint) error {
	item, err := s.GetByID(id)
	if err != nil {
		return err
	}
	s.tryDeleteStorageFiles(item)
	return s.Delete(id)
}

func (s *ImportRecordServiceImpl) tryDeleteStorageFiles(item *models.Import) {
	if item == nil {
		return
	}
	disk := item.Disk
	if disk == "" {
		disk = "local"
	}
	storage, err := utils.StorageDisk(disk)
	if err != nil {
		errorlog.Record(s.ctx, "import", "Failed to open storage for import delete", map[string]any{
			"error": err.Error(),
			"disk":  disk,
		}, "Open storage error: %v", err)
		return
	}
	for _, path := range []string{item.Path, item.ErrorFilePath} {
		if path == "" {
			continue
		}
		if err := storage.Delete(path); err != nil {
			errorlog.Record(s.ctx, "import", "Failed to delete import file", map[string]any{
				"error": err.Error(),
				"disk":  disk,
				"path":  path,
			}, "Delete import file error: %v", err)
		}
	}
}

func (s *ImportRecordServiceImpl) ImportRecordToJSON(record *models.Import) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	errorFileURL := ""
	if record.ErrorFilePath != "" {
		errorFileURL = fmt.Sprintf("/api/admin/imports/%d/error-file", record.ID)
	}
	payload := map[string]any{
		"id":             record.ID,
		"admin_id":       record.AdminID,
		"type":           record.Type,
		"status":         record.Status,
		"disk":           record.Disk,
		"path":           record.Path,
		"total_rows":     record.TotalRows,
		"success_rows":   record.SuccessRows,
		"failed_rows":    record.FailedRows,
		"error_file_path": record.ErrorFilePath,
		"error_file_url": errorFileURL,
		"error_msg":      record.ErrorMsg,
		"created_at":     record.CreatedAt,
		"updated_at":     record.UpdatedAt,
	}
	if record.Admin.ID > 0 {
		payload["admin"] = record.Admin
	}
	return payload
}
