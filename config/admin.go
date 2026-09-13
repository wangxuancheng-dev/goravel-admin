package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("admin", map[string]any{
		// Super Admin ID
		//
		// The super admin ID that cannot modify roles. This is usually the default admin account.
		// You can set this ID in the .env file.
		//
		// Example in .env file:
		//   ADMIN_SUPER_ADMIN_ID=1
		"super_admin_id": config.Env("ADMIN_SUPER_ADMIN_ID", "1"), // Default: ID 1 (default admin user)

		// Developer Admin IDs
		//
		// The developer admin IDs that should be hidden from the admin list.
		// These admins cannot be deleted or disabled, and will not appear in the list.
		// Usually these are robot accounts, script accounts, etc.
		// You can set multiple IDs separated by commas in the .env file.
		//
		// Examples in .env file:
		//   Single ID: ADMIN_DEVELOPER_IDS=2
		//   Multiple IDs: ADMIN_DEVELOPER_IDS=2,3,4
		//   Multiple IDs with spaces: ADMIN_DEVELOPER_IDS=2, 3, 4
		"developer_ids": config.Env("ADMIN_DEVELOPER_IDS", "2"), // Default: ID 2 (developer admin)

		// Reserved usernames for create/update (comma-separated, case-insensitive).
		// Conflicts return username_unavailable instead of username_exists (avoids leaking hidden accounts).
		"reserved_usernames": config.Env("ADMIN_RESERVED_USERNAMES", "admin,developer"),
		// 注意：验证码配置已迁移到数据库存储（configs表，group='captcha'）
		// 后台管理系统 -> 系统管理 -> 配置管理 -> 验证码配置
		// 如需在代码中使用验证码配置，请从数据库读取，而不是从环境变量读取
		// "login_captcha_enabled": config.Env("ADMIN_LOGIN_CAPTCHA_ENABLED", false),
		// "login_captcha_expire":  config.Env("ADMIN_LOGIN_CAPTCHA_EXPIRE", 120), // seconds

		// Show Buttons Without Permission
		//
		// Whether to show operation buttons (add, edit, delete) on the page when the user
		// does not have the corresponding permission. If set to false, buttons will be hidden
		// when the user lacks permission. If set to true, buttons will be shown but disabled.
		//
		// Example in .env file:
		//   ADMIN_SHOW_BUTTONS_WITHOUT_PERMISSION=false
		"show_buttons_without_permission": config.Env("ADMIN_SHOW_BUTTONS_WITHOUT_PERMISSION", false), // Default: false (hide buttons)

		// Monitor Hidden
		//
		// Whether to hide the service monitor menu from menu management and sidebar.
		// If set to a non-empty value, the monitor menu will be hidden for all users except developer admins.
		// Developer admins (ADMIN_DEVELOPER_IDS) will always see the monitor menu regardless of this setting.
		// If not set or empty, the monitor menu will be shown normally (default behavior).
		//
		// Example in .env file:
		//   ADMIN_MONITOR_HIDDEN=1
		"monitor_hidden": config.Env("ADMIN_MONITOR_HIDDEN", ""), // Default: empty (show monitor menu)
	})
	config.Add("role", map[string]any{
		// Protected Role Slugs
		//
		// These role slugs cannot be deleted or disabled. Usually these are system roles
		// or super administrator roles that are critical to the system operation.
		// You can add multiple slugs separated by commas in the .env file.
		//
		// Examples in .env file:
		//   Single slug: ROLE_PROTECTED_SLUGS=super-admin
		//   Multiple slugs: ROLE_PROTECTED_SLUGS=super-admin,admin
		//   Multiple slugs with spaces: ROLE_PROTECTED_SLUGS=super-admin, admin
		"protected_slugs": config.Env("ROLE_PROTECTED_SLUGS", "super-admin"), // Default: super-admin
	})
}
