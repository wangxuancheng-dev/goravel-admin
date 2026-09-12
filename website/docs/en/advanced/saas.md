# SaaS checklist

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/advanced/saas).

---

面向**公网可服务的中小 SaaS / 运营后台**（一户一库可选、非金融收单）。  
对照维度目标：**各项 ≥ 8.5 / 10**。详细运维步骤见 [生产清单](/deploy/production)。

## 1. 范围约定（功能完整度）

| 纳入评分 | 明确不纳入（默认关闭或骨架） |
|----------|------------------------------|
| 管理端 RBAC、菜单、日志、导出、代码生成 | 微信/支付宝真实验签与退款（用 mock 参考链路） |
| 一户一库 / Schema、provision 门禁、平台控制台 | 多活异地机房、金融级审计 |
| `/health` `/ready`、生产不安全默认告警 | 现成第三方告警面板 SaaS |
| 搜索扩展点（ES / Meili，订单索引可选） | 业务全文检索开箱即用 |
| 订单+支付 mock：`ApplyPaidResult` 幂等落库 | 生产收单 / 清算 |

支付边界见 [开源定位](/guide/opensource) §1.1、[支付参考](/advanced/payments)。

## 2. 上线前核对

### 安全（目标 ≥ 8.5）

- [ ] `APP_DEBUG=false`，强 `APP_KEY` / `JWT_SECRET`
- [ ] 全局安全头已启用（`SecurityHeaders`）；HTTPS 后设 `SECURITY_HSTS_MAX_AGE=31536000`
- [ ] 一户一库公网：`TENANCY_RESOLVER=subdomain`，`TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`
- [ ] 子域名与 Header 冲突返回 `tenant_hint_conflict`；未 migrate 返回 `tenant_not_ready`（403）
- [ ] 关闭 Swagger / 代码生成器 / pprof；默认管理员已改密

### 可运维（目标 ≥ 8.5）

- [ ] LB：`GET /health` 存活、`GET /ready` 就绪（含 DB；按需 Redis/Search）
- [ ] `/health` 可带 `app` / `env` / `version`（`APP_VERSION`）
- [ ] Web + Queue Worker 分离；`CACHE_STORE=redis`、`QUEUE_CONNECTION=redis`
- [ ] 平台库 + 租户库备份（`tenant:backup-all`）并有**异地**副本
- [ ] 对照 `app/production/warn.go` 启动 Warning 清零或已接受风险

### 测试与工程质量（目标 ≥ 8.5）

- [ ] CI unit gate + 关键 feature（含租户隔离、provision 门禁、权限/模块开关）通过
- [ ] 新开户：`pending` → `tenant:migrate` → `ready` 后再放业务流量

## 3. 证据索引

| 维度 | 主要证据 |
|------|----------|
| 安全 | `app/http/middleware/security_headers.go`、`tenant.go`、`app/production/warn.go` |
| 运维 | `app/health/`、[生产清单](/deploy/production)、本文件 |
| 多租户 | [多租户](/advanced/tenancy)、`tests/feature/tenant_*` |
| 扩展 | [搜索](/advanced/search)、代码生成器、模块开关 |
| 测试 | `tests/feature/`、`app/tenancy/tenancy_test.go`、`app/health/ready_test.go` |
