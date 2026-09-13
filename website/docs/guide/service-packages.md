# Service 包拆分路线

`app/services` 目前是按文件分域、同包共存。新安全/通知能力已拆为独立文件（如 `password_policy.go`、`login_anomaly.go`、`notification_channel.go`），避免继续堆进上帝服务。

## 原则

1. **CRUD / 生成器模块**继续留在 `app/services`，与模板一致。
2. **跨域编排**（支付落单、租户连接、导出/导入入队）保持小接口 + 独立文件。
3. **下一阶段**按域迁到子包（`orders`、`payment`、`tenancy`、`rbac`），通过薄 facade 保持控制器 import 稳定。
4. 优先拆 **高耦合高行数** 文件：`code_generator_service`、`order_service`、`payment_service`、`tenant_connection_service`。
5. **禁止循环依赖**：`services` 可 import `rbac` / `orders` / `payment` / `tenancy`；这些域包 **不得** import `services`。

## 已迁出（starter）

| 能力 | 包 / 文件 | 说明 |
|------|-----------|------|
| 支付成功态闸门 | `app/payment` | `PaidResult` + `ApplyPaidResultStatusGate`（纯逻辑） |
| 支付落库编排 | `app/services/payment_apply.go` | `ApplyPaidResult` 仍在 services，调用 `app/payment` 闸门；类型别名 `services.PaidResult` 保持兼容 |
| 数据权限 | `app/rbac` | `DataScopeResolved` / `ApplyDataScope` / `CanAccessOwnedBy` / `ResolveAdminDataScope`；`services` 保留同名薄封装（生成器/现有调用不变） |
| 订单筛选与 JSON | `app/orders` | `Filters`、时间解析、分表名映射、`ToJSON` / `DetailToJSON`；`OrderServiceImpl` 仍在 services，方法委托到 `orders` |
| 导入任务中心 | `ImportRecordService` + `ImportController` | 列表/详情/删除/错误文件下载；前端 `export/TaskCenter` |

后续可将网关驱动与 `ApplyPaidResult` 整段迁入 `app/payment`；`OrderServiceImpl`（约 700LOC）整包迁出仍待下一阶段。

## 租户（tenancy）清单

纯解析 / 前缀类逻辑已在 `app/tenancy`（如 `MergeTenantHints`、`SubdomainHint`、`CacheKey`、`StoragePrefix`、`PaymentNotifyPath`）。

| 仍留在 services | 原因 |
|-----------------|------|
| `tenant_connection_service.go` | ORM 连接注册、迁移、绑定编排，依赖 facades + DB |
| `tenant_ops_service.go` / `tenant_admin_service.go` / `tenant_scope.go` | 运营 CRUD 与作用域查询，与生成器式 service 同层 |

下一阶段再拆连接层时：保持 `app/tenancy` 无 `services` 依赖；连接服务可迁到 `app/tenancy/connection` 或类似子包，services 只留薄 facade。

## 双前端

React（`html-react/`）为**主发货 UI**；Vue 为对等参考实现。详见 [双前端对齐](/guide/frontend-parity)。
