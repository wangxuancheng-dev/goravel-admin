package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/go-pay/gopay/wechat/v3"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

// PaymentGatewayService 第三方支付下单/查询/回调（与后台支付记录 CRUD 解耦）。
// 注意：Query / Notify 仍为示例骨架（待实现），不要当作生产收单能力；见 docs/OPENSOURCE.md。
type PaymentGatewayService interface {
	CreatePaymentOrder(payment *models.Payment, clientIP string) (map[string]any, error)
	QueryPaymentOrder(payment *models.Payment) (map[string]any, error)
	HandlePaymentNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error)
}

type PaymentGatewayServiceImpl struct {
	ctx            context.Context
	paymentMethods PaymentMethodService
}

func NewPaymentGatewayService(ctx context.Context) PaymentGatewayService {
	return &PaymentGatewayServiceImpl{
		ctx:            ctx,
		paymentMethods: NewPaymentMethodService(ctx),
	}
}

// CreatePaymentOrder 创建支付订单（调用第三方支付）
func (s *PaymentGatewayServiceImpl) CreatePaymentOrder(payment *models.Payment, clientIP string) (map[string]any, error) {
	// 获取支付方式
	paymentMethod, err := s.paymentMethods.GetPaymentMethodByID(payment.PaymentMethodID)
	if err != nil {
		return nil, err
	}

	// 解析配置
	var config map[string]any
	if err := json.Unmarshal([]byte(paymentMethod.Config), &config); err != nil {
		return nil, apperrors.ErrPaymentConfigRequired.WithError(err)
	}

	// 根据支付类型调用不同的支付接口
	switch paymentMethod.Type {
	case "wechat":
		return s.createWechatPayment(payment, config, clientIP)
	case "alipay":
		return s.createAlipayPayment(payment, config, clientIP)
	default:
		return nil, apperrors.ErrInvalidPaymentType.WithMessage(fmt.Sprintf("不支持的支付类型: %s", paymentMethod.Type))
	}
}

// createWechatPayment 创建微信支付订单
func (s *PaymentGatewayServiceImpl) createWechatPayment(payment *models.Payment, config map[string]any, clientIP string) (map[string]any, error) {
	// 这里需要根据 gopay 的微信支付文档实现
	// 示例代码，实际使用时需要根据 gopay 的最新 API 调整
	// 参考: https://github.com/go-pay/gopay/tree/main/wechat/v3

	// 获取配置参数
	appID, _ := config["app_id"].(string)
	mchID, _ := config["mch_id"].(string)
	apiV3Key, _ := config["api_v3_key"].(string)
	certSerialNo, _ := config["cert_serial_no"].(string)
	privateKeyPath, _ := config["private_key_path"].(string)

	if appID == "" || mchID == "" || apiV3Key == "" {
		return nil, apperrors.ErrPaymentConfigRequired.WithMessage("微信支付配置不完整")
	}

	// 创建微信支付客户端
	client, err := wechat.NewClientV3(appID, mchID, apiV3Key, certSerialNo)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}
	// 设置私钥（如果需要）
	if privateKeyPath != "" {
		// 这里需要根据 gopay 的实际 API 设置私钥
		// client.SetPrivateKey(...)
	}

	// 设置回调地址（需要从配置中读取）
	notifyURL, _ := config["notify_url"].(string)
	if notifyURL == "" {
		notifyURL = fmt.Sprintf("%s/api/payment/notify/wechat", facades.Config().GetString("app.url"))
	}

	// 创建支付订单
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", payment.PaymentNo)
	bm.Set("description", payment.Remark)
	bm.Set("amount", map[string]any{
		"total":    int(payment.Amount * 100), // 转换为分
		"currency": "CNY",
	})
	bm.Set("notify_url", notifyURL)
	bm.Set("payer", map[string]any{
		"openid": config["openid"], // 需要从订单或用户信息中获取
	})

	// 注意：这里需要根据 gopay 的最新 API 调用
	// 示例代码，实际使用时需要根据 gopay 的最新文档调整
	ctx := context.Background()
	wxRsp, err := client.V3TransactionJsapi(ctx, bm)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	if wxRsp.Code != wechat.Success {
		return nil, apperrors.ErrCreatePaymentFailed.WithMessage(wxRsp.Error)
	}

	return map[string]any{
		"payment_no": payment.PaymentNo,
		"prepay_id":  wxRsp.Response.PrepayId,
		// PaySign 可能需要从其他地方获取或计算
	}, nil
}

