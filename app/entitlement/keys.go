package entitlement

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// ModuleKeyPrefix is the stable key namespace for generated CRUD modules.
	ModuleKeyPrefix = "module."

	// Source identifiers used in audit / grant provenance.
	SourcePlan     = "plan"
	SourceAddon    = "addon"
	SourceOverride = "override"
	SourceDefault  = "default"
)

var featureKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.]{1,62}$`)

// NormalizeModuleName lowercases and normalizes module identifiers.
func NormalizeModuleName(moduleName string) string {
	name := strings.ToLower(strings.TrimSpace(moduleName))
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

// ModuleFeatureKey builds module.{name} for a generated or hand-written module.
func ModuleFeatureKey(moduleName string) string {
	return ModuleKeyPrefix + NormalizeModuleName(moduleName)
}

// ModuleRowsLimitKey builds quota.module.{name}.rows for optional row quotas.
func ModuleRowsLimitKey(moduleName string) string {
	return "quota.module." + NormalizeModuleName(moduleName) + ".rows"
}

// IsModuleFeatureKey reports whether key uses the module.* namespace.
func IsModuleFeatureKey(key string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(key)), ModuleKeyPrefix)
}

// ValidateFeatureKey checks catalog key shape.
func ValidateFeatureKey(key string) error {
	key = strings.ToLower(strings.TrimSpace(key))
	if !featureKeyPattern.MatchString(key) {
		return fmt.Errorf("invalid feature key %q", key)
	}
	return nil
}
