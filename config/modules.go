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
		// 代码生成器前端目标：vue / react / vue,react（默认两者都生成）
		"code_generator_frontend": config.Env("CODE_GENERATOR_FRONTEND", "vue,react"),
	})
}
