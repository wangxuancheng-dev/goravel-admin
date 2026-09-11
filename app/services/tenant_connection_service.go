package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	mysqlfacades "github.com/goravel/mysql/facades"
	postgresfacades "github.com/goravel/postgres/facades"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

var (
	tenantIdentPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	registeredConns    sync.Map // connection_name -> struct{}
	migrateMu          sync.Mutex
)

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
	driverName = strings.ToLower(strings.TrimSpace(driverName))
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

// TenantConnectionName 生成运行时 connection 名
func TenantConnectionName(tenantID uint) string {
	return fmt.Sprintf("tenant_%d", tenantID)
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

// TenancyEnabled is an alias of tenancy.Enabled.
func TenancyEnabled() bool {
	return tenancy.Enabled()
}

type TenantConnectionService struct{}

func NewTenantConnectionService() *TenantConnectionService {
	return &TenantConnectionService{}
}

// EnsureRegistered 将租户连接写入 config 并可供 Orm.Connection 使用
func (s *TenantConnectionService) EnsureRegistered(tenant *models.Tenant) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if tenant.ConnectionName == "" {
		tenant.ConnectionName = TenantConnectionName(tenant.ID)
	}
	if _, loaded := registeredConns.LoadOrStore(tenant.ConnectionName, struct{}{}); loaded {
		return nil
	}

	cfg, err := s.buildConnectionConfig(tenant)
	if err != nil {
		registeredConns.Delete(tenant.ConnectionName)
		return err
	}
	facades.Config().Add("database.connections."+tenant.ConnectionName, cfg)
	return nil
}

func (s *TenantConnectionService) buildConnectionConfig(tenant *models.Tenant) (map[string]any, error) {
	driverName := tenant.Driver
	if driverName == "" {
		driverName = facades.Config().GetString("database.default", "mysql")
	}
	isolation, err := ResolveTenantIsolation(driverName, tenant.Isolation)
	if err != nil {
		return nil, err
	}

	host := tenant.Host
	if host == "" {
		host = facades.Config().GetString("database.connections."+driverName+".host", "127.0.0.1")
	}
	port := tenant.Port
	if port <= 0 {
		port = facades.Config().GetInt("database.connections."+driverName+".port", 0)
	}
	username := tenant.Username
	if username == "" {
		username = facades.Config().GetString("database.connections."+driverName+".username", "")
	}
	password := tenant.Password
	if password == "" {
		password = facades.Config().GetString("database.connections."+driverName+".password", "")
	}
	database := tenant.Database
	if database == "" {
		return nil, apperrors.ErrInvalidArgument.WithMessage("tenant database name is required")
	}

	connName := tenant.ConnectionName
	switch driverName {
	case models.TenantDriverMySQL:
		return map[string]any{
			"host":      host,
			"port":      port,
			"database":  database,
			"username":  username,
			"password":  password,
			"charset":   "utf8mb4",
			"collation": "utf8mb4_unicode_ci",
			"prefix":    "",
			"singular":  false,
			"via": func() (driver.Driver, error) {
				return mysqlfacades.Mysql(connName)
			},
		}, nil
	case models.TenantDriverPostgres:
		schemaName := "public"
		if isolation == models.TenantIsolationSchema {
			schemaName = tenant.Schema
			if schemaName == "" {
				return nil, apperrors.ErrInvalidArgument.WithMessage("postgres schema isolation requires schema name")
			}
		} else if tenant.Schema != "" {
			schemaName = tenant.Schema
		}
		return map[string]any{
			"host":     host,
			"port":     port,
			"database": database,
			"username": username,
			"password": password,
			"sslmode":  "disable",
			"singular": false,
			"prefix":   "",
			"schema":   schemaName,
			"via": func() (driver.Driver, error) {
				return postgresfacades.Postgres(connName)
			},
		}, nil
	default:
		return nil, apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}
}

// CreateStorage 在服务器上 CREATE DATABASE 或 CREATE SCHEMA
func (s *TenantConnectionService) CreateStorage(tenant *models.Tenant) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	isolation, err := ResolveTenantIsolation(tenant.Driver, tenant.Isolation)
	if err != nil {
		return err
	}

	switch tenant.Driver {
	case models.TenantDriverMySQL:
		if err := validateSQLIdent(tenant.Database); err != nil {
			return err
		}
		sql := fmt.Sprintf(
			"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
			tenant.Database,
		)
		_, err := appfacades.PlatformOrmQuery(nil).Exec(sql)
		return err
	case models.TenantDriverPostgres:
		if isolation == models.TenantIsolationDatabase {
			if err := validateSQLIdent(tenant.Database); err != nil {
				return err
			}
			var rows []struct {
				Exists int `gorm:"column:exists"`
			}
			if err := appfacades.PlatformOrmQuery(nil).
				Raw("SELECT 1 AS exists FROM pg_database WHERE datname = ?", tenant.Database).
				Scan(&rows); err != nil {
				return err
			}
			if len(rows) > 0 {
				return nil
			}
			_, err := appfacades.PlatformOrmQuery(nil).Exec(
				fmt.Sprintf("CREATE DATABASE %s", quotePGIdent(tenant.Database)),
			)
			return err
		}
		if err := validateSQLIdent(tenant.Schema); err != nil {
			return err
		}
		platformDB := facades.Config().GetString("database.connections."+tenant.Driver+".database", "")
		if tenant.Database == "" || tenant.Database == platformDB {
			_, err := appfacades.PlatformOrmQuery(nil).Exec(
				fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", quotePGIdent(tenant.Schema)),
			)
			return err
		}
		if err := s.EnsureRegistered(tenant); err != nil {
			return err
		}
		_, err := appfacades.Orm().Connection(tenant.ConnectionName).Query().Exec(
			fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", quotePGIdent(tenant.Schema)),
		)
		return err
	default:
		return apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}
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

