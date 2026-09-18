package jobs

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	contractsCache "github.com/goravel/framework/contracts/cache"
	"github.com/goravel/framework/facades"
	supportcarbon "github.com/goravel/framework/support/carbon"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

// ErrExportRecordMissing 导出记录被删除时的哨兵错误
var ErrExportRecordMissing = errors.New("export record missing (deleted)")

const defaultExportExecutionLockTTLSeconds = 7200

// ExportArgs 通用导出任务参数
type ExportArgs struct {
	ExportID         uint           `json:"export_id"`
	AdminID          uint           `json:"admin_id"`
	TenantID         uint           `json:"tenant_id,omitempty"`
	TenantConnection string         `json:"tenant_connection,omitempty"`
	Filters          map[string]any `json:"filters"`
	Type             string         `json:"type"`
	Language         string         `json:"language"`
	Timezone         string         `json:"timezone"` // 用户时区，用于时间格式化
}

// JobContext 为导出任务构建带租户连接的 context。
// tenancy 开启时必须带 tenant_id 且绑定成功，否则返回错误（fail-closed，避免写到平台库）。
func JobContext(args ExportArgs) (context.Context, error) {
	ctx := context.Background()
	if !tenancy.Enabled() {
		return ctx, nil
	}
	if args.TenantID == 0 {
		return ctx, apperrors.ErrTenantRequired
	}
	svc := services.NewTenantConnectionService()
	bound, err := svc.BindBackground(ctx, args.TenantID)
	if err != nil {
		facades.Log().Errorf("export job bind tenant failed: tenant_id=%d err=%v", args.TenantID, err)
		return ctx, err
	}
	return bound, nil
}

// FormatTimeWithTimezone 使用指定时区格式化时间
func FormatTimeWithTimezone(t time.Time, timezone string) string {
	if t.IsZero() {
		return ""
	}
	if timezone == "" {
		timezone = "UTC"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return t.In(loc).Format("2006-01-02 15:04:05")
}

// FormatCarbonWithTimezone 使用指定时区格式化 Carbon 时间
func FormatCarbonWithTimezone(t *supportcarbon.DateTime, timezone string) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return FormatTimeWithTimezone(t.StdTime(), timezone)
}

// ExportConfig 导出配置
type ExportConfig struct {
	// 文件名前缀，如 "payments"、"orders"
	FilePrefix string
	// 表头翻译键列表
	HeaderKeys []string
	// 数据写入函数，接收 CSV writer、筛选条件 map、语言、停止检查函数
	WriteData func(ctx context.Context, w *csv.Writer, filters map[string]any, lang string, shouldStop func() bool) error
}

// BaseExporter 通用导出器基类
type BaseExporter struct {
	config ExportConfig
}

type ExportExecutionLock struct {
	lock     contractsCache.Lock
	acquired bool
}

// NewBaseExporter 创建通用导出器
func NewBaseExporter(config ExportConfig) *BaseExporter {
	return &BaseExporter{config: config}
}

// ParseArgs 解析导出参数（通用逻辑）
func ParseArgs(args ...any) (ExportArgs, error) {
	if len(args) < 1 {
		return ExportArgs{}, apperrors.ErrInvalidArgument.WithMessage("missing export arguments")
	}

	var exportArgs ExportArgs
	switch v := args[0].(type) {
	case ExportArgs:
		exportArgs = v
	case string:
		if err := json.Unmarshal([]byte(v), &exportArgs); err != nil {
			facades.Log().Errorf("反序列化参数失败: %v, JSON: %s", err, v)
			return ExportArgs{}, apperrors.ErrInvalidArgument.WithMessage(fmt.Sprintf("failed to unmarshal export arguments: %v", err))
		}
	case map[string]any:
		if exportID, ok := utils.GetUint(v, "export_id"); ok {
			exportArgs.ExportID = exportID
		}
		if adminID, ok := utils.GetUint(v, "admin_id"); ok {
			exportArgs.AdminID = adminID
		}
		if filters, ok := utils.GetMap(v, "filters"); ok {
			exportArgs.Filters = filters
		}
		if exportType, ok := utils.GetString(v, "type"); ok {
			exportArgs.Type = exportType
		}
		if lang, ok := utils.GetString(v, "language"); ok {
			exportArgs.Language = lang
		}
		if tenantID, ok := utils.GetUint(v, "tenant_id"); ok {
			exportArgs.TenantID = tenantID
		}
		if tenantConn, ok := utils.GetString(v, "tenant_connection"); ok {
			exportArgs.TenantConnection = tenantConn
		}
		if tz, ok := utils.GetString(v, "timezone"); ok {
			exportArgs.Timezone = tz
		}
	default:
		return ExportArgs{}, apperrors.ErrInvalidArgument.WithMessage(fmt.Sprintf("invalid export arguments type: %T", args[0]))
	}

	if exportArgs.ExportID == 0 {
		return ExportArgs{}, apperrors.ErrInvalidArgument.WithMessage("export_id is required")
	}

	return exportArgs, nil
}

