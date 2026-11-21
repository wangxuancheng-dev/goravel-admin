package models

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// zeroIsInvalid lists numeric/bool fields that may keep a non-zero gorm default
// because 0 is never a value callers mean to store.
var zeroIsInvalid = map[string]string{
	"Menu.Type":                         "1 dir / 2 menu / 3 button; 0 is not a type",
	"Menu.LinkType":                     "1 internal / 2 external; 0 is not a type",
	"Menu.OpenType":                     "1 iframe / 2 new window; 0 is not a type",
	"Role.DataScope":                    "1..5; 0 is not a scope",
	"Currency.Rate":                     "an exchange rate of 0 is meaningless",
	"PlatformAlertDelivery.Attempt":     "attempts start at 1",
	"TenantEntitlementSnapshot.Version": "versions start at 1",
}

// TestNoNonZeroDefaultOnFlagFields guards against `gorm:"default:1"` (or true, 2, ...)
// on a numeric or bool field. gorm replaces a zero value with the tag default in Go
// before INSERT, so creating a row with Status 0 / IsPublic 0 / Enabled false silently
// stores the default: disabled admins came out enabled, private attachments public,
// failed operation logs recorded as successes.
func TestNoNonZeroDefaultOnFlagFields(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	pkgFiles, _ := filepath.Glob("../../packages/*/models/*.go")
	files = append(files, pkgFiles...)
	fset := token.NewFileSet()
	checked := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				if field.Tag == nil || len(field.Names) == 0 {
					continue
				}
				typ, ok := field.Type.(*ast.Ident)
				if !ok || !isNumericOrBool(typ.Name) {
					continue
				}
				raw, _ := strconv.Unquote(field.Tag.Value)
				def, ok := gormDefault(reflect.StructTag(raw).Get("gorm"))
				if !ok || isZeroLiteral(def) {
					continue
				}
				checked++
				key := ts.Name.Name + "." + field.Names[0].Name
				if _, allowed := zeroIsInvalid[key]; allowed {
					continue
				}
				t.Errorf("%s: %s has gorm default:%s; gorm stores that default whenever the field is 0/false, "+
					"so a deliberate 0/false can never be created. Drop the tag and set the value explicitly "+
					"(or add it to zeroIsInvalid if 0 is never a valid value)", fset.Position(field.Pos()), key, def)
			}
			return true
		})
	}
	if checked < len(zeroIsInvalid) {
		t.Fatalf("scanned only %d defaulted fields; is the test running in app/models?", checked)
	}
}

func gormDefault(tag string) (string, bool) {
	for _, part := range strings.Split(tag, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), ":")
		if ok && strings.EqualFold(strings.TrimSpace(k), "default") {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

func isZeroLiteral(v string) bool {
	switch strings.ToLower(strings.Trim(v, "'\"")) {
	case "", "0", "0.0", "false", "null":
		return true
	}
	return false
}

func isNumericOrBool(name string) bool {
	switch name {
	case "bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return true
	}
	return false
}
