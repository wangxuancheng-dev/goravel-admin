package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

type M20250130000004AddCurrencyIdToUsersTable struct {
}

func (r *M20250130000004AddCurrencyIdToUsersTable) Signature() string {
	return "20250130000004_add_currency_id_to_users_table"
}

func (r *M20250130000004AddCurrencyIdToUsersTable) Up() error {
	if !facades.Schema().HasTable("users") {
		return nil
	}

	// 检查字段是否已存在
	if facades.Schema().HasColumn("users", "currency_id") {
		return nil
	}

	// 先查找人民币的ID（货币表应该已经创建）
	var cnyID uint
	var cnyCurrency models.Currency
	if facades.Schema().HasTable("currencies") {
		err := facades.Orm().Query().Table("currencies").Where("code", "CNY").First(&cnyCurrency)
		if err == nil {
			cnyID = cnyCurrency.ID
		}
	}

	// 添加 currency_id 字段
	if err := facades.Schema().Table("users", func(table schema.Blueprint) {
		col := table.UnsignedBigInteger("currency_id").Nullable().Comment("货币ID")
		// 如果找到了人民币ID，设置默认值
		if cnyID > 0 {
			col.Default(cnyID)
		}
		table.Index("currency_id")
	}); err != nil {
		return err
	}

	// 如果找到了人民币，更新所有现有用户的 currency_id 为人民币ID
	if cnyID > 0 {
		// 更新所有 currency_id 为 NULL 的用户
		if _, err := facades.Orm().Query().Table("users").Where("currency_id IS NULL").Update("currency_id", cnyID); err != nil {
			return err
		}
	}

	return nil
}

func (r *M20250130000004AddCurrencyIdToUsersTable) Down() error {
	if facades.Schema().HasTable("users") {
		return facades.Schema().Table("users", func(table schema.Blueprint) {
			table.DropColumn("currency_id")
		})
	}
	return nil
}
