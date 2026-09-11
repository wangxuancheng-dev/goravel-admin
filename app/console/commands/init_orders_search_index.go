package commands

import (
	"context"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/search"
	searchorders "goravel/app/search/orders"
)

// InitOrdersSearchIndex 创建订单搜索索引（当前驱动）。
type InitOrdersSearchIndex struct{}

func (r *InitOrdersSearchIndex) Signature() string { return "search:init-orders-index" }
func (r *InitOrdersSearchIndex) Description() string {
	return "创建订单搜索索引（需 SEARCH_ENABLED=true）"
}
func (r *InitOrdersSearchIndex) Extend() command.Extend {
	return command.Extend{Category: "search"}
}

func (r *InitOrdersSearchIndex) Handle(ctx console.Context) error {
	if !search.Enabled() {
		ctx.Warning("未启用搜索。请设置 SEARCH_ENABLED=true 且 SEARCH_DRIVER 不为 null。")
		return nil
	}
	ctx.Info("driver=" + search.Driver() + " index=" + searchorders.IndexName())
	if err := searchorders.InitIndex(context.Background()); err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Success("索引已就绪")
	return nil
}
