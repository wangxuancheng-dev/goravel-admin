package facades

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
	"goravel/app/tenantdb"
)

func Orm() orm.Orm {
	return App().MakeOrm()
}

// OrmQuery returns an ORM query scoped to ctx.
// When tenancy is enabled and ctx carries tenant_connection, uses that connection.
// Business code should prefer OrmQuery(ctx) over Orm().Query().
//
// May return nil when Orm is unavailable or the tenant connection failed to init
// (framework Connection returns an Orm with a nil query on BuildQuery error).
func OrmQuery(ctx context.Context) orm.Query {
	o := Orm()
	if o == nil {
		return nil
	}
	if ctx == nil {
		return o.Query()
	}
	if tenancy.Enabled() {
		if conn, ok := tenancyctx.ConnectionFrom(ctx); ok {
			to := BuildOrmConnection(conn)
			if to == nil {
				return nil
			}
			return to.WithContext(ctx).Query()
		}
	}
	return o.WithContext(ctx).Query()
}

// BuildOrmConnection returns the Orm of a named connection. Tenant connections come
// from the app-owned registry (tenantdb), built on demand; other names (the platform
// connection) from the framework. Returns nil when the connection cannot be opened.
// Lookups of registered connections take no process-wide lock.
func BuildOrmConnection(name string) orm.Orm {
	if c, err := tenantdb.Default.Conn(name); err == nil {
		return c.Orm()
	} else if !errors.Is(err, tenantdb.ErrUnknownConnection) {
		logTenantConnError(name, err)
		return nil
	}
	o := Orm()
	if o == nil {
		return nil
	}
	if r, ok := o.(*ormRouter); ok {
		return r.base.Connection(name)
	}
	return o.Connection(name)
}

// ConnectionDatabaseName returns the database a connection points at: the tenant
// database for a tenant connection, else database.connections.<name>.database.
func ConnectionDatabaseName(name string) string {
	if c, err := tenantdb.Default.Conn(name); err == nil {
		return c.DatabaseName()
	}
	return facades.Config().GetString("database.connections." + name + ".database")
}

func logTenantConnError(name string, err error) {
	if l := facades.Log(); l != nil {
		l.Errorf("tenant connection %s: %v", name, err)
	}
}

// OrmTransaction runs fn inside a DB transaction on the same connection OrmQuery(ctx) would use.
// When ctx carries a tenant connection, never falls back to the platform DB (fail-closed).
func OrmTransaction(ctx context.Context, fn func(tx orm.Query) error) error {
	if ctx == nil {
		return Orm().Transaction(fn)
	}
	if tenancy.Enabled() {
		if conn, ok := tenancyctx.ConnectionFrom(ctx); ok {
			to := BuildOrmConnection(conn)
			if to == nil {
				// Match OrmQuery: do not write to landlord DB when tenant bind failed.
				return apperrors.ErrTenantConnectionFailed
			}
			return to.WithContext(ctx).Transaction(fn)
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
	po := BuildOrmConnection(PlatformConnectionName())
	if po == nil {
		return nil
	}
	if ctx == nil {
		return po.Query()
	}
	return po.WithContext(ctx).Query()
}

// SystemLogOrmQuery picks the DB for system_logs writes/reads.
// Tenant-bound ctx -> tenant connection; otherwise landlord PlatformOrmQuery
// (avoids writing into a flipped database.default during WithTenantConnection).
func SystemLogOrmQuery(ctx context.Context) orm.Query {
	if tenancy.Enabled() && !tenancy.Bound(ctx) {
		return PlatformOrmQuery(ctx)
	}
	q := OrmQuery(ctx)
	if q == nil {
		return PlatformOrmQuery(ctx)
	}
	return q
}
