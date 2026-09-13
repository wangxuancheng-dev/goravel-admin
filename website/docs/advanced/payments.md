# 支付 / 订单参考实现

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
| 网关注册表 | `app/payment/driver.go` | `RegisterGateway` / `LookupGateway`（services 有同名别名） |
| 网关实现 | `app/payment/gateways/*.go` | 各渠道 Create/Query/Notify（返回 `PaidResult`，不落库） |
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

在 `app/payment/gateways/wechat.go` / `alipay.go` 的 `Notify` / `Query` 中：

1. gopay 验签 / 查询  
2. 映射为 `payment.PaidResult{PaymentNo, ThirdPartyNo, PayTime, Amount, NotifyData}` 并 **return**  
3. `PaymentGatewayService.HandlePaymentNotify` 会调用 `ApplyPaidResult` — **不要**在驱动里直接改订单  

下单示例已在对应 Driver 的 `Create`（需真实商户配置）。

## 6. 接入全新渠道 / 多支付平台

**结论**：渠道驱动放在 `app/payment/gateways/`（一文件一渠道），**不要**堆在 `app/services`，也不要为每个渠道建 `app/stripe` 域包。注册表与 `PaidResult` 在 `app/payment`；落库编排仍在 services。回调路由已通用：`POST /api/payment/notify/{type}[/{tenant}]`。

### 6.1 每个渠道一份驱动

1. 新建例如 `app/payment/gateways/stripe.go`：

```go
package gateways

import "goravel/app/payment"

func init() {
    payment.RegisterGateway(&stripeDriver{})
}

type stripeDriver struct{}

func (d *stripeDriver) Type() string { return "stripe" }

func (d *stripeDriver) Create(ctx context.Context, pay *models.Payment, method *models.PaymentMethod, config map[string]any, clientIP string) (map[string]any, error) {
    // 调第三方下单，notify_url 可用 payment.DefaultNotifyURL(ctx, "stripe")
    return map[string]any{"payment_no": pay.PaymentNo}, nil
}

func (d *stripeDriver) Query(ctx context.Context, pay *models.Payment, method *models.PaymentMethod, config map[string]any) (map[string]any, error) {
    return nil, apperrors.ErrPaymentGatewayNotImplemented
}

func (d *stripeDriver) Notify(ctx context.Context, method *models.PaymentMethod, notifyData map[string]any) (*payment.PaidResult, error) {
    // 验签 → 返回 PaidResult；services 负责 ApplyPaidResult
    return &payment.PaidResult{PaymentNo: "...", ThirdPartyNo: "..."}, nil
}
```

确保 `app/services` 已 blank-import `goravel/app/payment/gateways`（见 `payment_gateway_boot.go`），新文件的 `init` 会自动注册。

2. 后台支付方式 `type` 与 `Type()` 同一字符串（如 `stripe`）
3. 把新 type 写入 `PAYMENT_GATEWAYS_ENABLED`（生产务必白名单，不要长期空 / `*`）
4. 前端（可选）：在 Vue/React 的 `PAYMENT_METHOD_TYPES` + `PAYMENT_TYPE_CONFIG_FIELDS` 增加同名项与 i18n（`payment_method.type_<name>`）；未配置字段时仍可出现在下拉（来自 `config.payment_gateways`），但表单无专用字段。只注册已实现的驱动类型。

已注册类型可用 `RegisteredPaymentGatewayTypes()` / `/api/admin/info` 的 `payment_gateways` 查看。参考：`app/payment/gateways/mock.go`。

### 6.1.1 两种实现方式（都支持）

驱动只负责实现 `PaymentGatewayDriver`；**怎么调第三方不限**，以下两种都是一等公民：

| 方式 | 何时用 | 做法 | 仓库内参考 |
|------|--------|------|------------|
| **外部 Go 模块 / SDK** | 有稳定官方或社区包（如 Stripe、gopay） | `go get` 写入 `go.mod`，在 `app/payment/gateways/<type>.go` 里调 SDK；**不要**再包一层 `app/<vendor>` 域包 | `gateways/wechat.go` / `alipay.go`（gopay） |
| **按文档手写** | 只有 HTTP/签名文档、无可靠 Go 包，或包过重不值得引 | 同文件内用 `net/http` + `crypto` 等自实现验签/下单/查单，返回 `PaidResult` | `gateways/mock.go` |

共同点：

- 注册、路由、落库路径相同（`payment.RegisterGateway` + `notify/{type}` + services `ApplyPaidResult`）
- 驱动 **禁止** import `app/services`（避免循环依赖）；金额回查等通过 `payment.ResolvePaymentAmount` 钩子由 services 注入
- 密钥 / 商户号放在 `payment_methods.config` JSON，不要硬编码

### 6.2 明确不做

| 项 | 原因 |
|----|------|
| 把几十个驱动平铺进 `app/services` | 与 CRUD 搅在一起；驱动已迁至 `app/payment/gateways` |
| 每渠道一个 `app/<vendor>` 包 | 与 [服务代码组织](/guide/service-packages) 约定不符；第三方用 **go.mod 依赖** 或驱动内手写即可 |
| 再引入 PaymentStore / OrderStore ports | 已删除未完成稿；编排留 services |
| 为新渠道改 routes / 复制 notify controller | 路由已按 `{type}` 分发 |
| 把 `OrderService` / `PaymentService` 整包迁出 | 订单/支付编排应留在 `app/services` |
| 强制所有渠道必须用同一 SDK | 不要求；SDK 与手写可并存 |
| 驱动内直接 `ApplyPaidResult` / 改订单 | 落库只在 services；Notify 只返回 `PaidResult` |

### 6.3 目录约定

| 路径 | 职责 |
|------|------|
| `app/payment/gateways/` | 所有渠道驱动（可扩展到几十个） |
| `app/payment/driver.go` | 注册表接口 |
| `app/services/payment_apply.go` | `ApplyPaidResult` 编排 |
| `app/services/payment_gateway_*.go` | 白名单、Service 门面、blank-import 驱动包 |

### 6.4 实现纪律

- **幂等**：重复回调依赖 `ApplyPaidResult` 状态闸门；驱动不要自己写 paid
- **金额**：能拿到实付金额就填 `PaidResult.Amount`，走现有校验
- **租户**：SaaS 回调必须带 `{tenant}`；`payment.DefaultNotifyURL` 已按租户拼 path
- **依赖**：可用外部仓库包（`go.mod`），也可纯文档手写 HTTP/验签；注册入口仍统一 `payment.RegisterGateway`（见 §6.1.1）

## 7. 环境与模块

| 变量 / 开关 | 说明 |
|-------------|------|
| `MODULE_PAYMENTS_ENABLED` | 管理端支付菜单与 API；**不影响**公开 notify |
| `PAYMENT_GATEWAYS_ENABLED` | 启用的网关类型（逗号分隔），如 `wechat,alipay`。空 / `*` / `all` = 全部已注册驱动。未列入的类型：不可创建支付方式、不可下单/查询/回调。**生产建议显式白名单** |
| `APP_URL` | 拼默认 `notify_url` |
| 多租户 | 回调必须带 `{tenant}`；见 `tenancy.PaymentNotifyPath` |

管理端 `/api/admin/info` 的 `config.payment_gateways` 会返回当前启用列表，前端支付方式「类型」下拉按此过滤。
