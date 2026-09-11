package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type TenantBackup struct{}

func (r *TenantBackup) Signature() string { return "tenant:backup" }
func (r *TenantBackup) Description() string {
	return "备份指定租户数据库到 storage/backups/tenants/{code}/（依赖 mysqldump/pg_dump）"
}
func (r *TenantBackup) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantBackup) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("需要 TENANCY_DRIVER=database")
		return nil
	}
	idOrCode := ctx.Argument(0)
	if idOrCode == "" {
		ctx.Error("用法: tenant:backup {id|code}")
		return nil
	}
	tenant, err := services.NewTenantConnectionService().FindTenantByIDOrCode(idOrCode)
	if err != nil {
		ctx.Error("租户不存在: " + idOrCode)
		return nil
	}
	dir := filepath.Join("storage", "backups", "tenants", tenant.Code)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		ctx.Error(err.Error())
		return err
	}
	stamp := time.Now().Format("20060102_150405")
	outFile := filepath.Join(dir, stamp+".sql")

	host, port, user, pass, database := resolveTenantDSN(tenant)
	var cmd *exec.Cmd
	switch strings.ToLower(tenant.Driver) {
	case models.TenantDriverPostgres, "pgsql", "postgresql":
		args := []string{"-h", host, "-p", strconv.Itoa(port), "-U", user, "-d", database, "-f", outFile}
		cmd = exec.Command("pg_dump", args...)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	default:
		args := []string{
			"-h", host,
			"-P", strconv.Itoa(port),
			"-u", user,
			"--result-file=" + outFile,
			"--single-transaction",
			"--routines",
			"--triggers",
			database,
		}
		if pass != "" {
			args = append([]string{"-p" + pass}, args...)
		}
		cmd = exec.Command("mysqldump", args...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Error(fmt.Sprintf("backup failed: %v\n%s", err, string(out)))
		return err
	}
	ctx.Success("备份完成: " + outFile)
	return nil
}

func resolveTenantDSN(tenant *models.Tenant) (host string, port int, user, pass, database string) {
	driver := tenant.Driver
	if driver == "" {
		driver = facades.Config().GetString("database.default", "mysql")
	}
	host = tenant.Host
	if host == "" {
		host = facades.Config().GetString("database.connections."+driver+".host", "127.0.0.1")
	}
	port = tenant.Port
	if port == 0 {
		port = facades.Config().GetInt("database.connections."+driver+".port", 3306)
	}
	user = tenant.Username
	if user == "" {
		user = facades.Config().GetString("database.connections."+driver+".username", "root")
	}
	pass = tenant.Password
	if pass == "" {
		pass = facades.Config().GetString("database.connections."+driver+".password", "")
	}
	database = tenant.Database
	return
}
