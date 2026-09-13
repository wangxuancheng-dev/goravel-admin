# Payments reference

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/advanced/payments).

---

订单 → 支付单 → 回调落库 → 订单已支付。能力边界见 [开源定位](/guide/opensource) §1.1。

## 1. 架构

```
CreatePayment (订单 pending)
    → CreatePaymentOrder (已注册 Driver)
    → 用户支付 / 本地模拟回调
    → POST /api/payment/notify/{type}[/{tenant}]
    → Driver.Notify（验签）
    → ApplyPaidResult（幂等：payment=paid + order=paid）
```

内置：`mock`（可跑通）、`wechat` / `alipay`（下单示例，回调/查询 501）。新渠道见下文「接入全新渠道」。
| 组件 | 路径 | 职责 |
|------|------|------|
| 订单 | `app/services/order_service.go` | 分表订单 CRUD；`UpdateOrderByOrderNo` |
| 支付记录 | `app/services/payment_service.go` | 分表支付单；创建时校验订单金额 |
| 落库编排 | `app/services/payment_apply.go` | **唯一**写成功态入口 `ApplyPaidResult` |
| 网关注册表 | `app/payment/driver.go` | `RegisterGateway` / `LookupGateway`（services aliases） |
| 网关实现 | `app/payment/gateways/*.go` | per-channel Create/Query/Notify（returns `PaidResult`, no DB writes） |
| 回调 | `payment_notify_controller.go` | `POST /api/payment/notify/{type}[/{tenant}]` |

## 2. 分表如何定位（回调一定找得到）

| 单号 | 格式 | 定位分表 |
|------|------|----------|
| `payment_no` | `PAY` + `YYYYMMDD` + ULID | `payments_YYYYMM` |
| `order_no` | `ORD` + `YYYYMM` + ULID | `orders_YYYYMM` |

回调流程：

1. `out_trade_no` = `payment_no` → `GetPaymentByPaymentNo` **直接定位**支付分表  
2. 取 `payment.order_no` → `GetOrderByOrderNo` **直接定位**订单分表  
3. `ApplyPaidResult` 幂等更新支付 + 订单  

跨月支付没问题：支付单创建在 2 月、3 月回调，仍用 2 月的 `payment_no` / `order_no` 定位原分表。

## 3. 完成时间与时区

- **存储**：`pay_time` 统一 **UTC** 写入  
- **展示**：请求带 `X-Timezone`（前端已自动带）时，响应里的 `pay_time` / `created_at` / `updated_at` 会转到该时区显示（`APP_RESPONSE_TIME_FIELDS` 已含 `pay_time`）  
- **筛选**：`start_time` / `end_time` 按请求时区解释后转 UTC 查库；`time_field=pay_time` 按**支付完成时间**筛，默认 `created_at`。按 `pay_time` 筛时会多扫前一个月分表，避免「上月建单、本月支付」漏查  

## 4. Mock 快速跑通

1. 后台创建支付方式：`type=mock`，可选 `shared_secret`
2. 创建待支付订单（管理端订单）
3. `POST /api/admin/payments`（需 `MODULE_PAYMENTS_ENABLED=true` + `payment.store`）：

```json
{
  "order_no": "ORD...",
  "payment_method_id": 1,
  "initiate": true
}
```

4. 模拟回调（单库）：

```bash
curl -X POST http://127.0.0.1:3000/api/payment/notify/mock \
  -H 'Content-Type: application/json' \
  -d '{"out_trade_no":"PAY...","trade_status":"SUCCESS"}'
```

多租户：`POST /api/payment/notify/mock/{tenant_code}`。

若配置了 `shared_secret`，需附带：

`sign = HMAC-SHA256(hex, out_trade_no + "|" + "SUCCESS" + "|" + amount(%.2f), secret)`

5. 查询：`POST /api/admin/payments/{payment_no}/query`（mock 读本地库状态）

## 5. WeChat / Alipay

In `app/payment/gateways/wechat.go` / `alipay.go` `Notify` / `Query`:

1. gopay verify / query  
2. Map to `payment.PaidResult{...}` and **return** it  
3. `PaymentGatewayService.HandlePaymentNotify` calls `ApplyPaidResult` — **do not** mutate orders in the driver  

Create examples already exist in those drivers (need real merchant config).

## 6. New channels / many payment platforms

**Summary**: put drivers in `app/payment/gateways/` (one file per channel). Do **not** pile them into `app/services`, and do **not** create per-vendor `app/stripe` packages. Registry + `PaidResult` live in `app/payment`; apply orchestration stays in services. Notify routing is already generic.

### 6.1 One driver file per channel

