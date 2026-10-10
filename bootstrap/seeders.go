package bootstrap

import (
	"github.com/goravel/framework/contracts/database/seeder"

	"goravel/database/seeders"
)

func Seeders() []seeder.Seeder {
	return []seeder.Seeder{
		&seeders.DatabaseSeeder{},
		&seeders.PlatformPlanSeeder{}, // landlord plans (free/pro); skipped on tenant databases
		&seeders.MenuSeeder{},       // 菜单（需要先创建，因为权限依赖）
		&seeders.PermissionSeeder{}, // 权限（依赖菜单）
		&seeders.GeneratedModulesSeeder{}, // 代码生成器模块（依赖菜单）
		&seeders.PositionSeeder{},   // 岗位基础数据（管理员可选关联）
		&seeders.AdminSeeder{},      // 管理员、部门、角色（最后执行，关联权限和菜单）
		&seeders.DictionarySeeder{}, // 字典数据
		&seeders.CurrencySeeder{},   // 货币数据（需要在用户表之前创建）
	}
}
