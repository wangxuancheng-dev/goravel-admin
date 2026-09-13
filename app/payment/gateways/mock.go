// Package gateways holds PaymentGatewayDriver implementations (one file per channel).
// Register via payment.RegisterGateway in init(). Do not import goravel/app/services.
package gateways

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/payment"
)

func init() {
	payment.RegisterGateway(&mockDriver{})
}

// mockDriver: hand-written reference (no third-party SDK).
// Prefer this style when the provider only publishes HTTP/signing docs.
// Config: shared_secret (optional), notify_url (optional).
type mockDriver struct{}

func (d *mockDriver) Type() string { return "mock" }

func (d *mockDriver) Create(ctx context.Context, pay *models.Payment, _ *models.PaymentMethod, config map[string]any, _ string) (map[string]any, error) {
	notifyURL := payment.DefaultNotifyURL(ctx, "mock")
	if v, _ := config["notify_url"].(string); strings.TrimSpace(v) != "" {
		notifyURL = strings.TrimSpace(v)
	}
	out := map[string]any{
		"payment_no":   pay.PaymentNo,
		"gateway":      "mock",
		"amount":       pay.Amount,
		"notify_url":   notifyURL,
		"trade_status": "SUCCESS",
		"hint":         "POST notify_url with out_trade_no + trade_status=SUCCESS (+ sign when shared_secret is set)",
	}
	if secret := strings.TrimSpace(payment.StringFromConfig(config, "shared_secret")); secret != "" {
		out["sign"] = mockNotifySign(pay.PaymentNo, "SUCCESS", pay.Amount, secret)
	}
	return out, nil
}

func (d *mockDriver) Query(_ context.Context, pay *models.Payment, _ *models.PaymentMethod, _ map[string]any) (map[string]any, error) {
	tradeState := "NOTPAY"
	switch pay.Status {
	case models.PaymentStatusPaid:
		tradeState = "SUCCESS"
	case models.PaymentStatusFailed:
		tradeState = "CLOSED"
	case models.PaymentStatusCancelled:
		tradeState = "REVOKED"
	}
	out := map[string]any{
		"payment_no":     pay.PaymentNo,
		"gateway":        "mock",
		"status":         pay.Status,
		"trade_state":    tradeState,
		"third_party_no": pay.ThirdPartyNo,
		"amount":         pay.Amount,
	}
	if pay.PayTime != nil {
		out["pay_time"] = pay.PayTime.Format(time.RFC3339)
	}
	return out, nil
}

func (d *mockDriver) Notify(ctx context.Context, paymentMethod *models.PaymentMethod, notifyData map[string]any) (*payment.PaidResult, error) {
	paymentNo := payment.FirstString(notifyData, "out_trade_no", "payment_no")
	if paymentNo == "" {
		return nil, apperrors.ErrPaymentNotifyInvalid.WithMessage("out_trade_no is required")
	}
	tradeStatus := strings.ToUpper(payment.FirstString(notifyData, "trade_status", "result_code", "status"))
	if tradeStatus == "" {
		tradeStatus = "SUCCESS"
	}
	if tradeStatus != "SUCCESS" && tradeStatus != "PAID" && tradeStatus != "TRADE_SUCCESS" {
		return nil, apperrors.ErrPaymentNotifyInvalid.WithMessage("trade_status is not success")
	}

	config, err := payment.ParseMethodConfig(paymentMethod)
	if err != nil {
		return nil, err
	}
	secret := strings.TrimSpace(payment.StringFromConfig(config, "shared_secret"))
	amount := payment.OptionalFloat(notifyData, "amount", "total_amount", "pay_amount")
	if secret != "" {
		sign := payment.FirstString(notifyData, "sign", "signature")
		expectedAmount := 0.0
		if amount != nil {
			expectedAmount = *amount
		} else if payment.ResolvePaymentAmount != nil {
			expectedAmount, err = payment.ResolvePaymentAmount(ctx, paymentNo)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, apperrors.ErrPaymentNotifyInvalid.WithMessage("amount required when shared_secret is set")
		}
		if !hmacEqual(sign, mockNotifySign(paymentNo, "SUCCESS", expectedAmount, secret)) {
			return nil, apperrors.ErrPaymentNotifyInvalid.WithMessage("invalid mock notify sign")
		}
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
