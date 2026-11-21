package gateways

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel-payment"
)

func testRSAKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}

func wechatTestConfig(extra map[string]any) map[string]any {
	cfg := map[string]any{
		"app_id":         "wx_app",
		"mch_id":         "1900000001",
		"api_v3_key":     "k",
		"cert_serial_no": "SERIAL",
		"private_key":    "PEM",
		"notify_url":     "https://host/notify/wechat",
	}
	for k, v := range extra {
		cfg[k] = v
	}
	return cfg
}

func TestWechatCreateRequiresSerialAndPrivateKey(t *testing.T) {
	order := &payment.Order{PaymentNo: "PAY1", Amount: 1}

	_, err := buildWechatCreate(context.Background(), order, wechatTestConfig(map[string]any{"cert_serial_no": ""}))
	assert.True(t, errors.Is(err, payment.ErrConfigRequired), "%v", err)

	_, err = buildWechatCreate(context.Background(), order, wechatTestConfig(map[string]any{"private_key": ""}))
	assert.True(t, errors.Is(err, payment.ErrConfigRequired), "%v", err)
}

func TestWechatCreateReadsPrivateKeyPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apiclient_key.pem")
	require.NoError(t, os.WriteFile(path, []byte("  FILE-PEM \n"), 0o600))

	p, err := buildWechatCreate(context.Background(), &payment.Order{PaymentNo: "PAY1", Amount: 1},
		wechatTestConfig(map[string]any{"private_key": "", "private_key_path": path}))
	require.NoError(t, err)
	assert.Equal(t, "FILE-PEM", p.privateKey)
}

func TestWechatCreateNativeBody(t *testing.T) {
	p, err := buildWechatCreate(context.Background(), &payment.Order{PaymentNo: "PAY1", Amount: 9.9}, wechatTestConfig(nil))
	require.NoError(t, err)
	assert.Equal(t, wechatTradeNative, p.tradeType)
	assert.Equal(t, "1900000001", p.mchID)
	assert.Equal(t, "SERIAL", p.serialNo)
	assert.Equal(t, "wx_app", p.body.GetString("appid"))
	assert.Equal(t, "Payment PAY1", p.body.GetString("description"))
	assert.Equal(t, int64(990), p.body["amount"].(map[string]any)["total"])
	assert.Nil(t, p.body["payer"], "native must not send a payer")
}

func TestWechatCreateJSAPINeedsPayer(t *testing.T) {
	cfg := wechatTestConfig(map[string]any{"trade_type": "jsapi", "openid": "merchant-config-openid"})
	_, err := buildWechatCreate(context.Background(), &payment.Order{PaymentNo: "PAY1", Amount: 1}, cfg)
	require.Error(t, err, "openid in merchant config must not be used")

	p, err := buildWechatCreate(context.Background(), &payment.Order{PaymentNo: "PAY1", Amount: 1, PayerID: "user-openid"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, "user-openid", p.body["payer"].(map[string]any)["openid"])

	_, err = buildWechatCreate(context.Background(), &payment.Order{PaymentNo: "PAY1"}, wechatTestConfig(map[string]any{"trade_type": "h5"}))
	assert.True(t, errors.Is(err, payment.ErrConfigRequired), "%v", err)
}

func TestAlipayCreateOrderString(t *testing.T) {
	pemKey := testRSAKeyPEM(t)
	bareKey := alipayKeyBody(pemKey) // what the Alipay console hands out
	require.NotContains(t, bareKey, "-----")
	cases := []struct {
		key     string
		sandbox any
	}{{pemKey, nil}, {bareKey, "1"}}
	for _, c := range cases {
		key, sandbox := c.key, c.sandbox
		cfg := map[string]any{"app_id": "2021000000", "private_key": key, "notify_url": "https://host/notify/alipay"}
		if sandbox != nil {
			cfg["sandbox"] = sandbox
		}
		out, err := (&alipayDriver{}).Create(context.Background(), &payment.Order{PaymentNo: "PAY1", Amount: 9.9}, cfg, "")
		require.NoError(t, err)
		assert.Equal(t, sandbox != nil, out["sandbox"])

		orderString, _ := out["order_string"].(string)
		values, err := url.ParseQuery(orderString)
		require.NoError(t, err)
		biz := values.Get("biz_content")
		assert.Contains(t, biz, `"out_trade_no":"PAY1"`)
		assert.Contains(t, biz, `"subject":"Payment PAY1"`, "empty remark falls back to the payment no")
		assert.True(t, strings.HasPrefix(values.Get("notify_url"), "https://host/notify/alipay"))
	}
}
