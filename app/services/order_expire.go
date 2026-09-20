package services

import (
	"context"
	"strings"
	"time"

	appfacades "goravel/app/facades"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/support"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

// EnqueueCancelExpiredOrderFn dispatches a delayed cancel job.
// Wired from QueueServiceProvider to avoid services<->jobs import cycle.
var EnqueueCancelExpiredOrderFn = func(ctx context.Context, orderNo string, expireAt time.Time, tenantID uint) error {
	return nil
}

// OrderIsExpired reports whether expire_at has passed (nil expire_at => false).
func OrderIsExpired(order *models.Order, now time.Time) bool {
	if order == nil || order.ExpireAt == nil {
		return false
	}
	return !now.UTC().Before(order.ExpireAt.UTC())
}

// CancelOrderIfExpired cancels a pending order when expire_at <= now.
// Idempotent: paid/cancelled/not-yet-due are no-ops.
func CancelOrderIfExpired(ctx context.Context, orderNo string) (cancelled bool, err error) {
	orderNo = trimOrderNo(orderNo)
	if orderNo == "" {
		return false, apperrors.ErrOrderNoRequired
	}
	order, err := FindOrderByOrderNo(ctx, orderNo)
	if err != nil || order == nil {
		return false, nil
	}
	if order.Status != models.OrderStatusPending {
		return false, nil
	}
	now := time.Now().UTC()
	if !OrderIsExpired(order, now) {
		return false, nil
	}

	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)
	tableName := utils.GetShardingTableName("orders", createdAt)

	res, err := appfacades.OrmQuery(ctx).Table(tableName).
		Where("id", order.ID).
		Where("status", models.OrderStatusPending).
		Where("expire_at IS NOT NULL").
		Where("expire_at <= ?", now).
		Update(map[string]any{
			"status": models.OrderStatusCancelled,
		})
	if err != nil {
		return false, err
	}
	if res == nil || res.RowsAffected == 0 {
		return false, nil
	}

	support.RequestOrderSearchSync(order.ID, order.OrderNo, "index", orderTenantID(ctx))
	return true, nil
}

// CancelExpiredOrdersSweep scans recent order shards for overdue pending rows.
func CancelExpiredOrdersSweep(ctx context.Context) (cancelled int, err error) {
	now := time.Now().UTC()
	start := now.AddDate(0, -utils.GetIDLookupScanMonths(), 0)
	tables := utils.GetShardingTableNames("orders", start, now)

	for _, tableName := range tables {
		if !appfacades.SchemaHasTable(ctx, tableName) {
			continue
		}
		if !appfacades.SchemaHasColumn(ctx, tableName, "expire_at") {
			continue
		}
		var orders []models.Order
		if qerr := appfacades.OrmQuery(ctx).Table(tableName).
			Where("status", models.OrderStatusPending).
			Where("expire_at IS NOT NULL").
			Where("expire_at <= ?", now).
			Limit(500).
			Find(&orders); qerr != nil {
			errorlog.Record(ctx, "order", "cancel expired sweep query failed", map[string]any{
				"table": tableName,
				"error": qerr.Error(),
			}, "cancel expired sweep query failed: %v", qerr)
			continue
		}
		for i := range orders {
			ok, cerr := CancelOrderIfExpired(ctx, orders[i].OrderNo)
			if cerr != nil {
				errorlog.Record(ctx, "order", "cancel expired failed", map[string]any{
					"order_no": orders[i].OrderNo,
					"error":    cerr.Error(),
				}, "cancel expired failed: %v", cerr)
				continue
			}
			if ok {
				cancelled++
			}
		}
	}
	return cancelled, nil
}

// ScheduleOrderExpireCancel sets expire_at semantics after create and enqueues Delay job.
func ScheduleOrderExpireCancel(ctx context.Context, order *models.Order, expireIn time.Duration) error {
	if order == nil || expireIn <= 0 {
		return nil
	}
	expireAt := time.Now().UTC().Add(expireIn)
	order.ExpireAt = &expireAt

	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)
	tableName := utils.GetShardingTableName("orders", createdAt)

	if _, err := appfacades.OrmQuery(ctx).Table(tableName).
		Where("id", order.ID).
		Update(map[string]any{"expire_at": expireAt}); err != nil {
		return err
	}

	return EnqueueCancelExpiredOrderFn(ctx, order.OrderNo, expireAt, orderTenantID(ctx))
}

func trimOrderNo(orderNo string) string {
	return strings.TrimSpace(orderNo)
}
