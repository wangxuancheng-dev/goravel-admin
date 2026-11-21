package gateways

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"

	"goravel-payment"
)

func init() {
	payment.RegisterGateway(&wechatDriver{})
}

const (
	wechatTradeNative = "native" // default: returns code_url for a QR code, no payer needed
	wechatTradeJSAPI  = "jsapi"  // needs Order.PayerID (the payer's openid under app_id)
)

// wechatDriver: external Go module example (github.com/go-pay/gopay), WeChat Pay API v3.
//
// Config: app_id, mch_id, api_v3_key, cert_serial_no (merchant API cert serial),
// private_key (apiclient_key.pem content) or private_key_path, trade_type (native|jsapi),
// notify_url (optional; else payment.ResolveNotifyURL).
// Query/notify return NotImplemented until verify is wired.
type wechatDriver struct{}

func (d *wechatDriver) Type() string { return "wechat" }

type wechatCreateParams struct {
	appID, mchID, apiV3Key, serialNo, privateKey string
	tradeType, notifyURL                          string
	body                                          gopay.BodyMap
}

func (d *wechatDriver) Create(ctx context.Context, order *payment.Order, config map[string]any, _ string) (map[string]any, error) {
	p, err := buildWechatCreate(ctx, order, config)
	if err != nil {
		return nil, err
	}
	client, err := wechat.NewClientV3(p.mchID, p.serialNo, p.apiV3Key, p.privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
	}

	out := map[string]any{"payment_no": order.PaymentNo, "trade_type": p.tradeType}
	switch p.tradeType {
	case wechatTradeJSAPI:
		rsp, err := client.V3TransactionJsapi(ctx, p.body)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
		}
		if rsp.Code != wechat.Success {
			return nil, fmt.Errorf("%w: %s", payment.ErrCreateFailed, rsp.Error)
		}
		// Params the WeChat JSAPI / mini-program client passes to requestPayment.
		params, err := client.PaySignOfJSAPI(p.appID, rsp.Response.PrepayId)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
		}
		out["prepay_id"] = rsp.Response.PrepayId
		out["jsapi_params"] = params
	default:
		rsp, err := client.V3TransactionNative(ctx, p.body)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
		}
		if rsp.Code != wechat.Success {
			return nil, fmt.Errorf("%w: %s", payment.ErrCreateFailed, rsp.Error)
		}
		out["code_url"] = rsp.Response.CodeUrl
	}
	return out, nil
}

// buildWechatCreate validates config and builds the request body (no network).
func buildWechatCreate(ctx context.Context, order *payment.Order, config map[string]any) (*wechatCreateParams, error) {
	if order == nil {
		return nil, fmt.Errorf("%w: order is nil", payment.ErrCreateFailed)
	}
	p := &wechatCreateParams{
		appID:    strings.TrimSpace(payment.StringFromConfig(config, "app_id")),
		mchID:    strings.TrimSpace(payment.StringFromConfig(config, "mch_id")),
		apiV3Key: strings.TrimSpace(payment.StringFromConfig(config, "api_v3_key")),
		serialNo: strings.TrimSpace(payment.StringFromConfig(config, "cert_serial_no")),
	}
	if p.appID == "" || p.mchID == "" || p.apiV3Key == "" || p.serialNo == "" {
		return nil, fmt.Errorf("%w: wechat needs app_id, mch_id, api_v3_key and cert_serial_no", payment.ErrConfigRequired)
	}
	key, err := wechatPrivateKey(config)
	if err != nil {
		return nil, err
	}
	p.privateKey = key

	p.tradeType = strings.ToLower(strings.TrimSpace(payment.StringFromConfig(config, "trade_type")))
	if p.tradeType == "" {
		p.tradeType = wechatTradeNative
	}
	if p.tradeType != wechatTradeNative && p.tradeType != wechatTradeJSAPI {
		return nil, fmt.Errorf("%w: unsupported wechat trade_type %q (native|jsapi)", payment.ErrConfigRequired, p.tradeType)
	}
	payerID := strings.TrimSpace(order.PayerID)
	if p.tradeType == wechatTradeJSAPI && payerID == "" {
		return nil, fmt.Errorf("%w: wechat jsapi needs the payer openid (Order.PayerID)", payment.ErrCreateFailed)
	}

	if p.notifyURL, err = payment.RequireNotifyURL(ctx, "wechat", config); err != nil {
		return nil, err
	}

	bm := make(gopay.BodyMap)
	bm.Set("appid", p.appID)
	bm.Set("out_trade_no", order.PaymentNo)
	bm.Set("description", orderSubject(order))
	bm.Set("notify_url", p.notifyURL)
	bm.Set("amount", map[string]any{
		"total":    int64(math.Round(order.Amount * 100)),
		"currency": "CNY",
	})
	if p.tradeType == wechatTradeJSAPI {
		bm.Set("payer", map[string]any{"openid": payerID})
	}
	p.body = bm
	return p, nil
}

// wechatPrivateKey returns the merchant API private key PEM from config
// (private_key content wins over private_key_path).
func wechatPrivateKey(config map[string]any) (string, error) {
	if key := strings.TrimSpace(payment.StringFromConfig(config, "private_key")); key != "" {
		return key, nil
	}
	path := strings.TrimSpace(payment.StringFromConfig(config, "private_key_path"))
	if path == "" {
		return "", fmt.Errorf("%w: wechat needs private_key (apiclient_key.pem content) or private_key_path", payment.ErrConfigRequired)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%w: read wechat private_key_path: %v", payment.ErrConfigRequired, err)
	}
	key := strings.TrimSpace(string(raw))
	if key == "" {
		return "", fmt.Errorf("%w: wechat private_key_path %s is empty", payment.ErrConfigRequired, path)
	}
	return key, nil
}

func (d *wechatDriver) Query(_ context.Context, _ *payment.Order, _ map[string]any) (map[string]any, error) {
	return nil, payment.ErrNotImplemented
}

func (d *wechatDriver) Notify(_ context.Context, _ map[string]any, _ map[string]any) (*payment.PaidResult, error) {
	return nil, payment.ErrNotImplemented
}
