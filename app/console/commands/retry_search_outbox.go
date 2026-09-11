package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/models"
	"goravel/app/queuejobs"
	"goravel/app/search"
	searchorders "goravel/app/search/orders"
	"goravel/app/tenancyctx"
	"goravel/app/utils"
)

// RetrySearchOutbox 重试 pending/failed 的搜索同步 outbox。
type RetrySearchOutbox struct{}

func (r *RetrySearchOutbox) Signature() string { return "search:retry-outbox" }
func (r *RetrySearchOutbox) Description() string {
	return "重试搜索同步 outbox（pending/failed；tenancy 开启时按租户执行）"
}
func (r *RetrySearchOutbox) Extend() command.Extend {
	return command.Extend{
		Category: "search",
		Flags: []command.Flag{
			&command.IntFlag{Name: "limit", Value: 100, Aliases: []string{"l"}, Usage: "单次处理条数上限"},
			&command.IntFlag{Name: "max-attempts", Value: 5, Usage: "最大重试次数，超过后跳过"},
			&command.BoolFlag{Name: "dispatch", Aliases: []string{"q"}, Usage: "重新入队（默认直接同步执行）"},
			TenantScopeFlag(),
		},
	}
}

func (r *RetrySearchOutbox) Handle(ctx console.Context) error {
	if !search.OrdersSyncEnabled() || !search.OutboxEnabled() {
		return nil
	}

	limit := cast.ToInt(ctx.Option("limit"))
	maxAttempts := cast.ToInt(ctx.Option("max-attempts"))
	dispatch := cast.ToBool(ctx.Option("dispatch"))

	return RunTenantScoped(ctx, func(tenant *models.Tenant, bound context.Context) error {
		records, err := search.ListOutboxRetryableCtx(bound, limit, maxAttempts)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			pending, failed := search.CountOutboxBacklogCtx(bound)
			ctx.Info(fmt.Sprintf("无待重试记录（pending=%d failed=%d）", pending, failed))
			return nil
		}

		ctx.Info(fmt.Sprintf("开始重试 %d 条 outbox 记录...", len(records)))
		var okCount, failCount int
		tenantID, _ := tenancyctx.IDFrom(bound)

		for _, record := range records {
			payload := map[string]any{
				"order_id": record.EntityID,
				"op":       record.Op,
			}
			if record.EntityKey != "" {
				payload["order_no"] = record.EntityKey
			}
			if record.Payload != "" {
				var parsed map[string]any
				if err := json.Unmarshal([]byte(record.Payload), &parsed); err == nil && parsed != nil {
					payload = parsed
				}
			}
			if tenantID > 0 {
				payload["tenant_id"] = tenantID
			} else if tenant != nil && tenant.ID > 0 {
				payload["tenant_id"] = tenant.ID
			}

			if dispatch {
				jsonBytes, err := json.Marshal(payload)
				if err != nil {
					search.MarkOutboxFailedByIDCtx(bound, record.ID, err.Error())
					failCount++
					continue
				}
				qargs := []queue.Arg{{Type: "string", Value: string(jsonBytes)}}
				if err := facades.Queue().Job(&queuejobs.SyncOrderSearch{}, qargs).OnQueue(search.SyncQueue()).Dispatch(); err != nil {
					search.MarkOutboxFailedByIDCtx(bound, record.ID, err.Error())
					failCount++
					continue
				}
				okCount++
				continue
			}

			orderID := cast.ToUint(payload["order_id"])
			op, _ := utils.GetString(payload, "op")
			orderNo, _ := utils.GetString(payload, "order_no")
			if orderID == 0 || (op != "index" && op != "delete") {
				search.MarkOutboxFailedByIDCtx(bound, record.ID, "invalid outbox payload")
				failCount++
				continue
			}
			if err := searchorders.Push(bound, orderID, orderNo, op); err != nil {
				search.MarkOutboxFailedByIDCtx(bound, record.ID, err.Error())
				failCount++
				continue
			}
			search.MarkOutboxProcessedByIDCtx(bound, record.ID)
			okCount++
		}

		if failCount > 0 {
			ctx.Warning(fmt.Sprintf("完成：成功 %d，失败 %d", okCount, failCount))
		} else {
			ctx.Success(fmt.Sprintf("完成：成功 %d", okCount))
		}
		return nil
	})
}
