package providers

import (
	"github.com/goravel/framework/contracts/foundation"

	"goravel/app/console"
	"goravel/app/facades"
	"goravel/app/services"
)

type ConsoleServiceProvider struct {
}

func (receiver *ConsoleServiceProvider) Register(app foundation.Application) {
	kernel := console.Kernel{}
	facades.Schedule().Register(kernel.Schedule())
	facades.Artisan().Register(kernel.Commands())
	services.SetScheduleCommandDescriptions(console.CommandDescriptionMap())
}

func (receiver *ConsoleServiceProvider) Boot(app foundation.Application) {

}
