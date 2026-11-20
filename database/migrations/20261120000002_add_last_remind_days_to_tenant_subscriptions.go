package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

// M20261120000002AddLastRemindDaysToTenantSubscriptions remembers which expiry notice
// a subscription last got (days-before threshold, 0 = "ended"), so the hourly
// tenant:subscription-expiry job sends each notice once.
type M20261120000002AddLastRemindDaysToTenantSubscriptions struct{}

func (r *M20261120000002AddLastRemindDaysToTenantSubscriptions) Signature() string {
	return "20261120000002_add_last_remind_days_to_tenant_subscriptions"
}

func (r *M20261120000002AddLastRemindDaysToTenantSubscriptions) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenant_subscriptions") || facades.Schema().HasColumn("tenant_subscriptions", "last_remind_days") {
		return nil
	}
	return facades.Schema().Table("tenant_subscriptions", func(table schema.Blueprint) {
		table.Integer("last_remind_days").Nullable().Comment("last expiry notice sent: days-before threshold, 0 = ended; null = none")
	})
}

func (r *M20261120000002AddLastRemindDaysToTenantSubscriptions) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenant_subscriptions") || !facades.Schema().HasColumn("tenant_subscriptions", "last_remind_days") {
		return nil
	}
	return facades.Schema().Table("tenant_subscriptions", func(table schema.Blueprint) {
		table.DropColumn("last_remind_days")
	})
}
