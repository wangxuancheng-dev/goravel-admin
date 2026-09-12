package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

// Mock gateway: local reference implementation for secondary developers.
// Config JSON keys (payment_methods.config):
//   - shared_secret (optional): when set, notify must include sign = HMAC-SHA256(out_trade_no|trade_status|amount, secret)
//   - notify_url (optional): override default APP_URL + /api/payment/notify/mock[/{tenant}]

func (s *PaymentGatewayServiceImpl) createMockPayment(payment *models.Payment, config map[string]any, _ string) (map[string]any, error) {
	notifyURL := defaultPaymentNotifyURL(s.ctx, "mock")
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

func (s *PaymentGatewayServiceImpl) queryMockPayment(payment *models.Payment, _ map[string]any) (map[string]any, error) {
	tradeState := "NOTPAY"
	switch payment.Status {
	case "paid":
		tradeState = "SUCCESS"
	case "failed":
		tradeState = "CLOSED"
	case "cancelled":
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

func (s *PaymentGatewayServiceImpl) handleMockNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
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
			payment, err := NewPaymentService(s.ctx).GetPaymentByPaymentNo(paymentNo)
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
	return ApplyPaidResult(s.ctx, PaidResult{
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

func parsePaymentMethodConfig(pm *models.PaymentMethod) (map[string]any, error) {
	if pm == nil || strings.TrimSpace(pm.Config) == "" {
		return map[string]any{}, nil
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(pm.Config), &config); err != nil {
		return nil, apperrors.ErrPaymentConfigRequired.WithError(err)
	}
	if config == nil {
		config = map[string]any{}
	}
	return config, nil
}

func stringFromConfig(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	v, _ := config[key].(string)
	return v
}

func firstString(data map[string]any, keys ...string) string {
	if data == nil {
		return ""
	}
	for _, k := range keys {
		v, ok := data[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			if s := strings.TrimSpace(t); s != "" {
				return s
			}
		case float64:
			return strconv.FormatInt(int64(t), 10)
		case int:
			return strconv.Itoa(t)
		case json.Number:
			return t.String()
		}
	}
	return ""
}

func optionalFloat(data map[string]any, keys ...string) *float64 {
	if data == nil {
		return nil
	}
	for _, k := range keys {
		v, ok := data[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case float64:
			return &t
		case float32:
			f := float64(t)
			return &f
		case int:
			f := float64(t)
			return &f
		case int64:
			f := float64(t)
			return &f
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
			if err == nil {
				return &f
			}
		case json.Number:
			f, err := t.Float64()
			if err == nil {
				return &f
			}
		}
	}
	return nil
}
