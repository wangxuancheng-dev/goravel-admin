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

// Export<<.ModelName>>sArgs export task args alias.
type Export<<.ModelName>>sArgs = ExportArgs

// Export<<.ModelName>>s exports <<.ModelName>> records asynchronously.
type Export<<.ModelName>>s struct{}

func (r *Export<<.ModelName>>s) Signature() string {
	return "export_<<.ModuleName>>s"
}

func (r *Export<<.ModelName>>s) Handle(args ...any) (retErr error) {
	var exportID uint
	var jobCtx context.Context

	defer func() {
		if rec := recover(); rec != nil {
			errorMsg := fmt.Sprintf("panic: %v", rec)
			facades.Log().Errorf("Export<<.ModelName>>s Job panic: %v", rec)
			MarkExportFailed(jobCtx, exportID, errorMsg)
			retErr = fmt.Errorf("%s", errorMsg)
		}
	}()

	exportArgs, err := ParseArgs(args...)
	if err != nil {
		return err
	}
	exportID = exportArgs.ExportID
	jobCtx, err = JobContext(exportArgs)
	if err != nil {
		return err
	}

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
		FilePrefix: "<<.ModuleName>>s",
		HeaderKeys: []string{
			<<- range .ListFields>>
			<<- if and .ShowInList (ne .Name "operation")>>
			"<<.Name>>",
			<<- end>>
			<<- end>>
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

func (r *Export<<.ModelName>>s) writeToCSV(ctx context.Context, w *csv.Writer, filters map[string]any, lang string, shouldStop func() bool) error {
	var modelFilters services.<<.ModelName>>Filters
	utils.FillFiltersFromMap(filters, &modelFilters)

	timezone, _ := utils.GetString(filters, "_timezone")

	const chunkSize = 2000
	lastID := uint(0)

	for {
		if shouldStop != nil && shouldStop() {
			return ErrExportRecordMissing
		}

		q := services.Build<<.ModelName>>Query(ctx, modelFilters)
		if lastID > 0 {
			q = q.Where("id < ?", lastID)
		}
		q = q.Order("id desc").Limit(chunkSize)

		var rows []models.<<.ModelName>>
		if err := q.Get(&rows); err != nil {
			return fmt.Errorf("query <<.ModuleName>>s for export failed: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			record := []string{
				<<- range .ListFields>>
				<<- if and .ShowInList (ne .Name "operation")>>
				<<- if eq .Name "created_at">>
				FormatCarbonWithTimezone(row.CreatedAt, timezone),
				<<- else if eq .Name "updated_at">>
				FormatCarbonWithTimezone(row.UpdatedAt, timezone),
				<<- else if eq .GoType "time.Time">>
				FormatTimeWithTimezone(row.<<.FieldName>>, timezone),
				<<- else if eq .GoType "string">>
				row.<<.FieldName>>,
				<<- else>>
				cast.ToString(row.<<.FieldName>>),
				<<- end>>
				<<- end>>
				<<- end>>
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
