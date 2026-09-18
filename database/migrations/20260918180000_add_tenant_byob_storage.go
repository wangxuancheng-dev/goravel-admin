package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260918180000AddTenantByobStorage struct{}

func (m *M20260918180000AddTenantByobStorage) Signature() string {
	return "20260918180000_add_tenant_byob_storage"
}

func (m *M20260918180000AddTenantByobStorage) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenants", "storage_mode") {
			table.String("storage_mode", 16).Default("shared").Comment("shared|custom")
		}
		if !facades.Schema().HasColumn("tenants", "storage_driver") {
			table.String("storage_driver", 20).Default("").Comment("s3|oss|cos|minio when custom")
		}
		if !facades.Schema().HasColumn("tenants", "storage_key") {
			table.String("storage_key", 255).Default("").Comment("access key id")
		}
		if !facades.Schema().HasColumn("tenants", "storage_secret") {
			table.Text("storage_secret").Comment("APP_KEY sealed secret")
		}
		if !facades.Schema().HasColumn("tenants", "storage_region") {
			table.String("storage_region", 64).Default("").Comment("region")
		}
		if !facades.Schema().HasColumn("tenants", "storage_bucket") {
			table.String("storage_bucket", 255).Default("").Comment("bucket name")
		}
		if !facades.Schema().HasColumn("tenants", "storage_url") {
			table.String("storage_url", 512).Default("").Comment("public base url")
		}
		if !facades.Schema().HasColumn("tenants", "storage_endpoint") {
			table.String("storage_endpoint", 512).Default("").Comment("endpoint")
		}
		if !facades.Schema().HasColumn("tenants", "storage_use_path_style") {
			table.Boolean("storage_use_path_style").Default(false).Comment("S3 path-style")
		}
		if !facades.Schema().HasColumn("tenants", "storage_ssl") {
			table.Boolean("storage_ssl").Default(true).Comment("MinIO SSL")
		}
	})
}

func (m *M20260918180000AddTenantByobStorage) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	cols := []string{
		"storage_mode", "storage_driver", "storage_key", "storage_secret",
		"storage_region", "storage_bucket", "storage_url", "storage_endpoint",
		"storage_use_path_style", "storage_ssl",
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		for _, col := range cols {
			if facades.Schema().HasColumn("tenants", col) {
				table.DropColumn(col)
			}
		}
	})
}
