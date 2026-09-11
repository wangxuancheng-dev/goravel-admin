package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/search"
	searchorders "goravel/app/search/orders"
	"goravel/app/services"
	"goravel/app/utils"
)

// SyncOrdersSearch 手动将订单同步到当前搜索驱动。
type SyncOrdersSearch struct {
	cancel context.CancelFunc
}

func (r *SyncOrdersSearch) Signature() string {
	return "search:sync-orders"
}

func (r *SyncOrdersSearch) Description() string {
	return "手动同步订单到当前搜索引擎（默认最近 3 个月）"
}

func (r *SyncOrdersSearch) Extend() command.Extend {
	return command.Extend{
		Category: "search",
		Flags: []command.Flag{
			&command.StringFlag{Name: "from", Usage: "开始时间 RFC3339，可选；与 --to 成对使用"},
			&command.StringFlag{Name: "to", Usage: "结束时间 RFC3339，可选"},
			&command.StringFlag{Name: "order-id", Usage: "仅同步指定订单 ID"},
		},
	}
}

func (r *SyncOrdersSearch) Handle(ctx console.Context) error {
	if !search.OrdersSyncEnabled() {
		ctx.Warning("未开启订单搜索同步。需 SEARCH_ENABLED=true、SEARCH_DRIVER≠null，且 SEARCH_SYNC_ORDERS=true。")
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	defer cancel()

	svc := services.NewOrderService(context.Background())

	if id := cast.ToUint(ctx.Option("order-id")); id > 0 {
		ctx.Info(fmt.Sprintf("同步订单 id=%d driver=%s ...", id, search.Driver()))
		if err := searchorders.InitIndex(runCtx); err != nil {
			ctx.Error(err.Error())
			return err
		}
		if err := searchorders.Push(runCtx, id, "", "index"); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("完成")
		return nil
	}

	var filters services.OrderFilters
	fromStr := ctx.Option("from")
	toStr := ctx.Option("to")
	if fromStr != "" && toStr != "" {
		var err error
		filters.StartTime, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			ctx.Error("from 时间解析失败，请使用 RFC3339")
			return err
		}
		filters.EndTime, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			ctx.Error("to 时间解析失败，请使用 RFC3339")
			return err
		}
	} else {
		now := time.Now().UTC()
		filters.EndTime = now
		filters.StartTime = now.AddDate(0, -3, 0)
		ctx.Info(fmt.Sprintf("时间范围: %s ~ %s（未指定 from/to 时使用最近 3 个月）",
			filters.StartTime.Format(time.RFC3339), filters.EndTime.Format(time.RFC3339)))
	}

	valid, err := utils.ValidateTimeRange(filters.StartTime, filters.EndTime)
	if !valid {
		ctx.Error(err.Error())
		return err
	}

	list, err := svc.GetAllOrdersWithDetailsForExport(filters)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Info(fmt.Sprintf("共 %d 条，开始写入 %s...", len(list), search.Driver()))
	if err := searchorders.InitIndex(runCtx); err != nil {
		ctx.Error(err.Error())
		return err
	}
	var fail int
	for i, row := range list {
		if runCtx.Err() != nil {
			ctx.Warning("收到停止信号，已中断同步")
			return runCtx.Err()
		}
		if err := searchorders.Push(runCtx, row.ID, row.OrderNo, "index"); err != nil {
			fail++
			facades.Log().Warningf("search sync order id=%d: %v", row.ID, err)
		}
		if (i+1)%500 == 0 {
			ctx.Info(fmt.Sprintf("已处理 %d/%d", i+1, len(list)))
		}
	}
	if fail > 0 {
		ctx.Warning(fmt.Sprintf("完成，失败 %d 条（见日志）", fail))
	} else {
		ctx.Success(fmt.Sprintf("完成，共 %d 条", len(list)))
	}
	return nil
}

func (r *SyncOrdersSearch) Shutdown(ctx console.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}
