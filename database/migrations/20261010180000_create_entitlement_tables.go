package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20261010180000CreateEntitlementTables struct{}

func (m *M20261010180000CreateEntitlementTables) Signature() string {
	return "20261010180000_create_entitlement_tables"
}

func (m *M20261010180000CreateEntitlementTables) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}

	if !facades.Schema().HasTable("platform_features") {
		if err := facades.Schema().Create("platform_features", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.String("key", 64)
			table.String("name", 100)
			table.String("description", 500).Nullable()
			table.String("type", 16).Default("boolean")
			table.String("default_value", 255).Default("false")
			table.Text("menu_slugs").Nullable()
			table.Text("channels").Nullable()
			table.Boolean("is_test").Default(false)
			table.Boolean("requires_install").Default(false)
			table.TinyInteger("status").Default(1)
			table.Integer("sort").Default(0)
			table.Timestamps()
			table.Unique("key")
			table.Index("status")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasTable("platform_plans") {
		if err := facades.Schema().Create("platform_plans", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.String("code", 64)
			table.String("name", 100)
			table.String("description", 500).Nullable()
			table.Boolean("is_default").Default(false)
			table.Boolean("is_public").Default(true)
			table.BigInteger("price_monthly").Default(0)
			table.BigInteger("price_yearly").Default(0)
			table.String("currency", 8).Default("CNY")
			table.TinyInteger("status").Default(1)
			table.Integer("sort").Default(0)
			table.Timestamps()
			table.Unique("code")
			table.Index("status")
			table.Index("is_default")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasTable("platform_plan_entitlements") {
		if err := facades.Schema().Create("platform_plan_entitlements", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("plan_id")
			table.String("feature_key", 64)
			table.String("value", 255)
			table.Timestamps()
			table.Unique("plan_id", "feature_key")
			table.Index("plan_id")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasTable("tenant_subscriptions") {
		if err := facades.Schema().Create("tenant_subscriptions", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("tenant_id")
			table.UnsignedBigInteger("plan_id")
			table.String("status", 32).Default("active")
			table.String("billing_cycle", 16).Default("manual")
			table.DateTime("starts_at").Nullable()
			table.DateTime("ends_at").Nullable()
			table.DateTime("trial_ends_at").Nullable()
			table.String("external_ref", 128).Nullable()
			table.String("note", 500).Nullable()
			table.Timestamps()
			table.Index("tenant_id")
			table.Index("plan_id")
			table.Index("status")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasTable("tenant_entitlement_overrides") {
		if err := facades.Schema().Create("tenant_entitlement_overrides", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("tenant_id")
			table.String("feature_key", 64)
			table.String("value", 255)
			table.Boolean("pinned").Default(true)
			table.DateTime("expires_at").Nullable()
			table.String("reason", 500).Nullable()
			table.UnsignedBigInteger("created_by").Default(0)
			table.DateTime("installed_at").Nullable()
			table.Timestamps()
			table.Unique("tenant_id", "feature_key")
			table.Index("expires_at")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasTable("tenant_entitlement_snapshots") {
		if err := facades.Schema().Create("tenant_entitlement_snapshots", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("tenant_id")
			table.BigInteger("version").Default(1)
			table.String("plan_code", 64).Nullable()
			table.LongText("payload")
			table.DateTime("computed_at")
			table.Timestamps()
			table.Unique("tenant_id")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasTable("tenant_usage_counters") {
		if err := facades.Schema().Create("tenant_usage_counters", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("tenant_id")
			table.String("feature_key", 64)
			table.String("period_key", 32).Default("lifetime")
			table.BigInteger("used").Default(0)
			table.Timestamps()
			table.Unique("tenant_id", "feature_key", "period_key")
		}); err != nil {
			return err
		}
	}

	if !facades.Schema().HasTable("platform_entitlement_audit_logs") {
		if err := facades.Schema().Create("platform_entitlement_audit_logs", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.UnsignedBigInteger("admin_id").Default(0)
			table.UnsignedBigInteger("tenant_id").Default(0)
			table.String("action", 64)
			table.Text("detail").Nullable()
			table.Timestamps()
			table.Index("admin_id")
			table.Index("tenant_id")
			table.Index("action")
		}); err != nil {
			return err
		}
	}

	return nil
}

func (m *M20261010180000CreateEntitlementTables) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	_ = facades.Schema().DropIfExists("platform_entitlement_audit_logs")
	_ = facades.Schema().DropIfExists("tenant_usage_counters")
	_ = facades.Schema().DropIfExists("tenant_entitlement_snapshots")
	_ = facades.Schema().DropIfExists("tenant_entitlement_overrides")
	_ = facades.Schema().DropIfExists("tenant_subscriptions")
	_ = facades.Schema().DropIfExists("platform_plan_entitlements")
	_ = facades.Schema().DropIfExists("platform_plans")
	_ = facades.Schema().DropIfExists("platform_features")
	return nil
}
