package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260920220000PlatformOpsExtensions struct{}

func (m *M20260920220000PlatformOpsExtensions) Signature() string {
	return "20260920220000_platform_ops_extensions"
}

func (m *M20260920220000PlatformOpsExtensions) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}

	if facades.Schema().HasTable("platform_admins") {
		if !facades.Schema().HasColumn("platform_admins", "google_secret") {
			if err := facades.Schema().Table("platform_admins", func(table schema.Blueprint) {
				table.String("google_secret", 255).Nullable().Comment("TOTP secret")
			}); err != nil {
				return err
			}
		}
		if !facades.Schema().HasColumn("platform_admins", "allowed_ips") {
			if err := facades.Schema().Table("platform_admins", func(table schema.Blueprint) {
				table.String("allowed_ips", 500).Nullable().Comment("comma-separated IPs/CIDRs; empty=any")
			}); err != nil {
				return err
			}
		}
	}

	if facades.Schema().HasTable("platform_alert_deliveries") {
		return nil
	}
	return facades.Schema().Create("platform_alert_deliveries", func(table schema.Blueprint) {
		table.ID()
		table.String("channel", 32).Comment("webhook|mail")
		table.String("event", 64).Comment("tenant_op_failed|tenant_health_inspect|...")
		table.UnsignedBigInteger("tenant_id").Nullable()
		table.String("tenant_code", 64).Nullable()
		table.String("op", 64).Nullable()
		table.String("status", 32).Default("pending").Comment("success|failed|pending")
		table.UnsignedInteger("http_status").Default(0)
		table.String("target_masked", 255).Nullable()
		table.Text("payload").Nullable()
		table.Text("error_message").Nullable()
		table.UnsignedInteger("attempt").Default(1)
		table.Timestamp("delivered_at").Nullable()
		table.Timestamps()
		table.Index("event")
		table.Index("tenant_id")
		table.Index("tenant_code")
		table.Index("status")
		table.Index("created_at")
	})
}

func (m *M20260920220000PlatformOpsExtensions) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	_ = facades.Schema().DropIfExists("platform_alert_deliveries")
	if facades.Schema().HasTable("platform_admins") {
		if facades.Schema().HasColumn("platform_admins", "allowed_ips") {
			_ = facades.Schema().Table("platform_admins", func(table schema.Blueprint) {
				table.DropColumn("allowed_ips")
			})
		}
		if facades.Schema().HasColumn("platform_admins", "google_secret") {
			_ = facades.Schema().Table("platform_admins", func(table schema.Blueprint) {
				table.DropColumn("google_secret")
			})
		}
	}
	return nil
}
