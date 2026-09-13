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

// Export users job args (ExportArgs alias)
type ExportUsersArgs = ExportArgs

// Users export job
type ExportUsers struct{}

func (r *ExportUsers) Signature() string {
	return "export_users"
}

func (r *ExportUsers) Handle(args ...any) (retErr error) {
	var exportID uint
	var jobCtx context.Context

	defer func() {
		if rec := recover(); rec != nil {
			errorMsg := fmt.Sprintf("panic: %v", rec)
			facades.Log().Errorf("ExportUsers Job panic: %v", rec)
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
		facades.Log().Infof("export execution lock not acquired: export_id=%d", exportID)
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
		FilePrefix: "users",
		HeaderKeys: []string{
			"id",
			"username",
			"nickname",
			"email",
			"phone",
			"balance",
			"currency",
			"status",
			"last_login_at",
			"created_at",
		},
		WriteData: r.writeUsersToCSV,
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

// Write users to CSV
func (r *ExportUsers) writeUsersToCSV(ctx context.Context, w *csv.Writer, filters map[string]any, lang string, shouldStop func() bool) error {
	// helper
	var userFilters services.UserFilters
	utils.FillFiltersFromMap(filters, &userFilters)

	orderBy, _ := utils.GetString(filters, "order_by")
	_, direction := ParseOrderBy(orderBy)

	// helper
	timezone, _ := utils.GetString(filters, "_timezone")

	const chunkSize = 2000
	var lastID uint = 0

	for {
		if shouldStop != nil && shouldStop() {
			return ErrExportRecordMissing
		}

		// helper
		query := services.BuildUserQuery(ctx, userFilters).With("Currency")

		// Keyset ??
		if lastID > 0 {
			if direction == "desc" {
				query = query.Where("id < ?", lastID)
			} else {
				query = query.Where("id > ?", lastID)
			}
		}

		if direction == "desc" {
			query = query.OrderByDesc("id")
		} else {
			query = query.OrderBy("id")
		}

		var users []models.User
		if err := query.Limit(chunkSize).Get(&users); err != nil {
			return fmt.Errorf("failed: %v", err)
		}

		if len(users) == 0 {
			break
		}

		// ?? CSV
		for _, user := range users {
			row := r.formatUserRow(user, lang, timezone)
			if err := w.Write(row); err != nil {
				return fmt.Errorf("??CSV??: %v", err)
			}
		}

		// helper
		lastID = users[len(users)-1].ID

		if len(users) < chunkSize {
			break
		}
	}

	return nil
}

// formatUserRow
func (r *ExportUsers) formatUserRow(user models.User, lang, timezone string) []string {
	// helper
	statusKey := "disabled"
	if user.Status == 1 {
		statusKey = "enabled"
	}
	statusText := utils.TranslateKey(statusKey, lang, cast.ToString(user.Status))

	// helper
	currencyName := ""
	if user.Currency != nil {
		currencyName = user.Currency.Name
	}

	// helper
	lastLoginAt := ""
	if user.LastLoginAt != nil {
		lastLoginAt = FormatTimeWithTimezone(*user.LastLoginAt, timezone)
	}

	// helper
	createdAt := FormatCarbonWithTimezone(user.CreatedAt, timezone)

	return []string{
		cast.ToString(user.ID),
		user.Username,
		user.Nickname,
		user.Email,
		user.Phone,
		fmt.Sprintf("%.8f", user.Balance),
		currencyName,
		statusText,
		lastLoginAt,
		createdAt,
	}
}
