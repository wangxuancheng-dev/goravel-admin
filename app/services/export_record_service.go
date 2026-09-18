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

type ExportRecordService interface {
	// GetByID 根据ID获取导出记录
	GetByID(id uint) (*models.Export, error)
	// GetByIDs 根据ID列表获取导出记录
	GetByIDs(ids []uint) ([]models.Export, error)
	// GetList 获取导出记录列表
	GetList(filters ExportRecordFilters, page, pageSize int) ([]models.Export, int64, error)
	// Delete 删除导出记录
	Delete(id uint) error
	// BatchDelete 批量删除导出记录
	BatchDelete(ids []uint) error
	// DeleteWithFile 删除导出记录及源文件（源文件删除失败仅记日志）
	DeleteWithFile(id uint) error
	// BatchDeleteWithFiles 批量删除导出记录及源文件（源文件删除失败仅记日志）
	BatchDeleteWithFiles(ids []uint) error
	// ExportRecordToJSON 列表展示字段（含 file_url）
	ExportRecordToJSON(export *models.Export) map[string]any
}

// ExportRecordFilters 导出记录查询过滤器
type ExportRecordFilters struct {
	AdminID   string
	Type      string
	Filename  string
	Disk      string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

func BuildExportRecordFiltersFromHTTP(ctx http.Context) ExportRecordFilters {
	return ExportRecordFilters{
		AdminID:   ctx.Request().Query("admin_id", ""),
		Type:      ctx.Request().Query("type", ""),
		Filename:  ctx.Request().Query("filename", ""),
		Disk:      ctx.Request().Query("disk", ""),
		Status:    ctx.Request().Query("status", ""),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Query("order_by", ""),
	}
}

type ExportRecordServiceImpl struct {
	ctx context.Context
}

func NewExportRecordService(ctx context.Context) ExportRecordService {
	return &ExportRecordServiceImpl{ctx: ctx}
}

// GetByID 根据ID获取导出记录
func (s *ExportRecordServiceImpl) GetByID(id uint) (*models.Export, error) {
	var export models.Export
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&export); err != nil {
		return nil, apperrors.ErrExportRecordNotFound.WithError(err)
	}
	return &export, nil
}

// GetByIDs 根据ID列表获取导出记录
func (s *ExportRecordServiceImpl) GetByIDs(ids []uint) ([]models.Export, error) {
	if len(ids) == 0 {
		return []models.Export{}, nil
	}

	idsAny := helpers.ConvertUintSliceToAny(ids)
	var exports []models.Export
	if err := appfacades.OrmQuery(s.ctx).WhereIn("id", idsAny).Get(&exports); err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}
	return exports, nil
}

// GetList 获取导出记录列表
func (s *ExportRecordServiceImpl) GetList(filters ExportRecordFilters, page, pageSize int) ([]models.Export, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Export{})

	if filters.AdminID != "" {
		query = query.Where("admin_id", filters.AdminID)
	}
	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}
	if filters.Filename != "" {
		query = query.Where("filename LIKE ?", "%"+filters.Filename+"%")
	}
	if filters.Disk != "" {
		query = query.Where("disk = ?", filters.Disk)
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

	var exports []models.Export
	var total int64
	if err := query.With("Admin").Paginate(page, pageSize, &exports, &total); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}

	return exports, total, nil
}

// Delete 删除导出记录
func (s *ExportRecordServiceImpl) Delete(id uint) error {
	var export models.Export
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&export); err != nil {
		return apperrors.ErrExportRecordNotFound.WithError(err)
	}

	if _, err := appfacades.OrmQuery(s.ctx).Delete(&export); err != nil {
		errorlog.Record(s.ctx, "export-record", "删除导出记录失败", map[string]any{
			"export_id": id,
			"error":     err.Error(),
		}, "删除导出记录失败: %v", err)
		return apperrors.ErrDeleteFailed.WithError(err)
	}

	return nil
}

