package config

// Installs the goroutine-safe config wrapper before any config init() in this
// package runs (imported packages are initialized first). See app/support/safeconfig.
import _ "goravel/app/support/safeconfig"
