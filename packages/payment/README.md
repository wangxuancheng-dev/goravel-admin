# goravel-payment

Reusable Goravel payment **gateway core**: registry, `PaidResult`, helpers, and built-in drivers (`mock` / `wechat` / `alipay`).

Host apps own orders/payments DB writes, notify HTTP routes, and menus. This package only implements Create / Query / Notify drivers.

## Install

### Option A — copy directory

Copy `packages/payment` into your Goravel project, then in your root `go.mod`:

```go
require goravel-payment v0.0.0
replace goravel-payment => ./packages/payment
```

### Option B — local / published module

```bash
# local path
go mod edit -require=goravel-payment@v0.0.0
go mod edit -replace=goravel-payment=../path/to/payment

# or after you publish this module under your own module path, go get that path
```

Run `go mod tidy` in `packages/payment` (and in the host) after copy.

## Wire in a Goravel host

```go
import (
    "context"
    "strings"

    "github.com/goravel/framework/facades"

    "goravel-payment"
    _ "goravel-payment/gateways" // mock, wechat, alipay
)

func Boot() {
    payment.ResolvePaymentAmount = func(ctx context.Context, paymentNo string) (float64, error) {
        // look up your payment row amount
        return 0, nil
    }
    payment.ResolveNotifyURL = func(ctx context.Context, gatewayType string) string {
        base := strings.TrimRight(facades.Config().GetString("app.url"), "/")
        return base + "/api/payment/notify/" + gatewayType
    }
}
```

`ResolveNotifyURL` is required for `wechat` / `alipay`: unless the method config sets `notify_url`, their `Create` fails with `ErrConfigRequired` when it is not wired (see `payment.RequireNotifyURL`). Run this module's own tests in CI too (`cd packages/payment && go test ./...`) — a root `go list ./...` skips nested modules.

Call drivers:

```go
order := &payment.Order{PaymentNo: "PAY...", Amount: 9.9, Status: payment.StatusPending}
cfg, _ := payment.ParseConfigJSON(`{"shared_secret":"..."}`)
driver, _ := payment.LookupGateway("mock")
out, err := driver.Create(ctx, order, cfg, clientIP)
result, err := driver.Notify(ctx, cfg, notifyBody)
// then persist result in your app (orders/payments)
```

## Built-in gateways

| Type | Style | Notes |
|------|--------|------|
| `mock` | Hand-written HMAC | Full create / query / notify |
| `wechat` | `gopay` SDK | Create sample; query/notify NotImplemented |
| `alipay` | `gopay` SDK | Create sample; query/notify NotImplemented |

Add a channel: new file under `gateways/`, implement `payment.GatewayDriver`, `RegisterGateway` in `init()`.

## What stays in the host

- Payment / order models and migrations
- `ApplyPaidResult` (or equivalent) DB orchestration
- `POST /api/payment/notify/{type}` controller
- Allowlist env (optional), admin CRUD, frontend

## Host CLI smoke (this repo)

Driver-only (no DB write). Pass config JSON yourself — no admin UI required:

```bash
# default --config already has shared_secret=cli-secret (mock)
go run . artisan payment:try roundtrip --type=mock --amount=9.9
go run . artisan payment:try create --type=mock --amount=0.01
go run . artisan payment:try notify --type=mock --payment-no=PAY1 --amount=9.9
go run . artisan payment:try query --type=mock --payment-no=PAY1 --status=paid --amount=9.9

# custom config: prefer a JSON file on Windows PowerShell
# echo {"shared_secret":"secret","app_id":"..."} > tmp/pay.json
go run . artisan payment:try create --type=wechat --config-file=tmp/pay.json
```

Actions: `create` | `query` | `notify` | `roundtrip`. Driver-only (no DB). Non-mock notify needs `--notify=...` JSON.
