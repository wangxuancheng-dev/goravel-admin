package seeders

import (
	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

type AdminSeeder struct {
}

func (s *AdminSeeder) Signature() string {
	return "AdminSeeder"
}

func (s *AdminSeeder) Run() error {
	hasGoogleSecret := facades.Schema().HasColumn("admins", "google_secret")
	hasPositionID := facades.Schema().HasColumn("admins", "position_id")

	adminToMap := func(username, password, nickname string) map[string]any {
		data := map[string]any{
			"username": username,
			"password": password,
			"nickname": nickname,
			"status":   1,
		}
		if hasGoogleSecret {
			data["google_secret"] = ""
		}
		if hasPositionID {
			data["position_id"] = 0
		}
		return data
	}

	// 创建超级管理员
	var superAdmin models.Admin
	exists, _ := facades.Orm().Query().Model(&models.Admin{}).Where("username", "admin").Exists()
	if !exists {
		hashedPassword, _ := facades.Hash().Make("admin123")
		superAdmin = models.Admin{
			Username: "admin",
			Password: hashedPassword,
			Nickname: "超级管理员",
			Status:   1,
		}
		_ = facades.Orm().Query().Table("admins").Create(adminToMap(superAdmin.Username, superAdmin.Password, superAdmin.Nickname))
		_ = facades.Orm().Query().Where("username", "admin").First(&superAdmin)
	} else {
		_ = facades.Orm().Query().Where("username", "admin").First(&superAdmin)
	}

	// 创建开发者管理员（受保护，不显示在列表中）
	var developerAdmin models.Admin
	exists, _ = facades.Orm().Query().Model(&models.Admin{}).Where("username", "developer").Exists()
	if !exists {
		developerPassword, _ := facades.Hash().Make("developer123")
		developerAdmin = models.Admin{
			Username: "developer",
			Password: developerPassword,
			Nickname: "开发者管理员",
			Status:   1,
		}
		_ = facades.Orm().Query().Table("admins").Create(adminToMap(developerAdmin.Username, developerAdmin.Password, developerAdmin.Nickname))
		_ = facades.Orm().Query().Where("username", "developer").First(&developerAdmin)
	} else {
		_ = facades.Orm().Query().Where("username", "developer").First(&developerAdmin)
	}

	// 创建部门
	var rootDept models.Department
	exists, _ = facades.Orm().Query().Model(&models.Department{}).Where("code", "ROOT").Exists()
	if !exists {
		rootDept = models.Department{
			Name:   "总公司",
			Code:   "ROOT",
			Status: 1,
			Sort:   0,
		}
		facades.Orm().Query().Create(&rootDept)
	} else {
		facades.Orm().Query().Where("code", "ROOT").First(&rootDept)
	}

	var itDept models.Department
	exists, _ = facades.Orm().Query().Model(&models.Department{}).Where("code", "IT").Exists()
	if !exists {
		itDept = models.Department{
			ParentID: rootDept.ID,
			Name:     "技术部",
			Code:     "IT",
			Status:   1,
			Sort:     1,
		}
		facades.Orm().Query().Create(&itDept)
	} else {
		facades.Orm().Query().Where("code", "IT").First(&itDept)
	}

	// 创建角色
	var superRole models.Role
	exists, _ = facades.Orm().Query().Model(&models.Role{}).Where("slug", "super-admin").Exists()
	if !exists {
		superRole = models.Role{
			Name:        "超级管理员",
			Slug:        "super-admin",
			Description: "拥有所有权限",
			Status:      1,
			Sort:        0,
			DataScope:   models.DataScopeAll,
		}
		facades.Orm().Query().Create(&superRole)
	} else {
		facades.Orm().Query().Where("slug", "super-admin").First(&superRole)
		if superRole.DataScope == 0 || superRole.DataScope != models.DataScopeAll {
			_, _ = facades.Orm().Query().Model(&superRole).Update("data_scope", models.DataScopeAll)
			superRole.DataScope = models.DataScopeAll
		}
	}

	var adminRole models.Role
	exists, _ = facades.Orm().Query().Model(&models.Role{}).Where("slug", "admin").Exists()
	if !exists {
		adminRole = models.Role{
			Name:        "管理员",
			Slug:        "admin",
			Description: "普通管理员",
			Status:      1,
			Sort:        1,
			DataScope:   models.DataScopeAll,
		}
		facades.Orm().Query().Create(&adminRole)
	} else {
		facades.Orm().Query().Where("slug", "admin").First(&adminRole)
	}

	// 关联超级管理员和超级角色（增量添加，不覆盖已有角色）
	assignRole := func(adminID, roleID uint) {
		count, err := facades.Orm().Query().Table("admin_role").Where("admin_id", adminID).Where("role_id", roleID).Count()
		if err == nil && count == 0 {
			_ = facades.Orm().Query().Table("admin_role").Create(map[string]any{"admin_id": adminID, "role_id": roleID})
		}
	}
	assignRole(superAdmin.ID, superRole.ID)
	assignRole(developerAdmin.ID, superRole.ID)

	// super-admin 角色不需要分配权限和菜单，因为它在权限中间件中会跳过权限检查
	// 在获取用户信息时，会特殊处理 super-admin 角色，返回所有菜单用于前端显示

	// 关联默认岗位（若已存在岗位数据）
	var gmPosition models.Position
	if hasPositionID && facades.Schema().HasTable("positions") {
		if err := facades.Orm().Query().Where("code", "GM").First(&gmPosition); err == nil && gmPosition.ID > 0 {
		_, _ = facades.Orm().Query().Model(&superAdmin).Update("position_id", gmPosition.ID)
		_, _ = facades.Orm().Query().Model(&developerAdmin).Update("position_id", gmPosition.ID)
		}
	}

	return nil
}
