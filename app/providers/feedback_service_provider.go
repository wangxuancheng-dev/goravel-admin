package providers

import (
	"context"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/foundation"

	appfacades "goravel/app/facades"
	"goravel/app/rbac"
	feedbackpkg "goravel/packages/feedback"
)

// FeedbackServiceProvider wires the example Goravel package packages/feedback.
// It injects tenant-safe OrmQuery and the current admin id; menus/permissions stay in the host.
type FeedbackServiceProvider struct{}

func (r *FeedbackServiceProvider) Register(app foundation.Application) {}

func (r *FeedbackServiceProvider) Boot(app foundation.Application) {
	feedbackpkg.ResolveQuery = func(ctx context.Context) orm.Query {
		return appfacades.OrmQuery(ctx)
	}
	feedbackpkg.ResolveActorID = func(ctx context.Context) uint {
		if admin := rbac.AdminFromContext(ctx); admin != nil {
			return admin.ID
		}
		return 0
	}
}
