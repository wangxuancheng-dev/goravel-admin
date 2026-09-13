# Service 包拆分路线

`app/services` 以**同包按文件分域**为主；仅把「纯逻辑、可单测、且不依赖 HTTP/facades 编排」的小块抽到 `app/*` 域包。

## 原则（冻结）

1. **CRUD / 生成器模块**继续留在 `app/services`，与模板一致。
2. **跨域编排**（`ApplyPaidResult`、租户连接、导出/导入入队）**留在 services**，不要再为编排单独建 `app` 子包或 ports 接口层。
3. **已迁出的域包点到为止**：`payment` / `rbac`（及既有 `tenancy`）。**不再继续**把 OrderService / PaymentService / TenantConnection 整段搬进新目录；**默认不新增** `app/<业务>` 包。
4. **禁止循环依赖**：`services` 可 import `rbac` / `payment` / `tenancy`；域包 **不得** import `services`。
5. 新代码默认写 `app/services`；只有纯算法 / 强复用 / 打断 import 循环时才考虑新域包。

## 保留的拆分（有价值）

| 能力 | 包 / 文件 | 说明 |
|------|-----------|------|
| 支付成功态闸门 | `app/payment` | `PaidResult` + `ApplyPaidResultStatusGate`（纯逻辑） |
| 支付网关驱动 | `app/payment/gateways/` | 每渠道一文件；`payment.RegisterGateway`；SDK 或手写均可；`Notify` 只返回 `PaidResult`。详见 [支付参考](/advanced/payments) §6 |
| 支付落库编排 | `app/services/payment_apply.go` | 编排留 services；调用闸门；别名 `services.PaidResult` |
| 数据权限 | `app/rbac` | `ApplyDataScope` / `CanAccessOwnedBy` 等；services 直接调用 `rbac` |
| 订单筛选与 JSON | `app/services/order_filters.go` / `order_json.go` | 与 CRUD 同包，避免为订单再开 `app/orders` |
| 租户解析类 | `app/tenancy` | Hint / CacheKey / StoragePrefix 等（原本即独立） |
| 导入任务中心 | services + controllers | 不必再拆独立 `app/imports` |

## 明确不拆 / 已回退

| 项 | 处理 |
|----|------|
| `ApplyPaidResult` 整段 + `PaymentStore`/`OrderStore` ports | **不拆**；未完成的 `app/payment/store.go` 已删除 |
| 每支付渠道一个 `app/<vendor>` 包 | **不拆**；驱动放 `app/payment/gateways` |
| `app/orders`（filters/JSON） | **已收回** `app/services` |
| `tenant_connection_service` / `tenant_ops_service` | **留 services** |
| `OrderServiceImpl` / `PaymentService` 整包 | **留 services** |

## 双前端

React（`html-react/`）为**主发货 UI**；Vue 为对等参考实现。详见 [双前端对齐](/guide/frontend-parity)。
