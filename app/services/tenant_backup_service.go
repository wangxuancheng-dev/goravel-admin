package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// BackupTenant dumps the tenant database to storage/backups/tenants/{code}/.
// keep<=0 uses TENANCY_BACKUP_KEEP (0 means do not prune).
func BackupTenant(tenant *models.Tenant, keep int) (string, error) {
	if tenant == nil || tenant.ID == 0 {
		return "", apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	dir := filepath.Join("storage", "backups", "tenants", tenant.Code)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	stamp := time.Now().Format("20060102_150405")
	finalFile := filepath.Join(dir, stamp+".sql")
	tmpFile := finalFile + ".tmp"
	_ = os.Remove(tmpFile)

	host, port, user, pass, database, err := ResolveTenantDSN(tenant)
	if err != nil {
		return "", err
	}
	var cmd *exec.Cmd
	var cleanup func()
	switch strings.ToLower(tenant.Driver) {
	case models.TenantDriverPostgres, "pgsql", "postgresql":
		args := []string{"-h", host, "-p", strconv.Itoa(port), "-U", user, "-d", database, "-f", tmpFile}
		if tenant.Isolation == models.TenantIsolationSchema && strings.TrimSpace(tenant.Schema) != "" {
			args = append(args, "-n", tenant.Schema)
		}
		cmd = exec.Command("pg_dump", args...)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	default:
		defaultsFile, err := WriteMySQLDefaultsFile(user, pass)
		if err != nil {
			return "", err
		}
		cleanup = func() { _ = os.Remove(defaultsFile) }
		args := []string{
			"--defaults-extra-file=" + defaultsFile,
			"-h", host,
			"-P", strconv.Itoa(port),
			"--result-file=" + tmpFile,
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
		_ = os.Remove(tmpFile)
		return "", fmt.Errorf("backup failed: %w\n%s", err, string(out))
	}
	info, statErr := os.Stat(tmpFile)
	if statErr != nil || info.Size() == 0 {
		_ = os.Remove(tmpFile)
		return "", fmt.Errorf("backup failed: empty dump file")
	}
	if err := os.Rename(tmpFile, finalFile); err != nil {
		_ = os.Remove(tmpFile)
		return "", err
	}

	if keep < 0 {
		keep = facades.Config().GetInt("tenancy.backup_keep", 10)
	}
	if keep > 0 {
		_, _ = pruneTenantBackups(dir, keep)
	}
	return finalFile, nil
}

// ResolveTenantDSN returns connection fields for dump tools.
func ResolveTenantDSN(tenant *models.Tenant) (host string, port int, user, pass, database string, err error) {
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
		pass, err = RevealTenantPassword(pass)
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

func WriteMySQLDefaultsFile(user, pass string) (string, error) {
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
	if strings.ContainsAny(v, " #\t\"'\\") {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return v
}

// MarkTenantBackupResult persists backup path on the platform tenants row.
func MarkTenantBackupResult(tenant *models.Tenant, path string) error {
	if tenant == nil || tenant.ID == 0 {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	now := time.Now()
	if _, err := appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
		"last_backup_path": path,
		"last_op_at":       now,
	}); err != nil {
		return err
	}
	tenant.LastBackupPath = path
	tenant.LastOpAt = &now
	return nil
}

// TenantBackupDir returns the relative backup directory for a tenant code.
func TenantBackupDir(code string) string {
	code = strings.TrimSpace(code)
	return filepath.Join("storage", "backups", "tenants", code)
}

// TenantBackupFile is a dump under storage/backups/tenants/{code}/.
type TenantBackupFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// ListTenantBackups lists .sql dumps for the tenant (newest first).
func ListTenantBackups(tenant *models.Tenant) ([]TenantBackupFile, string, error) {
	if tenant == nil || tenant.Code == "" {
		return nil, "", apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	dir := TenantBackupDir(tenant.Code)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []TenantBackupFile{}, dir, nil
		}
		return nil, dir, err
	}
	var files []TenantBackupFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".sql") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(dir, name))
		files = append(files, TenantBackupFile{
			Name:    name,
			Path:    rel,
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name > files[j].Name
	})
	return files, dir, nil
}

// ResolveTenantBackupAbsPath validates name and returns an absolute path under the tenant backup dir.
func ResolveTenantBackupAbsPath(tenant *models.Tenant, name string) (string, error) {
	if tenant == nil || tenant.Code == "" {
		return "", apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return "", apperrors.ErrInvalidArgument.WithMessage("invalid backup name")
	}
	if !strings.HasSuffix(strings.ToLower(name), ".sql") || strings.Contains(name, "..") {
		return "", apperrors.ErrInvalidArgument.WithMessage("invalid backup name")
	}
	absDir, err := filepath.Abs(TenantBackupDir(tenant.Code))
	if err != nil {
		return "", err
	}
	absFile := filepath.Join(absDir, name)
	rel, err := filepath.Rel(absDir, absFile)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", apperrors.ErrInvalidArgument.WithMessage("invalid backup name")
	}
	if _, err := os.Stat(absFile); err != nil {
		if os.IsNotExist(err) {
			return "", apperrors.ErrInvalidArgument.WithMessage("backup file not found")
		}
		return "", err
	}
	return absFile, nil
}

