package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
)

// Every platform write route must get a "platform.*" title, and both consoles must
// translate it (platform_op.* in each locale), or the operation log shows raw
// "POST /api/platform/..." or an untranslated key.
func TestPlatformOperationTitleCoversRoutesAndLocales(t *testing.T) {
	root := filepath.Join("..", "..")
	src, err := os.ReadFile(filepath.Join(root, "routes", "platform.go"))
	require.NoError(t, err)

	routeRe := regexp.MustCompile(`\.(Post|Put|Delete|Patch)\("([^"]+)"`)
	paramRe := regexp.MustCompile(`\{[^}]+\}`)
	matches := routeRe.FindAllStringSubmatch(string(src), -1)
	require.NotEmpty(t, matches)

	locales := map[string]map[string]any{}
	for _, app := range []string{"html", "html-react"} {
		for _, lang := range []string{"zh-CN", "en-US"} {
			file := filepath.Join(root, app, "src", "i18n", "locales", lang+".json")
			raw, err := os.ReadFile(file)
			require.NoError(t, err, file)
			var tree map[string]any
			require.NoError(t, json.Unmarshal(raw, &tree), file)
			locales[app+"/"+lang] = tree
		}
	}

	for _, m := range matches {
		method, route := strings.ToUpper(m[1]), m[2]
		if route == "login" {
			continue // logged in the login log, not the operation log
		}
		path := "/api/platform/" + paramRe.ReplaceAllString(route, "1")
		title := PlatformOperationTitle(method, path)
		if !assert.True(t, strings.HasPrefix(title, "platform."), "%s %s has no title key (got %q)", method, path, title) {
			continue
		}
		key := "platform_op." + strings.TrimPrefix(title, "platform.")
		for name, tree := range locales {
			assert.True(t, hasLocaleString(tree, key), "%s: missing %s (%s %s)", name, key, method, path)
		}
	}
}

func hasLocaleString(tree map[string]any, key string) bool {
	var node any = tree
	for _, part := range strings.Split(key, ".") {
		obj, ok := node.(map[string]any)
		if !ok {
			return false
		}
		node, ok = obj[part]
		if !ok {
			return false
		}
	}
	s, ok := node.(string)
	return ok && s != ""
}

func TestPlatformOperationLogToJSONMapsLegacyFallbackTitle(t *testing.T) {
	row := &models.PlatformOperationLog{
		Method: "POST",
		Path:   "/api/platform/login-logs/refresh-location",
		Title:  "POST /api/platform/login-logs/refresh-location",
	}
	assert.Equal(t, "platform.login_log.refresh_location", PlatformOperationLogToJSON(row)["title"])

	row.Title = "platform.tenant.ping"
	assert.Equal(t, "platform.tenant.ping", PlatformOperationLogToJSON(row)["title"])
}
