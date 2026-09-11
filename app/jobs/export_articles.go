package jobs

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

// ExportArticlesArgs export task args alias.
type ExportArticlesArgs = ExportArgs

// ExportArticles exports Article records asynchronously.
type ExportArticles struct{}

func (r *ExportArticles) Signature() string {
	return "export_articles"
}

func (r *ExportArticles) Handle(args ...any) (retErr error) {
	var exportID uint
	var jobCtx context.Context

	defer func() {
		if rec := recover(); rec != nil {
			errorMsg := fmt.Sprintf("panic: %v", rec)
			facades.Log().Errorf("ExportArticles Job panic: %v", rec)
			MarkExportFailed(jobCtx, exportID, errorMsg)
			retErr = fmt.Errorf("%s", errorMsg)
		}
	}()

	exportArgs, err := ParseArgs(args...)
	if err != nil {
		return err
	}
	exportID = exportArgs.ExportID
	jobCtx = JobContext(exportArgs)

	lock, err := AcquireExportExecutionLock(jobCtx, exportID)
	if err != nil {
		return err
	}
	if lock == nil {
		return nil
	}
	defer lock.Release()

	exportRecord, err := CheckAndUpdateExportStatus(jobCtx, exportID)
	if err != nil {
		return err
	}
	if exportRecord == nil {
		return nil
	}

	exporter := NewBaseExporter(ExportConfig{
		FilePrefix: "articles",
		HeaderKeys: []string{
			"admin_id",
			"title",
			"content",
			"status",
			"created_at",
			"updated_at",
		},
		WriteData: r.writeToCSV,
	})

	jobErr := exporter.Execute(exportArgs)
	if errors.Is(jobErr, ErrExportRecordMissing) {
		return nil
	}
	if jobErr != nil {
		MarkExportFailed(jobCtx, exportID, jobErr.Error())
		return jobErr
	}

	return nil
}

func (r *ExportArticles) writeToCSV(ctx context.Context, w *csv.Writer, filters map[string]any, lang string, shouldStop func() bool) error {
	var modelFilters services.ArticleFilters
	utils.FillFiltersFromMap(filters, &modelFilters)

	timezone, _ := utils.GetString(filters, "_timezone")

	const chunkSize = 2000
	lastID := uint(0)

	for {
		if shouldStop != nil && shouldStop() {
			return ErrExportRecordMissing
		}

		q := services.BuildArticleQuery(ctx, modelFilters)
		if lastID > 0 {
			q = q.Where("id < ?", lastID)
		}
		q = q.Order("id desc").Limit(chunkSize)

		var rows []models.Article
		if err := q.Get(&rows); err != nil {
			return fmt.Errorf("query articles for export failed: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			record := []string{
				cast.ToString(row.AdminId),
				row.Title,
				row.Content,
				cast.ToString(row.Status),
				FormatCarbonWithTimezone(row.CreatedAt, timezone),
				FormatCarbonWithTimezone(row.UpdatedAt, timezone),
			}
			if err := w.Write(record); err != nil {
				return fmt.Errorf("write csv row failed: %w", err)
			}
		}

		lastID = rows[len(rows)-1].ID
		if len(rows) < chunkSize {
			break
		}
	}

	return nil
}
