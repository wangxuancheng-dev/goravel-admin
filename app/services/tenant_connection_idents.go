package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goravel/framework/facades"
	goravelmysql "github.com/goravel/mysql"
	goravelpostgres "github.com/goravel/postgres"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

var tenantIdentPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// NormalizeTenantCode 校验并规范化租户短码（小写字母开头，仅 a-z0-9_）
func NormalizeTenantCode(code string) (string, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if !tenantIdentPattern.MatchString(code) {
		return "", apperrors.ErrInvalidArgument.WithMessage("tenant code must match ^[a-z][a-z0-9_]{0,62}$")
	}
	return code, nil
}

// ResolveTenantIsolation MySQL 强制 database；PG 允许 database|schema
func ResolveTenantIsolation(driverName, isolation string) (string, error) {
	driverName = NormalizeTenantDriver(driverName)
	isolation = strings.ToLower(strings.TrimSpace(isolation))
	if isolation == "" {
		isolation = models.TenantIsolationDatabase
	}
	switch driverName {
	case models.TenantDriverMySQL:
		if isolation != models.TenantIsolationDatabase {
			return "", apperrors.ErrInvalidArgument.WithMessage("mysql tenants must use isolation=database")
		}
		return models.TenantIsolationDatabase, nil
	case models.TenantDriverPostgres:
		if isolation != models.TenantIsolationDatabase && isolation != models.TenantIsolationSchema {
			return "", apperrors.ErrInvalidArgument.WithMessage("postgres isolation must be database or schema")
		}
		return isolation, nil
	default:
		return "", apperrors.ErrInvalidArgument.WithMessage("driver must be mysql or postgres")
	}
}

// NormalizeTenantDriver returns lowercase mysql|postgres (empty stays empty for caller default).
func NormalizeTenantDriver(driverName string) string {
	driverName = strings.ToLower(strings.TrimSpace(driverName))
	switch driverName {
	case "pgsql", "postgresql":
		return models.TenantDriverPostgres
	default:
		return driverName
	}
}

// TenantConnectionName 生成运行时 connection 名
func TenantConnectionName(tenantID uint) string {
	return fmt.Sprintf("tenant_%d", tenantID)
}

// samePlatformDriver reports whether driverName (normalized tenant driver) is the
// driver of the platform connection, i.e. whether the tenant can live on the platform
// database server.
func samePlatformDriver(driverName string) bool {
	q := appfacades.PlatformQuery()
	if down, _ := appfacades.QueryUnavailable(q); down {
		return true // cannot tell; CreateStorage reports the real error
	}
	// Query.Driver() reports the Goravel driver's name ("MySQL" / "PostgreSQL").
	platform := ""
	switch q.Driver() {
	case goravelmysql.Name:
		platform = models.TenantDriverMySQL
	case goravelpostgres.Name:
		platform = models.TenantDriverPostgres
	}
	return platform == NormalizeTenantDriver(driverName)
}

// pgTenantOwner is the role that must own a postgres tenant database / schema created
// with the platform account: the tenant's own account, or "" when it has none (it then
// uses the platform account, which owns it anyway). Naming the creating role itself is
// harmless. For another role the platform role needs the right to create objects it
// owns (superuser, or membership / SET ROLE); without it the CREATE fails with the
// server's reason instead of leaving a database the tenant account cannot write to.
func pgTenantOwner(tenant *models.Tenant) string {
	return strings.TrimSpace(tenant.Username)
}

// DefaultTenantDatabaseName 默认库名 = prefix + code
func DefaultTenantDatabaseName(code string) string {
	prefix := facades.Config().GetString("tenancy.database_prefix", "tenant_")
	return prefix + code
}

// DefaultTenantSchemaName 默认 schema 名 = prefix + code
func DefaultTenantSchemaName(code string) string {
	prefix := facades.Config().GetString("tenancy.schema_prefix", "tenant_")
	return prefix + code
}

func rejectDroppingPlatformDatabase(target, platformDB string) error {
	target = strings.ToLower(strings.TrimSpace(target))
	platformDB = strings.ToLower(strings.TrimSpace(platformDB))
	if target == "" {
		return apperrors.ErrInvalidArgument.WithMessage("tenant database is empty")
	}
	if platformDB != "" && target == platformDB {
		return apperrors.ErrInvalidArgument.WithMessage("refusing to drop platform database")
	}
	switch target {
	case "mysql", "information_schema", "performance_schema", "sys", "postgres", "template0", "template1":
		return apperrors.ErrInvalidArgument.WithMessage("refusing to drop system database: " + target)
	}
	prefix := strings.ToLower(strings.TrimSpace(facades.Config().GetString("tenancy.database_prefix", "tenant_")))
	if prefix != "" && !strings.HasPrefix(target, prefix) {
		return apperrors.ErrInvalidArgument.WithMessage("refusing to drop database outside tenancy prefix")
	}
	return nil
}

func rejectDroppingPlatformSchema(schemaName string) error {
	schemaName = strings.ToLower(strings.TrimSpace(schemaName))
	switch schemaName {
	case "", "public", "pg_catalog", "information_schema":
		return apperrors.ErrInvalidArgument.WithMessage("refusing to drop reserved schema: " + schemaName)
	}
	prefix := strings.ToLower(strings.TrimSpace(facades.Config().GetString("tenancy.schema_prefix", "tenant_")))
	if prefix != "" && !strings.HasPrefix(schemaName, prefix) {
		return apperrors.ErrInvalidArgument.WithMessage("refusing to drop schema outside tenancy prefix")
	}
	return nil
}

func validateSQLIdent(name string) error {
	name = strings.TrimSpace(name)
	if !tenantIdentPattern.MatchString(name) {
		return apperrors.ErrInvalidArgument.WithMessage("invalid SQL identifier: " + name)
	}
	return nil
}

func quotePGIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func tenantPostgresSSLMode() string {
	mode := strings.TrimSpace(facades.Config().GetString("tenancy.postgres_sslmode", ""))
	if mode == "" {
		mode = strings.TrimSpace(facades.Config().GetString("database.connections.postgres.sslmode", "disable"))
	}
	if mode == "" {
		return "disable"
	}
	return mode
}

// tenantMySQLTLSMode is the go-sql-driver tls= value for tenant MySQL connections.
// Empty / false / disable → plain TCP. true / required → tls=true. skip-verify → insecure.
func tenantMySQLTLSMode() string {
	mode := strings.ToLower(strings.TrimSpace(facades.Config().GetString("tenancy.mysql_tls", "")))
	switch mode {
	case "", "false", "disable", "off", "0", "no":
		return ""
	case "true", "required", "1", "yes", "on":
		return "true"
	case "skip-verify", "skip_verify", "insecure":
		return "skip-verify"
	case "preferred":
		return "preferred"
	default:
		return mode
	}
}
