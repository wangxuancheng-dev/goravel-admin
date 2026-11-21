package commands

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel-payment"
	_ "goravel-payment/gateways"
)

// PaymentTry exercises gateway Create / Query / Notify from the CLI (no admin UI).
//
// Examples:
//
//	go run . artisan payment:try roundtrip --type=mock --amount=9.9
//	go run . artisan payment:try create --type=mock --config-file=tmp/mock.json --amount=0.01
//	go run . artisan payment:try notify --type=mock --payment-no=PAY1 --amount=9.9
//	go run . artisan payment:try query --type=mock --payment-no=PAY1 --amount=9.9 --status=paid
type PaymentTry struct{}

func (receiver *PaymentTry) Signature() string {
	return "payment:try"
}

func (receiver *PaymentTry) Description() string {
	return "Try payment gateway Create/Query/Notify with CLI args (driver only, no DB write)"
}

func (receiver *PaymentTry) Extend() command.Extend {
	return command.Extend{
		Category: "payment",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "type",
				Aliases: []string{"t"},
				Value:   "mock",
				Usage:   "Gateway type (mock / wechat / alipay / custom)",
			},
			&command.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   `{"shared_secret":"cli-secret"}`,
				Usage:   "Gateway config JSON string (prefer --config-file on Windows shells)",
			},
			&command.StringFlag{
				Name:  "config-file",
				Value: "",
				Usage: "Path to gateway config JSON file (overrides --config when set)",
			},
			&command.StringFlag{
				Name:  "payment-no",
				Value: "",
				Usage: "Payment no (default: auto PAYCLI...)",
			},
			&command.StringFlag{
				Name:    "amount",
				Aliases: []string{"a"},
				Value:   "0.01",
				Usage:   "Order amount",
			},
			&command.StringFlag{
				Name:  "status",
				Value: payment.StatusPending,
				Usage: "Order status for query (pending/paid/failed/cancelled)",
			},
			&command.StringFlag{
				Name:  "remark",
				Value: "payment:try",
				Usage: "Order remark / subject",
			},
			&command.StringFlag{
				Name:  "client-ip",
				Value: "127.0.0.1",
				Usage: "Client IP passed to Create",
			},
			&command.StringFlag{
				Name:  "notify",
				Value: "",
				Usage: "Raw notify JSON body (notify action). Empty = build mock SUCCESS payload",
			},
			&command.StringFlag{
				Name:  "third-party-no",
				Value: "",
				Usage: "Optional third_party_no for query",
			},
		},
	}
}

func (receiver *PaymentTry) Handle(ctx console.Context) error {
	action := strings.ToLower(strings.TrimSpace(ctx.Argument(0)))
	if action == "" {
		action = "create"
	}
	switch action {
	case "create", "query", "notify", "roundtrip":
	default:
		return fmt.Errorf("unknown action %q (use create|query|notify|roundtrip)", action)
	}

	typ := payment.NormalizeGatewayType(ctx.Option("type"))
	driver, ok := payment.LookupGateway(typ)
	if !ok || driver == nil {
		return fmt.Errorf("gateway not registered: %s (known: %s)", typ, strings.Join(payment.RegisteredGatewayTypes(), ", "))
	}

	cfg, err := loadTryConfig(ctx)
	if err != nil {
		return err
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(ctx.Option("amount")), 64)
	if err != nil {
		return fmt.Errorf("invalid --amount: %w", err)
	}

	paymentNo := strings.TrimSpace(ctx.Option("payment-no"))
	if paymentNo == "" {
		paymentNo = fmt.Sprintf("PAYCLI%s", time.Now().UTC().Format("20060102150405.000000000"))
		paymentNo = strings.ReplaceAll(paymentNo, ".", "")
	}

	order := &payment.Order{
		PaymentNo:    paymentNo,
		Amount:       amount,
		Status:       strings.TrimSpace(ctx.Option("status")),
		Remark:       ctx.Option("remark"),
		ThirdPartyNo: strings.TrimSpace(ctx.Option("third-party-no")),
	}
	if order.Status == "" {
		order.Status = payment.StatusPending
	}

	bg := context.Background()
	clientIP := ctx.Option("client-ip")

	switch action {
	case "create":
		out, err := driver.Create(bg, order, cfg, clientIP)
		if err != nil {
			return err
		}
		return printJSON(ctx, map[string]any{
			"action":     "create",
			"type":       typ,
			"payment_no": paymentNo,
			"result":     out,
		})
	case "query":
		out, err := driver.Query(bg, order, cfg)
		if err != nil {
			return err
		}
		return printJSON(ctx, map[string]any{
			"action":     "query",
			"type":       typ,
			"payment_no": paymentNo,
			"result":     out,
		})
	case "notify":
		notifyData, err := buildNotifyPayload(ctx, typ, order, cfg)
		if err != nil {
			return err
		}
		result, err := driver.Notify(bg, cfg, notifyData)
		if err != nil {
			return err
		}
		return printJSON(ctx, map[string]any{
			"action":     "notify",
			"type":       typ,
			"payment_no": paymentNo,
			"notify":     notifyData,
			"result":     result,
		})
	case "roundtrip":
		created, err := driver.Create(bg, order, cfg, clientIP)
		if err != nil {
			return err
		}
		notifyData, err := notifyFromCreateResult(typ, order, cfg, created)
		if err != nil {
			return err
		}
		result, err := driver.Notify(bg, cfg, notifyData)
		if err != nil {
			return err
		}
		return printJSON(ctx, map[string]any{
			"action":     "roundtrip",
			"type":       typ,
			"payment_no": paymentNo,
			"create":     created,
			"notify":     notifyData,
			"result":     result,
		})
	}
	return nil
}

