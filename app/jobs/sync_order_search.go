package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/errors"
	"goravel/app/search"
	searchorders "goravel/app/search/orders"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/utils"
)

// SyncOrderSearch 订单搜索同步队列任务。
// 以后文章等模块可同目录新增 sync_article_search.go，不必再拆 app/queuejobs。
type SyncOrderSearch struct{}

func (r *SyncOrderSearch) Signature() string {
	return "sync_order_search"
}

func (r *SyncOrderSearch) Handle(args ...any) error {
	if !search.OrdersSyncEnabled() {
		return nil
	}
	if len(args) < 1 {
		return errors.ErrInvalidArgument.WithMessage("missing sync args")
	}
	var m map[string]any
	switch v := args[0].(type) {
	case map[string]any:
		m = v
	case string:
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return errors.ErrInvalidArgument.WithMessage("sync args json invalid")
		}
	default:
		return errors.ErrInvalidArgument.WithMessage("sync args must be map or json string")
	}
	if m == nil {
		return errors.ErrInvalidArgument.WithMessage("sync args empty")
	}
	orderID := cast.ToUint(m["order_id"])
	if orderID == 0 {
		return errors.ErrInvalidArgument.WithMessage("invalid order_id")
	}
	op, _ := utils.GetString(m, "op")
	if op != "index" && op != "delete" {
		return fmt.Errorf("invalid op: %s", op)
	}
	orderNo, _ := utils.GetString(m, "order_no")
	tenantID := cast.ToUint(m["tenant_id"])
	ctx := context.Background()
	if tenancy.Enabled() {
		if tenantID == 0 {
			return errors.ErrTenantRequired
		}
		bound, err := services.NewTenantConnectionService().BindBackground(ctx, tenantID)
		if err != nil {
			facades.Log().Errorf("SyncOrderSearch bind tenant failed: tenant_id=%d err=%v", tenantID, err)
			return err
		}
		ctx = bound
	}
	if err := searchorders.Push(ctx, orderID, orderNo, op); err != nil {
		facades.Log().Errorf("SyncOrderSearch failed: order_id=%d op=%s err=%v", orderID, op, err)
		search.MarkSyncOutboxFailedCtx(ctx, orderID, op, err.Error())
		return err
	}
	search.MarkSyncOutboxProcessedCtx(ctx, orderID, op)
	return nil
}
