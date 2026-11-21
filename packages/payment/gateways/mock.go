// Package gateways holds payment.GatewayDriver implementations (one file per channel).
// Register via payment.RegisterGateway in init(). Do not import host app/services.
package gateways

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"goravel-payment"
)

func init() {
	payment.RegisterGateway(&mockDriver{})
}

// mockDriver: hand-written reference (no third-party SDK).
// Prefer this style when the provider only publishes HTTP/signing docs.
// Config: shared_secret (required for notify), notify_url (optional).
type mockDriver struct{}

func (d *mockDriver) Type() string { return "mock" }

func (d *mockDriver) Create(ctx context.Context, order *payment.Order, config map[string]any, _ string) (map[string]any, error) {
	if order == nil {
		return nil, fmt.Errorf("%w: order is nil", payment.ErrCreateFailed)
	}
	if config == nil {
		config = map[string]any{}
	}
	notifyURL := payment.DefaultNotifyURL(ctx, "mock")
	if v, _ := config["notify_url"].(string); strings.TrimSpace(v) != "" {
		notifyURL = strings.TrimSpace(v)
	}
	secret := strings.TrimSpace(payment.StringFromConfig(config, "shared_secret"))
	if secret == "" {
		return nil, fmt.Errorf("%w: shared_secret is required for mock gateway", payment.ErrConfigRequired)
	}
	out := map[string]any{
		"payment_no":   order.PaymentNo,
		"gateway":      "mock",
		"amount":       order.Amount,
		"notify_url":   notifyURL,
		"trade_status": "SUCCESS",
		"sign":         mockNotifySign(order.PaymentNo, "SUCCESS", order.Amount, secret),
		"hint":         "POST notify_url with out_trade_no, trade_status=SUCCESS, amount, and sign (HMAC-SHA256)",
	}
	return out, nil
}

func (d *mockDriver) Query(_ context.Context, order *payment.Order, _ map[string]any) (map[string]any, error) {
	if order == nil {
		return nil, fmt.Errorf("%w: order is nil", payment.ErrCreateFailed)
	}
	tradeState := "NOTPAY"
	switch order.Status {
	case payment.StatusPaid:
		tradeState = "SUCCESS"
	case payment.StatusFailed:
		tradeState = "CLOSED"
	case payment.StatusCancelled:
		tradeState = "REVOKED"
	}
	out := map[string]any{
		"payment_no":     order.PaymentNo,
		"gateway":        "mock",
		"status":         order.Status,
		"trade_state":    tradeState,
		"third_party_no": order.ThirdPartyNo,
		"amount":         order.Amount,
	}
	if order.PayTime != nil {
		out["pay_time"] = order.PayTime.Format(time.RFC3339)
	}
	return out, nil
}

func (d *mockDriver) Notify(ctx context.Context, config map[string]any, notifyData map[string]any) (*payment.PaidResult, error) {
	paymentNo := payment.FirstString(notifyData, "out_trade_no", "payment_no")
	if paymentNo == "" {
		return nil, fmt.Errorf("%w: out_trade_no is required", payment.ErrNotifyInvalid)
	}
	tradeStatus := strings.ToUpper(payment.FirstString(notifyData, "trade_status", "result_code", "status"))
	if tradeStatus == "" {
		tradeStatus = "SUCCESS"
	}
	if tradeStatus != "SUCCESS" && tradeStatus != "PAID" && tradeStatus != "TRADE_SUCCESS" {
		return nil, fmt.Errorf("%w: trade_status is not success", payment.ErrNotifyInvalid)
	}

	secret := strings.TrimSpace(payment.StringFromConfig(config, "shared_secret"))
	if secret == "" {
		return nil, fmt.Errorf("%w: shared_secret is required for mock notify", payment.ErrNotifyInvalid)
	}
	amount := payment.OptionalFloat(notifyData, "amount", "total_amount", "pay_amount")
	sign := payment.FirstString(notifyData, "sign", "signature")
	expectedAmount := 0.0
	var err error
	if amount != nil {
		expectedAmount = *amount
	} else if payment.ResolvePaymentAmount != nil {
		expectedAmount, err = payment.ResolvePaymentAmount(ctx, paymentNo)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("%w: amount required for mock notify sign", payment.ErrNotifyInvalid)
	}
	if !hmacEqual(sign, mockNotifySign(paymentNo, "SUCCESS", expectedAmount, secret)) {
		return nil, fmt.Errorf("%w: invalid mock notify sign", payment.ErrNotifyInvalid)
	}

	thirdPartyNo := payment.FirstString(notifyData, "transaction_id", "trade_no", "third_party_no")
	if thirdPartyNo == "" {
		thirdPartyNo = "MOCK-" + paymentNo
	}
	now := time.Now()
	return &payment.PaidResult{
		PaymentNo:    paymentNo,
		ThirdPartyNo: thirdPartyNo,
		PayTime:      &now,
		Amount:       amount,
		NotifyData:   notifyData,
	}, nil
}

// MockNotifySign returns the mock gateway's notify signature:
// hex(HMAC-SHA256(secret, "paymentNo|tradeStatus|amount(%.2f)")).
// Exported so CLIs / tests build callbacks with the exact algorithm Notify verifies.
func MockNotifySign(paymentNo, tradeStatus string, amount float64, secret string) string {
	return mockNotifySign(paymentNo, tradeStatus, amount, secret)
}

func mockNotifySign(paymentNo, tradeStatus string, amount float64, secret string) string {
	payload := fmt.Sprintf("%s|%s|%.2f", paymentNo, tradeStatus, amount)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func hmacEqual(got, want string) bool {
	got = strings.TrimSpace(strings.ToLower(got))
	want = strings.TrimSpace(strings.ToLower(want))
	if got == "" || want == "" {
		return false
	}
	return hmac.Equal([]byte(got), []byte(want))
}
