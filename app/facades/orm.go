package facades

import (
	"context"

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

// PlatformOrmQuery always uses the default (platform) connection — tenants metadata / DDL only.
func PlatformOrmQuery(ctx context.Context) orm.Query {
	defaultConn := facades.Config().GetString("database.default", "mysql")
	o := Orm().Connection(defaultConn)
	if ctx == nil {
		return o.Query()
	}
	return o.WithContext(ctx).Query()
}
