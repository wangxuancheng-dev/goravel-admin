package commands

import (
	"context"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/search"
	searchorders "goravel/app/search/orders"
)

// InitOrdersSearchIndex 创建订单搜索索引（当前驱动）。
type InitOrdersSearchIndex struct{}

func (r *InitOrdersSearchIndex) Signature() string { return "search:init-orders-index" }
func (r *InitOrdersSearchIndex) Description() string {
	return "创建订单搜索索引（需 SEARCH_ENABLED=true；tenancy 开启时按租户执行）"
}
func (r *InitOrdersSearchIndex) Extend() command.Extend {
	return command.Extend{
		Category: "search",
		Flags:    []command.Flag{TenantScopeFlag()},
	}
}

func (r *InitOrdersSearchIndex) Handle(ctx console.Context) error {
	if !search.Enabled() {
		ctx.Warning("未启用搜索。请设置 SEARCH_ENABLED=true 且 SEARCH_DRIVER 不为 null。")
		return nil
	}
	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		ctx.Info("driver=" + search.Driver() + " index=" + searchorders.IndexName(bound))
		if err := searchorders.InitIndex(bound); err != nil {
			return err
		}
		ctx.Success("索引已就绪")
		return nil
	})
}
