# 中小 SaaS 上线核对清单

面向**公网可服务的中小 SaaS / 运营后台**（一户一库可选、非金融收单）。  
详细运维步骤见 [生产清单](/deploy/production)。

## 1. 范围约定

| 范围内 | 不在范围内（默认关闭） |
|--------|------------------------------|
| 管理端 RBAC、菜单、日志、导出、代码生成 | 微信/支付宝真实验签与退款（用 mock 参考链路） |
| 一户一库 / Schema、provision 门禁、平台控制台 | 多活异地机房、金融级审计 |
| `/health` `/ready`、生产不安全默认告警 | 现成第三方告警面板 SaaS |
| 搜索扩展点（ES / Meili，订单索引可选） | 业务全文检索开箱即用 |
| 订单+支付 mock：`ApplyPaidResult` 幂等落库 | 生产收单 / 清算 |

支付边界见 [开源定位](/guide/opensource) §1.1、[支付参考](/advanced/payments)。

## 2. 上线前核对

### 安全

- [ ] `APP_DEBUG=false`，强 `APP_KEY` / `JWT_SECRET`
- [ ] 全局安全头已启用（`SecurityHeaders`）；HTTPS 后设 `SECURITY_HSTS_MAX_AGE=31536000`
- [ ] 一户一库公网：`TENANCY_RESOLVER=subdomain`，`TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`
- [ ] 子域名与 Header 冲突返回 `tenant_hint_conflict`；未 migrate 返回 `tenant_not_ready`（403）
- [ ] 关闭 Swagger / 代码生成器 / pprof；默认管理员已改密

### 可运维

- [ ] LB：`GET /health` 存活、`GET /ready` 就绪（含 DB；按需 Redis/Search）
- [ ] `/health` 可带 `app` / `env` / `version`（`APP_VERSION`）
- [ ] Web + Queue Worker 分离（API：`APP_DISABLED_RUNNERS=queue-*`；Worker：勿禁用）；`CACHE_STORE=redis`、`QUEUE_CONNECTION=redis`（详见 [生产清单](/deploy/production) §4.1）
- [ ] 平台库 + 租户库备份（`tenant:backup-all`）并有**异地**副本
- [ ] 对照 `app/production/warn.go` 启动 Warning 清零或已接受风险

### 测试与工程质量

- [ ] CI unit gate + 关键 feature（含租户隔离、provision 门禁、权限/模块开关）通过
- [ ] 新开户：`pending` → 平台 UI 异步迁移（或 `tenant:migrate`）→ `ready` 后再放业务流量；生产有 `long-running` worker
