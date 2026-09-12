package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260913000002AddMustChangePasswordToAdmins struct {
}

func (r *M20260913000002AddMustChangePasswordToAdmins) Signature() string {
	return "20260913000002_add_must_change_password_to_admins"
}

func (r *M20260913000002AddMustChangePasswordToAdmins) Up() error {
	if !facades.Schema().HasTable("admins") {
		return nil
	}
	if facades.Schema().HasColumn("admins", "must_change_password") {
		return nil
	}
	return facades.Schema().Table("admins", func(table schema.Blueprint) {
		table.UnsignedTinyInteger("must_change_password").Default(0).Comment("是否必须修改密码 1:是 0:否")
	})
}

func (r *M20260913000002AddMustChangePasswordToAdmins) Down() error {
	if !facades.Schema().HasTable("admins") {
		return nil
	}
	if !facades.Schema().HasColumn("admins", "must_change_password") {
		return nil
	}
	return facades.Schema().Table("admins", func(table schema.Blueprint) {
		table.DropColumn("must_change_password")
	})
}
