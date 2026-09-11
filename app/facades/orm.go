package facades

import (
	"context"
	"strings"
	"sync"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"

	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

func Orm() orm.Orm {
	return App().MakeOrm()
}

// OrmQuery returns an ORM query scoped to ctx.
// When tenancy is enabled and ctx carries tenant_connection, uses that connection.
// Business code should prefer OrmQuery(ctx) over Orm().Query().
func OrmQuery(ctx context.Context) orm.Query {
	if ctx == nil {
		return Orm().Query()
	}
	if tenancy.Enabled() {
		if conn, ok := tenancyctx.ConnectionFrom(ctx); ok {
			return Orm().Connection(conn).WithContext(ctx).Query()
		}
	}
	return Orm().WithContext(ctx).Query()
}

// OrmTransaction runs fn inside a DB transaction on the same connection OrmQuery(ctx) would use.
func OrmTransaction(ctx context.Context, fn func(tx orm.Query) error) error {
	if ctx == nil {
		return Orm().Transaction(fn)
	}
	if tenancy.Enabled() {
		if conn, ok := tenancyctx.ConnectionFrom(ctx); ok {
			return Orm().Connection(conn).WithContext(ctx).Transaction(fn)
		}
	}
	return Orm().WithContext(ctx).Transaction(fn)
}

var (
	platformConnOnce sync.Once
	platformConnName string
)

// ResetPlatformConnectionNameForTest clears the pinned platform connection (tests only).
func ResetPlatformConnectionNameForTest() {
	platformConnOnce = sync.Once{}
	platformConnName = ""
	facades.Config().Add("tenancy.platform_connection", "")
}

// PlatformConnectionName returns the fixed landlord connection name.
// It is pinned on first use and never follows a temporary database.default flip
// during tenant migrate/seed (WithTenantConnection).
func PlatformConnectionName() string {
	platformConnOnce.Do(func() {
		name := strings.TrimSpace(facades.Config().GetString("tenancy.platform_connection", ""))
		if name == "" || strings.HasPrefix(name, "tenant_") {
			name = strings.TrimSpace(facades.Config().GetString("database.default", "mysql"))
		}
		if name == "" || strings.HasPrefix(name, "tenant_") {
			name = "mysql"
		}
		platformConnName = name
		facades.Config().Add("tenancy.platform_connection", platformConnName)
	})
	return platformConnName
}

// PlatformOrmQuery always uses the pinned platform (landlord) connection — tenants metadata / DDL only.
// Safe to call while WithTenantConnection temporarily flips database.default for Artisan migrate/seed.
func PlatformOrmQuery(ctx context.Context) orm.Query {
	o := Orm().Connection(PlatformConnectionName())
	if ctx == nil {
		return o.Query()
	}
	return o.WithContext(ctx).Query()
}
