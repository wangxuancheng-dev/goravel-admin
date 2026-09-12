package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("login_security", map[string]any{
		// 登录失败锁定：同一 IP + 同一账号维度
		// 同一 IP 下不同账号互不影响

		// 允许的最大连续失败次数，超过后锁定
		"max_attempts": config.Env("LOGIN_SECURITY_MAX_ATTEMPTS", 5),

		// 锁定时长（分钟）
		"lock_duration_minutes": config.Env("LOGIN_SECURITY_LOCK_DURATION_MINUTES", 15),

		// 失败计数的衰减窗口（分钟），在此时间内未再次失败则自动清零
		"decay_minutes": config.Env("LOGIN_SECURITY_DECAY_MINUTES", 5),

		// 密码策略
		"password_min_length":      config.Env("LOGIN_SECURITY_PASSWORD_MIN_LENGTH", 8),
		"password_require_letter":  config.Env("LOGIN_SECURITY_PASSWORD_REQUIRE_LETTER", true),
		"password_require_number":  config.Env("LOGIN_SECURITY_PASSWORD_REQUIRE_NUMBER", true),
		"password_require_special": config.Env("LOGIN_SECURITY_PASSWORD_REQUIRE_SPECIAL", false),

		// 登录异地/换 IP 告警
		"anomaly_alert_enabled": config.Env("LOGIN_SECURITY_ANOMALY_ALERT_ENABLED", true),

		// 角色维度 API 限流（次/分钟）
		"role_rate_limits": map[string]any{
			"super_admin_per_minute": config.Env("LOGIN_SECURITY_RATE_SUPER_ADMIN", 1200),
			"default_per_minute":     config.Env("LOGIN_SECURITY_RATE_DEFAULT", 300),
			"write_per_minute":       config.Env("LOGIN_SECURITY_RATE_WRITE", 120),
		},
	})
}