// FindTenantByIDOrCode 平台库查找租户
func (s *TenantConnectionService) FindTenantByIDOrCode(idOrCode string) (*models.Tenant, error) {
	var tenant models.Tenant
	q := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{})
	if id := cast.ToUint(idOrCode); id > 0 && fmt.Sprintf("%d", id) == idOrCode {
		if err := q.Where("id", id).First(&tenant); err != nil {
			return nil, apperrors.ErrRecordNotFound.WithError(err)
		}
		return &tenant, nil
	}
	if err := q.Where("code", strings.ToLower(strings.TrimSpace(idOrCode))).First(&tenant); err != nil {
		return nil, apperrors.ErrRecordNotFound.WithError(err)
	}
	return &tenant, nil
}

// ExtractTenantHint 从 Header / Query 取租户标识
func ExtractTenantHint(ctx http.Context) string {
	return tenancy.HTTPHint(ctx)
}

// BindHTTP 解析租户、注册连接并写入 HTTP context。hint 可空则从 Header/Query 取。
func (s *TenantConnectionService) BindHTTP(ctx http.Context, hint string) error {
	if !tenancy.Enabled() {
		return nil
	}
	raw := strings.TrimSpace(hint)
	if raw == "" {
		raw = tenancy.HTTPHint(ctx)
	}
	if raw == "" {
		return apperrors.ErrTenantRequired
	}
	tenant, err := s.FindTenantByIDOrCode(raw)
	if err != nil {
		return apperrors.ErrTenantNotFound.WithError(err)
	}
	if tenant.Status != models.TenantStatusActive {
		return apperrors.ErrTenantDisabled
	}
	if err := s.EnsureRegistered(tenant); err != nil {
		return apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	helpers.SetTenantContext(ctx, tenant.ID, tenant.ConnectionName, tenant.Code)
	return nil
}

// BindBackground 供队列任务切换到租户连接
func (s *TenantConnectionService) BindBackground(ctx context.Context, tenantID uint) (context.Context, error) {
	if !tenancy.Enabled() || tenantID == 0 {
		return ctx, nil
	}
	tenant, err := s.FindTenantByIDOrCode(fmt.Sprintf("%d", tenantID))
	if err != nil {
		return ctx, err
	}
	if err := s.EnsureRegistered(tenant); err != nil {
		return ctx, err
	}
	return tenancyctx.WithTenant(ctx, tenant.ID, tenant.ConnectionName, tenant.Code), nil
}

// WithTenantConnection 在租户连接上串行执行（migrate / seed 共用）。
func (s *TenantConnectionService) WithTenantConnection(tenant *models.Tenant, fn func() error) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if err := s.EnsureRegistered(tenant); err != nil {
		return err
	}
	migrateMu.Lock()
	defer migrateMu.Unlock()

	prevDefault := facades.Config().GetString("database.default")
	facades.Config().Add("database.default", tenant.ConnectionName)
	defer facades.Config().Add("database.default", prevDefault)

	schema := facades.Schema()
	prevConn := schema.GetConnection()
	schema.SetConnection(tenant.ConnectionName)
	defer schema.SetConnection(prevConn)

	return fn()
}

// SeedTenant 在租户连接上执行 db:seed。
// seeders 为空则跑全部；否则等价于 db:seed --seeder=Name（可多个）。
func (s *TenantConnectionService) SeedTenant(tenant *models.Tenant, seeders ...string) error {
	return s.WithTenantConnection(tenant, func() error {
		cmd := "db:seed --force"
		for _, name := range seeders {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			cmd += " --seeder=" + name
		}
		return facades.Artisan().Call(cmd)
	})
}

// MigrateTenant 在租户连接上执行 migrate
func (s *TenantConnectionService) MigrateTenant(tenant *models.Tenant) error {
	return s.WithTenantConnection(tenant, func() error {
		return facades.Artisan().Call("migrate")
	})
}
