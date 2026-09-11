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
	host, port, user, pass, database := resolveTenantDSN(tenant)
	var cmd *exec.Cmd
	switch strings.ToLower(tenant.Driver) {
	case models.TenantDriverPostgres, "pgsql", "postgresql":
		args := []string{"-h", host, "-p", strconv.Itoa(port), "-U", user, "-d", database, "-f", file}
		cmd = exec.Command("psql", args...)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	default:
		args := []string{"-h", host, "-P", strconv.Itoa(port), "-u", user, database}
		if pass != "" {
			args = append([]string{"-p" + pass}, args...)
		}
		cmd = exec.Command("mysql", args...)
		f, err := os.Open(file)
		if err != nil {
			ctx.Error(err.Error())
			return err
		}
		defer f.Close()
		cmd.Stdin = f
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Error(fmt.Sprintf("restore failed: %v\n%s", err, string(out)))
		return err
	}
	ctx.Success(fmt.Sprintf("已恢复租户 %s ← %s", tenant.Code, file))
	return nil
}
