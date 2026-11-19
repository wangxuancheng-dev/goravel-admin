package feedback

import (
	"context"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
)

// ResolveQuery lets the host inject a tenant-safe query (e.g. OrmQuery(ctx)).
// When nil, the package uses facades.Orm().Query() for single-DB Goravel apps.
var ResolveQuery func(ctx context.Context) orm.Query

// ResolveActorID optionally fills created_by from the host auth context.
var ResolveActorID func(ctx context.Context) uint

func query(ctx context.Context) orm.Query {
	if ResolveQuery != nil {
		return ResolveQuery(ctx)
	}
	return facades.Orm().Query()
}

func actorID(ctx context.Context) uint {
	if ResolveActorID != nil {
		return ResolveActorID(ctx)
	}
	return 0
}
