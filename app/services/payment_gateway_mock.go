package services

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
)

func init() {
	RegisterPaymentGateway(&mockPaymentDriver{})
}

// mockPaymentDriver: local reference for secondary developers — hand-written (no third-party SDK).
// Prefer this style when the provider only publishes HTTP/signing docs and no stable Go module.
// Config JSON keys (payment_methods.config):
//   - shared_secret (optional): notify must include sign = HMAC-SHA256(out_trade_no|trade_status|amount, secret)
//   - notify_url (optional): override default APP_URL + /api/payment/notify/mock[/{tenant}]
type mockPaymentDriver struct{}

func (d *mockPaymentDriver) Type() string { return "mock" }

func (d *mockPaymentDriver) Create(ctx context.Context, payment *models.Payment, _ *models.PaymentMethod, config map[string]any, _ string) (map[string]any, error) {
	notifyURL := defaultPaymentNotifyURL(ctx, "mock")
	if v, _ := config["notify_url"].(string); strings.TrimSpace(v) != "" {
		notifyURL = strings.TrimSpace(v)
	}
	out := map[string]any{
		"payment_no":   payment.PaymentNo,
		"gateway":      "mock",
		"amount":       payment.Amount,
		"notify_url":   notifyURL,
		"trade_status": "SUCCESS",
		"hint":         "POST notify_url with out_trade_no + trade_status=SUCCESS (+ sign when shared_secret is set)",
	}
	if secret := strings.TrimSpace(stringFromConfig(config, "shared_secret")); secret != "" {
		out["sign"] = mockNotifySign(payment.PaymentNo, "SUCCESS", payment.Amount, secret)
	}
	return out, nil
}

func (d *mockPaymentDriver) Query(_ context.Context, payment *models.Payment, _ *models.PaymentMethod, _ map[string]any) (map[string]any, error) {
	tradeState := "NOTPAY"
	switch payment.Status {
	case models.PaymentStatusPaid:
		tradeState = "SUCCESS"
	case models.PaymentStatusFailed:
		tradeState = "CLOSED"
	case models.PaymentStatusCancelled:
		tradeState = "REVOKED"
	}
	out := map[string]any{
		"payment_no":     payment.PaymentNo,
		"gateway":        "mock",
		"status":         payment.Status,
		"trade_state":    tradeState,
		"third_party_no": payment.ThirdPartyNo,
		"amount":         payment.Amount,
	}
	if payment.PayTime != nil {
		out["pay_time"] = payment.PayTime.Format(time.RFC3339)
	}
	return out, nil
}

func (d *mockPaymentDriver) Notify(ctx context.Context, paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	paymentNo := firstString(notifyData, "out_trade_no", "payment_no")
	if paymentNo == "" {
		return nil, apperrors.ErrPaymentNotifyInvalid.WithMessage("out_trade_no is required")
	}
	tradeStatus := strings.ToUpper(firstString(notifyData, "trade_status", "result_code", "status"))
	if tradeStatus == "" {
		tradeStatus = "SUCCESS"
	}
	if tradeStatus != "SUCCESS" && tradeStatus != "PAID" && tradeStatus != "TRADE_SUCCESS" {
		return nil, apperrors.ErrPaymentNotifyInvalid.WithMessage("trade_status is not success")
	}

	config, err := parsePaymentMethodConfig(paymentMethod)
	if err != nil {
		return nil, err
	}
	secret := strings.TrimSpace(stringFromConfig(config, "shared_secret"))
	amount := optionalFloat(notifyData, "amount", "total_amount", "pay_amount")
	if secret != "" {
		sign := firstString(notifyData, "sign", "signature")
		expectedAmount := 0.0
		if amount != nil {
			expectedAmount = *amount
		} else {
			payment, err := NewPaymentService(ctx).GetPaymentByPaymentNo(paymentNo)
			if err != nil {
				return nil, err
			}
			expectedAmount = payment.Amount
		}
		if !hmacEqual(sign, mockNotifySign(paymentNo, "SUCCESS", expectedAmount, secret)) {
			return nil, apperrors.ErrPaymentNotifyInvalid.WithMessage("invalid mock notify sign")
		}
	}

	thirdPartyNo := firstString(notifyData, "transaction_id", "trade_no", "third_party_no")
	if thirdPartyNo == "" {
		thirdPartyNo = "MOCK-" + paymentNo
	}
	now := time.Now()
	return ApplyPaidResult(ctx, PaidResult{
		PaymentNo:    paymentNo,
		ThirdPartyNo: thirdPartyNo,
		PayTime:      &now,
		Amount:       amount,
		NotifyData:   notifyData,
	})
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
