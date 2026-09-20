package seeders

import (
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

type MenuSeeder struct {
}

func (s *MenuSeeder) Signature() string {
	return "MenuSeeder"
}

func (s *MenuSeeder) Run() error {
	hasLinkType := facades.Schema().HasColumn("menus", "link_type")
	hasOpenType := facades.Schema().HasColumn("menus", "open_type")
	hasNoCache := facades.Schema().HasColumn("menus", "no_cache")

	menuToMap := func(menuData models.Menu) map[string]any {
		now := time.Now().Format("2006-01-02 15:04:05")
		data := map[string]any{
			"parent_id":  menuData.ParentID,
			"title":      menuData.Title,
			"slug":       menuData.Slug,
			"icon":       menuData.Icon,
			"path":       menuData.Path,
			"component":  menuData.Component,
			"permission": menuData.Permission,
			"type":       menuData.Type,
			"status":     menuData.Status,
			"sort":       menuData.Sort,
			"is_hidden":  menuData.IsHidden,
			"created_at": now,
			"updated_at": now,
		}
		if hasLinkType {
			linkType := menuData.LinkType
			if linkType == 0 {
				linkType = 1
			}
			data["link_type"] = linkType
		}
		if hasOpenType {
			openType := menuData.OpenType
			if openType == 0 {
				openType = 1
			}
			data["open_type"] = openType
		}
		if hasNoCache {
			data["no_cache"] = menuData.NoCache
		}
		return data
	}

	// 根据Slug查找或创建菜单，如果存在则只更新 Component 字段，不存在则创建
	createOrUpdateMenu := func(menuData models.Menu) models.Menu {
		if menuData.Slug == "" {
			return menuData
		}

		var existingMenu models.Menu
		exists, _ := facades.Orm().Query().Model(&models.Menu{}).Where("slug", menuData.Slug).Exists()
		if exists {
			facades.Orm().Query().Where("slug", menuData.Slug).First(&existingMenu)

			hasUpdates := false
			updateData := map[string]any{}

			// 检查并更新 Component
			if menuData.Component != "" && menuData.Component != existingMenu.Component {
				updateData["component"] = menuData.Component
				hasUpdates = true
			}

			// 检查并更新 Path
			if menuData.Path != "" && menuData.Path != existingMenu.Path {
				updateData["path"] = menuData.Path
				hasUpdates = true
			}

			// 历史 seed 可能没有写入 created_at，这里补齐一次
			if existingMenu.CreatedAt == nil || existingMenu.CreatedAt.IsZero() {
				updateData["created_at"] = time.Now().Format("2006-01-02 15:04:05")
				hasUpdates = true
			}

			// 检查并更新 ParentID
			// if menuData.ParentID != existingMenu.ParentID {
			// 	existingMenu.ParentID = menuData.ParentID
			// 	hasUpdates = true
			// }

			// 检查并更新 Type
			// if menuData.Type != existingMenu.Type {
			// 	existingMenu.Type = menuData.Type
			// 	hasUpdates = true
			// }

			// 检查并更新 IsHidden
			// if menuData.IsHidden != existingMenu.IsHidden {
			// 	existingMenu.IsHidden = menuData.IsHidden
			// 	hasUpdates = true
			// }

			// 历史 seed / 部署菜单可能未写入 link_type，补齐默认值
			if hasLinkType && existingMenu.LinkType == 0 {
				updateData["link_type"] = uint8(1)
				hasUpdates = true
			}
			if hasOpenType && existingMenu.OpenType == 0 {
				updateData["open_type"] = uint8(1)
				hasUpdates = true
			}

			if hasUpdates {
				updateData["updated_at"] = time.Now().Format("2006-01-02 15:04:05")
				_, _ = facades.Orm().Query().Model(&models.Menu{}).Where("id", existingMenu.ID).Update(updateData)
				_ = facades.Orm().Query().Where("id", existingMenu.ID).First(&existingMenu)
			}

			return existingMenu
		}

		// 菜单不存在，创建新菜单
		_ = facades.Orm().Query().Table("menus").Create(menuToMap(menuData))
		var createdMenu models.Menu
		_ = facades.Orm().Query().Where("slug", menuData.Slug).First(&createdMenu)
		if createdMenu.ID > 0 {
			return createdMenu
		}
		return menuData
	}

	// 创建菜单
	systemMenu := createOrUpdateMenu(models.Menu{
		ParentID:  0,
		Title:     "系统管理",
		Slug:      "system",
		Icon:      "Setting",
		Path:      "/system",
		Component: "Layout",
		Type:      1,
		Status:    1,
		Sort:      1,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "管理员管理",
		Slug:      "admin",
		Icon:      "User",
		Path:      "/admins",
		Component: "admin/AdminList",
		Type:      2,
		Status:    1,
		Sort:      1,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "角色管理",
		Slug:      "role",
		Icon:      "UserFilled",
		Path:      "/roles",
		Component: "role/RoleList",
		Type:      2,
		Status:    1,
		Sort:      2,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "权限管理",
		Slug:      "permission",
		Icon:      "Lock",
		Path:      "/permissions",
		Component: "permission/PermissionList",
		Type:      2,
		Status:    1,
		Sort:      3,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "菜单管理",
		Slug:      "menu",
		Icon:      "Menu",
		Path:      "/menus",
		Component: "menu/MenuList",
		Type:      2,
		Status:    1,
		Sort:      4,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "部门管理",
		Slug:      "department",
		Icon:      "OfficeBuilding",
		Path:      "/departments",
		Component: "department/DepartmentList",
		Type:      2,
		Status:    1,
		Sort:      5,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "岗位管理",
		Slug:      "position",
		Icon:      "Briefcase",
		Path:      "/positions",
		Component: "position/PositionList",
		Type:      2,
		Status:    1,
		Sort:      6,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "在线管理员",
		Slug:      "online-admin",
		Icon:      "User",
		Path:      "/online-admins",
		Component: "onlineAdmin/OnlineAdminList",
		Type:      2,
		Status:    1,
		Sort:      7,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "字典管理",
		Slug:      "dictionary",
		Icon:      "Document",
		Path:      "/dictionaries",
		Component: "dictionary/DictionaryList",
		Type:      2,
		Status:    1,
		Sort:      8,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "配置管理",
		Slug:      "config",
		Icon:      "Setting",
		Path:      "/configs",
		Component: "config/ConfigList",
		Type:      2,
		Status:    1,
		Sort:      9,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "导入导出",
		Slug:      "export",
		Icon:      "Document",
		Path:      "/exports",
		Component: "export/TaskCenter",
		Type:      2,
		Status:    1,
		Sort:      10,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "附件管理",
		Slug:      "attachment",
		Icon:      "Folder",
		Path:      "/attachments",
		Component: "attachment/AttachmentList",
		Type:      2,
		Status:    1,
		Sort:      11,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "IP黑名单",
		Slug:      "blacklist",
		Icon:      "Warning",
		Path:      "/blacklists",
		Component: "blacklist/BlacklistList",
		Type:      2,
		Status:    1,
		Sort:      12,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "活动调度演示",
		Slug:      "demo-activity",
		Icon:      "Calendar",
		Path:      "/demo-activities",
		Component: "demoActivity/DemoActivityList",
		Type:      2,
		Status:    1,
		Sort:      14,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "订单管理",
		Slug:      "order",
		Icon:      "ShoppingCart",
		Path:      "/orders",
		Component: "order/OrderList",
		Type:      2,
		Status:    1,
		Sort:      13,
		IsHidden:  0,
	})

	// 创建支付管理菜单
	paymentMenu := createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "支付管理",
		Slug:      "payment",
		Icon:      "CreditCard",
		Path:      "/payments",
		Component: "Layout",
		Type:      1,
		Status:    1,
		Sort:      15,
		IsHidden:  0,
	})

	// 支付方式管理
	createOrUpdateMenu(models.Menu{
		ParentID:  paymentMenu.ID,
		Title:     "支付方式管理",
		Slug:      "payment-method",
		Icon:      "CreditCard",
		Path:      "/payment-methods",
		Component: "payment/PaymentMethodList",
		Type:      2,
		Status:    1,
		Sort:      1,
		IsHidden:  0,
	})

	// 支付记录管理
	createOrUpdateMenu(models.Menu{
		ParentID:  paymentMenu.ID,
		Title:     "支付记录管理",
		Slug:      "payment-record",
		Icon:      "Document",
		Path:      "/payment-records",
		Component: "payment/PaymentList",
		Type:      2,
		Status:    1,
		Sort:      2,
		IsHidden:  0,
	})

	userMenu := createOrUpdateMenu(models.Menu{
		ParentID:  systemMenu.ID,
		Title:     "用户管理",
		Slug:      "user",
		Icon:      "User",
		Path:      "/users",
		Component: "user/UserList",
		Type:      2,
		Status:    1,
		Sort:      14,
		IsHidden:  0,
	})

	// 创建用户余额变动记录菜单（隐藏，从用户列表跳转）
	createOrUpdateMenu(models.Menu{
		ParentID:  userMenu.ID,
		Title:     "用户余额变动记录",
		Slug:      "user-balance-log",
		Icon:      "Document",
		Path:      "/user-balance-logs",
		Component: "user/UserBalanceLogList",
		Type:      2,
		Status:    1,
		Sort:      1,
		IsHidden:  1, // 隐藏，不在菜单中显示
	})

	// 创建日志管理父菜单
	logMenu := createOrUpdateMenu(models.Menu{
		ParentID:  0,
		Title:     "日志管理",
		Slug:      "log",
		Icon:      "Document",
		Path:      "/logs",
		Component: "Layout",
		Type:      1,
		Status:    1,
		Sort:      2,
		IsHidden:  0,
	})

	// 创建日志管理子菜单
	createOrUpdateMenu(models.Menu{
		ParentID:  logMenu.ID,
		Title:     "操作日志",
		Slug:      "operation-log",
		Icon:      "Document",
		Path:      "/operation-logs",
		Component: "log/OperationLogList",
		Type:      2,
		Status:    1,
		Sort:      1,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  logMenu.ID,
		Title:     "登录日志",
		Slug:      "login-log",
		Icon:      "Document",
		Path:      "/login-logs",
		Component: "log/LoginLogList",
		Type:      2,
		Status:    1,
		Sort:      2,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  logMenu.ID,
		Title:     "系统日志",
		Slug:      "system-log",
		Icon:      "Document",
		Path:      "/system-logs",
		Component: "log/SystemLogList",
		Type:      2,
		Status:    1,
		Sort:      3,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  logMenu.ID,
		Title:     "日志与观测",
		Slug:      "observability",
		Icon:      "DataAnalysis",
		Path:      "/observability",
		Component: "log/ObservabilityHub",
		Type:      2,
		Status:    1,
		Sort:      4,
		IsHidden:  0,
	})

	// 服务监控（一级菜单）
	createOrUpdateMenu(models.Menu{
		ParentID:  0,
		Title:     "服务监控",
		Slug:      "monitor",
		Icon:      "Monitor",
		Path:      "/monitor",
		Component: "monitor/Monitor",
		Type:      2,
		Status:    1,
		Sort:      1,
		IsHidden:  0,
	})

	// 定时任务管理（一级菜单）
	createOrUpdateMenu(models.Menu{
		ParentID:  0,
		Title:     "定时任务",
		Slug:      "schedule",
		Icon:      "Timer",
		Path:      "/schedules",
		Component: "schedule/ScheduleList",
		Type:      2,
		Status:    1,
		Sort:      2,
		IsHidden:  0,
	})

	// 创建个人中心菜单
	createOrUpdateMenu(models.Menu{
		ParentID:  0,
		Title:     "个人中心",
		Slug:      "profile",
		Icon:      "User",
		Path:      "/profile",
		Component: "profile/Profile",
		Type:      2,
		Status:    1,
		Sort:      4,
		IsHidden:  1,
	})

	// 创建通知中心菜单
	createOrUpdateMenu(models.Menu{
		ParentID:  0,
		Title:     "通知中心",
		Slug:      "notification",
		Icon:      "Bell",
		Path:      "/notifications",
		Component: "notification/NotificationList",
		Type:      2,
		Status:    1,
		Sort:      5,
		IsHidden:  0,
	})

	// 创建 AI 实验室菜单（顶级；未配置 AI 时由运行时过滤隐藏）
	createOrUpdateMenu(models.Menu{
		ParentID:  0,
		Title:     "AI 实验室",
		Slug:      "ai_lab",
		Icon:      "Cpu",
		Path:      "/ai-lab",
		Component: "dev/AiLab",
		Type:      2,
		Status:    1,
		Sort:      98,
		IsHidden:  0,
	})

	// 创建开发工具菜单（是否展示由运行时过滤控制：开发模式/开发者管理员）
	devMenu := createOrUpdateMenu(models.Menu{
		ParentID:  0,
		Title:     "开发工具",
		Slug:      "dev",
		Icon:      "Tools",
		Path:      "/dev",
		Component: "Layout",
		Type:      1,
		Status:    1,
		Sort:      99,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  devMenu.ID,
		Title:     "代码生成器",
		Slug:      "code-generator",
		Icon:      "MagicStick",
		Path:      "/code-generator",
		Component: "dev/CodeGenerator",
		Type:      2,
		Status:    1,
		Sort:      1,
		IsHidden:  0,
	})

	createOrUpdateMenu(models.Menu{
		ParentID:  devMenu.ID,
		Title:     "表单组件演示",
		Slug:      "form_demo",
		Icon:      "EditPen",
		Path:      "/dev/form-demo",
		Component: "dev/FormDemo",
		Type:      2,
		Status:    1,
		Sort:      2,
		IsHidden:  0,
	})

	return nil
}
