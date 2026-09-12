package jobs

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

// ImportArgs 异步导入任务参数
type ImportArgs struct {
	ImportID         uint   `json:"import_id"`
	AdminID          uint   `json:"admin_id"`
	TenantID         uint   `json:"tenant_id,omitempty"`
	TenantConnection string `json:"tenant_connection,omitempty"`
	Type             string `json:"type"`
	Language         string `json:"language"`
	Timezone         string `json:"timezone"`
}

// ImportConfig 通用导入配置
type ImportConfig struct {
	// ProcessCSV 解析并导入 CSV 内容，返回行级结果
	ProcessCSV func(ctx context.Context, csvContent string) (*services.ImportResult, error)
}

// BaseImporter 通用导入器
type BaseImporter struct {
	config ImportConfig
}

// NewBaseImporter 创建通用导入器
func NewBaseImporter(config ImportConfig) *BaseImporter {
	return &BaseImporter{config: config}
}

// ImportJobContext 为导入任务构建带租户连接的 context。
func ImportJobContext(args ImportArgs) (context.Context, error) {
	return JobContext(ExportArgs{
		TenantID:         args.TenantID,
		TenantConnection: args.TenantConnection,
	})
}

// ParseImportArgs 解析导入参数（通用逻辑）
func ParseImportArgs(args ...any) (ImportArgs, error) {
	if len(args) < 1 {
		return ImportArgs{}, apperrors.ErrInvalidArgument.WithMessage("missing import arguments")
	}
	var importArgs ImportArgs
	switch v := args[0].(type) {
	case ImportArgs:
		importArgs = v
	case string:
		if err := json.Unmarshal([]byte(v), &importArgs); err != nil {
			return ImportArgs{}, apperrors.ErrInvalidArgument.WithMessage(fmt.Sprintf("failed to unmarshal import arguments: %v", err))
		}
	case map[string]any:
		b, _ := json.Marshal(v)
		if err := json.Unmarshal(b, &importArgs); err != nil {
			return ImportArgs{}, apperrors.ErrInvalidArgument.WithError(err)
		}
	default:
		return ImportArgs{}, apperrors.ErrInvalidArgument.WithMessage(fmt.Sprintf("invalid import arguments type: %T", args[0]))
	}
	if importArgs.ImportID == 0 {
		return ImportArgs{}, apperrors.ErrInvalidArgument.WithMessage("import_id is required")
	}
	return importArgs, nil
}

// MarkImportFailed 标记导入失败
func MarkImportFailed(ctx context.Context, importID uint, msg string) {
	if importID == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var record models.Import
	if err := appfacades.OrmQuery(ctx).Where("id", importID).First(&record); err != nil {
		return
	}
	record.Status = models.ImportStatusFailed
	record.ErrorMsg = msg
	_ = appfacades.OrmQuery(ctx).Save(&record)
}

// WriteImportErrorCSV 将失败行写入 CSV 并返回相对路径
func WriteImportErrorCSV(ctx context.Context, importID uint, errors []string) (string, error) {
	headers := []string{"row", "error"}
	rows := make([][]string, 0, len(errors))
	for i, e := range errors {
		rows = append(rows, []string{fmt.Sprintf("%d", i+1), e})
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("import_errors_%d_%s.csv", importID, timestamp)
	rel := path.Join("imports", "errors", filename)
	if p := helpers.TenantStoragePrefix(ctx); p != "" {
		rel = path.Join(strings.TrimSuffix(p, "/"), rel)
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	if err := w.Write(headers); err != nil {
		return "", err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}

	storage, err := utils.StorageDisk("local")
	if err != nil {
		return "", err
	}
	if err := storage.Put(rel, buf.String()); err != nil {
		return "", err
	}
	return rel, nil
}

// Execute 执行导入（通用流程：读记录 → 读文件 → ProcessCSV → 写错误文件 → 更新状态）
func (b *BaseImporter) Execute(args ImportArgs) error {
	if b == nil || b.config.ProcessCSV == nil {
		return apperrors.ErrInvalidArgument.WithMessage("import ProcessCSV is required")
	}

	ctx, err := ImportJobContext(args)
	if err != nil {
		return err
	}
	importID := args.ImportID

	var importRecord models.Import
	if err := appfacades.OrmQuery(ctx).Where("id", importID).FirstOrFail(&importRecord); err != nil {
		return apperrors.ErrRecordNotFound.WithError(err)
	}
	if importRecord.Status == models.ImportStatusSuccess {
		return nil
	}

	importRecord.Status = models.ImportStatusProcessing
	importRecord.ErrorMsg = ""
	_ = appfacades.OrmQuery(ctx).Save(&importRecord)

	if importRecord.Path == "" || importRecord.Disk == "" {
		MarkImportFailed(ctx, importID, "import file path is empty")
		return apperrors.ErrFilePathRequired
	}

	storage, err := utils.StorageDisk(importRecord.Disk)
	if err != nil {
		MarkImportFailed(ctx, importID, err.Error())
		return err
	}
	csvContent, err := storage.Get(importRecord.Path)
	if err != nil {
		MarkImportFailed(ctx, importID, err.Error())
		return err
	}

	result, err := b.config.ProcessCSV(ctx, csvContent)
	if err != nil {
		MarkImportFailed(ctx, importID, err.Error())
		return err
	}

	errorFilePath := ""
	if result != nil && result.FailedCount > 0 && len(result.Errors) > 0 {
		errorFilePath, err = WriteImportErrorCSV(ctx, importRecord.ID, result.Errors)
		if err != nil {
			facades.Log().Warningf("write import error csv failed: import_id=%d err=%v", importID, err)
		}
	}

	importRecord.TotalRows = 0
	importRecord.SuccessRows = 0
	importRecord.FailedRows = 0
	if result != nil {
		importRecord.TotalRows = result.TotalRows
		importRecord.SuccessRows = result.SuccessCount
		importRecord.FailedRows = result.FailedCount
	}
	importRecord.ErrorFilePath = errorFilePath
	if result != nil && result.FailedCount > 0 && result.SuccessCount == 0 {
		importRecord.Status = models.ImportStatusFailed
		importRecord.ErrorMsg = "all rows failed"
	} else {
		importRecord.Status = models.ImportStatusSuccess
		importRecord.ErrorMsg = ""
	}
	if err := appfacades.OrmQuery(ctx).Save(&importRecord); err != nil {
		return err
	}
	return nil
}
