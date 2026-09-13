package jobs

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

// Export payments job args (ExportArgs alias)
type ExportPaymentsArgs = ExportArgs

// Payments export job
type ExportPayments struct{}

func (r *ExportPayments) Signature() string {
	return "export_payments"
}

func (r *ExportPayments) Handle(args ...any) (retErr error) {
	var exportID uint
	var jobCtx context.Context

	// panic recover
	defer func() {
		if rec := recover(); rec != nil {
			errorMsg := fmt.Sprintf("panic: %v", rec)
			facades.Log().Errorf("ExportPayments Job panic: %v", rec)
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
		FilePrefix: "payments",
		HeaderKeys: []string{
			"id",
			"payment_no",
			"order_no",
			"payment_method",
			"user_id",
			"amount",
			"status",
			"third_party_no",
			"pay_time",
			"fail_reason",
			"remark",
			"created_at",
		},
		WriteData: r.writePaymentsToCSV,
	})

	jobErr := exporter.Execute(exportArgs)

	if errors.Is(jobErr, ErrExportRecordMissing) {
		facades.Log().Infof("export record missing, stop: export_id=%d", exportID)
		return nil
	}

	if jobErr != nil {
		MarkExportFailed(jobCtx, exportID, jobErr.Error())
		return jobErr
	}

	return nil
}

// Write payments to CSV
func (r *ExportPayments) writePaymentsToCSV(ctx context.Context, w *csv.Writer, filters map[string]any, lang string, shouldStop func() bool) error {
	// helper
	var paymentFilters services.PaymentFilters
	utils.FillFiltersFromMap(filters, &paymentFilters)

	// helper
	paymentFilters.StartTime, paymentFilters.EndTime = GetDefaultTimeRange(filters)

	// helper
	timezone, _ := utils.GetString(filters, "_timezone")

	// helper
	tableNames := r.getTableNames(paymentFilters)
	if len(tableNames) == 0 {
		return nil
	}

	// helper
	_, direction := ParseOrderBy(paymentFilters.OrderBy)
	if direction == "desc" {
		for i, j := 0, len(tableNames)-1; i < j; i, j = i+1, j-1 {
			tableNames[i], tableNames[j] = tableNames[j], tableNames[i]
		}
	}

	// helper
	paymentMethodMap := r.loadPaymentMethods(ctx)

	const chunkSize = 2000

	for _, tableName := range tableNames {
		if err := r.exportTable(ctx, w, tableName, paymentFilters, paymentMethodMap, lang, timezone, direction, chunkSize, shouldStop); err != nil {
			return err
		}
	}

	return nil
}

// getTableNames
func (r *ExportPayments) getTableNames(filters services.PaymentFilters) []string {
	// helper
	if filters.PaymentNo != "" && len(filters.PaymentNo) >= 11 {
		dateStr := filters.PaymentNo[3:11]
		if t, err := time.Parse("20060102", dateStr); err == nil {
			return []string{utils.GetShardingTableName("payments", t)}
		}
	}
	return utils.GetShardingTableNames("payments", filters.StartTime, filters.EndTime)
}

// Export one sharding table
func (r *ExportPayments) exportTable(ctx context.Context, w *csv.Writer, tableName string, filters services.PaymentFilters, paymentMethodMap map[uint]models.PaymentMethod, lang, timezone, direction string, chunkSize int, shouldStop func() bool) error {
	lastTimeStr := ""
	var lastID uint = 0
	field := "created_at"

	for {
		if shouldStop != nil && shouldStop() {
			return ErrExportRecordMissing
		}

		query := services.BuildPaymentQuery(ctx, tableName, filters)

		// Keyset ??
		if lastTimeStr != "" {
			if direction == "desc" {
				query = query.Where(fmt.Sprintf("(%s < ? OR (%s = ? AND id < ?))", field, field), lastTimeStr, lastTimeStr, lastID)
			} else {
				query = query.Where(fmt.Sprintf("(%s > ? OR (%s = ? AND id > ?))", field, field), lastTimeStr, lastTimeStr, lastID)
			}
		} else if lastID > 0 {
			if direction == "desc" {
				query = query.Where("id < ?", lastID)
			} else {
				query = query.Where("id > ?", lastID)
			}
		}

		if direction == "desc" {
			query = query.OrderByDesc(field).OrderByDesc("id")
		} else {
			query = query.OrderBy(field).OrderBy("id")
		}

		var payments []models.Payment
		if err := query.Limit(chunkSize).Get(&payments); err != nil {
			if IsTableNotExistsError(err) {
				facades.Log().Warningf("????????????: table=%s", tableName)
				return nil
			}
			return fmt.Errorf("????????: %v", err)
		}

		if len(payments) == 0 {
			break
		}

		// ?? CSV
		for _, payment := range payments {
			row := r.formatPaymentRow(payment, paymentMethodMap, lang, timezone)
			if err := w.Write(row); err != nil {
				return fmt.Errorf("??CSV??: %v", err)
			}
		}

		// helper
		lastPayment := payments[len(payments)-1]
		prevID := lastID
		if lastPayment.CreatedAt != nil && !lastPayment.CreatedAt.IsZero() {
			lastTimeStr = lastPayment.CreatedAt.ToDateTimeString()
		} else {
			lastTimeStr = ""
		}
		lastID = lastPayment.ID

		if lastID == prevID {
			return fmt.Errorf("???????: table=%s, last_id=%d", tableName, lastID)
		}

		if len(payments) < chunkSize {
			break
		}
	}

	return nil
}

// formatPaymentRow
func (r *ExportPayments) formatPaymentRow(payment models.Payment, paymentMethodMap map[uint]models.PaymentMethod, lang, timezone string) []string {
	paymentMethodName := ""
	if pm, ok := paymentMethodMap[payment.PaymentMethodID]; ok {
		paymentMethodName = pm.Name
	}

	payTime := ""
	if payment.PayTime != nil {
		payTime = FormatTimeWithTimezone(*payment.PayTime, timezone)
	}

	createdAt := FormatCarbonWithTimezone(payment.CreatedAt, timezone)

	statusKey := fmt.Sprintf("payment_status_%s", payment.Status)

	return []string{
		cast.ToString(payment.ID),
		payment.PaymentNo,
		payment.OrderNo,
		paymentMethodName,
		cast.ToString(payment.UserID),
		fmt.Sprintf("%.2f", payment.Amount),
		utils.TranslateKey(statusKey, lang, payment.Status),
		payment.ThirdPartyNo,
		payTime,
		payment.FailReason,
		payment.Remark,
		createdAt,
	}
}

// loadPaymentMethods
func (r *ExportPayments) loadPaymentMethods(ctx context.Context) map[uint]models.PaymentMethod {
	var methods []models.PaymentMethod
	appfacades.OrmQuery(ctx).Model(&models.PaymentMethod{}).Get(&methods)

	result := make(map[uint]models.PaymentMethod)
	for _, m := range methods {
		result[m.ID] = m
	}
	return result
}
