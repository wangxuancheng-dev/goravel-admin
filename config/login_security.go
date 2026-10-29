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

		// 账号维度（跨 IP，防代理池轮换 IP 撞库）
		// 在衰减窗口内累计失败达到 captcha_after 次后，所有 IP 登录该账号都需人机验证（0 关闭）
		"account_captcha_after": config.Env("LOGIN_SECURITY_ACCOUNT_CAPTCHA_AFTER", 5),
		// 累计失败达到 max_attempts 次后锁定该账号（0 关闭）；阈值宜宽松，避免他人恶意锁死正常用户
		"account_max_attempts":          config.Env("LOGIN_SECURITY_ACCOUNT_MAX_ATTEMPTS", 20),
		"account_lock_duration_minutes": config.Env("LOGIN_SECURITY_ACCOUNT_LOCK_DURATION_MINUTES", 15),
		"account_decay_minutes":         config.Env("LOGIN_SECURITY_ACCOUNT_DECAY_MINUTES", 60),

		// 密码策略（管理员 / C 端用户 / 平台管理员通用；后台“登录安全”配置可按租户覆盖，平台管理员只认这里）
		// 另外始终拒绝常见弱口令（如 Admin@123）、包含用户名的密码，改密时拒绝与旧密码相同
		"password_min_length":      config.Env("LOGIN_SECURITY_PASSWORD_MIN_LENGTH", 8),
		"password_require_letter":  config.Env("LOGIN_SECURITY_PASSWORD_REQUIRE_LETTER", true),
		"password_require_number":  config.Env("LOGIN_SECURITY_PASSWORD_REQUIRE_NUMBER", true),
		"password_require_special": config.Env("LOGIN_SECURITY_PASSWORD_REQUIRE_SPECIAL", true),

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