1. Add e.g. `app/payment/gateways/stripe.go`:

```go
package gateways

import "goravel/app/payment"

func init() {
    payment.RegisterGateway(&stripeDriver{})
}

type stripeDriver struct{}

func (d *stripeDriver) Type() string { return "stripe" }

func (d *stripeDriver) Create(ctx context.Context, pay *models.Payment, method *models.PaymentMethod, config map[string]any, clientIP string) (map[string]any, error) {
    // Call provider; notify_url via payment.DefaultNotifyURL(ctx, "stripe")
    return map[string]any{"payment_no": pay.PaymentNo}, nil
}

func (d *stripeDriver) Query(ctx context.Context, pay *models.Payment, method *models.PaymentMethod, config map[string]any) (map[string]any, error) {
    return nil, apperrors.ErrPaymentGatewayNotImplemented
}

func (d *stripeDriver) Notify(ctx context.Context, method *models.PaymentMethod, notifyData map[string]any) (*payment.PaidResult, error) {
    // Verify → return PaidResult; services ApplyPaidResult
    return &payment.PaidResult{PaymentNo: "...", ThirdPartyNo: "..."}, nil
}
```

`app/services` blank-imports `goravel/app/payment/gateways` (`payment_gateway_boot.go`) so new `init` registrations load automatically.

2. Admin `payment_methods.type` must match `Type()` (e.g. `stripe`)
3. Add the type to `PAYMENT_GATEWAYS_ENABLED` (production: explicit allowlist)
4. Frontend (optional): extend `PAYMENT_METHOD_TYPES` + `PAYMENT_TYPE_CONFIG_FIELDS` + i18n

List types via `RegisteredPaymentGatewayTypes()` / `/api/admin/info` → `payment_gateways`. Reference: `app/payment/gateways/mock.go`.

### 6.1.1 Two implementation styles (both supported)

| Style | When | How | In-repo reference |
|-------|------|-----|-------------------|
| **External Go module / SDK** | Stable package (Stripe, gopay, …) | `go get` into `go.mod`; call from `gateways/<type>.go` | `gateways/wechat.go` / `alipay.go` |
| **Hand-written from docs** | Docs only / no good Go package | `net/http` + crypto in the same file; return `PaidResult` | `gateways/mock.go` |

Shared rules:

- Same register / route / apply path (`payment.RegisterGateway` + `notify/{type}` + services `ApplyPaidResult`)
- Drivers must **not** import `app/services` (no cycles); amount lookup uses `payment.ResolvePaymentAmount` wired by services
- Secrets in `payment_methods.config` JSON

### 6.2 Explicitly do not

| Item | Why |
|------|-----|
| Dozens of drivers under `app/services` | Clutters CRUD; drivers live in `app/payment/gateways` |
| Per-vendor `app/<vendor>` package | Against [service code layout](/en/guide/service-packages); use a go.mod dependency or hand-written driver |
| Revive PaymentStore / OrderStore ports | Orchestration stays in services |
| New routes / notify controllers per channel | `{type}` already dispatches |
| Move entire `OrderService` / `PaymentService` | Outside frozen split |
| Force one SDK for all channels | SDK and hand-written may coexist |
| Call `ApplyPaidResult` inside drivers | Notify returns `PaidResult` only |

### 6.3 Layout

| Path | Role |
|------|------|
| `app/payment/gateways/` | All channel drivers (scales to dozens) |
| `app/payment/driver.go` | Registry |
| `app/services/payment_apply.go` | `ApplyPaidResult` |
| `app/services/payment_gateway_*.go` | Allowlist, service facade, blank-import |

### 6.4 Implementation discipline

- **Idempotency**: rely on `ApplyPaidResult` status gate
- **Amount**: set `PaidResult.Amount` when available
- **Tenancy**: SaaS notify must include `{tenant}`; use `payment.DefaultNotifyURL`
- **Deps**: external modules or hand-written HTTP; register via `payment.RegisterGateway`

## 7. Environment / modules

| Variable | Notes |
|----------|-------|
| `MODULE_PAYMENTS_ENABLED` | Admin payment menu/API; does **not** affect public notify |
| `PAYMENT_GATEWAYS_ENABLED` | Comma-separated enabled types, e.g. `wechat,alipay`. Empty / `*` / `all` = all registered. Unlisted types cannot create methods or create/query/notify. **Prefer an explicit production allowlist** |
| `APP_URL` | Default `notify_url` base |
| Multi-tenant | Notify must include `{tenant}`; see `tenancy.PaymentNotifyPath` |

Admin `/api/admin/info` → `config.payment_gateways` drives the payment-method type dropdown.
