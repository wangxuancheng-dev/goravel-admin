package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260912000003AddDataScopeToRoles struct{}

func (r *M20260912000003AddDataScopeToRoles) Signature() string {
	return "20260912000003_add_data_scope_to_roles"
}

func (r *M20260912000003AddDataScopeToRoles) Up() error {
	if !facades.Schema().HasTable("roles") {
		return nil
	}
	columns, err := facades.Schema().GetColumns("roles")
	if err != nil {
		return err
	}
	hasDataScope := false
	for _, column := range columns {
		if column.Name == "data_scope" {
			hasDataScope = true
			break
		}
	}
	if !hasDataScope {
		if err := facades.Schema().Table("roles", func(table schema.Blueprint) {
			table.UnsignedTinyInteger("data_scope").Default(1).Comment("数据范围 1全部 2自定义 3本部门 4本部门及以下 5仅本人")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasTable("role_department") {
		return facades.Schema().Create("role_department", func(table schema.Blueprint) {
			table.UnsignedBigInteger("role_id")
			table.UnsignedBigInteger("department_id")
			table.Timestamps()
			table.Comment("角色自定义数据权限-部门")
			table.Unique("role_id", "department_id")
			table.Index("role_id")
			table.Index("department_id")
		})
	}
	return nil
}

func (r *M20260912000003AddDataScopeToRoles) Down() error {
	_ = facades.Schema().DropIfExists("role_department")
	if facades.Schema().HasTable("roles") && facades.Schema().HasColumn("roles", "data_scope") {
		return facades.Schema().Table("roles", func(table schema.Blueprint) {
			table.DropColumn("data_scope")
		})
	}
	return nil
}
