package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/modify"
)

func main() {
	setup := packages.Setup(os.Args)
	serviceProvider := "&feedback.ServiceProvider{}"
	// Local package in this monorepo; other apps should use their module import path.
	moduleImport := "feedbackpkg " + setup.Paths().Module().Import()

	setup.Install(
		modify.RegisterProvider(moduleImport, serviceProvider),
	).Uninstall(
		modify.UnregisterProvider(moduleImport, serviceProvider),
	).Execute()
}
