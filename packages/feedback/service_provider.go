package feedback

import (
	"github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/foundation"
)

const Binding = "feedback"

// App is set during Register for facades.Feedback().
var App foundation.Application

type ServiceProvider struct{}

func (r *ServiceProvider) Relationship() binding.Relationship {
	return binding.Relationship{
		Bindings:     []string{Binding},
		Dependencies: []string{},
		ProvideFor:   []string{},
	}
}

func (r *ServiceProvider) Register(app foundation.Application) {
	App = app
	app.Bind(Binding, func(app foundation.Application) (any, error) {
		return NewFeedback(), nil
	})
}

func (r *ServiceProvider) Boot(app foundation.Application) {
	// Publish migration for other Goravel apps:
	//   ./artisan vendor:publish --package=goravel/packages/feedback
	app.Publishes("goravel/packages/feedback", map[string]string{
		"migrations": app.DatabasePath("migrations"),
	}, "feedback-migrations")
}
