package feature_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppayment "goravel/app/payment"
	"goravel/app/services"
	"goravel/tests"
)

func mockNotifyHMAC(paymentNo, tradeStatus string, amount float64, secret string) string {
	payload := fmt.Sprintf("%s|%s|%.2f", paymentNo, tradeStatus, amount)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestMockPaymentNotifyMarksOrderPaid(t *testing.T) {
	withTenancyDriver(t, "off")
	ctx := t.Context()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	secret := "mock-feature-secret-" + suffix

	pm, err := services.NewPaymentMethodService(ctx).CreatePaymentMethod(
		"Mock Feature "+suffix, "mock_feat_"+suffix, "mock",
		map[string]any{"shared_secret": secret}, true, 0, "feature",
	)
	require.NoError(t, err)
	pmID := pm.ID
	t.Cleanup(func() {
		_ = services.NewPaymentMethodService(context.Background()).DeletePaymentMethod(pmID)
	})

	order, _, err := services.NewOrderService(ctx).CreateOrder(1, 8.5, []services.OrderProduct{
		{ProductID: 1, ProductName: "sku", Price: 8.5, Quantity: 1},
	}, "req-mock-"+suffix, "mock notify")
	require.NoError(t, err)

	payment, err := services.NewPaymentService(ctx).CreatePayment(order.OrderNo, pm.ID, order.UserID, order.Amount, "")
	require.NoError(t, err)

	sign := mockNotifyHMAC(payment.PaymentNo, "SUCCESS", order.Amount, secret)
	body := fmt.Sprintf(
		`{"out_trade_no":%q,"trade_status":"SUCCESS","amount":%.2f,"sign":%q,"transaction_id":"TX-%s"}`,
		payment.PaymentNo, order.Amount, sign, suffix,
	)
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/payment/notify/mock", strings.NewReader(body))
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)
	var payload struct {
		Code int `json:"code"`
		Data struct {
			Status    string `json:"status"`
			PaymentNo string `json:"payment_no"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
	assert.Equal(t, 200, payload.Code, content)
	assert.Equal(t, "paid", payload.Data.Status)

	paidPayment, err := services.NewPaymentService(ctx).GetPaymentByPaymentNo(payment.PaymentNo)
	require.NoError(t, err)
	assert.Equal(t, "paid", paidPayment.Status)

	paidOrder, _, err := services.NewOrderService(ctx).GetOrderByOrderNo(order.OrderNo)
	require.NoError(t, err)
	assert.Equal(t, "paid", paidOrder.Status)

	// Idempotent second notify
	resp2, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/payment/notify/mock", strings.NewReader(body))
	require.NoError(t, err)
	content2, err := resp2.Content()
	require.NoError(t, err)
	assert.Contains(t, content2, `"status":"paid"`)

	query, err := services.NewPaymentGatewayService(ctx).QueryPaymentOrder(paidPayment)
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", query["trade_state"])
}

func TestApplyPaidResultIdempotentFeature(t *testing.T) {
	withTenancyDriver(t, "off")
	ctx := t.Context()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)

	pm, err := services.NewPaymentMethodService(ctx).CreatePaymentMethod(
		"Mock Apply "+suffix, "mock_apply_"+suffix, "mock",
		map[string]any{"shared_secret": ""}, true, 0, "feature",
	)
	require.NoError(t, err)
	pmID := pm.ID
	t.Cleanup(func() {
		_ = services.NewPaymentMethodService(context.Background()).DeletePaymentMethod(pmID)
	})

	order, _, err := services.NewOrderService(ctx).CreateOrder(1, 12.34, []services.OrderProduct{
		{ProductID: 1, ProductName: "demo", Price: 12.34, Quantity: 1},
	}, "req-apply-"+suffix, "apply test")
	require.NoError(t, err)

	payment, err := services.NewPaymentService(ctx).CreatePayment(order.OrderNo, pm.ID, order.UserID, order.Amount, "")
	require.NoError(t, err)

	now := time.Now()
	paid, err := services.ApplyPaidResult(ctx, apppayment.PaidResult{
		PaymentNo:    payment.PaymentNo,
		ThirdPartyNo: "MOCK-UT-1",
		PayTime:      &now,
	})
	require.NoError(t, err)
	assert.Equal(t, "paid", paid.Status)

	again, err := services.ApplyPaidResult(ctx, apppayment.PaidResult{PaymentNo: payment.PaymentNo})
	require.NoError(t, err)
	assert.Equal(t, "paid", again.Status)

	updatedOrder, _, err := services.NewOrderService(ctx).GetOrderByOrderNo(order.OrderNo)
	require.NoError(t, err)
	assert.Equal(t, "paid", updatedOrder.Status)
}
