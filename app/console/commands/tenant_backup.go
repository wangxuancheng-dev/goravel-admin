package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{Name: "keep", Usage: "保留最近 N 份备份（0=不清理；默认读 TENANCY_BACKUP_KEEP）"},
		},
	}
}

func (r *TenantBackup) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("需要 TENANCY_DRIVER=database")
		return nil
	}
	idOrCode := ctx.Argument(0)
	if idOrCode == "" {
		ctx.Error("用法: tenant:backup {id|code} [--keep=N]")
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

	host, port, user, pass, database, err := resolveTenantDSN(tenant)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	var cmd *exec.Cmd
	var cleanup func()
	switch strings.ToLower(tenant.Driver) {
	case models.TenantDriverPostgres, "pgsql", "postgresql":
		args := []string{"-h", host, "-p", strconv.Itoa(port), "-U", user, "-d", database, "-f", outFile}
		if tenant.Isolation == models.TenantIsolationSchema && strings.TrimSpace(tenant.Schema) != "" {
			args = append(args, "-n", tenant.Schema)
		}
		cmd = exec.Command("pg_dump", args...)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	default:
		defaultsFile, err := writeMySQLDefaultsFile(user, pass)
		if err != nil {
			ctx.Error(err.Error())
			return err
		}
		cleanup = func() { _ = os.Remove(defaultsFile) }
		args := []string{
			"--defaults-extra-file=" + defaultsFile,
			"-h", host,
			"-P", strconv.Itoa(port),
			"--result-file=" + outFile,
			"--single-transaction",
			"--routines",
			"--triggers",
			database,
		}
		cmd = exec.Command("mysqldump", args...)
	}
	if cleanup != nil {
		defer cleanup()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Error(fmt.Sprintf("backup failed: %v\n%s", err, string(out)))
		return err
	}
	ctx.Success("备份完成: " + outFile)

	keep := facades.Config().GetInt("tenancy.backup_keep", 10)
	if raw := strings.TrimSpace(ctx.Option("keep")); raw != "" {
		keep = ctx.OptionInt("keep")
	}
	if keep > 0 {
		removed, pruneErr := pruneTenantBackups(dir, keep)
		if pruneErr != nil {
			ctx.Warning("备份保留清理失败: " + pruneErr.Error())
		} else if removed > 0 {
			ctx.Info(fmt.Sprintf("已清理旧备份 %d 个（保留最近 %d 份）", removed, keep))
		}
	}
	return nil
}

func pruneTenantBackups(dir string, keep int) (int, error) {
	if keep <= 0 {
		return 0, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	var files []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			files = append(files, e)
		}
	}
	if len(files) <= keep {
		return 0, nil
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})
	removed := 0
	for _, e := range files[keep:] {
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func resolveTenantDSN(tenant *models.Tenant) (host string, port int, user, pass, database string, err error) {
	driver := tenant.Driver
	if driver == "" {
		driver = facades.Config().GetString("database.default", "mysql")
	}
	host = strings.TrimSpace(tenant.Host)
	if host == "" {
		host = facades.Config().GetString("database.connections."+driver+".host", "127.0.0.1")
	}
	port = tenant.Port
	if port == 0 {
		port = facades.Config().GetInt("database.connections."+driver+".port", 3306)
	}
	user = strings.TrimSpace(tenant.Username)
	if user == "" {
		user = facades.Config().GetString("database.connections."+driver+".username", "root")
	}
	pass = tenant.Password
	if pass != "" {
		pass, err = services.RevealTenantPassword(pass)
		if err != nil {
			return "", 0, "", "", "", err
		}
	}
	if pass == "" {
		pass = facades.Config().GetString("database.connections."+driver+".password", "")
	}
	database = tenant.Database
	return host, port, user, pass, database, nil
}

// writeMySQLDefaultsFile avoids putting the password on the process argv.
func writeMySQLDefaultsFile(user, pass string) (string, error) {
	f, err := os.CreateTemp("", "goravel-mysql-*.cnf")
	if err != nil {
		return "", err
	}
	path := f.Name()
	content := fmt.Sprintf("[client]\nuser=%s\npassword=%s\n", user, escapeMySQLCnf(pass))
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	_ = os.Chmod(path, 0o600)
	return path, nil
}

func escapeMySQLCnf(v string) string {
	// Option file values with special chars should be quoted.
	if strings.ContainsAny(v, " #\t\"'\\") {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return v
}
