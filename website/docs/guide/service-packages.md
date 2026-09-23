# 服务代码组织

业务逻辑默认放在 `app/services`（同包按文件分域）。只有纯逻辑、可单测、且不依赖 HTTP/Facades 编排时，才抽到 `app/*` 域包。

## 约定

1. **CRUD / 代码生成器模块**写在 `app/services`，与生成模板一致。
2. **跨域编排**（如支付落库、租户连接、导出/导入入队）留在 `app/services`，不要为编排单独建 ports 层。
3. **现有域包**：`app/payment`、`app/search`、`app/rbac`、`app/tenancy`。不要默认再新建 `app/<业务>`（例如不要为公告建 `app/notice`）。
4. **依赖方向**：`services` 可 import 域包；域包 **不得** import `services`，避免循环依赖。
5. 新代码优先 `app/services`；确有纯算法、强复用或需要打断 import 循环时再考虑域包。

## 现有域包一览

| 能力 | 位置 | 说明 |
|------|------|------|
| 支付状态闸门 | `app/payment` | `PaidResult`、`ApplyPaidResultStatusGate` |
| 支付网关驱动 | `app/payment/gateways/` | 每渠道一文件；`payment.RegisterGateway`；详见 [支付参考](/advanced/payments) |
| 支付接线 | `providers.PaymentServiceProvider` | blank-import gateways；注入 `ResolvePaymentAmount` |
| 支付落库编排 | `app/services/payment_apply.go` | 编排在 services，类型用 `payment.PaidResult`；白名单 `EnabledPaymentGateways` |
| 搜索引擎 | `app/search` | `Engine` / `Resolve` / drivers；禁止 import services |
| 订单搜索适配 | `app/search/orders` | `Document` / `Push` / `Query`；loader 由 SearchServiceProvider 注入 |
| 数据权限 | `app/rbac` | `ApplyDataScope` / `CanAccessOwnedBy` 等 |
| 订单筛选与 JSON | `app/services/order_filters.go`、`order_json.go` | 与 CRUD 同包 |
| 租户解析 | `app/tenancy` | Hint / CacheKey / StoragePrefix 等 |
| 导入任务中心 | services + controllers | 不必再拆 `app/imports` |

域包可外迁时：搬模块代码 + 保留对应 Provider 作为 blank-import / 钩子注入点；不要让域包反向依赖 `app/services`。

## 双前端

React（`html-react/`）为主界面；Vue（`html/`）为对等实现。详见 [双前端对齐](/guide/frontend-parity)。
