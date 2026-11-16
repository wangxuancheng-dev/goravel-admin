package migrations

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"goravel/app/models"
)

// models.PlatformTables (what the tenant connection refuses) must be exactly the
// tables created by landlord-only migrations (Up guarded by SkipOnTenantConnection).
// A new platform table is added to both, or tenant code reading it fails / misses it.
func TestPlatformTablesMatchLandlordOnlyMigrations(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	created := map[string]bool{}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Up" || fn.Body == nil || !startsWithTenantSkip(fn.Body) {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Create" {
					return true
				}
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if s, err := strconv.Unquote(lit.Value); err == nil {
						created[s] = true
					}
				}
				return true
			})
		}
	}
	var fromMigrations []string
	for s := range created {
		fromMigrations = append(fromMigrations, s)
	}
	sort.Strings(fromMigrations)

	got := strings.Join(models.PlatformTables(), ",")
	want := strings.Join(fromMigrations, ",")
	if got != want {
		t.Fatalf("models.PlatformTables() = %s\nlandlord-only migrations create %s", got, want)
	}
}

// startsWithTenantSkip reports whether body begins with `if SkipOnTenantConnection() {`.
func startsWithTenantSkip(body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}
	ifs, ok := body.List[0].(*ast.IfStmt)
	if !ok {
		return false
	}
	call, ok := ifs.Cond.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := call.Fun.(*ast.Ident)
	return ok && id.Name == "SkipOnTenantConnection"
}
