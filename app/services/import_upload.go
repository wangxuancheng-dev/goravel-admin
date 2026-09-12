package services

import (
	"context"
	"encoding/csv"
	"strings"

	"github.com/goravel/framework/contracts/filesystem"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
)

// SaveUploadedCSVForAsync persists an uploaded CSV for a background import job (not deleted).
func SaveUploadedCSVForAsync(ctx context.Context, file filesystem.File) (disk, savedPath, filename string, dataRows int, err error) {
	if file == nil {
		return "", "", "", 0, apperrors.ErrFileRequired
	}
	filename = file.GetClientOriginalName()
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		return "", "", filename, 0, apperrors.ErrInvalidFileType
	}

	disk = "local"
	storage := facades.Storage().Disk(disk)
	tmpDir := strings.TrimSuffix(helpers.TenantStoragePrefix(ctx), "/")
	if tmpDir == "" {
		tmpDir = "imports/pending"
	} else {
		tmpDir = tmpDir + "/imports/pending"
	}
	savedPath, err = storage.PutFile(tmpDir, file)
	if err != nil {
		return disk, "", filename, 0, err
	}
	csvContent, err := storage.Get(savedPath)
	if err != nil {
		_ = storage.Delete(savedPath)
		return disk, "", filename, 0, err
	}
	dataRows = CountCSVDataRows(csvContent)
	return disk, savedPath, filename, dataRows, nil
}

// CountCSVDataRows counts non-empty data rows (excluding header).
func CountCSVDataRows(csvContent string) int {
	reader := csv.NewReader(strings.NewReader(csvContent))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return 0
	}
	count := 0
	for _, row := range records[1:] {
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			continue
		}
		count++
	}
	return count
}
