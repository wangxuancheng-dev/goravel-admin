package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260916120000CreateTenantDomainsTable struct{}

func (r *M20260916120000CreateTenantDomainsTable) Signature() string {
	return "20260916120000_create_tenant_domains_table"
}

func (r *M20260916120000CreateTenantDomainsTable) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if facades.Schema().HasTable("tenant_domains") {
		return nil
	}
	return facades.Schema().Create("tenant_domains", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("tenant_id").Comment("tenants.id")
		table.String("host", 255).Comment("normalized custom host")
		table.Boolean("is_primary").Default(false).Comment("primary vanity host")
		table.String("status", 32).Default("pending").Comment("pending|verified|active|disabled")
		table.String("ssl_mode", 32).Default("edge").Comment("edge|customer_cdn")
		table.String("verify_type", 32).Default("dns_txt").Comment("dns_txt")
		table.String("verify_token", 64).Comment("dns txt token")
		table.Timestamp("verified_at").Nullable()
		table.Timestamp("last_check_at").Nullable()
		table.Text("last_check_error").Nullable()
		table.Timestamps()
		table.SoftDeletes()
		table.Unique("host")
		table.Index("tenant_id")
		table.Index("status")
		table.Index("ssl_mode")
	})
}

func (r *M20260916120000CreateTenantDomainsTable) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().DropIfExists("tenant_domains")
}