package http

import (
	"github.com/goravel/framework/contracts/route"
)

// RegisterAdminRoutes mounts feedback APIs on an already-authenticated admin router group.
// The host chooses middleware (Tenant, JWT, Permission, ...). No menus/permissions are installed.
func RegisterAdminRoutes(router route.Router) {
	c := NewController()
	router.Get("package-demos/feedbacks", c.Index)
	router.Post("package-demos/feedbacks", c.Store)
	router.Get("package-demos/feedbacks/{id}", c.Show)
	router.Post("package-demos/feedbacks/{id}/done", c.MarkDone)
	router.Delete("package-demos/feedbacks/{id}", c.Destroy)
}