func AcquireExportExecutionLock(ctx context.Context, exportID uint) (*ExportExecutionLock, error) {
	lockKey := tenancy.CacheKey(ctx, fmt.Sprintf("export:execution:%d", exportID))
	lock := facades.Cache().Lock(lockKey, getExportExecutionLockTTL())
	if !lock.Get() {
		return nil, nil
	}

	return &ExportExecutionLock{
		lock:     lock,
		acquired: true,
	}, nil
}

func getExportExecutionLockTTL() time.Duration {
	raw := os.Getenv("QUEUE_EXPORT_EXECUTION_LOCK_TTL_SECONDS")
	if raw == "" {
		return defaultExportExecutionLockTTLSeconds * time.Second
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return defaultExportExecutionLockTTLSeconds * time.Second
	}

	return time.Duration(seconds) * time.Second
}

func (l *ExportExecutionLock) Release() {
	if l == nil || !l.acquired {
		return
	}

	l.lock.Release()
	l.acquired = false
}

// MarkExportFailed 标记导出失败
func MarkExportFailed(ctx context.Context, exportID uint, errorMsg string) {
	if exportID == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var failedRecord models.Export
	if queryErr := appfacades.OrmQuery(ctx).Where("id", exportID).First(&failedRecord); queryErr == nil {
		failedRecord.Status = models.ExportStatusFailed
		failedRecord.ErrorMsg = errorMsg
		if saveErr := appfacades.OrmQuery(ctx).Save(&failedRecord); saveErr != nil {
			facades.Log().Errorf("更新导出记录失败状态失败: export_id=%d, error=%v", exportID, saveErr)
		}
	}
}

// CheckAndUpdateExportStatus 检查导出记录并更新状态为处理中
func CheckAndUpdateExportStatus(ctx context.Context, exportID uint) (*models.Export, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	exists, err := appfacades.OrmQuery(ctx).Model(&models.Export{}).Where("id", exportID).Exists()
	if err != nil {
		errorlog.Record(ctx, "export", "检查导出记录是否存在失败", map[string]any{
			"export_id": exportID,
			"error":     err.Error(),
		}, "检查导出记录是否存在失败: %v", err)
		return nil, err
	}
	if !exists {
		facades.Log().Infof("导出记录不存在，停止任务: export_id=%d", exportID)
		return nil, nil
	}

	var exportRecords []models.Export
	if err := appfacades.OrmQuery(ctx).Where("id", exportID).Limit(1).Get(&exportRecords); err != nil {
		errorlog.Record(ctx, "export", "查询导出记录失败", map[string]any{
			"export_id": exportID,
			"error":     err.Error(),
		}, "查询导出记录失败: %v", err)
		return nil, err
	}
	if len(exportRecords) == 0 {
		facades.Log().Infof("导出记录已被删除，停止任务: export_id=%d", exportID)
		return nil, nil
	}

	exportRecord := &exportRecords[0]
	if exportRecord.Status == models.ExportStatusSuccess {
		facades.Log().Infof("导出记录已成功，跳过重复执行: export_id=%d", exportID)
		return nil, nil
	}
	exportRecord.Status = models.ExportStatusProcessing
	exportRecord.ErrorMsg = ""
	if err := appfacades.OrmQuery(ctx).Save(exportRecord); err != nil {
		facades.Log().Errorf("更新导出状态为处理中失败: export_id=%d, error=%v", exportID, err)
		return nil, err
	}

	return exportRecord, nil
}

