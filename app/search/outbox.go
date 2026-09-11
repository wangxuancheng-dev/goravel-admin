package search

import (
	"context"
	"encoding/json"
	"time"

	appfacades "goravel/app/facades"

	"github.com/goravel/framework/facades"
)

// OutboxTable 搜索同步 outbox 表名。
const OutboxTable = "search_sync_outbox"

const (
	outboxStatusPending   = "pending"
	outboxStatusProcessed = "processed"
	outboxStatusFailed    = "failed"
)

// OutboxRecord 待重试记录。
type OutboxRecord struct {
	ID        uint
	EntityID  uint
	EntityKey string
	Op        string
	Payload   string
	Attempts  int
}

// CreateSyncOutbox 写入同步 outbox（ctx 应带租户连接，若 tenancy 开启）。
func CreateSyncOutbox(ctx context.Context, orderID uint, orderNo, op string, payload map[string]any) {
	if !OutboxEnabled() || !facades.Schema().HasTable(OutboxTable) {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		facades.Log().Errorf("search outbox marshal failed: %v", err)
		return
	}
	now := time.Now().UTC()
	record := map[string]any{
		"entity_type": "order",
		"entity_id":   orderID,
		"entity_key":  orderNo,
		"op":          op,
		"payload":     string(payloadJSON),
		"status":      outboxStatusPending,
		"attempts":    0,
		"created_at":  now,
		"updated_at":  now,
	}
	if err := appfacades.OrmQuery(ctx).Table(OutboxTable).Create(record); err != nil {
		facades.Log().Errorf("search outbox create failed: order_id=%d err=%v", orderID, err)
	}
}

func findLatestPendingOutboxID(ctx context.Context, orderID uint, op string) uint {
	if orderID == 0 || !facades.Schema().HasTable(OutboxTable) {
		return 0
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var row struct {
		ID uint `gorm:"column:id"`
	}
	if err := appfacades.OrmQuery(ctx).Table(OutboxTable).
		Where("entity_type", "order").
		Where("entity_id", orderID).
		Where("op", op).
		Where("status", outboxStatusPending).
		Order("id DESC").
		Limit(1).
		First(&row); err != nil || row.ID == 0 {
		return 0
	}
	return row.ID
}

func updateOutboxByID(ctx context.Context, id uint, data map[string]any) {
	if id == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	data["updated_at"] = time.Now().UTC()
	_, _ = appfacades.OrmQuery(ctx).Table(OutboxTable).Where("id", id).Update(data)
}

// ListOutboxRetryable 列出可重试 outbox（CLI 在 WithTenantConnection 下调用时走当前 default）。
func ListOutboxRetryable(limit, maxAttempts int) ([]OutboxRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if !OutboxEnabled() || !facades.Schema().HasTable(OutboxTable) {
		return nil, nil
	}

	ctx := context.Background()
	var rows []struct {
		ID        uint   `gorm:"column:id"`
		EntityID  uint   `gorm:"column:entity_id"`
		EntityKey string `gorm:"column:entity_key"`
		Op        string `gorm:"column:op"`
		Payload   string `gorm:"column:payload"`
		Attempts  int    `gorm:"column:attempts"`
	}
	err := appfacades.OrmQuery(ctx).Table(OutboxTable).
		Where("status IN ?", []string{outboxStatusPending, outboxStatusFailed}).
		Where("attempts < ?", maxAttempts).
		Order("id ASC").
		Limit(limit).
		Find(&rows)
	if err != nil {
		return nil, err
	}

	records := make([]OutboxRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, OutboxRecord{
			ID:        row.ID,
			EntityID:  row.EntityID,
			EntityKey: row.EntityKey,
			Op:        row.Op,
			Payload:   row.Payload,
			Attempts:  row.Attempts,
		})
	}
	return records, nil
}

// MarkOutboxProcessedByID 按 ID 标记已处理。
func MarkOutboxProcessedByID(id uint) {
	MarkOutboxProcessedByIDCtx(context.Background(), id)
}

// MarkOutboxProcessedByIDCtx 按 ID 标记已处理（可带租户 ctx）。
func MarkOutboxProcessedByIDCtx(ctx context.Context, id uint) {
	if id == 0 {
		return
	}
	now := time.Now().UTC()
	updateOutboxByID(ctx, id, map[string]any{
		"status":       outboxStatusProcessed,
		"processed_at": now,
	})
}

// MarkOutboxFailedByID 按 ID 标记失败。
func MarkOutboxFailedByID(id uint, errMsg string) {
	MarkOutboxFailedByIDCtx(context.Background(), id, errMsg)
}

// MarkOutboxFailedByIDCtx 按 ID 标记失败（可带租户 ctx）。
func MarkOutboxFailedByIDCtx(ctx context.Context, id uint, errMsg string) {
	if id == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var attempts int
	_ = appfacades.OrmQuery(ctx).Table(OutboxTable).Where("id", id).Pluck("attempts", &attempts)
	updateOutboxByID(ctx, id, map[string]any{
		"status":     outboxStatusFailed,
		"attempts":   attempts + 1,
		"last_error": errMsg,
	})
}

// CountOutboxBacklog 统计积压。
func CountOutboxBacklog() (pending int64, failed int64) {
	if !facades.Schema().HasTable(OutboxTable) {
		return 0, 0
	}
	ctx := context.Background()
	pending, _ = appfacades.OrmQuery(ctx).Table(OutboxTable).
		Where("status", outboxStatusPending).Count()
	failed, _ = appfacades.OrmQuery(ctx).Table(OutboxTable).
		Where("status", outboxStatusFailed).Count()
	return pending, failed
}

// MarkSyncOutboxProcessed 标记最近一条 pending 为已处理。
func MarkSyncOutboxProcessed(orderID uint, op string) {
	MarkSyncOutboxProcessedCtx(context.Background(), orderID, op)
}

// MarkSyncOutboxProcessedCtx 标记最近一条 pending 为已处理（可带租户 ctx）。
func MarkSyncOutboxProcessedCtx(ctx context.Context, orderID uint, op string) {
	id := findLatestPendingOutboxID(ctx, orderID, op)
	if id == 0 {
		return
	}
	now := time.Now().UTC()
	updateOutboxByID(ctx, id, map[string]any{
		"status":       outboxStatusProcessed,
		"processed_at": now,
	})
}

// MarkSyncOutboxFailed 标记最近一条 pending 为失败。
func MarkSyncOutboxFailed(orderID uint, op, errMsg string) {
	MarkSyncOutboxFailedCtx(context.Background(), orderID, op, errMsg)
}

// MarkSyncOutboxFailedCtx 标记最近一条 pending 为失败（可带租户 ctx）。
func MarkSyncOutboxFailedCtx(ctx context.Context, orderID uint, op, errMsg string) {
	id := findLatestPendingOutboxID(ctx, orderID, op)
	if id == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var attempts int
	_ = appfacades.OrmQuery(ctx).Table(OutboxTable).Where("id", id).Pluck("attempts", &attempts)
	updateOutboxByID(ctx, id, map[string]any{
		"status":     outboxStatusFailed,
		"attempts":   attempts + 1,
		"last_error": errMsg,
	})
}
