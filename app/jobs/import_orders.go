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

// ImportOrders 异步订单导入任务
type ImportOrders struct{}

func (r *ImportOrders) Signature() string {
	return "import_orders"
}

func (r *ImportOrders) Handle(args ...any) (retErr error) {
	var importID uint
	var jobCtx context.Context

	defer func() {
		if rec := recover(); rec != nil {
			errorMsg := fmt.Sprintf("panic: %v", rec)
			facades.Log().Errorf("ImportOrders Job panic: %v", rec)
			markImportFailed(jobCtx, importID, errorMsg)
			retErr = fmt.Errorf("%s", errorMsg)
		}
	}()

	importArgs, err := ParseImportArgs(args...)
	if err != nil {
		return err
	}
	importID = importArgs.ImportID

	exportArgs := ExportArgs{
		TenantID:         importArgs.TenantID,
		TenantConnection: importArgs.TenantConnection,
	}
	jobCtx, err = JobContext(exportArgs)
	if err != nil {
		return err
	}

	var importRecord models.Import
	if err := appfacades.OrmQuery(jobCtx).Where("id", importID).FirstOrFail(&importRecord); err != nil {
		return apperrors.ErrRecordNotFound.WithError(err)
	}
	if importRecord.Status == models.ImportStatusSuccess {
		return nil
	}

	importRecord.Status = models.ImportStatusProcessing
	importRecord.ErrorMsg = ""
	_ = appfacades.OrmQuery(jobCtx).Save(&importRecord)

	if importRecord.Path == "" || importRecord.Disk == "" {
		markImportFailed(jobCtx, importID, "import file path is empty")
		return apperrors.ErrFilePathRequired
	}

	storage, err := utils.StorageDisk(importRecord.Disk)
	if err != nil {
		markImportFailed(jobCtx, importID, err.Error())
		return err
	}
	csvContent, err := storage.Get(importRecord.Path)
	if err != nil {
		markImportFailed(jobCtx, importID, err.Error())
		return err
	}

	result, err := services.NewImportOrderService(jobCtx).ImportOrders(csvContent)
	if err != nil {
		markImportFailed(jobCtx, importID, err.Error())
		return err
	}

	errorFilePath := ""
	if result != nil && result.FailedCount > 0 && len(result.Errors) > 0 {
		errorFilePath, err = writeImportErrorCSV(jobCtx, importRecord.ID, result.Errors)
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
	if err := appfacades.OrmQuery(jobCtx).Save(&importRecord); err != nil {
		return err
	}
	return nil
}

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

func markImportFailed(ctx context.Context, importID uint, msg string) {
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

func writeImportErrorCSV(ctx context.Context, importID uint, errors []string) (string, error) {
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
