package seeders

import (
	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

type PermissionSeeder struct {
}

func (s *PermissionSeeder) Signature() string {
	return "PermissionSeeder"
}

func (s *PermissionSeeder) Run() error {
	// 获取菜单（权限需要关联菜单）
	var adminMenu, roleMenu, permissionMenu, menuMenu, departmentMenu, positionMenu, dictionaryMenu, configMenu, blacklistMenu, onlineAdminMenu, orderMenu, userMenu, userBalanceLogMenu models.Menu
	var operationLogMenu, loginLogMenu, systemLogMenu, observabilityMenu, monitorMenu, scheduleMenu, profileMenu, exportMenu, attachmentMenu, dashboardMenu, notificationMenu models.Menu
	var paymentMethodMenu, paymentRecordMenu models.Menu

	// 辅助函数：查找菜单
	findMenu := func(slug string, menu *models.Menu) {
		*menu = models.Menu{}
		facades.Orm().Query().Where("slug", slug).First(menu)
	}

	findMenu("admin", &adminMenu)
	findMenu("role", &roleMenu)
	findMenu("permission", &permissionMenu)
	findMenu("menu", &menuMenu)
	findMenu("department", &departmentMenu)
	findMenu("position", &positionMenu)
	findMenu("dictionary", &dictionaryMenu)
	findMenu("config", &configMenu)
	findMenu("blacklist", &blacklistMenu)
	findMenu("online-admin", &onlineAdminMenu)
	findMenu("operation-log", &operationLogMenu)
	findMenu("login-log", &loginLogMenu)
	findMenu("system-log", &systemLogMenu)
	findMenu("observability", &observabilityMenu)
	findMenu("monitor", &monitorMenu)
	findMenu("schedule", &scheduleMenu)
	findMenu("profile", &profileMenu)
	findMenu("export", &exportMenu)
	findMenu("attachment", &attachmentMenu)
	findMenu("notification", &notificationMenu)
	findMenu("order", &orderMenu)
	findMenu("user", &userMenu)
	findMenu("user-balance-log", &userBalanceLogMenu)
	findMenu("payment-method", &paymentMethodMenu)
	findMenu("payment-record", &paymentRecordMenu)

	// Dashboard 可能没有单独的菜单，使用 profile 菜单作为关联
	facades.Orm().Query().Where("slug", "dashboard").First(&dashboardMenu)
	if dashboardMenu.ID == 0 {
		dashboardMenu = profileMenu
	}

	// 创建权限（关联菜单ID）
	permissions := []models.Permission{
		// 管理员管理
		{Name: "管理员列表", Slug: "admin.index", Method: "GET", Path: "/api/admin/admins", Description: "查看管理员列表", Status: 1, Sort: 1, MenuID: adminMenu.ID},
		{Name: "管理员详情", Slug: "admin.show", Method: "GET", Path: "/api/admin/admins/*", Description: "查看管理员详情", Status: 1, Sort: 2, MenuID: adminMenu.ID},
		{Name: "管理员创建", Slug: "admin.store", Method: "POST", Path: "/api/admin/admins", Description: "创建管理员", Status: 1, Sort: 3, MenuID: adminMenu.ID},
		{Name: "管理员更新", Slug: "admin.update", Method: "PUT", Path: "/api/admin/admins/*", Description: "更新管理员", Status: 1, Sort: 4, MenuID: adminMenu.ID},
		{Name: "管理员删除", Slug: "admin.destroy", Method: "DELETE", Path: "/api/admin/admins/*", Description: "删除管理员", Status: 1, Sort: 5, MenuID: adminMenu.ID},
		{Name: "管理员导出", Slug: "admin.export", Method: "POST", Path: "/api/admin/admins/export", Description: "导出管理员列表", Status: 1, Sort: 6, MenuID: adminMenu.ID},
		{Name: "管理员重置密码", Slug: "admin.password", Method: "PUT", Path: "/api/admin/admins/*/password", Description: "重置管理员密码", Status: 1, Sort: 7, MenuID: adminMenu.ID},
		{Name: "踢出用户", Slug: "admin.kick_out", Method: "DELETE", Path: "/api/admin/admins/*/tokens", Description: "踢出指定用户的所有token", Status: 1, Sort: 8, MenuID: adminMenu.ID},
		{Name: "解绑谷歌验证码", Slug: "admin.unbind_google_auth", Method: "POST", Path: "/api/admin/admins/*/unbind-google-auth", Description: "解绑管理员的谷歌验证码", Status: 1, Sort: 9, MenuID: adminMenu.ID},
		{Name: "重置谷歌验证码", Slug: "admin.reset_google_auth", Method: "POST", Path: "/api/admin/admins/*/reset-google-auth", Description: "重置管理员的谷歌验证码（无需验证码，用于丢失手机等场景）", Status: 1, Sort: 10, MenuID: adminMenu.ID},
		// 角色管理
		{Name: "角色列表", Slug: "role.index", Method: "GET", Path: "/api/admin/roles", Description: "查看角色列表", Status: 1, Sort: 1, MenuID: roleMenu.ID},
		{Name: "角色详情", Slug: "role.show", Method: "GET", Path: "/api/admin/roles/*", Description: "查看角色详情", Status: 1, Sort: 2, MenuID: roleMenu.ID},
		{Name: "角色创建", Slug: "role.store", Method: "POST", Path: "/api/admin/roles", Description: "创建角色", Status: 1, Sort: 3, MenuID: roleMenu.ID},
		{Name: "角色更新", Slug: "role.update", Method: "PUT", Path: "/api/admin/roles/*", Description: "更新角色", Status: 1, Sort: 4, MenuID: roleMenu.ID},
		{Name: "角色删除", Slug: "role.destroy", Method: "DELETE", Path: "/api/admin/roles/*", Description: "删除角色", Status: 1, Sort: 5, MenuID: roleMenu.ID},
		// 权限管理
		{Name: "权限列表", Slug: "permission.index", Method: "GET", Path: "/api/admin/permissions", Description: "查看权限列表", Status: 1, Sort: 1, MenuID: permissionMenu.ID},
		{Name: "权限详情", Slug: "permission.show", Method: "GET", Path: "/api/admin/permissions/*", Description: "查看权限详情", Status: 1, Sort: 2, MenuID: permissionMenu.ID},
		{Name: "权限创建", Slug: "permission.store", Method: "POST", Path: "/api/admin/permissions", Description: "创建权限", Status: 1, Sort: 3, MenuID: permissionMenu.ID},
		{Name: "权限更新", Slug: "permission.update", Method: "PUT", Path: "/api/admin/permissions/*", Description: "更新权限", Status: 1, Sort: 4, MenuID: permissionMenu.ID},
		{Name: "权限删除", Slug: "permission.destroy", Method: "DELETE", Path: "/api/admin/permissions/*", Description: "删除权限", Status: 1, Sort: 5, MenuID: permissionMenu.ID},
		// 菜单管理
		{Name: "菜单列表", Slug: "menu.index", Method: "GET", Path: "/api/admin/menus", Description: "查看菜单列表", Status: 1, Sort: 1, MenuID: menuMenu.ID},
		{Name: "菜单详情", Slug: "menu.show", Method: "GET", Path: "/api/admin/menus/*", Description: "查看菜单详情", Status: 1, Sort: 2, MenuID: menuMenu.ID},
		{Name: "菜单创建", Slug: "menu.store", Method: "POST", Path: "/api/admin/menus", Description: "创建菜单", Status: 1, Sort: 3, MenuID: menuMenu.ID},
		{Name: "菜单更新", Slug: "menu.update", Method: "PUT", Path: "/api/admin/menus/*", Description: "更新菜单", Status: 1, Sort: 4, MenuID: menuMenu.ID},
		{Name: "菜单删除", Slug: "menu.destroy", Method: "DELETE", Path: "/api/admin/menus/*", Description: "删除菜单", Status: 1, Sort: 5, MenuID: menuMenu.ID},
		// 部门管理
		{Name: "部门列表", Slug: "department.index", Method: "GET", Path: "/api/admin/departments", Description: "查看部门列表", Status: 1, Sort: 1, MenuID: departmentMenu.ID},
		{Name: "部门详情", Slug: "department.show", Method: "GET", Path: "/api/admin/departments/*", Description: "查看部门详情", Status: 1, Sort: 2, MenuID: departmentMenu.ID},
		{Name: "部门创建", Slug: "department.store", Method: "POST", Path: "/api/admin/departments", Description: "创建部门", Status: 1, Sort: 3, MenuID: departmentMenu.ID},
		{Name: "部门更新", Slug: "department.update", Method: "PUT", Path: "/api/admin/departments/*", Description: "更新部门", Status: 1, Sort: 4, MenuID: departmentMenu.ID},
		{Name: "部门删除", Slug: "department.destroy", Method: "DELETE", Path: "/api/admin/departments/*", Description: "删除部门", Status: 1, Sort: 5, MenuID: departmentMenu.ID},
		{Name: "部门管理员批量转移", Slug: "department.transfer_admins", Method: "POST", Path: "/api/admin/departments/*/transfer-admins", Description: "批量转移部门管理员", Status: 1, Sort: 6, MenuID: departmentMenu.ID},
		// 岗位管理
		{Name: "岗位列表", Slug: "position.index", Method: "GET", Path: "/api/admin/positions", Description: "查看岗位列表", Status: 1, Sort: 1, MenuID: positionMenu.ID},
		{Name: "岗位详情", Slug: "position.show", Method: "GET", Path: "/api/admin/positions/*", Description: "查看岗位详情", Status: 1, Sort: 2, MenuID: positionMenu.ID},
		{Name: "岗位创建", Slug: "position.store", Method: "POST", Path: "/api/admin/positions", Description: "创建岗位", Status: 1, Sort: 3, MenuID: positionMenu.ID},
		{Name: "岗位更新", Slug: "position.update", Method: "PUT", Path: "/api/admin/positions/*", Description: "更新岗位", Status: 1, Sort: 4, MenuID: positionMenu.ID},
		{Name: "岗位删除", Slug: "position.destroy", Method: "DELETE", Path: "/api/admin/positions/*", Description: "删除岗位", Status: 1, Sort: 5, MenuID: positionMenu.ID},
		// 字典管理
		{Name: "字典列表", Slug: "dictionary.index", Method: "GET", Path: "/api/admin/dictionaries", Description: "查看字典列表", Status: 1, Sort: 1, MenuID: dictionaryMenu.ID},
		{Name: "字典详情", Slug: "dictionary.show", Method: "GET", Path: "/api/admin/dictionaries/*", Description: "查看字典详情", Status: 1, Sort: 2, MenuID: dictionaryMenu.ID},
		{Name: "字典创建", Slug: "dictionary.store", Method: "POST", Path: "/api/admin/dictionaries", Description: "创建字典", Status: 1, Sort: 3, MenuID: dictionaryMenu.ID},
		{Name: "字典更新", Slug: "dictionary.update", Method: "PUT", Path: "/api/admin/dictionaries/*", Description: "更新字典", Status: 1, Sort: 4, MenuID: dictionaryMenu.ID},
		{Name: "字典删除", Slug: "dictionary.destroy", Method: "DELETE", Path: "/api/admin/dictionaries/*", Description: "删除字典", Status: 1, Sort: 5, MenuID: dictionaryMenu.ID},
		{Name: "字典查询", Slug: "dictionary.type", Method: "GET", Path: "/api/admin/dictionaries/type/*", Description: "根据类型查询字典", Status: 1, Sort: 6, MenuID: dictionaryMenu.ID},
		// 配置管理
		{Name: "获取配置", Slug: "config.group", Method: "GET", Path: "/api/admin/configs/group/*", Description: "根据分组获取配置", Status: 1, Sort: 1, MenuID: configMenu.ID},
		{Name: "保存配置", Slug: "config.save", Method: "POST", Path: "/api/admin/configs/save", Description: "保存配置", Status: 1, Sort: 2, MenuID: configMenu.ID},
		{Name: "测试邮箱", Slug: "config.test_email", Method: "POST", Path: "/api/admin/configs/test-email", Description: "测试邮箱配置", Status: 1, Sort: 3, MenuID: configMenu.ID},
		// 黑名单管理
		{Name: "黑名单列表", Slug: "blacklist.index", Method: "GET", Path: "/api/admin/blacklists", Description: "查看黑名单列表", Status: 1, Sort: 1, MenuID: blacklistMenu.ID},
		{Name: "黑名单详情", Slug: "blacklist.show", Method: "GET", Path: "/api/admin/blacklists/*", Description: "查看黑名单详情", Status: 1, Sort: 2, MenuID: blacklistMenu.ID},
		{Name: "黑名单创建", Slug: "blacklist.store", Method: "POST", Path: "/api/admin/blacklists", Description: "创建黑名单", Status: 1, Sort: 3, MenuID: blacklistMenu.ID},
		{Name: "黑名单更新", Slug: "blacklist.update", Method: "PUT", Path: "/api/admin/blacklists/*", Description: "更新黑名单", Status: 1, Sort: 4, MenuID: blacklistMenu.ID},
		{Name: "黑名单删除", Slug: "blacklist.destroy", Method: "DELETE", Path: "/api/admin/blacklists/*", Description: "删除黑名单", Status: 1, Sort: 5, MenuID: blacklistMenu.ID},

		// 在线管理员管理
		{Name: "在线管理员列表", Slug: "online-admin.index", Method: "GET", Path: "/api/admin/online-admins", Description: "查看在线管理员列表", Status: 1, Sort: 1, MenuID: onlineAdminMenu.ID},
		{Name: "踢下线", Slug: "online-admin.kick-out", Method: "DELETE", Path: "/api/admin/online-admins/*", Description: "踢下线管理员", Status: 1, Sort: 2, MenuID: onlineAdminMenu.ID},
		{Name: "批量踢下线", Slug: "online-admin.batch-kick-out", Method: "POST", Path: "/api/admin/online-admins/batch-kick-out", Description: "批量踢下线管理员", Status: 1, Sort: 3, MenuID: onlineAdminMenu.ID},
		// 操作日志
		{Name: "操作日志列表", Slug: "operation_log.index", Method: "GET", Path: "/api/admin/operation-logs", Description: "查看操作日志列表", Status: 1, Sort: 1, MenuID: operationLogMenu.ID},
		{Name: "操作日志详情", Slug: "operation_log.show", Method: "GET", Path: "/api/admin/operation-logs/*", Description: "查看操作日志详情", Status: 1, Sort: 2, MenuID: operationLogMenu.ID},
		{Name: "操作日志删除", Slug: "operation_log.destroy", Method: "DELETE", Path: "/api/admin/operation-logs/*", Description: "删除操作日志", Status: 1, Sort: 3, MenuID: operationLogMenu.ID},
		{Name: "操作日志批量删除", Slug: "operation_log.batch_delete", Method: "POST", Path: "/api/admin/operation-logs/batch-delete", Description: "批量删除操作日志", Status: 1, Sort: 4, MenuID: operationLogMenu.ID},
		{Name: "操作日志清理", Slug: "operation_log.clean", Method: "POST", Path: "/api/admin/operation-logs/clean", Description: "清理操作日志", Status: 1, Sort: 5, MenuID: operationLogMenu.ID},
		{Name: "操作日志归档", Slug: "operation_log.archive", Method: "POST", Path: "/api/admin/operation-logs/archive", Description: "归档并清理操作日志", Status: 1, Sort: 6, MenuID: operationLogMenu.ID},
		// 登录日志
		{Name: "登录日志列表", Slug: "login_log.index", Method: "GET", Path: "/api/admin/login-logs", Description: "查看登录日志列表", Status: 1, Sort: 1, MenuID: loginLogMenu.ID},
		{Name: "登录日志详情", Slug: "login_log.show", Method: "GET", Path: "/api/admin/login-logs/*", Description: "查看登录日志详情", Status: 1, Sort: 2, MenuID: loginLogMenu.ID},
		{Name: "登录日志删除", Slug: "login_log.destroy", Method: "DELETE", Path: "/api/admin/login-logs/*", Description: "删除登录日志", Status: 1, Sort: 3, MenuID: loginLogMenu.ID},
		{Name: "登录日志批量删除", Slug: "login_log.batch_delete", Method: "POST", Path: "/api/admin/login-logs/batch-delete", Description: "批量删除登录日志", Status: 1, Sort: 4, MenuID: loginLogMenu.ID},
		// {Name: "登录日志清理", Slug: "login_log.clean", Method: "POST", Path: "/api/admin/login-logs/clean", Description: "清理登录日志", Status: 1, Sort: 5, MenuID: loginLogMenu.ID},
		// 系统日志
		{Name: "系统日志列表", Slug: "system_log.index", Method: "GET", Path: "/api/admin/system-logs", Description: "查看系统日志列表", Status: 1, Sort: 1, MenuID: systemLogMenu.ID},
		{Name: "系统日志详情", Slug: "system_log.show", Method: "GET", Path: "/api/admin/system-logs/*", Description: "查看系统日志详情", Status: 1, Sort: 2, MenuID: systemLogMenu.ID},
		{Name: "系统日志删除", Slug: "system_log.destroy", Method: "DELETE", Path: "/api/admin/system-logs/*", Description: "删除系统日志", Status: 1, Sort: 3, MenuID: systemLogMenu.ID},
		{Name: "系统日志批量删除", Slug: "system_log.batch_delete", Method: "POST", Path: "/api/admin/system-logs/batch-delete", Description: "批量删除系统日志", Status: 1, Sort: 4, MenuID: systemLogMenu.ID},
		// {Name: "系统日志清理", Slug: "system_log.clean", Method: "POST", Path: "/api/admin/system-logs/clean", Description: "清理系统日志", Status: 1, Sort: 5, MenuID: systemLogMenu.ID},
		// 观测与调试
		{Name: "追踪聚合查询", Slug: "observability.trace", Method: "GET", Path: "/api/admin/observability/trace", Description: "按 trace_id 聚合请求链路", Status: 1, Sort: 1, MenuID: observabilityMenu.ID},
		{Name: "慢SQL TopN", Slug: "observability.slow_sql_top", Method: "GET", Path: "/api/admin/observability/slow-sql/top", Description: "查询慢 SQL TopN 统计", Status: 1, Sort: 2, MenuID: observabilityMenu.ID},
		{Name: "审计时间线", Slug: "observability.audit_timeline", Method: "GET", Path: "/api/admin/observability/audit-timeline", Description: "查询统一审计时间线", Status: 1, Sort: 3, MenuID: observabilityMenu.ID},
		{Name: "队列看板", Slug: "observability.queue_dashboard", Method: "GET", Path: "/api/admin/observability/queue-dashboard", Description: "轻量队列统计看板", Status: 1, Sort: 4, MenuID: observabilityMenu.ID},
		{Name: "接口性能概览", Slug: "observability.api_performance_overview", Method: "GET", Path: "/api/admin/observability/api-performance/overview", Description: "接口性能 TopN 与错误率概览", Status: 1, Sort: 5, MenuID: observabilityMenu.ID},
		{Name: "接口性能 Trace 下钻", Slug: "observability.api_performance_traces", Method: "GET", Path: "/api/admin/observability/api-performance/traces", Description: "按路由模板查询关联 trace 列表", Status: 1, Sort: 6, MenuID: observabilityMenu.ID},
		// {Name: "PPROF状态", Slug: "observability.pprof_status", Method: "GET", Path: "/api/admin/observability/pprof/status", Description: "查询 pprof 功能状态", Status: 1, Sort: 7, MenuID: observabilityMenu.ID},
		// {Name: "PPROF验证", Slug: "observability.pprof_verify", Method: "POST", Path: "/api/admin/observability/pprof/verify", Description: "验证 pprof token", Status: 1, Sort: 8, MenuID: observabilityMenu.ID},
		// {Name: "CPU热点采样", Slug: "observability.pprof_cpu_hotspots", Method: "POST", Path: "/api/admin/observability/pprof/cpu-hotspots", Description: "采样并查询 CPU 热点函数", Status: 1, Sort: 9, MenuID: observabilityMenu.ID},
		// {Name: "内存热点采样", Slug: "observability.pprof_memory_hotspots", Method: "POST", Path: "/api/admin/observability/pprof/memory-hotspots", Description: "采样并查询内存分配热点", Status: 1, Sort: 10, MenuID: observabilityMenu.ID},
		// 服务监控
		{Name: "系统监控", Slug: "monitor.system_info", Method: "GET", Path: "/api/admin/monitor/system-info", Description: "查看系统监控信息", Status: 1, Sort: 1, MenuID: monitorMenu.ID},
		{Name: "系统监控实时流", Slug: "monitor.system_info_stream", Method: "GET", Path: "/api/admin/monitor/system-info/stream", Description: "系统监控实时数据流", Status: 1, Sort: 2, MenuID: monitorMenu.ID},
		// 定时任务
		{Name: "定时任务列表", Slug: "schedule.index", Method: "GET", Path: "/api/admin/schedules", Description: "查看定时任务列表", Status: 1, Sort: 1, MenuID: scheduleMenu.ID},
		{Name: "手动执行定时任务", Slug: "schedule.run", Method: "POST", Path: "/api/admin/schedules/run", Description: "手动执行已注册的定时任务", Status: 1, Sort: 2, MenuID: scheduleMenu.ID},
		// 个人中心
		{Name: "修改资料", Slug: "profile.update", Method: "PUT", Path: "/api/admin/profile", Description: "修改当前登录管理员资料", Status: 1, Sort: 1, MenuID: profileMenu.ID},
		{Name: "修改密码", Slug: "password.update", Method: "PUT", Path: "/api/admin/password", Description: "修改当前登录管理员密码", Status: 1, Sort: 2, MenuID: profileMenu.ID},
		// 导出管理
		{Name: "导出列表", Slug: "export.index", Method: "GET", Path: "/api/admin/exports", Description: "查看导出记录列表", Status: 1, Sort: 1, MenuID: exportMenu.ID},
		{Name: "导出数据下载", Slug: "export.download", Method: "GET", Path: "/api/admin/exports/*/download", Description: "下载导出数据文件", Status: 1, Sort: 2, MenuID: exportMenu.ID},
		// {Name: "导出进度", Slug: "export.progress", Method: "GET", Path: "/api/admin/exports/*/progress", Description: "查看导出任务进度", Status: 1, Sort: 3, MenuID: exportMenu.ID},
		{Name: "删除导出", Slug: "export.destroy", Method: "DELETE", Path: "/api/admin/exports/*", Description: "删除导出记录及源文件", Status: 1, Sort: 4, MenuID: exportMenu.ID},
		{Name: "导出批量删除", Slug: "export.batch_delete", Method: "POST", Path: "/api/admin/exports/batch-delete", Description: "批量删除导出记录", Status: 1, Sort: 5, MenuID: exportMenu.ID},
		// 附件管理
		{Name: "附件列表", Slug: "attachment.index", Method: "GET", Path: "/api/admin/attachments", Description: "查看附件列表", Status: 1, Sort: 1, MenuID: attachmentMenu.ID},
		{Name: "附件上传", Slug: "attachment.upload", Method: "POST", Path: "/api/admin/attachments/upload", Description: "上传附件", Status: 1, Sort: 2, MenuID: attachmentMenu.ID},
		{Name: "大文件分片上传", Slug: "attachment.chunk", Method: "POST", Path: "/api/admin/attachments/chunk", Description: "大文件分片上传（包含初始化、上传分片、合并分片）", Status: 1, Sort: 3, MenuID: attachmentMenu.ID},
		{Name: "获取上传进度", Slug: "attachment.chunk_progress", Method: "GET", Path: "/api/admin/attachments/chunk", Description: "获取大文件分片上传进度", Status: 1, Sort: 4, MenuID: attachmentMenu.ID},
		// {Name: "上传进度推送", Slug: "attachment.upload_progress", Method: "GET", Path: "/api/admin/attachments/upload/progress", Description: "文件上传进度实时推送", Status: 1, Sort: 4, MenuID: attachmentMenu.ID},
		{Name: "附件预览", Slug: "attachment.preview", Method: "GET", Path: "/api/admin/attachments/*/preview", Description: "预览附件", Status: 1, Sort: 4, MenuID: attachmentMenu.ID},
		{Name: "附件下载", Slug: "attachment.download", Method: "GET", Path: "/api/admin/attachments/*/download", Description: "下载附件", Status: 1, Sort: 5, MenuID: attachmentMenu.ID},
		{Name: "附件更新显示名称", Slug: "attachment.update_display_name", Method: "PUT", Path: "/api/admin/attachments/*/display-name", Description: "更新附件显示名称", Status: 1, Sort: 6, MenuID: attachmentMenu.ID},
		{Name: "附件更新分类", Slug: "attachment.update_category", Method: "PUT", Path: "/api/admin/attachments/*/category", Description: "更新附件分类", Status: 1, Sort: 7, MenuID: attachmentMenu.ID},
		{Name: "附件更新公开状态", Slug: "attachment.update_visibility", Method: "PUT", Path: "/api/admin/attachments/*/visibility", Description: "更新附件公开/私有状态", Status: 1, Sort: 8, MenuID: attachmentMenu.ID},
		{Name: "附件删除", Slug: "attachment.destroy", Method: "DELETE", Path: "/api/admin/attachments/*", Description: "删除附件", Status: 1, Sort: 9, MenuID: attachmentMenu.ID},
		{Name: "附件批量删除", Slug: "attachment.batch_delete", Method: "POST", Path: "/api/admin/attachments/batch-delete", Description: "批量删除附件", Status: 1, Sort: 10, MenuID: attachmentMenu.ID},
		{Name: "附件分类列表", Slug: "attachment_category.index", Method: "GET", Path: "/api/admin/attachment-categories", Description: "查看附件分类列表", Status: 1, Sort: 11, MenuID: attachmentMenu.ID},
		{Name: "附件分类详情", Slug: "attachment_category.show", Method: "GET", Path: "/api/admin/attachment-categories/*", Description: "查看附件分类详情", Status: 1, Sort: 12, MenuID: attachmentMenu.ID},
		{Name: "附件分类创建", Slug: "attachment_category.store", Method: "POST", Path: "/api/admin/attachment-categories", Description: "创建附件分类", Status: 1, Sort: 13, MenuID: attachmentMenu.ID},
		{Name: "附件分类更新", Slug: "attachment_category.update", Method: "PUT", Path: "/api/admin/attachment-categories/*", Description: "更新附件分类", Status: 1, Sort: 14, MenuID: attachmentMenu.ID},
		{Name: "附件分类删除", Slug: "attachment_category.destroy", Method: "DELETE", Path: "/api/admin/attachment-categories/*", Description: "删除附件分类", Status: 1, Sort: 15, MenuID: attachmentMenu.ID},
		// Dashboard 统计
		{Name: "Dashboard数据", Slug: "dashboard.data", Method: "GET", Path: "/api/admin/dashboard/*", Description: "查看Dashboard统计数据", Status: 1, Sort: 1, MenuID: dashboardMenu.ID},
		// 通知管理
		{Name: "创建通知", Slug: "notification.store", Method: "POST", Path: "/api/admin/notifications", Description: "创建通知/公告/私信", Status: 1, Sort: 1, MenuID: notificationMenu.ID},
		// 订单管理
		{Name: "订单列表", Slug: "order.index", Method: "GET", Path: "/api/admin/orders", Description: "查看订单列表", Status: 1, Sort: 1, MenuID: orderMenu.ID},
		{Name: "订单详情", Slug: "order.show", Method: "GET", Path: "/api/admin/orders/*", Description: "查看订单详情", Status: 1, Sort: 2, MenuID: orderMenu.ID},
		{Name: "订单创建", Slug: "order.store", Method: "POST", Path: "/api/admin/orders", Description: "创建订单", Status: 1, Sort: 3, MenuID: orderMenu.ID},
		{Name: "订单更新", Slug: "order.update", Method: "PUT", Path: "/api/admin/orders/*", Description: "更新订单", Status: 1, Sort: 4, MenuID: orderMenu.ID},
		{Name: "订单删除", Slug: "order.destroy", Method: "DELETE", Path: "/api/admin/orders/*", Description: "删除订单", Status: 1, Sort: 5, MenuID: orderMenu.ID},
		{Name: "订单导出", Slug: "order.export", Method: "POST", Path: "/api/admin/orders/export", Description: "导出订单列表", Status: 1, Sort: 6, MenuID: orderMenu.ID},
		{Name: "订单导入", Slug: "order.import", Method: "POST", Path: "/api/admin/orders/import", Description: "导入订单列表", Status: 1, Sort: 7, MenuID: orderMenu.ID},
		{Name: "导入状态", Slug: "order.import_status", Method: "GET", Path: "/api/admin/imports/*", Description: "查看异步导入状态", Status: 1, Sort: 8, MenuID: orderMenu.ID},
		{Name: "导入错误文件", Slug: "order.import_error_file", Method: "GET", Path: "/api/admin/imports/*/error-file", Description: "下载导入失败行文件", Status: 1, Sort: 9, MenuID: orderMenu.ID},
		// 用户管理
		{Name: "用户列表", Slug: "user.index", Method: "GET", Path: "/api/admin/users", Description: "查看用户列表", Status: 1, Sort: 1, MenuID: userMenu.ID},
		{Name: "用户详情", Slug: "user.show", Method: "GET", Path: "/api/admin/users/*", Description: "查看用户详情", Status: 1, Sort: 2, MenuID: userMenu.ID},
		{Name: "用户创建", Slug: "user.store", Method: "POST", Path: "/api/admin/users", Description: "创建用户", Status: 1, Sort: 3, MenuID: userMenu.ID},
		{Name: "用户更新", Slug: "user.update", Method: "PUT", Path: "/api/admin/users/*", Description: "更新用户", Status: 1, Sort: 4, MenuID: userMenu.ID},
		{Name: "用户删除", Slug: "user.destroy", Method: "DELETE", Path: "/api/admin/users/*", Description: "删除用户", Status: 1, Sort: 5, MenuID: userMenu.ID},
		{Name: "用户重置密码", Slug: "user.password", Method: "PUT", Path: "/api/admin/users/*/password", Description: "重置用户密码", Status: 1, Sort: 6, MenuID: userMenu.ID},
		{Name: "更新余额", Slug: "user.update_balance", Method: "POST", Path: "/api/admin/users/*/update-balance", Description: "更新用户余额", Status: 1, Sort: 7, MenuID: userMenu.ID},
		// 用户余额变动记录
		{Name: "余额记录列表", Slug: "user_balance_log.index", Method: "GET", Path: "/api/admin/user-balance-logs", Description: "查看用户余额变动记录列表", Status: 1, Sort: 8, MenuID: userBalanceLogMenu.ID},
		{Name: "余额记录创建", Slug: "user_balance_log.store", Method: "POST", Path: "/api/admin/user-balance-logs", Description: "创建用户余额变动记录", Status: 1, Sort: 9, MenuID: userBalanceLogMenu.ID},
		{Name: "余额统计", Slug: "user_balance_log.statistics", Method: "GET", Path: "/api/admin/user-balance-logs/statistics", Description: "查看用户余额统计", Status: 1, Sort: 10, MenuID: userBalanceLogMenu.ID},
		// 支付方式管理
		{Name: "支付方式列表", Slug: "payment_method.index", Method: "GET", Path: "/api/admin/payment-methods", Description: "查看支付方式列表", Status: 1, Sort: 1, MenuID: paymentMethodMenu.ID},
		{Name: "支付方式详情", Slug: "payment_method.show", Method: "GET", Path: "/api/admin/payment-methods/*", Description: "查看支付方式详情", Status: 1, Sort: 2, MenuID: paymentMethodMenu.ID},
		{Name: "支付方式创建", Slug: "payment_method.store", Method: "POST", Path: "/api/admin/payment-methods", Description: "创建支付方式", Status: 1, Sort: 3, MenuID: paymentMethodMenu.ID},
		{Name: "支付方式更新", Slug: "payment_method.update", Method: "PUT", Path: "/api/admin/payment-methods/*", Description: "更新支付方式", Status: 1, Sort: 4, MenuID: paymentMethodMenu.ID},
		{Name: "支付方式删除", Slug: "payment_method.destroy", Method: "DELETE", Path: "/api/admin/payment-methods/*", Description: "删除支付方式", Status: 1, Sort: 5, MenuID: paymentMethodMenu.ID},
		// 支付记录管理
		{Name: "支付记录列表", Slug: "payment.index", Method: "GET", Path: "/api/admin/payments", Description: "查看支付记录列表", Status: 1, Sort: 1, MenuID: paymentRecordMenu.ID},
		{Name: "创建支付记录", Slug: "payment.store", Method: "POST", Path: "/api/admin/payments", Description: "为待支付订单创建支付单（参考下单）", Status: 1, Sort: 2, MenuID: paymentRecordMenu.ID},
		{Name: "支付记录详情", Slug: "payment.show", Method: "GET", Path: "/api/admin/payments/*", Description: "查看支付记录详情", Status: 1, Sort: 3, MenuID: paymentRecordMenu.ID},
		{Name: "支付结果查询", Slug: "payment.query", Method: "POST", Path: "/api/admin/payments/*/query", Description: "查询第三方/ mock 支付状态", Status: 1, Sort: 4, MenuID: paymentRecordMenu.ID},
	}

	for _, perm := range permissions {
		// 检查 slug 是否有效
		if perm.Slug == "" {
			facades.Log().Errorf("Permission has empty slug, skipping")
			continue
		}

		// 检查菜单ID是否有效
		if perm.MenuID == 0 {
			continue
		}

		// 使用 FirstOrCreate：如果 slug 或 name 存在则使用已存在的记录，否则创建
		facades.Orm().Query().Where("slug", perm.Slug).OrWhere("name", perm.Name).FirstOrCreate(&perm, perm)
	}

	return nil
}
