# Service 包拆分路线

`app/services` 目前是按文件分域、同包共存。新安全/通知能力已拆为独立文件（如 `password_policy.go`、`login_anomaly.go`、`notification_channel.go`），避免继续堆进上帝服务。

## 原则

1. **CRUD / 生成器模块**继续留在 `app/services`，与模板一致。
2. **跨域编排**（支付落单、租户连接、导出/导入入队）保持小接口 + 独立文件。
3. **下一阶段**按域迁到子包（`orders`、`payments`、`tenancy`、`rbac`），通过薄 facade 保持控制器 import 稳定。
4. 优先拆 **高耦合高行数** 文件：`code_generator_service`、`order_service`、`payment_service`、`tenant_connection_service`。

## 双前端

Vue（`html/`）为**主发货 UI**；React 为对等实现。详见 [双前端对齐](/guide/frontend-parity)。
