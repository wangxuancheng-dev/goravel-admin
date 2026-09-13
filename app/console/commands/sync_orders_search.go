package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/models"
	"goravel/app/search"
	searchorders "goravel/app/search/orders"
	"goravel/app/services"
	"goravel/app/utils"
)

// SyncOrdersSearch ???????????????
type SyncOrdersSearch struct {
	cancel context.CancelFunc
}

func (r *SyncOrdersSearch) Signature() string {
	return "search:sync-orders"
}

func (r *SyncOrdersSearch) Description() string {
	return "?????????????????? 3 ???tenancy ?????????"
}

func (r *SyncOrdersSearch) Extend() command.Extend {
	return command.Extend{
		Category: "search",
		Flags: []command.Flag{
			&command.StringFlag{Name: "from", Usage: "???? RFC3339????? --to ????"},
			&command.StringFlag{Name: "to", Usage: "???? RFC3339???"},
			&command.StringFlag{Name: "order-id", Usage: "??????? ID"},
			TenantScopeFlag(),
		},
	}
}

func (r *SyncOrdersSearch) Handle(ctx console.Context) error {
	if !search.OrdersSyncEnabled() {
		ctx.Warning("??????????? SEARCH_ENABLED=true?SEARCH_DRIVER?null?? SEARCH_SYNC_ORDERS=true?")
		return nil
	}

	runParent, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	defer cancel()

	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		runCtx, cancelTenant := context.WithCancel(bound)
		defer cancelTenant()
		go func() {
			select {
			case <-runParent.Done():
				cancelTenant()
			case <-runCtx.Done():
			}
		}()

		svc := services.NewOrderService(bound)

		if id := cast.ToUint(ctx.Option("order-id")); id > 0 {
			ctx.Info(fmt.Sprintf("???? id=%d driver=%s index=%s ...", id, search.Driver(), searchorders.IndexName(bound)))
			if err := searchorders.InitIndex(runCtx); err != nil {
				return err
			}
			if err := searchorders.Push(runCtx, id, "", "index"); err != nil {
				return err
			}
			ctx.Success("??")
			return nil
		}

		var filters services.OrderFilters
		fromStr := ctx.Option("from")
		toStr := ctx.Option("to")
		if fromStr != "" && toStr != "" {
			var err error
			filters.StartTime, err = time.Parse(time.RFC3339, fromStr)
			if err != nil {
				return fmt.Errorf("from ?????????? RFC3339")
			}
			filters.EndTime, err = time.Parse(time.RFC3339, toStr)
			if err != nil {
				return fmt.Errorf("to ?????????? RFC3339")
			}
		} else {
			now := time.Now().UTC()
			filters.EndTime = now
			filters.StartTime = now.AddDate(0, -3, 0)
			ctx.Info(fmt.Sprintf("????: %s ~ %s???? from/to ????? 3 ???",
				filters.StartTime.Format(time.RFC3339), filters.EndTime.Format(time.RFC3339)))
		}

		valid, err := utils.ValidateTimeRange(filters.StartTime, filters.EndTime)
		if !valid {
			return err
		}

		list, err := svc.GetAllOrdersWithDetailsForExport(filters)
		if err != nil {
			return err
		}
		ctx.Info(fmt.Sprintf("? %d ?????? %s index=%s...", len(list), search.Driver(), searchorders.IndexName(bound)))
		if err := searchorders.InitIndex(runCtx); err != nil {
			return err
		}
		var fail int
		for i, row := range list {
			if runParent.Err() != nil {
				ctx.Warning("????????????")
				return runParent.Err()
			}
			if err := searchorders.Push(runCtx, row.ID, row.OrderNo, "index"); err != nil {
				fail++
				facades.Log().Warningf("search sync order id=%d: %v", row.ID, err)
			}
			if (i+1)%500 == 0 {
				ctx.Info(fmt.Sprintf("??? %d/%d", i+1, len(list)))
			}
		}
		if fail > 0 {
			ctx.Warning(fmt.Sprintf("????? %d ??????", fail))
		} else {
			ctx.Success(fmt.Sprintf("???? %d ?", len(list)))
		}
		return nil
	})
}

func (r *SyncOrdersSearch) Shutdown(ctx console.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}
