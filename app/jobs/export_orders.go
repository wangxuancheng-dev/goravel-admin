package jobs

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

// ExportOrdersArgs 导出订单任务参数（ExportArgs 别名）
type ExportOrdersArgs = ExportArgs

// ExportOrders 订单导出任务
type ExportOrders struct{}

func (r *ExportOrders) Signature() string {
	return "export_orders"
}

func (r *ExportOrders) Handle(args ...any) (retErr error) {
	var exportID uint
	var jobCtx context.Context

	defer func() {
		if rec := recover(); rec != nil {
			errorMsg := fmt.Sprintf("panic: %v", rec)
			facades.Log().Errorf("ExportOrders Job panic: %v", rec)
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
		FilePrefix: "orders",
		HeaderKeys: []string{
			"id",
			"order_no",
			"user_id",
			"amount",
			"status",
			"item_index",
			"product_id",
			"product_name",
			"price",
			"quantity",
			"subtotal",
			"remark",
			"created_at",
		},
		WriteData: r.writeOrdersToCSV,
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

// writeOrdersToCSV 将订单写入 CSV
func (r *ExportOrders) writeOrdersToCSV(ctx context.Context, w *csv.Writer, filters map[string]any, lang string, shouldStop func() bool) error {
	var orderFilters services.OrderFilters
	utils.FillFiltersFromMap(filters, &orderFilters)

	orderFilters.StartTime, orderFilters.EndTime = GetDefaultTimeRange(filters)

	timezone, _ := utils.GetString(filters, "_timezone")

	tableNames := utils.GetShardingTableNames("orders", orderFilters.StartTime, orderFilters.EndTime)
	if len(tableNames) == 0 {
		return nil
	}

	_, direction := ParseOrderBy(orderFilters.OrderBy)
	if direction == "desc" {
		for i, j := 0, len(tableNames)-1; i < j; i, j = i+1, j-1 {
			tableNames[i], tableNames[j] = tableNames[j], tableNames[i]
		}
	}

	const chunkSize = 2000

	for _, tableName := range tableNames {
		if err := r.exportTable(ctx, w, tableName, orderFilters, lang, timezone, direction, chunkSize, shouldStop); err != nil {
			return err
		}
	}

	return nil
}

// exportTable 导出单个分表
func (r *ExportOrders) exportTable(ctx context.Context, w *csv.Writer, tableName string, filters services.OrderFilters, lang, timezone, direction string, chunkSize int, shouldStop func() bool) error {
	suffix := strings.TrimPrefix(tableName, "orders_")
	detailTableName := "order_details_" + suffix

	lastTimeStr := ""
	var lastID uint = 0

	for {
		if shouldStop != nil && shouldStop() {
			return ErrExportRecordMissing
		}

		query := services.BuildOrderQuery(ctx, tableName, filters)

		if lastTimeStr != "" {
			if direction == "desc" {
				query = query.Where("(created_at < ? OR (created_at = ? AND id < ?))", lastTimeStr, lastTimeStr, lastID)
			} else {
				query = query.Where("(created_at > ? OR (created_at = ? AND id > ?))", lastTimeStr, lastTimeStr, lastID)
			}
		}

		if direction == "desc" {
			query = query.Order("created_at desc").Order("id desc")
		} else {
			query = query.Order("created_at asc").Order("id asc")
		}

		var orders []models.Order
		if err := query.Limit(chunkSize).Get(&orders); err != nil {
			return fmt.Errorf("query orders failed: table=%s, err=%v", tableName, err)
		}

		if len(orders) == 0 {
			break
		}

		orderIDsAny := make([]any, 0, len(orders))
		for _, o := range orders {
			orderIDsAny = append(orderIDsAny, o.ID)
		}

		var details []models.OrderDetail
		if len(orderIDsAny) > 0 {
			_ = appfacades.OrmQuery(ctx).Table(detailTableName).
				WhereIn("order_id", orderIDsAny).
				Get(&details)
		}

		detailMap := make(map[uint][]models.OrderDetail, len(orders))
		for _, d := range details {
			detailMap[d.OrderID] = append(detailMap[d.OrderID], d)
		}

		// ?? CSV
		for _, order := range orders {
			if err := r.writeOrderRows(w, order, detailMap[order.ID], lang, timezone); err != nil {
				return err
			}
		}

		if shouldStop != nil && shouldStop() {
			return ErrExportRecordMissing
		}

		last := orders[len(orders)-1]
		if last.CreatedAt != nil && !last.CreatedAt.IsZero() {
			lastTimeStr = last.CreatedAt.ToDateTimeString()
		} else {
			lastTimeStr = ""
		}
		lastID = last.ID

		if len(orders) < chunkSize {
			break
		}
	}

	return nil
}

// writeOrderRows 写入订单行
func (r *ExportOrders) writeOrderRows(w *csv.Writer, order models.Order, details []models.OrderDetail, lang, timezone string) error {
	statusText := r.translateStatus(order.Status, lang)

	timeStr := FormatCarbonWithTimezone(order.CreatedAt, timezone)

	if len(details) == 0 {
		row := []string{
			cast.ToString(order.ID),
			order.OrderNo,
			cast.ToString(order.UserID),
			fmt.Sprintf("%.2f", order.Amount),
			statusText,
			"", "", "", "", "", "",
			order.Remark,
			timeStr,
		}
		return w.Write(row)
	}

	totalItems := len(details)
	for idx, detail := range details {
		row := []string{
			cast.ToString(order.ID),
			order.OrderNo,
			cast.ToString(order.UserID),
			fmt.Sprintf("%.2f", order.Amount),
			statusText,
			fmt.Sprintf("%d/%d", idx+1, totalItems),
			cast.ToString(detail.ProductID),
			detail.ProductName,
			fmt.Sprintf("%.2f", detail.Price),
			cast.ToString(detail.Quantity),
			fmt.Sprintf("%.2f", detail.Subtotal),
			order.Remark,
			timeStr,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// translateStatus 翻译订单状态
func (r *ExportOrders) translateStatus(status, lang string) string {
	switch status {
	case models.OrderStatusPending:
		return utils.TranslateKey("order_status_pending", lang, models.OrderStatusPending)
	case models.OrderStatusPaid:
		return utils.TranslateKey("order_status_paid", lang, models.OrderStatusPaid)
	case models.OrderStatusCancelled:
		return utils.TranslateKey("order_status_cancelled", lang, models.OrderStatusCancelled)
	default:
		return status
	}
}
