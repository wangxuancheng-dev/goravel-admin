package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// Finding is one rule violation.
type Finding struct {
	File string
	Line int
	Rule string
	Msg  string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s", f.File, f.Line, f.Rule, f.Msg)
}

const (
	pathFrameworkFacades = "github.com/goravel/framework/facades"
	pathAppFacades       = "goravel/app/facades"
	pathTenancy          = "goravel/app/tenancy"
	pathContext          = "context"
	pathHTTPContract     = "github.com/goravel/framework/contracts/http"
)

// Orm() methods that hand out a query on whatever connection the Orm points at.
var ormQueryMethods = map[string]bool{"Query": true, "Transaction": true, "WithContext": true, "Connection": true}

// Query helpers whose first argument is the routing ctx.
var ctxRoutedFuncs = map[string]bool{
	"OrmQuery": true, "PlatformOrmQuery": true, "OrmTransaction": true,
	"SystemLogOrmQuery": true, "OrmDriver": true,
}

// Names a captured request ctx usually has (used when the declared type is unknown).
var ctxNames = map[string]bool{"ctx": true, "reqCtx": true, "requestCtx": true, "httpCtx": true}

var allowRe = regexp.MustCompile(`tenantlint:allow\s+([a-z]+)\s*:\s*\S`)

type fileLint struct {
	path     string
	fset     *token.FileSet
	file     *ast.File
	imports  map[string]string // local name -> import path
	allows   map[int][]string  // line -> allowed rules
	keyFuncs map[string]bool   // package funcs / methods that build a tenancy.CacheKey
	findings []Finding
}

// srcFile is one parsed file of a package.
type srcFile struct {
	path string
	fset *token.FileSet
	file *ast.File
}

// lintPackage lints the files of one package (one directory). Key-building helpers
// are resolved package-wide: a function that calls lockKey(...), which returns
// tenancy.CacheKey(...), uses the tenant prefix.
func lintPackage(files map[string][]byte) ([]Finding, error) {
	var parsed []srcFile
	for path, src := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, srcFile{path: path, fset: fset, file: f})
	}
	keyFuncs := cacheKeyFuncs(parsed)
	var out []Finding
	for _, sf := range parsed {
		out = append(out, lintParsed(sf, keyFuncs)...)
	}
	return out, nil
}

// lintFile lints a single file as a package of its own (tests).
func lintFile(path string, src []byte) ([]Finding, error) {
	return lintPackage(map[string][]byte{path: src})
}

// cacheKeyFuncs returns the names of the package's functions and methods that call
// tenancy.CacheKey, directly or through another such function.
func cacheKeyFuncs(files []srcFile) map[string]bool {
	calls := map[string]map[string]bool{} // func name -> names it calls
	keys := map[string]bool{}
	for _, sf := range files {
		tenancyNames := map[string]bool{}
		for _, imp := range sf.file.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == pathTenancy {
				name := "tenancy"
				if imp.Name != nil {
					name = imp.Name.Name
				}
				tenancyNames[name] = true
			}
		}
		for _, decl := range sf.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			name := fn.Name.Name
			if calls[name] == nil {
				calls[name] = map[string]bool{}
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch f := call.Fun.(type) {
				case *ast.Ident:
					calls[name][f.Name] = true
				case *ast.SelectorExpr:
					if id, ok := f.X.(*ast.Ident); ok && id.Obj == nil && tenancyNames[id.Name] && f.Sel.Name == "CacheKey" {
						keys[name] = true
					} else if id, ok := f.X.(*ast.Ident); ok && id.Obj != nil {
						calls[name][f.Sel.Name] = true // method on a receiver / local value: s.lockKey(...)
					}
				}
				return true
			})
		}
	}
	for changed := true; changed; {
		changed = false
		for name, callees := range calls {
			if keys[name] {
				continue
			}
			for c := range callees {
				if keys[c] {
					keys[name] = true
					changed = true
					break
				}
			}
		}
	}
	return keys
}

func lintParsed(sf srcFile, keyFuncs map[string]bool) []Finding {
	path, fset, f := sf.path, sf.fset, sf.file
	l := &fileLint{path: path, fset: fset, file: f, imports: map[string]string{}, allows: map[int][]string{}, keyFuncs: keyFuncs}
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := p[strings.LastIndex(p, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		l.imports[name] = p
	}
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			for _, m := range allowRe.FindAllStringSubmatch(c.Text, -1) {
				line := fset.Position(c.Slash).Line
				l.allows[line] = append(l.allows[line], m[1])
			}
		}
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		l.lintFunc(fn)
	}
	return l.findings
}

