package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("module", map[string]any{
		// 示例业务模块开关（默认开启；纯 RBAC 后台可关闭）
		"orders_enabled":   config.Env("MODULE_ORDERS_ENABLED", true),
		"payments_enabled": config.Env("MODULE_PAYMENTS_ENABLED", false),
		// 启用的支付网关类型（逗号分隔）。空=全部已注册驱动；例：wechat,alipay
		"payment_gateways_enabled": config.Env("PAYMENT_GATEWAYS_ENABLED", ""),
		// 代码生成器前端目标：react / vue / react,vue（默认两者都生成；顺序建议 react 优先）
		"code_generator_frontend": config.Env("CODE_GENERATOR_FRONTEND", "react,vue"),
	})
}
