package facades

import (
	"log"

	"goravel/packages/feedback"
	"goravel/packages/feedback/contracts"
)

// Feedback returns the bound feedback service (after ServiceProvider Register).
func Feedback() contracts.Service {
	if feedback.App == nil {
		return nil
	}
	instance, err := feedback.App.Make(feedback.Binding)
	if err != nil {
		log.Println(err)
		return nil
	}
	svc, ok := instance.(contracts.Service)
	if !ok {
		return nil
	}
	return svc
}