// Execute 执行导出（通用流程）
func (e *BaseExporter) Execute(args ExportArgs) error {
	ctx, err := JobContext(args)
	if err != nil {
		return err
	}

	// 获取语言
	lang := args.Language
	if lang == "" {
		lang = facades.Config().GetString("app.locale", "cn")
	}
	lang = utils.NormalizeLanguage(lang)

	// 将时区放入 filters 供 WriteData 使用
	if args.Timezone != "" {
		if args.Filters == nil {
			args.Filters = make(map[string]any)
		}
		args.Filters["_timezone"] = args.Timezone
	}

	// 翻译表头
	headers := utils.TranslateHeaders(e.config.HeaderKeys, lang)
	exportFormat := strings.ToLower(strings.TrimSpace(utils.GetConfigValue(ctx, "storage", "export_format", "csv")))
	if exportFormat != "xlsx" && exportFormat != "csv" {
		exportFormat = "csv"
	}

	diskHint := ""
	var exportMeta models.Export
	if err := appfacades.OrmQuery(ctx).Where("id", args.ExportID).First(&exportMeta); err == nil {
		diskHint = strings.TrimSpace(exportMeta.Disk)
	}
	exportService := services.NewExportServiceForJob(ctx, diskHint)
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%d_%s.%s", e.config.FilePrefix, args.ExportID, timestamp, exportFormat)
	filePath := path.Join("exports", filename)
	if p := helpers.TenantStoragePrefix(ctx); p != "" {
		filePath = path.Join(strings.TrimSuffix(p, "/"), filePath)
	}

	// 预写入文件信息
	e.preWriteFileInfo(ctx, args.ExportID, filePath, filename)

	// 创建 shouldStop 和进度更新回调
	lastUpdateAt := time.Now().Add(-10 * time.Second)
	lastExistCheckAt := time.Now().Add(-10 * time.Second)
	shouldStop := func() bool {
		if time.Since(lastExistCheckAt) < 2*time.Second {
			return false
		}
		lastExistCheckAt = time.Now()
		exists, err := appfacades.OrmQuery(ctx).Model(&models.Export{}).Where("id", args.ExportID).Exists()
		if err != nil {
			return false
		}
		return !exists
	}

	var filePathResult string
	if exportFormat == "xlsx" {
		var csvBuf bytes.Buffer
		cw := csv.NewWriter(&csvBuf)
		if err := e.config.WriteData(ctx, cw, args.Filters, lang, shouldStop); err != nil {
			return err
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			return err
		}

		cr := csv.NewReader(strings.NewReader(csvBuf.String()))
		rows, err := cr.ReadAll()
		if err != nil {
			return err
		}

		filePathResult, err = exportService.ExportToXLSXAt(headers, rows, filePath, true)
	} else {
		// 执行流式导出
		filePathResult, err = exportService.ExportToCSVStreamAtWithProgress(headers, filePath, func(w *csv.Writer) error {
			return e.config.WriteData(ctx, w, args.Filters, lang, shouldStop)
		}, func(writtenBytes int64) {
			if shouldStop() {
				return
			}
			if time.Since(lastUpdateAt) < 3*time.Second {
				return
			}
			lastUpdateAt = time.Now()
			result, _ := appfacades.OrmQuery(ctx).Model(&models.Export{}).Where("id", args.ExportID).Update(map[string]any{
				"size": writtenBytes,
			})
			if result == nil || result.RowsAffected == 0 {
				lastExistCheckAt = time.Now().Add(-10 * time.Second)
			}
		}, true)
	}

	if err != nil {
		if shouldStop() {
			return ErrExportRecordMissing
		}
		errorlog.Record(ctx, "export", "导出文件失败", map[string]any{
			"export_id": args.ExportID,
			"filename":  filename,
			"error":     err.Error(),
		}, "导出文件失败: %v", err)
		return fmt.Errorf("导出文件失败: %v", err)
	}

	// 更新导出记录为成功
	return e.finalizeExport(ctx, args.ExportID, filePathResult)
}

