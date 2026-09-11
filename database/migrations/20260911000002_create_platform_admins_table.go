package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260911000002CreatePlatformAdminsTable struct{}

func (r *M20260911000002CreatePlatformAdminsTable) Signature() string {
	return "20260911000002_create_platform_admins_table"
}

func (r *M20260911000002CreatePlatformAdminsTable) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().Create("platform_admins", func(table schema.Blueprint) {
		table.ID()
		table.String("username", 50).Comment("平台管理员用户名")
		table.String("password", 255).Comment("密码哈希")
		table.String("name", 100).Nullable().Comment("显示名")
		table.UnsignedTinyInteger("status").Default(1).Comment("1启用 0禁用")
		table.Timestamps()
		table.SoftDeletes()

		table.Unique("username")
		table.Index("status")
	})
}

func (r *M20260911000002CreatePlatformAdminsTable) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().DropIfExists("platform_admins")
}