func (l *fileLint) line(n ast.Node) int { return l.fset.Position(n.Pos()).Line }

// allowed reports whether rule is allowed at node n inside fn (same line, line
// above, or fn's doc comment).
func (l *fileLint) allowed(rule string, n ast.Node, fn *ast.FuncDecl) bool {
	ln := l.line(n)
	for _, at := range []int{ln, ln - 1} {
		for _, r := range l.allows[at] {
			if r == rule {
				return true
			}
		}
	}
	if fn != nil && fn.Doc != nil {
		for _, c := range fn.Doc.List {
			for _, m := range allowRe.FindAllStringSubmatch(c.Text, -1) {
				if m[1] == rule {
					return true
				}
			}
		}
	}
	return false
}

func (l *fileLint) report(rule string, n ast.Node, fn *ast.FuncDecl, format string, args ...any) {
	if l.allowed(rule, n, fn) {
		return
	}
	l.findings = append(l.findings, Finding{File: l.path, Line: l.line(n), Rule: rule, Msg: fmt.Sprintf(format, args...)})
}

// pkgCall reports whether call is <local>.Name(...) where local imports one of paths.
func (l *fileLint) pkgCall(call *ast.CallExpr, name string, paths ...string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok || id.Obj != nil { // a local variable, not a package
		return false
	}
	p := l.imports[id.Name]
	for _, want := range paths {
		if p == want {
			return true
		}
	}
	return false
}

func (l *fileLint) lintFunc(fn *ast.FuncDecl) {
	var cacheCalls []*ast.CallExpr
	usesCacheKey := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			l.checkOrm(n, fn)
			l.checkNilCtx(n, fn)
			if l.pkgCall(n, "Cache", pathFrameworkFacades, pathAppFacades) {
				cacheCalls = append(cacheCalls, n)
			}
			if l.pkgCall(n, "CacheKey", pathTenancy) || l.callsKeyFunc(n) {
				usesCacheKey = true
			}
		case *ast.GoStmt:
			l.checkGo(n, fn)
		}
		return true
	})

	if !usesCacheKey {
		for _, c := range cacheCalls {
			l.report("cache", c, fn, "facades.Cache() without tenancy.CacheKey(ctx, key) in %s: tenant keys need the tenant prefix", fn.Name.Name)
		}
	}
}

// callsKeyFunc reports whether call calls one of the package's key-building helpers.
func (l *fileLint) callsKeyFunc(call *ast.CallExpr) bool {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return l.keyFuncs[f.Name]
	case *ast.SelectorExpr:
		// Only s.lockKey(...) on a receiver / local value: not pkg.Func of another
		// package, and not a method of a call chain (facades.Cache().Put).
		if id, ok := f.X.(*ast.Ident); ok && id.Obj != nil {
			return l.keyFuncs[f.Sel.Name]
		}
	}
	return false
}

// orm: facades.Orm().Query() / .Transaction() / .WithContext() / .Connection().
func (l *fileLint) checkOrm(call *ast.CallExpr, fn *ast.FuncDecl) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !ormQueryMethods[sel.Sel.Name] {
		return
	}
	inner, ok := sel.X.(*ast.CallExpr)
	if !ok || !l.pkgCall(inner, "Orm", pathFrameworkFacades, pathAppFacades) {
		return
	}
	hint := "use appfacades.OrmQuery(ctx) for tenant data, PlatformQuery() / PlatformOrmQuery(ctx) for platform data"
	if strings.HasPrefix(l.path, "database/migrations/") {
		hint = "use migrationQuery(): facades.Orm() runs outside the migration's transaction"
	}
	l.report("orm", call, fn, "facades.Orm().%s(): %s", sel.Sel.Name, hint)
}