func loadTryConfig(ctx console.Context) (map[string]any, error) {
	path := strings.TrimSpace(ctx.Option("config-file"))
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read --config-file: %w", err)
		}
		return payment.ParseConfigJSON(string(raw))
	}
	return payment.ParseConfigJSON(ctx.Option("config"))
}

func buildNotifyPayload(ctx console.Context, typ string, order *payment.Order, cfg map[string]any) (map[string]any, error) {
	raw := strings.TrimSpace(ctx.Option("notify"))
	if raw != "" {
		var data map[string]any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			return nil, fmt.Errorf("invalid --notify JSON: %w", err)
		}
		return data, nil
	}
	if typ != "mock" {
		return nil, fmt.Errorf("--notify JSON is required for gateway %q (auto payload only for mock)", typ)
	}
	secret := payment.StringFromConfig(cfg, "shared_secret")
	if secret == "" {
		return nil, fmt.Errorf("mock notify needs shared_secret in --config or a full --notify JSON")
	}
	return map[string]any{
		"out_trade_no": order.PaymentNo,
		"trade_status": "SUCCESS",
		"amount":       order.Amount,
		"sign":         mockCLINotifySign(order.PaymentNo, "SUCCESS", order.Amount, secret),
	}, nil
}

func notifyFromCreateResult(typ string, order *payment.Order, cfg map[string]any, created map[string]any) (map[string]any, error) {
	if typ != "mock" {
		return nil, fmt.Errorf("roundtrip auto-notify is only implemented for mock (got %q); use notify with --notify JSON", typ)
	}
	secret := payment.StringFromConfig(cfg, "shared_secret")
	sign := payment.FirstString(created, "sign")
	if sign == "" && secret != "" {
		sign = mockCLINotifySign(order.PaymentNo, "SUCCESS", order.Amount, secret)
	}
	if sign == "" {
		return nil, fmt.Errorf("mock create did not return sign and shared_secret is empty")
	}
	amount := order.Amount
	if v := payment.OptionalFloat(created, "amount"); v != nil {
		amount = *v
	}
	return map[string]any{
		"out_trade_no": order.PaymentNo,
		"trade_status": "SUCCESS",
		"amount":       amount,
		"sign":         sign,
	}, nil
}

func mockCLINotifySign(paymentNo, tradeStatus string, amount float64, secret string) string {
	payload := fmt.Sprintf("%s|%s|%.2f", paymentNo, tradeStatus, amount)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func printJSON(ctx console.Context, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	ctx.Info(string(b))
	return nil
}
