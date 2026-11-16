package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const header = `package p

import (
	"context"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/tenancy"
)
`

func rules(t *testing.T, path, body string) []string {
	t.Helper()
	fs, err := lintFile(path, []byte(header+body))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range fs {
		out = append(out, f.Rule)
	}
	return out
}

func expect(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("rules = %v, want %v", got, want)
	}
}

func TestOrmRule(t *testing.T) {
	expect(t, rules(t, "app/services/a.go", `
func f() { facades.Orm().Query().Count() }
func g() { appfacades.Orm().Connection("x") }
func h(ctx context.Context) { appfacades.OrmQuery(ctx).Count() }
func k() bool { return appfacades.Orm() == nil }
func assigned() {
	o := facades.Orm()
	if o == nil { return }
	o.Query().Count()
}
func varAssigned() {
	var o = facades.Orm()
	o.Query().Count()
}
`), "orm", "orm", "orm", "orm")
}

func TestNilCtxRule(t *testing.T) {
	expect(t, rules(t, "app/services/a.go", `
func f() { appfacades.OrmQuery(nil) }
func g() { appfacades.PlatformOrmQuery(nil) }
func h() { appfacades.PlatformQuery() }
func k() error { return appfacades.OrmTransaction(nil, nil) }
`), "nilctx", "nilctx", "nilctx")
}

func TestCacheRule(t *testing.T) {
	expect(t, rules(t, "app/services/a.go", `
type S struct{ ctx context.Context }

func (s *S) lockKey(id string) string { return tenancy.CacheKey(s.ctx, "lock:"+id) }

func direct(ctx context.Context) { facades.Cache().Put(tenancy.CacheKey(ctx, "k"), 1, 0) }
func (s *S) viaHelper() { facades.Cache().Forget(s.lockKey("1")) }
func global() { facades.Cache().Forget("k") }

// allowed is platform-wide.
//
//tenantlint:allow cache: platform key
func allowed() { facades.Cache().Forget("k") }

func line() {
	//tenantlint:allow cache: platform key
	facades.Cache().Forget("k")
}

func noReason() {
	//tenantlint:allow cache:
	facades.Cache().Forget("k")
}
`), "cache", "cache")
}

// A method named like a key helper on a call chain does not count as one.
func TestCacheRuleIgnoresChainMethodNames(t *testing.T) {
	expect(t, rules(t, "app/services/a.go", `
func Put(ctx context.Context) string { return tenancy.CacheKey(ctx, "k") }
func f() { facades.Cache().Put("k", 1, 0) }
`), "cache")
}

func TestGoCtxRule(t *testing.T) {
	expect(t, rules(t, "app/services/a.go", `
type S struct{ ctx http.Context }

func param(ctx context.Context) { go func() { _ = ctx }() }
func arg(ctx context.Context) { go run(ctx) }
func httpCtx(c http.Context) { go func() { _ = c }() }
func field(s *S) { go func() { _ = s.ctx }() }
func run(context.Context) {}

func inner() { go func() { ctx := context.Background(); _ = ctx }() }
func owned() {
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel
	go run(ctx)
}
func joined(ctx context.Context) {
	var wg interface{ Done() }
	go func() {
		defer wg.Done()
		_ = ctx
	}()
}
func notCtx(n int) { go func() { _ = n }() }
`), "goctx", "goctx", "goctx", "goctx")
}

func TestMigrationOrmHint(t *testing.T) {
	fs, err := lintFile("database/migrations/m.go", []byte(header+`func f() { facades.Orm().Query() }`))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || !strings.Contains(fs[0].Msg, "migrationQuery") {
		t.Fatalf("findings = %v", fs)
	}
}

// Code generator templates (not valid Go until rendered) must not emit what the rules
// forbid: every generated module would carry it.
func TestGeneratorTemplatesFollowTheRules(t *testing.T) {
	forbidden := []string{"facades.Orm()", "OrmQuery(nil)", "PlatformOrmQuery(nil)", "OrmTransaction(nil", "go func", "facades.Cache()"}
	paths, err := filepath.Glob("../../app/services/templates/*.tpl")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no templates found: %v", err)
	}
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range forbidden {
			if strings.Contains(string(src), f) {
				t.Errorf("%s contains %q", filepath.Base(p), f)
			}
		}
	}
}

// The repository itself must stay clean (CI also runs the command).
func TestRepositoryIsClean(t *testing.T) {
	fs, err := lintTree("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		t.Error(f)
	}
}
