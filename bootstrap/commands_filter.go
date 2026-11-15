package bootstrap

// ProductionCommandsAllowlist returns the artisan commands kept in production.
//
// WithCommandsFilter is a POSITIVE list: any command (framework or app) whose
// signature matches no entry is not registered at all. Since framework v1.18.1
// this also covers framework commands such as migrate, schedule:run and
// queue:retry, and scheduled commands that are filtered out silently never run.
// Every new app command prefix must be added here (TestProductionCommandsAllowlist
// checks this against the Kernel).
func ProductionCommandsAllowlist(allowMaintenance bool) []string {
	allow := []string{
		"about", "list", "key:generate",
		"schedule:*", "queue:*", "migrate*",
		"db:*", "cache:*", "config:*", "lang:*",
		"app:*", "order:*", "payment:*", "search:*", "token:*",
		"tenant:*", "platform:*",
		"activity:*", "flexible-schedule:*", "login_log:*", "operation_log:*",
	}
	// up/down are disabled in production unless APP_ALLOW_MAINTENANCE_COMMANDS=true.
	if allowMaintenance {
		allow = append(allow, "up", "down")
	}
	return allow
}
