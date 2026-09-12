package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type TenantRestore struct{}

func (r *TenantRestore) Signature() string { return "tenant:restore" }
func (r *TenantRestore) Description() string {
	return "从 SQL 文件恢复租户库（依赖 mysql / psql 客户端）"
}
func (r *TenantRestore) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantRestore) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("需要 TENANCY_DRIVER=database")
		return nil
	}
	idOrCode := ctx.Argument(0)
	file := strings.TrimSpace(ctx.Argument(1))
	if idOrCode == "" || file == "" {
		ctx.Error("用法: tenant:restore {id|code} {sql文件路径}")
		return nil
	}
	if !filepath.IsAbs(file) {
		wd, _ := os.Getwd()
		file = filepath.Join(wd, file)
	}
	if _, err := os.Stat(file); err != nil {
		ctx.Error("文件不存在: " + file)
		return nil
	}
	tenant, err := services.NewTenantConnectionService().FindTenantByIDOrCode(idOrCode)
	if err != nil {
		ctx.Error("租户不存在: " + idOrCode)
		return nil
	}
	host, port, user, pass, database, err := services.ResolveTenantDSN(tenant)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	var cmd *exec.Cmd
	var cleanup func()
	switch strings.ToLower(tenant.Driver) {
	case models.TenantDriverPostgres, "pgsql", "postgresql":
		args, env := postgresRestoreArgs(host, port, user, pass, database, file, tenant)
		cmd = exec.Command("psql", args...)
		cmd.Env = env
	default:
		defaultsFile, err := services.WriteMySQLDefaultsFile(user, pass)
		if err != nil {
			ctx.Error(err.Error())
			return err
		}
		cleanup = func() { _ = os.Remove(defaultsFile) }
		args := []string{
			"--defaults-extra-file=" + defaultsFile,
			"-h", host,
			"-P", strconv.Itoa(port),
			database,
		}
		cmd = exec.Command("mysql", args...)
		f, err := os.Open(file)
		if err != nil {
			cleanup()
			ctx.Error(err.Error())
			return err
		}
		defer f.Close()
		cmd.Stdin = f
	}
	if cleanup != nil {
		defer cleanup()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Error(fmt.Sprintf("restore failed: %v\n%s", err, string(out)))
		return err
	}
	ctx.Success(fmt.Sprintf("已恢复租户 %s ← %s", tenant.Code, file))
	return nil
}

// postgresRestoreArgs builds psql argv/env; schema isolation sets search_path like backup -n.
func postgresRestoreArgs(host string, port int, user, pass, database, file string, tenant *models.Tenant) (args []string, env []string) {
	args = []string{"-h", host, "-p", strconv.Itoa(port), "-U", user, "-d", database, "-v", "ON_ERROR_STOP=1", "-f", file}
	env = append(os.Environ(), "PGPASSWORD="+pass)
	if tenant != nil && tenant.Isolation == models.TenantIsolationSchema {
		if schema := strings.TrimSpace(tenant.Schema); schema != "" {
			env = append(env, "PGOPTIONS=--search_path="+schema)
		}
	}
	return args, env
}

