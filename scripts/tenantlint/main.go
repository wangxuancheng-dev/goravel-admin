// Command tenantlint enforces the tenancy coding rules that the compiler cannot:
// every rule below is a way to write code that runs fine in single-database mode and
// leaks across tenants (or onto the platform database) with TENANCY_DRIVER=database.
//
//	go run ./scripts/tenantlint            # from the repository root; exit 1 on findings
//
// Rules (rule names are what an allow comment names):
//
//	orm     facades.Orm().Query/Transaction/WithContext/Connection in app code.
//	        Use appfacades.OrmQuery(ctx) (tenant data) or PlatformQuery() /
//	        PlatformOrmQuery(ctx) (platform data). Migrations use migrationQuery().
//	nilctx  OrmQuery(nil) / PlatformOrmQuery(nil) / OrmTransaction(nil) / ...
//	        A nil ctx says nothing about which database is meant: pass the ctx, or
//	        call PlatformQuery() for platform data.
//	cache   facades.Cache() in a function that never calls tenancy.CacheKey: a tenant
//	        key without the tenant prefix is read by every tenant.
//	goctx   a `go` statement that captures or is handed a request ctx (context.Context,
//	        http.Context, s.ctx ...). The request ctx is cancelled (and Goravel's
//	        http.Context recycled) when the request returns. Use tenancyctx.Go.
//
// A finding that is intended is kept with a comment on the line, the line above, or
// (for every finding in a function) the function's doc comment:
//
//	//tenantlint:allow cache: platform-wide key, not tenant data
//
// The reason after the colon is required.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Directories checked, relative to the repository root. Infrastructure that
// implements the rules (app/facades, app/tenancyctx, app/tenantdb, app/tenantcache,
// app/providers) is out of scope.
var (
	scanDirs = []string{"app", "database/migrations"}
	skipDirs = []string{
		"app/facades",
		"app/providers",
		"app/tenancyctx",
		"app/tenantcache",
		"app/tenantdb",
	}
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	findings, err := lintTree(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tenantlint:", err)
		os.Exit(2)
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	if len(findings) > 0 {
		fmt.Printf("\ntenantlint: %d finding(s). See the rules at the top of scripts/tenantlint/main.go;\n", len(findings))
		fmt.Println("keep an intended one with  //tenantlint:allow <rule>: <reason>")
		os.Exit(1)
	}
	fmt.Println("tenantlint: OK")
}

func lintTree(root string) ([]Finding, error) {
	pkgs := map[string]map[string][]byte{} // dir -> rel path -> source
	for _, dir := range scanDirs {
		base := filepath.Join(root, dir)
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				for _, s := range skipDirs {
					if rel == s {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			dir := filepath.ToSlash(filepath.Dir(rel))
			if pkgs[dir] == nil {
				pkgs[dir] = map[string][]byte{}
			}
			pkgs[dir][rel] = src
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	var all []Finding
	for _, files := range pkgs {
		fs, err := lintPackage(files)
		if err != nil {
			return nil, err
		}
		all = append(all, fs...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		return all[i].Line < all[j].Line
	})
	return all, nil
}
