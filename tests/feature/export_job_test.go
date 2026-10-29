package feature_test

import (
	"context"
	"encoding/csv"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/utils"
)

// withExportFormat sets configs storage.export_format for the test and restores the previous value.
func withExportFormat(t *testing.T, format string) {
	t.Helper()
	ctx := context.Background()
	var prev models.Config
	_ = appfacades.OrmQuery(ctx).Where("group", "storage").Where("key", "export_format").First(&prev)
	if prev.ID > 0 {
		old := prev.Value
		_, err := appfacades.OrmQuery(ctx).Model(&models.Config{}).Where("id", prev.ID).Update("value", format)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = appfacades.OrmQuery(ctx).Model(&models.Config{}).Where("id", prev.ID).Update("value", old)
		})
		return
	}
	row := models.Config{Group: "storage", Key: "export_format", Value: format, Label: "export format (test)"}
	require.NoError(t, appfacades.OrmQuery(ctx).Create(&row))
	t.Cleanup(func() {
		_, _ = appfacades.OrmQuery(ctx).Where("id", row.ID).Delete(&models.Config{})
	})
}

func createExportRecord(t *testing.T, exportType string) models.Export {
	t.Helper()
	ctx := context.Background()
	record := models.Export{Type: exportType, Disk: "local", Status: models.ExportStatusProcessing}
	require.NoError(t, appfacades.OrmQuery(ctx).Create(&record))
	t.Cleanup(func() {
		var final models.Export
		if err := appfacades.OrmQuery(ctx).Where("id", record.ID).First(&final); err == nil && final.Path != "" {
			if disk, err := utils.StorageDisk("local"); err == nil {
				_ = disk.Delete(final.Path)
			}
		}
		_, _ = appfacades.OrmQuery(ctx).Where("id", record.ID).Delete(&models.Export{})
	})
	return record
}

func loadExportRecord(t *testing.T, id uint) models.Export {
	t.Helper()
	var record models.Export
	require.NoError(t, appfacades.OrmQuery(context.Background()).Where("id", id).First(&record))
	return record
}

// The full job writes a real file and only then marks the record successful.
func TestExportUsersJobWritesFile(t *testing.T) {
	withTenancyDriver(t, "off")

	for _, format := range []string{"csv", "xlsx"} {
		t.Run(format, func(t *testing.T) {
			withExportFormat(t, format)
			record := createExportRecord(t, models.ExportTypeUsers)

			err := (&jobs.ExportUsers{}).Handle(jobs.ExportArgs{ExportID: record.ID, Type: models.ExportTypeUsers, Language: "en"})
			require.NoError(t, err)

			got := loadExportRecord(t, record.ID)
			require.Equal(t, models.ExportStatusSuccess, got.Status, got.ErrorMsg)
			assert.Equal(t, format, got.Extension)
			assert.Positive(t, got.Size, "file size is read back from storage")

			disk, err := utils.StorageDisk("local")
			require.NoError(t, err)
			assert.True(t, disk.Exists(got.Path), "export file %s should exist", got.Path)
		})
	}
}

// A failing data writer must surface as an error, never as a successful export.
func TestExportFailureIsNotMarkedSuccess(t *testing.T) {
	withTenancyDriver(t, "off")

	for _, format := range []string{"csv", "xlsx"} {
		t.Run(format, func(t *testing.T) {
			withExportFormat(t, format)
			record := createExportRecord(t, models.ExportTypeUsers)

			exporter := jobs.NewBaseExporter(jobs.ExportConfig{
				FilePrefix: "users",
				HeaderKeys: []string{"id"},
				WriteData: func(context.Context, *csv.Writer, map[string]any, string, func() bool) error {
					return errors.New("boom")
				},
			})
			err := exporter.Execute(jobs.ExportArgs{ExportID: record.ID, Type: models.ExportTypeUsers, Language: "en"})
			require.Error(t, err)

			got := loadExportRecord(t, record.ID)
			assert.NotEqual(t, models.ExportStatusSuccess, got.Status)
		})
	}
}

// A job whose export record was deleted (user cancelled) exits quietly.
func TestExportJobSkipsMissingRecord(t *testing.T) {
	withTenancyDriver(t, "off")

	err := (&jobs.ExportUsers{}).Handle(jobs.ExportArgs{ExportID: 999999999, Type: models.ExportTypeUsers})
	require.NoError(t, err)

	exists, err := appfacades.OrmQuery(context.Background()).Model(&models.Export{}).Where("id", 999999999).Exists()
	require.NoError(t, err)
	assert.False(t, exists)
}
