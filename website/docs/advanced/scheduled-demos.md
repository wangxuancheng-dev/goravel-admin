# 调度演示（活动窗口 + 订单超时）

> 开源示例，不是生产营销/收单中台。仅开发工具环境可调 HTTP（`DevelopmentOnly` / `APP_ENABLE_DEV_TOOL`）。

对照两种常见需求：

| 场景 | 做法 | 精度 |
|------|------|------|
| 活动 once / 每天时点 | 表 `demo_activities` + `activity:sync-status`（每 10 秒）+ 读路径 `IsActive` 现算 | 门禁约等于服务器时钟；`status` 字段最多约 10 秒滞后 |
| 订单到期取消 | `orders.expire_at` + Delay Job `cancel_expired_order` + `order:cancel-expired` 每分钟兜底 | 通常秒级；支付前会再按 `expire_at` 校验 |

## 1. 前置

1. `go run . artisan migrate`
2. 队列 Worker 在跑（Delay 才生效；`QUEUE_CONNECTION=sync` 时 Delay 行为依赖驱动）
3. Schedule runner 在跑（或手动 artisan）
4. 开发环境：`APP_ENV=local` 或 `APP_ENABLE_DEV_TOOL=true`
5. 多租户开启时请求需带租户（与其它租户 API 相同）

## 2. 活动窗口 API

管理端前缀：`/api/admin/demo-activities`（需登录；模块开关 `MODULE_SCHEDULE_DEMO_ENABLED`，默认 true）。

菜单：系统管理 → 活动调度演示。关闭模块后菜单与 API 均不可用。

开发态订单超时演示仍可用：`/api/schedule-demo/orders/expire`。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/demo-activities` | 列表 |
| POST | `/api/admin/demo-activities` | 创建 |
| GET | `/api/admin/demo-activities/{id}` | 详情 |
| PUT | `/api/admin/demo-activities/{id}` | 更新 |
| DELETE | `/api/admin/demo-activities/{id}` | 删除 |
| GET | `/api/admin/demo-activities/{id}/active-check` | 对比库 `status` 与现算 `is_active_live` |
| POST | `/api/admin/demo-activities/sync` | 立即扫库翻状态 |

### once 示例

```json
{
  "title": "flash sale",
  "schedule_type": "once",
  "start_at": "2026-09-20T10:00:00Z",
  "end_at": "2026-09-20T12:00:00Z",
  "timezone": "Asia/Shanghai",
  "enabled": true
}
```

`once` 窗口按 **UTC** 绝对时间比较（半开区间 `[start, end)`）。

\n### 调度类型\n\n| type | 含义 | 关键字段 |\n|------|------|----------|\n| once | 指定绝对时间段 | start_at / end_at (UTC) |\n| daily | 每天固定时点 | daily_start / daily_end (HH:MM) |\n| weekly | 每周若干天 + 时点 | weekdays(1=Mon..7=Sun) + daily_* |\n| monthly | 每月若干号或日期区间 | month_days 或 month_day_start/end；可选 daily_* |\n| yearly | 每年日期区间 | year_start/year_end (MM-DD)；可选 daily_* |\n\n循环类（daily/weekly/monthly/yearly）窗外为 pending，窗内 running；once 过期后为 ended。\n### daily 示例

```json
{
  "title": "daily window",
  "schedule_type": "daily",
  "daily_start": "09:00",
  "daily_end": "18:00",
  "timezone": "Asia/Shanghai",
  "enabled": true
}
```

`daily_*` 按活动 `timezone` 的本地 `HH:MM`；支持跨天（如 `22:00`–`02:00`）。

定时：`activity:sync-status` → `EveryTenSeconds` + `OnOneServer`（见管理端「定时任务」列表）。

**业务门禁请用现算**（`DemoActivityIsActive` / `active-check` 的 `is_active_live`），不要只信扫库写的 `status`。

## 3. 订单超时取消

1. 创建 pending 订单时可选 `expire_in_seconds`（管理端下单 JSON）或调用演示接口。
2. 写入 `expire_at`，并 `Delay(expire_at)` 投递 `cancel_expired_order`。
3. Job / 扫库均幂等：仅 `pending` 且已到期才改为 `cancelled`；已支付跳过。
4. 创建支付时若已过期会先取消再拒绝（`order_not_payable`）。

### 一键演示

```http
POST /api/schedule-demo/orders/expire?seconds=60
```

可选 `user_id`、`amount`。默认取库中第一个用户、金额 `0.01`。

兜底：`go run . artisan order:cancel-expired`（Kernel 每分钟也会跑）。

## 4. 相关代码

| 能力 | 路径 |
|------|------|
| 活动模型/服务 | `app/models/demo_activity.go`、`app/services/demo_activity_service.go` |
| 活动命令 | `app/console/commands/sync_demo_activities.go` |
| 订单过期 | `app/services/order_expire.go`、`app/jobs/cancel_expired_order.go` |
| 关单命令 | `app/console/commands/cancel_expired_orders.go` |
| 演示路由 | admin `demo-activities` + `/api/schedule-demo/orders/expire` |

## 5. 边界说明

- 不是动态 cron 管理台；固定两条 Schedule + DB 规则。
- 演示站/二次开发样板；生产营销活动请按业务重做校验、库存与审计。