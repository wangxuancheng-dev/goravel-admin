package jobs

import (
	"context"
	"fmt"

	"github.com/goravel/framework/facades"

	"goravel/app/services"
)

// ImportArticlesArgs import task args alias.
type ImportArticlesArgs = ImportArgs

// ImportArticles imports Article records asynchronously from CSV.
type ImportArticles struct{}

func (r *ImportArticles) Signature() string {
	return "import_articles"
}

func (r *ImportArticles) Handle(args ...any) (retErr error) {
	var importID uint
	var jobCtx context.Context

	defer func() {
		if rec := recover(); rec != nil {
			errorMsg := fmt.Sprintf("panic: %v", rec)
			facades.Log().Errorf("ImportArticles Job panic: %v", rec)
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
			return services.NewArticleService(ctx).ImportFromCSV(csvContent)
		},
	})
	return importer.Execute(importArgs)
}
