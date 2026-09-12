# 支付 / 订单参考实现（二次开发）

本仓库**不是**生产收单系统。目标是提供一条可跑通的「订单 → 支付单 → 回调落库 → 订单已支付」参考链路，方便在此基础上接入微信 / 支付宝。

边界总览见 [OPENSOURCE.md](./OPENSOURCE.md) §1.1。

## 1. 架构

```
CreatePayment (订单 pending)
    → CreatePaymentOrder (gateway: mock|wechat|alipay)
    → 用户支付 / 本地模拟回调
    → POST /api/payment/notify/{type}[/{tenant}]
    → HandlePaymentNotify（验签 / mock 校验）
    → ApplyPaidResult（幂等：payment=paid + order=paid）
```

| 组件 | 路径 | 职责 |
|------|------|------|
| 订单 | `app/services/order_service.go` | 分表订单 CRUD；`UpdateOrderByOrderNo` |
| 支付记录 | `app/services/payment_service.go` | 分表支付单；创建时校验订单金额 |
| 落库编排 | `app/services/payment_apply.go` | **唯一**写成功态入口 `ApplyPaidResult` |
| 网关 | `app/services/payment_gateway_*.go` | 验签 + 归一化；mock 完整，微信/支付宝挂 TODO |
| 回调 | `app/http/controllers/api/payment_notify_controller.go` | 公开路由；按支付单解析 `PaymentMethod` |

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

## 3. 接入微信 / 支付宝

1. 在 `handleWechatNotify` / `handleAlipayNotify` 用 gopay 验签
2. 映射为 `PaidResult{PaymentNo, ThirdPartyNo, PayTime, Amount, NotifyData}`
3. `return ApplyPaidResult(ctx, result)` — **不要**在网关里直接改订单
4. `queryWechatPayment` / `queryAlipayPayment` 同样：查第三方成功后可调用 `ApplyPaidResult` 做对账

下单示例仍在 `createWechatPayment` / `createAlipayPayment`（需真实商户配置）；生产请关掉 demo、自备证书与密钥。

## 4. 环境与模块

| 变量 / 开关 | 说明 |
|-------------|------|
| `MODULE_PAYMENTS_ENABLED` | 管理端支付菜单与 API；**不影响**公开 notify |
| `APP_URL` | 拼默认 `notify_url` |
| 多租户 | 回调必须带 `{tenant}`；见 `tenancy.PaymentNotifyPath` |

## 5. 明确不做

- 退款 / 部分退款 API
- 微信/支付宝真实验签与生产密钥管理
- C 端完整收银台 UI
- 金融级对账与分账