// createAlipayPayment 创建支付宝支付订单
func (s *PaymentGatewayServiceImpl) createAlipayPayment(payment *models.Payment, config map[string]any, clientIP string) (map[string]any, error) {
	// 这里需要根据 gopay 的支付宝文档实现
	// 示例代码，实际使用时需要根据 gopay 的最新 API 调整
	// 参考: https://github.com/go-pay/gopay/tree/main/alipay

	// 获取配置参数
	appID, _ := config["app_id"].(string)
	privateKey, _ := config["private_key"].(string)
	// appCertPublicKey, _ := config["app_cert_public_key"].(string)
	// alipayRootCert, _ := config["alipay_root_cert"].(string)
	// alipayPublicCert, _ := config["alipay_public_cert"].(string)

	if appID == "" || privateKey == "" {
		return nil, apperrors.ErrPaymentConfigRequired.WithMessage("支付宝配置不完整")
	}

	// 创建支付宝客户端
	client, err := alipay.NewClient(appID, privateKey, false)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	// 设置证书（如果需要）
	// 注意：这里需要根据 gopay 的最新 API 设置证书
	// 示例代码，实际使用时需要根据 gopay 的最新文档调整
	// if appCertPublicKey != "" {
	// 	client.SetAppCertPublicKey(appCertPublicKey)
	// }
	// if alipayRootCert != "" {
	// 	client.SetAlipayRootCert(alipayRootCert)
	// }
	// if alipayPublicCert != "" {
	// 	client.SetAlipayPublicCert(alipayPublicCert)
	// }

	// 设置回调地址
	notifyURL, _ := config["notify_url"].(string)
	if notifyURL == "" {
		notifyURL = fmt.Sprintf("%s/api/payment/notify/alipay", facades.Config().GetString("app.url"))
	}
	client.SetNotifyUrl(notifyURL)

	// 创建支付订单
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", payment.PaymentNo)
	bm.Set("subject", payment.Remark)
	bm.Set("total_amount", fmt.Sprintf("%.2f", payment.Amount))
	bm.Set("product_code", "QUICK_MSECURITY_PAY")

	ctx := context.Background()
	payUrl, err := client.TradeAppPay(ctx, bm)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	return map[string]any{
		"payment_no": payment.PaymentNo,
		"pay_url":    payUrl,
	}, nil
}

// QueryPaymentOrder 查询支付订单状态
func (s *PaymentGatewayServiceImpl) QueryPaymentOrder(payment *models.Payment) (map[string]any, error) {
	// 获取支付方式
	paymentMethod, err := s.paymentMethods.GetPaymentMethodByID(payment.PaymentMethodID)
	if err != nil {
		return nil, err
	}

	// 解析配置
	var config map[string]any
	if err := json.Unmarshal([]byte(paymentMethod.Config), &config); err != nil {
		return nil, apperrors.ErrPaymentConfigRequired.WithError(err)
	}

	// 根据支付类型查询
	switch paymentMethod.Type {
	case "wechat":
		return s.queryWechatPayment(payment, config)
	case "alipay":
		return s.queryAlipayPayment(payment, config)
	default:
		return nil, apperrors.ErrInvalidPaymentType.WithMessage(fmt.Sprintf("不支持的支付类型: %s", paymentMethod.Type))
	}
}

// queryWechatPayment 查询微信支付订单状态
func (s *PaymentGatewayServiceImpl) queryWechatPayment(payment *models.Payment, config map[string]any) (map[string]any, error) {
	// 实现微信支付查询逻辑
	// 这里需要根据 gopay 的微信支付文档实现
	return nil, fmt.Errorf("微信支付查询功能待实现")
}

// queryAlipayPayment 查询支付宝支付订单状态
func (s *PaymentGatewayServiceImpl) queryAlipayPayment(payment *models.Payment, config map[string]any) (map[string]any, error) {
	// 实现支付宝支付查询逻辑
	// 这里需要根据 gopay 的支付宝文档实现
	return nil, fmt.Errorf("支付宝支付查询功能待实现")
}

// HandlePaymentNotify 处理支付回调通知
func (s *PaymentGatewayServiceImpl) HandlePaymentNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	// 根据支付类型处理回调
	switch paymentMethod.Type {
	case "wechat":
		return s.handleWechatNotify(paymentMethod, notifyData)
	case "alipay":
		return s.handleAlipayNotify(paymentMethod, notifyData)
	default:
		return nil, apperrors.ErrInvalidPaymentType.WithMessage(fmt.Sprintf("不支持的支付类型: %s", paymentMethod.Type))
	}
}

// handleWechatNotify 处理微信支付回调
func (s *PaymentGatewayServiceImpl) handleWechatNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	// 实现微信支付回调处理逻辑
	// 这里需要根据 gopay 的微信支付文档实现
	return nil, fmt.Errorf("微信支付回调处理功能待实现")
}

// handleAlipayNotify 处理支付宝支付回调
func (s *PaymentGatewayServiceImpl) handleAlipayNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	// 实现支付宝支付回调处理逻辑
	// 这里需要根据 gopay 的支付宝文档实现
	return nil, fmt.Errorf("支付宝支付回调处理功能待实现")
}
