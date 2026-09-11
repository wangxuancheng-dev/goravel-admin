package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260911000001CreateTenantsTable struct{}

func (m *M20260911000001CreateTenantsTable) Signature() string {
	return "20260911000001_create_tenants_table"
}

func (m *M20260911000001CreateTenantsTable) Up() error {
	return facades.Schema().Create("tenants", func(table schema.Blueprint) {
		table.ID()
		table.String("code", 64).Comment("租户短码")
		table.String("name", 100).Comment("显示名称")
		table.UnsignedTinyInteger("status").Default(1).Comment("1启用 0禁用")
		table.String("driver", 20).Comment("mysql|postgres")
		table.String("isolation", 20).Comment("database|schema")
		table.String("host", 255).Nullable().Comment("空则回落平台 DB_HOST")
		table.Integer("port").Default(0).Comment("0 则回落平台 DB_PORT")
		table.String("database", 128).Comment("目标 database 名")
		table.String("schema", 128).Nullable().Comment("PG schema；database 隔离可空")
		table.String("username", 128).Nullable().Comment("空则回落平台用户")
		table.Text("password").Nullable().Comment("空则回落平台密码；存 APP_KEY 加密密文")
		table.String("connection_name", 64).Comment("运行时 connection 名")
		table.Timestamps()
		table.SoftDeletes()

		table.Unique("code")
		table.Unique("connection_name")
		table.Index("status")
	})
}

func (m *M20260911000001CreateTenantsTable) Down() error {
	return facades.Schema().DropIfExists("tenants")
}