// BatchDelete 批量删除导出记录
func (s *ExportRecordServiceImpl) BatchDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	idsAny := helpers.ConvertUintSliceToAny(ids)
	if _, err := appfacades.OrmQuery(s.ctx).WhereIn("id", idsAny).Delete(&models.Export{}); err != nil {
		errorlog.Record(s.ctx, "export-record", "批量删除导出记录失败", map[string]any{
			"ids":   ids,
			"count": len(ids),
			"error": err.Error(),
		}, "批量删除导出记录失败: %v", err)
		return apperrors.ErrBatchDeleteExportFailed.WithError(err)
	}

	return nil
}

// DeleteWithFile 删除导出记录并尝试删除源文件（源文件失败仅记日志，不阻断）
func (s *ExportRecordServiceImpl) DeleteWithFile(id uint) error {
	export, err := s.GetByID(id)
	if err != nil {
		return err
	}
	s.tryDeleteStorageFile(export, false)
	return s.Delete(id)
}

// BatchDeleteWithFiles 批量删除导出记录并尝试删除源文件（源文件失败仅记日志，不阻断）
func (s *ExportRecordServiceImpl) BatchDeleteWithFiles(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	exports, err := s.GetByIDs(ids)
	if err == nil {
		for i := range exports {
			s.tryDeleteStorageFile(&exports[i], true)
		}
	}

	return s.BatchDelete(ids)
}

func (s *ExportRecordServiceImpl) tryDeleteStorageFile(export *models.Export, batch bool) {
	if export == nil || export.Path == "" || export.Disk == "" {
		return
	}

	openMsg := "Failed to open storage for delete"
	deleteMsg := "Failed to delete export source file"
	if batch {
		openMsg = "Failed to open storage for batch delete"
		deleteMsg = "Failed to delete export source file in batch delete"
	}

	storage, err := utils.StorageDisk(export.Disk)
	if err != nil {
		s.recordStorageError(openMsg, export, err, "Open storage error: %v")
		return
	}
	if err := storage.Delete(export.Path); err != nil {
		s.recordStorageError(deleteMsg, export, err, "Delete export source file error: %v")
	}
}

func (s *ExportRecordServiceImpl) recordStorageError(message string, export *models.Export, err error, format string) {
	attrs := map[string]any{
		"error": err.Error(),
		"disk":  export.Disk,
		"path":  export.Path,
	}
	if httpCtx, ok := s.ctx.(http.Context); ok {
		errorlog.RecordHTTP(httpCtx, "export", message, attrs, format, err)
		return
	}
	errorlog.Record(s.ctx, "export", message, attrs, format, err)
}

// ExportRecordToJSON 导出记录列表展示字段（含 file_url）
func (s *ExportRecordServiceImpl) ExportRecordToJSON(export *models.Export) map[string]any {
	if export == nil {
		return map[string]any{}
	}

	payload := map[string]any{
		"id":         export.ID,
		"admin_id":   export.AdminID,
		"type":       export.Type,
		"disk":       export.Disk,
		"path":       export.Path,
		"filename":   export.Filename,
		"extension":  export.Extension,
		"size":       export.Size,
		"status":     export.Status,
		"error_msg":  export.ErrorMsg,
		"created_at": export.CreatedAt,
		"updated_at": export.UpdatedAt,
		"file_url":   s.exportFileURL(export),
	}
	if export.Admin.ID > 0 {
		payload["admin"] = export.Admin
	}
	return payload
}

func (s *ExportRecordServiceImpl) exportFileURL(export *models.Export) string {
	if export == nil || export.Path == "" {
		return ""
	}
	if export.Disk == "local" || export.Disk == "public" {
		return fmt.Sprintf("/api/admin/exports/%d/download", export.ID)
	}
	return NewExportServiceForJob(s.ctx, export.Disk).GetExportURL(export.Path)
}
