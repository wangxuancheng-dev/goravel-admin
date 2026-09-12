package jobs

import (
	"context"
	"fmt"

	"github.com/goravel/framework/facades"

	"goravel/app/services"
)

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
			MarkImportFailed(jobCtx, importID, errorMsg)
			retErr = fmt.Errorf("%s", errorMsg)
		}
	}()

	importArgs, err := ParseImportArgs(args...)
	if err != nil {
		return err
	}
	importID = importArgs.ImportID

	jobCtx, err = ImportJobContext(importArgs)
	if err != nil {
		return err
	}

	importer := NewBaseImporter(ImportConfig{
		ProcessCSV: func(ctx context.Context, csvContent string) (*services.ImportResult, error) {
			return services.NewImportOrderService(ctx).ImportOrders(csvContent)
		},
	})
	return importer.Execute(importArgs)
}
