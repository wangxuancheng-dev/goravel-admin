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
| 网关注册表 | `app/services/payment_gateway_driver.go` | `RegisterPaymentGateway` / `LookupPaymentGateway` |
| 网关实现 | `payment_gateway_mock.go` 等 | 各渠道 Create/Query/Notify |
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

## 5. 接入微信 / 支付宝

在 `payment_gateway_wechat.go` / `payment_gateway_alipay.go` 的 `Notify` / `Query` 中：

1. gopay 验签 / 查询  
2. 映射为 `PaidResult{PaymentNo, ThirdPartyNo, PayTime, Amount, NotifyData}`  
3. `return ApplyPaidResult(ctx, result)` — **不要**在驱动里直接改订单  

下单示例已在对应 Driver 的 `Create`（需真实商户配置）。

## 6. New channels / many payment platforms

**Summary**: scale by adding drivers — do **not** create per-vendor `app/stripe`-style packages or revive `PaymentStore` / `OrderStore` ports. Notify routing is already generic: `POST /api/payment/notify/{type}[/{tenant}]` — **no route changes**.

### 6.1 One driver file per channel

1. Add e.g. `app/services/payment_gateway_stripe.go`:

```go
package services

func init() {
    RegisterPaymentGateway(&stripePaymentDriver{})
}

type stripePaymentDriver struct{}

func (d *stripePaymentDriver) Type() string { return "stripe" }

func (d *stripePaymentDriver) Create(ctx context.Context, payment *models.Payment, method *models.PaymentMethod, config map[string]any, clientIP string) (map[string]any, error) {
    // Call provider; notify_url via defaultPaymentNotifyURL(ctx, "stripe")
    return map[string]any{"payment_no": payment.PaymentNo}, nil
}

func (d *stripePaymentDriver) Query(ctx context.Context, payment *models.Payment, method *models.PaymentMethod, config map[string]any) (map[string]any, error) {
    return nil, apperrors.ErrPaymentGatewayNotImplemented
}

func (d *stripePaymentDriver) Notify(ctx context.Context, method *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
    // Verify → PaidResult → ApplyPaidResult(ctx, result)
    return ApplyPaidResult(ctx, PaidResult{PaymentNo: "...", ThirdPartyNo: "..."})
}
```

2. Admin `payment_methods.type` must match `Type()` (e.g. `stripe`)
3. Add the type to `PAYMENT_GATEWAYS_ENABLED` (production: explicit allowlist; avoid long-lived empty / `*`)
4. Frontend (optional): extend Vue/React `PAYMENT_METHOD_TYPES` + `PAYMENT_TYPE_CONFIG_FIELDS` and i18n (`payment_method.type_<name>`). Without field defs the type still appears in the dropdown from `config.payment_gateways`, but the form has no dedicated fields. Only register implemented drivers.

List types via `RegisteredPaymentGatewayTypes()` / `/api/admin/info` → `payment_gateways`. Reference: `payment_gateway_mock.go`.

### 6.1.1 Two implementation styles (both supported)

The driver only implements `PaymentGatewayDriver`. **How** you talk to the provider is unrestricted:

| Style | When | How | In-repo reference |
|-------|------|-----|-------------------|
| **External Go module / SDK** | Stable official or community package (Stripe, gopay, …) | `go get` into `go.mod`; call SDK from `payment_gateway_<type>.go`. Do **not** wrap in an `app/<vendor>` domain package | `payment_gateway_wechat.go` / `alipay` (gopay) |
| **Hand-written from docs** | HTTP/signing docs only, no reliable Go package, or SDK too heavy | Same file: `net/http` + `crypto` for create/query/verify; still map to `PaidResult` → `ApplyPaidResult` | `payment_gateway_mock.go` |

Shared rules:

- Same register / route / apply path (`RegisterPaymentGateway` + `notify/{type}` + `ApplyPaidResult`)
- Secrets live in `payment_methods.config` JSON
- Heavy SDKs are `require`d modules; hand-written logic stays in the driver (or small same-package helpers). **No** per-channel `app/stripe`-style domain packages

### 6.2 Explicitly do not

| Item | Why |
|------|-----|
| Per-vendor `app/<vendor>` package | Against [service packages](/en/guide/service-packages); use a **go.mod dependency** or hand-written driver logic |
| Revive PaymentStore / OrderStore ports | Unfinished draft removed; orchestration stays in services |
| New routes / notify controllers per channel | `{type}` already dispatches |
| Move entire `OrderService` / `PaymentService` | Outside frozen split |
| Force every channel onto one SDK | Not required; SDK and hand-written drivers may coexist |

### 6.3 When files proliferate

If `payment_gateway_*.go` exceeds ~8–10 files and clutters services, **move driver implementations only** to e.g. `app/payment/gateways/` (still `init` register); keep the registry and `ApplyPaidResult` **in services**. Today (mock / wechat / alipay) **do not move yet**.

### 6.4 Implementation discipline

- **Idempotency**: rely on `ApplyPaidResult` status gate; drivers must not write paid themselves
- **Amount**: set `PaidResult.Amount` when the provider returns it
- **Tenancy**: SaaS notify must include `{tenant}`; `defaultPaymentNotifyURL` already builds the path
- **Heavy SDKs or hand-written HTTP**: either is fine; registration stays `RegisterPaymentGateway` (see §6.1.1)

## 7. Environment / modules

| Variable | Notes |
|----------|-------|
| `MODULE_PAYMENTS_ENABLED` | Admin payment menu/API; does **not** affect public notify |
| `PAYMENT_GATEWAYS_ENABLED` | Comma-separated enabled types, e.g. `wechat,alipay`. Empty / `*` / `all` = all registered. Unlisted types cannot create methods or create/query/notify. **Prefer an explicit production allowlist** |
| `APP_URL` | Default `notify_url` base |
| Multi-tenant | Notify must include `{tenant}`; see `tenancy.PaymentNotifyPath` |

Admin `/api/admin/info` → `config.payment_gateways` drives the payment-method type dropdown.
