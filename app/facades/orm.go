package facades

import (
	"context"
	"errors"
	"fmt"
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
// Never nil: when the Orm is unavailable or the connection could not be built (the
// framework hands back an Orm with a nil query), it returns a stand-in whose calls
// fail with that error (see QueryUnavailable). A tenant ctx never falls back to the
// platform connection.
//
// With tenancy on, a ctx bound to no tenant (nil, context.Background(), the platform
// console's request) would silently read and write the platform database, so it fails
// closed with ErrNoTenantContext — unless the goroutine is bound to a tenant Schema
// (migrations / seeders), which routes it. Platform data goes through PlatformQuery /
// PlatformOrmQuery, which say so.
func OrmQuery(ctx context.Context) orm.Query {
	o := Orm()
	if o == nil {
		return newUnavailableQuery(ctx, errors.New("orm unavailable"))
	}
	if tenancy.Enabled() {
		if conn, ok := tenancyctx.ConnectionFrom(ctx); ok {
			to := BuildOrmConnection(conn)
			if to == nil {
				return newUnavailableQuery(ctx, apperrors.ErrTenantConnectionFailed.WithParams(map[string]any{"connection": conn}))
			}
			return orUnavailable(ctx, to.WithContext(ctx).Query(), conn)
		}
		if err := noTenantGuard("OrmQuery"); err != nil {
			return newUnavailableQuery(ctx, err)
		}
	}
	if ctx == nil {
		return orUnavailable(ctx, o.Query(), "default")
	}
	return orUnavailable(ctx, o.WithContext(ctx).Query(), "default")
}

// orUnavailable returns q, or the unavailable stand-in when the framework built none.
func orUnavailable(ctx context.Context, q orm.Query, conn string) orm.Query {
	if q != nil {
		return q
	}
	return newUnavailableQuery(ctx, fmt.Errorf("database connection %s unavailable", conn))
}

// OrmDriver returns the Goravel driver name of ctx's connection (mysql.Name "MySQL",
// postgres.Name "PostgreSQL"), or "" when the connection is unavailable.
func OrmDriver(ctx context.Context) string {
	return OrmQuery(ctx).Driver()
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
	if tenancy.Enabled() {
		if _, ok := tenancyctx.ConnectionFrom(ctx); !ok {
			if err := noTenantGuard("OrmTransaction"); err != nil {
				return err
			}
		}
	}
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

// ErrNoTenantContext is returned by OrmQuery / OrmTransaction, with tenancy on, for a
// ctx bound to no tenant on a goroutine bound to none (see OrmQuery).
var ErrNoTenantContext = errors.New("OrmQuery without a tenant (tenancy on): pass the request/job ctx bound to the tenant, or use PlatformQuery() / PlatformOrmQuery(ctx) for platform data")

// noTenantGuard returns ErrNoTenantContext (and logs the caller) unless the goroutine
// is bound to a tenant Schema (migrate / seed). Callers check tenancy and the ctx.
func noTenantGuard(fn string) error {
	if BoundTenantSchema() != nil {
		return nil
	}
	if l := facades.Log(); l != nil {
		l.Errorf("%s: %v", fn, ErrNoTenantContext)
	}
	return ErrNoTenantContext
}

// PlatformConnectionName returns the fixed landlord connection name.
// It is pinned on first use, so it never follows a tenant Schema bound to the
// goroutine during tenant migrate/seed (WithSchemaContext).
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

// PlatformQuery is PlatformOrmQuery without a ctx: the platform (landlord) database,
// for code that has no request/job ctx at hand (console commands, schedulers, tenant
// ops). Prefer PlatformOrmQuery(ctx) when a ctx exists, so cancellation and tracing
// follow the request. PlatformOrmQuery(nil) is rejected by tenantlint: write this.
func PlatformQuery() orm.Query {
	return PlatformOrmQuery(context.Background())
}

// PlatformOrmQuery always uses the pinned platform (landlord) connection — tenants metadata / DDL only.
// Safe to call inside WithSchemaContext (a tenant Schema bound to the goroutine for migrate/seed).
// Never nil, like OrmQuery.
func PlatformOrmQuery(ctx context.Context) orm.Query {
	name := PlatformConnectionName()
	po := BuildOrmConnection(name)
	if po == nil {
		return newUnavailableQuery(ctx, fmt.Errorf("database connection %s unavailable", name))
	}
	if ctx == nil {
		return orUnavailable(ctx, po.Query(), name)
	}
	return orUnavailable(ctx, po.WithContext(ctx).Query(), name)
}

// SystemLogOrmQuery picks the DB for system_logs writes/reads.
// Tenant-bound ctx -> tenant connection; otherwise landlord PlatformOrmQuery
// (never the tenant Schema a migrate/seed goroutine may be bound to).
// When the tenant connection is down, the log goes to the platform database: that
// failure is exactly what the log should record.
func SystemLogOrmQuery(ctx context.Context) orm.Query {
	if tenancy.Enabled() && !tenancy.Bound(ctx) {
		return PlatformOrmQuery(ctx)
	}
	q := OrmQuery(ctx)
	if down, _ := QueryUnavailable(q); down {
		return PlatformOrmQuery(ctx)
	}
	return q
}