// preWriteFileInfo 预写入文件信息
func (e *BaseExporter) preWriteFileInfo(ctx context.Context, exportID uint, filePath, filename string) {
	if ctx == nil {
		ctx = context.Background()
	}
	var exportRecord models.Export
	if err := appfacades.OrmQuery(ctx).Where("id", exportID).First(&exportRecord); err == nil {
		changed := false
		if exportRecord.Path == "" {
			exportRecord.Path = filePath
			changed = true
		}
		if exportRecord.Filename == "" {
			exportRecord.Filename = filename
			changed = true
		}
		if exportRecord.Extension == "" {
			if ext := path.Ext(filePath); ext != "" {
				exportRecord.Extension = strings.TrimPrefix(ext, ".")
			} else {
				exportRecord.Extension = "csv"
			}
			changed = true
		}
		if changed {
			_ = appfacades.OrmQuery(ctx).Save(&exportRecord)
		}
	}
}

// finalizeExport 完成导出，更新记录
func (e *BaseExporter) finalizeExport(ctx context.Context, exportID uint, filePath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var exportRecord models.Export
	if err := appfacades.OrmQuery(ctx).Where("id", exportID).First(&exportRecord); err != nil {
		return nil
	}

	exportRecord.Path = filePath
	exportRecord.Filename = path.Base(filePath)

	if ext := path.Ext(filePath); ext != "" {
		exportRecord.Extension = ext[1:]
	} else {
		exportRecord.Extension = "csv"
	}

	if exportRecord.Disk != "" {
		if storage, err := utils.StorageDisk(exportRecord.Disk); err != nil {
			facades.Log().Warningf("Skip export file size: %v", err)
		} else if fileInfo, err := storage.Size(filePath); err == nil {
			exportRecord.Size = fileInfo
		}
	}

	exportRecord.Status = models.ExportStatusSuccess
	exportRecord.ErrorMsg = ""

	if err := appfacades.OrmQuery(ctx).Save(&exportRecord); err != nil {
		facades.Log().Errorf("保存导出记录失败: export_id=%d, error=%v", exportID, err)
		return fmt.Errorf("更新导出记录失败: %v", err)
	}

	return nil
}

// IsTableNotExistsError 判断是否是表不存在错误
func IsTableNotExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "1146") || strings.Contains(msg, "42s02") {
		return true
	}
	return strings.Contains(msg, "table") && (strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "does not exist"))
}

// ParseOrderBy 解析排序字符串
func ParseOrderBy(orderBy string) (field, direction string) {
	parts := strings.Split(orderBy, ":")
	field = "created_at"
	direction = "desc"

	if len(parts) >= 1 && parts[0] != "" {
		field = parts[0]
	}
	if len(parts) >= 2 && (parts[1] == "asc" || parts[1] == "desc") {
		direction = parts[1]
	}
	return
}

// GetDefaultTimeRange 获取默认时间范围（7天前到现在）
func GetDefaultTimeRange(filters map[string]any) (startTime, endTime time.Time) {
	if startTimeStr, ok := utils.GetString(filters, "start_time"); ok && startTimeStr != "" {
		if t, err := utils.ParseDateTimeUTC(startTimeStr); err == nil {
			startTime = t
		}
	}
	if endTimeStr, ok := utils.GetString(filters, "end_time"); ok && endTimeStr != "" {
		if t, err := utils.ParseDateTimeUTC(endTimeStr); err == nil {
			endTime = t
		}
	}

	if startTime.IsZero() {
		startTime = time.Now().UTC().AddDate(0, 0, -7)
	}
	if endTime.IsZero() {
		endTime = time.Now().UTC()
	}
	return
}