// nilctx: OrmQuery(nil) and friends.
func (l *fileLint) checkNilCtx(call *ast.CallExpr, fn *ast.FuncDecl) {
	if len(call.Args) == 0 {
		return
	}
	if id, ok := call.Args[0].(*ast.Ident); !ok || id.Name != "nil" {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !ctxRoutedFuncs[sel.Sel.Name] {
		return
	}
	if id, ok := sel.X.(*ast.Ident); !ok || l.imports[id.Name] != pathAppFacades {
		return
	}
	if sel.Sel.Name == "PlatformOrmQuery" {
		l.report("nilctx", call, fn, "PlatformOrmQuery(nil): write PlatformQuery(), or pass the ctx you have")
		return
	}
	l.report("nilctx", call, fn, "%s(nil): pass the request/job ctx (with tenancy on a nil ctx fails closed)", sel.Sel.Name)
}

// goctx: a go statement that captures or is handed a request ctx.
func (l *fileLint) checkGo(g *ast.GoStmt, fn *ast.FuncDecl) {
	const msg = "goroutine uses request ctx %q, which is cancelled when the request ends: use tenancyctx.Go(ctx, name, func(ctx context.Context) {...})"
	for _, arg := range g.Call.Args {
		if name, ok := l.ctxExpr(arg, nil); ok {
			l.report("goctx", g, fn, msg, name)
			return
		}
	}
	lit, ok := g.Call.Fun.(*ast.FuncLit)
	if !ok {
		return
	}
	if joinedByWaitGroup(lit) {
		return // the caller waits for it (wg.Wait): it does not outlive the request
	}
	var found string
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		if e, ok := n.(ast.Expr); ok {
			if name, ok := l.ctxExpr(e, lit); ok {
				found = name
				return false
			}
		}
		return true
	})
	if found != "" {
		l.report("goctx", g, fn, msg, found)
	}
}

// ctxExpr reports whether e is a request ctx from outside lit (lit nil: any): an
// identifier declared outside lit as context.Context / http.Context (or, with no
// declared type, named like one), or a field such as s.ctx / s.reqCtx.
func (l *fileLint) ctxExpr(e ast.Expr, lit *ast.FuncLit) (string, bool) {
	switch e := e.(type) {
	case *ast.Ident:
		if e.Obj == nil || e.Obj.Kind != ast.Var {
			return "", false
		}
		decl, ok := e.Obj.Decl.(ast.Node)
		if !ok {
			return "", false
		}
		if lit != nil && decl.Pos() >= lit.Pos() && decl.Pos() < lit.End() {
			return "", false // declared inside the goroutine
		}
		if typ := declType(e.Obj.Decl, e.Name); typ != nil {
			if l.isCtxType(typ) {
				return e.Name, true
			}
			return "", false
		}
		if l.madeByContextPkg(e.Obj.Decl) {
			return "", false // ctx, cancel := context.WithTimeout(...): this function owns its lifetime
		}
		return e.Name, ctxNames[e.Name]
	case *ast.SelectorExpr:
		if !ctxNames[e.Sel.Name] {
			return "", false
		}
		id, ok := e.X.(*ast.Ident)
		if !ok || id.Obj == nil {
			return "", false
		}
		if decl, ok := id.Obj.Decl.(ast.Node); ok && lit != nil && decl.Pos() >= lit.Pos() && decl.Pos() < lit.End() {
			return "", false
		}
		return id.Name + "." + e.Sel.Name, true
	}
	return "", false
}

// joinedByWaitGroup reports whether the goroutine body starts with `defer wg.Done()`.
func joinedByWaitGroup(lit *ast.FuncLit) bool {
	for _, st := range lit.Body.List {
		d, ok := st.(*ast.DeferStmt)
		if !ok {
			continue
		}
		if sel, ok := d.Call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Done" && len(d.Call.Args) == 0 {
			return true
		}
	}
	return false
}

// madeByContextPkg reports whether decl is `x, ... := context.Something(...)`.
func (l *fileLint) madeByContextPkg(decl any) bool {
	as, ok := decl.(*ast.AssignStmt)
	if !ok {
		return false
	}
	for _, rhs := range as.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Obj == nil && l.imports[id.Name] == pathContext {
				return true
			}
		}
	}
	return false
}

// declType returns the declared type expression of name in decl, or nil when the
// declaration does not state one (x := ...).
func declType(decl any, name string) ast.Expr {
	switch d := decl.(type) {
	case *ast.Field:
		return d.Type
	case *ast.ValueSpec:
		return d.Type
	}
	return nil
}

func (l *fileLint) isCtxType(t ast.Expr) bool {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Context" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	p := l.imports[id.Name]
	return p == pathContext || p == pathHTTPContract
}
