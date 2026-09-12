# 支付 / 订单参考实现（二次开发）

本仓库**不是**生产收单系统。目标是提供一条可跑通的「订单 → 支付单 → 回调落库 → 订单已支付」参考链路，方便在此基础上接入微信 / 支付宝。

边界总览见 [OPENSOURCE.md](./OPENSOURCE.md) §1.1。

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

## 6. 接入全新渠道（推荐）

回调路由已通用：`POST /api/payment/notify/{type}[/{tenant}]`，**不必再改 routes**。

1. 新建例如 `app/services/payment_gateway_stripe.go`：

```go
package services

func init() {
    RegisterPaymentGateway(&stripePaymentDriver{})
}

type stripePaymentDriver struct{}

func (d *stripePaymentDriver) Type() string { return "stripe" }

func (d *stripePaymentDriver) Create(ctx context.Context, payment *models.Payment, method *models.PaymentMethod, config map[string]any, clientIP string) (map[string]any, error) {
    // 调第三方下单，notify_url 可用 defaultPaymentNotifyURL(ctx, "stripe")
    return map[string]any{"payment_no": payment.PaymentNo}, nil
}

func (d *stripePaymentDriver) Query(ctx context.Context, payment *models.Payment, method *models.PaymentMethod, config map[string]any) (map[string]any, error) {
    return nil, apperrors.ErrPaymentGatewayNotImplemented
}

func (d *stripePaymentDriver) Notify(ctx context.Context, method *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
    // 验签 → PaidResult → ApplyPaidResult(ctx, result)
    return ApplyPaidResult(ctx, PaidResult{PaymentNo: "...", ThirdPartyNo: "..."})
}
```

2. 后台支付方式 `type` 填同一字符串（如 `stripe`）  
3. 前端（可选）：`PAYMENT_METHOD_TYPES` + 配置字段 + i18n，管理端才能选该类型  

已注册类型可用 `RegisteredPaymentGatewayTypes()` 查看。参考实现：`payment_gateway_mock.go`。

## 7. 环境与模块

| 变量 / 开关 | 说明 |
|-------------|------|
| `MODULE_PAYMENTS_ENABLED` | 管理端支付菜单与 API；**不影响**公开 notify |
| `PAYMENT_GATEWAYS_ENABLED` | 启用的网关类型（逗号分隔），如 `wechat,alipay`。空 / `*` / `all` = 全部已注册驱动。未列入的类型：不可创建支付方式、不可下单/查询/回调 |
| `APP_URL` | 拼默认 `notify_url` |
| 多租户 | 回调必须带 `{tenant}`；见 `tenancy.PaymentNotifyPath` |

管理端 `/api/admin/info` 的 `config.payment_gateways` 会返回当前启用列表，前端支付方式「类型」下拉按此过滤。

## 8. 明确不做

- 退款 / 部分退款 API
- 微信/支付宝真实验签与生产密钥管理
- C 端完整收银台 UI
- 金融级对账与分账
