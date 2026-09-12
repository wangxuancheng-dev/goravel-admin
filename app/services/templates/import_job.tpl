package jobs

import (
	"context"
	"fmt"

	"github.com/goravel/framework/facades"

	"goravel/app/services"
)

// Import<<.ModelName>>sArgs import task args alias.
type Import<<.ModelName>>sArgs = ImportArgs

// Import<<.ModelName>>s imports <<.ModelName>> records asynchronously from CSV.
type Import<<.ModelName>>s struct{}

func (r *Import<<.ModelName>>s) Signature() string {
	return "import_<<.ModuleName>>s"
}

func (r *Import<<.ModelName>>s) Handle(args ...any) (retErr error) {
	var importID uint
	var jobCtx context.Context

	defer func() {
		if rec := recover(); rec != nil {
			errorMsg := fmt.Sprintf("panic: %v", rec)
			facades.Log().Errorf("Import<<.ModelName>>s Job panic: %v", rec)
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
			return services.New<<.ModelName>>Service(ctx).ImportFromCSV(csvContent)
		},
	})
	return importer.Execute(importArgs)
}
