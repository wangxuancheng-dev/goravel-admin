package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/contracts/database/orm"
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

// Forget drops a cached tenant connection registration and closes its sql.DB pool when possible.
func (s *TenantConnectionService) Forget(connectionName string) {
	if connectionName == "" {
		return
	}
	registeredConns.Delete(connectionName)
	o := appfacades.Orm().Connection(connectionName)
	if db, err := o.DB(); err == nil && db != nil {
		_ = db.Close()
	}
	o.Fresh()
}

// ValidateTenantCredentials enforces dedicated DB users for remote hosts.
// Same-host (empty host) may reuse platform credentials when tenancy.allow_platform_db_credentials=true.
func ValidateTenantCredentials(host, username, password string) error {
	host = strings.TrimSpace(host)
	username = strings.TrimSpace(username)
	allowShared := facades.Config().GetBool("tenancy.allow_platform_db_credentials", true)
	if host != "" {
		if username == "" || strings.TrimSpace(password) == "" {
			return apperrors.ErrTenantCredentialsRequired
		}
		return nil
	}
	if !allowShared && (username == "" || strings.TrimSpace(password) == "") {
		return apperrors.ErrTenantCredentialsRequired
	}
	return nil
}

// SetProvisionStatus persists provision lifecycle on the platform tenants row.
func (s *TenantConnectionService) SetProvisionStatus(tenant *models.Tenant, status string) error {
	if tenant == nil || tenant.ID == 0 {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	status = strings.TrimSpace(status)
	switch status {
	case models.TenantProvisionPending, models.TenantProvisionReady, models.TenantProvisionFailed:
	default:
		return apperrors.ErrInvalidArgument.WithMessage("invalid provision_status")
	}
	if _, err := appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
		"provision_status": status,
	}); err != nil {
		return err
	}
	tenant.ProvisionStatus = status
	return nil
}

func (s *TenantConnectionService) buildConnectionConfig(tenant *models.Tenant) (map[string]any, error) {
	driverName := NormalizeTenantDriver(tenant.Driver)
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
	if password != "" {
		plain, err := RevealTenantPassword(password)
		if err != nil {
			return nil, apperrors.ErrTenantConnectionFailed.WithError(err)
		}
		password = plain
	}
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
			"sslmode":  tenantPostgresSSLMode(),
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

// TenantUsesCustomHost reports whether the tenant points at an explicit DB host
// (empty host means fall back to platform DB_*).
func TenantUsesCustomHost(tenant *models.Tenant) bool {
	return tenant != nil && strings.TrimSpace(tenant.Host) != ""
}

func (s *TenantConnectionService) resolveEndpoint(tenant *models.Tenant) (driverName, host string, port int, username, password string, err error) {
	driverName = NormalizeTenantDriver(tenant.Driver)
	if driverName == "" {
		driverName = facades.Config().GetString("database.default", "mysql")
	}
	host = strings.TrimSpace(tenant.Host)
	if host == "" {
		host = facades.Config().GetString("database.connections."+driverName+".host", "127.0.0.1")
	}
	port = tenant.Port
	if port <= 0 {
		port = facades.Config().GetInt("database.connections."+driverName+".port", 0)
	}
	username = strings.TrimSpace(tenant.Username)
	if username == "" {
		username = facades.Config().GetString("database.connections."+driverName+".username", "")
	}
	password = tenant.Password
	if password != "" {
		password, err = RevealTenantPassword(password)
		if err != nil {
			return "", "", 0, "", "", err
		}
	}
	if password == "" {
		password = facades.Config().GetString("database.connections."+driverName+".password", "")
	}
	return driverName, host, port, username, password, nil
}

// withMaintenanceOrmQuery opens a short-lived connection to the tenant endpoint's
// system database (mysql / postgres) so CREATE DATABASE can run on the correct host.
func (s *TenantConnectionService) withMaintenanceOrmQuery(tenant *models.Tenant, fn func(q orm.Query) error) error {
	driverName, host, port, username, password, err := s.resolveEndpoint(tenant)
	if err != nil {
		return err
	}
	maintName := strings.TrimSpace(tenant.ConnectionName) + "_maint"
	if maintName == "_maint" || tenant.ConnectionName == "" {
		maintName = fmt.Sprintf("tenant_maint_%s", tenant.Code)
	}

	var cfg map[string]any
	switch driverName {
	case models.TenantDriverMySQL:
		cfg = map[string]any{
			"host":      host,
			"port":      port,
			"database":  "mysql",
			"username":  username,
			"password":  password,
			"charset":   "utf8mb4",
			"collation": "utf8mb4_unicode_ci",
			"prefix":    "",
			"singular":  false,
			"via": func() (driver.Driver, error) {
				return mysqlfacades.Mysql(maintName)
			},
		}
	case models.TenantDriverPostgres:
		cfg = map[string]any{
			"host":     host,
			"port":     port,
			"database": "postgres",
			"username": username,
			"password": password,
			"sslmode":  tenantPostgresSSLMode(),
			"singular": false,
			"prefix":   "",
			"schema":   "public",
			"via": func() (driver.Driver, error) {
				return postgresfacades.Postgres(maintName)
			},
		}
	default:
		return apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}

	facades.Config().Add("database.connections."+maintName, cfg)
	return fn(appfacades.Orm().Connection(maintName).Query())
}

// CreateStorage 在目标库所在主机上 CREATE DATABASE / SCHEMA。
// host 为空：在平台库所在实例执行；host 有值：用租户凭据连该主机的系统库执行。
func (s *TenantConnectionService) CreateStorage(tenant *models.Tenant) error {
	return s.CreateStorageWithOptions(tenant, false)
}

// CreateStorageWithOptions skipCreate=true 时假定库已由 DBA 建好，仅登记元数据。
func (s *TenantConnectionService) CreateStorageWithOptions(tenant *models.Tenant, skipCreate bool) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if skipCreate {
		return nil
	}
	isolation, err := ResolveTenantIsolation(tenant.Driver, tenant.Isolation)
	if err != nil {
		return err
	}

	usePlatform := !TenantUsesCustomHost(tenant)
	driverName := NormalizeTenantDriver(tenant.Driver)

	switch driverName {
	case models.TenantDriverMySQL:
		if err := validateSQLIdent(tenant.Database); err != nil {
			return err
		}
		sql := fmt.Sprintf(
			"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
			tenant.Database,
		)
		if usePlatform {
			_, err := appfacades.PlatformOrmQuery(nil).Exec(sql)
			return err
		}
		return s.withMaintenanceOrmQuery(tenant, func(q orm.Query) error {
			_, err := q.Exec(sql)
			return err
		})
	case models.TenantDriverPostgres:
		if isolation == models.TenantIsolationDatabase {
			if err := validateSQLIdent(tenant.Database); err != nil {
				return err
			}
			createSQL := fmt.Sprintf("CREATE DATABASE %s", quotePGIdent(tenant.Database))
			existsSQL := "SELECT 1 AS exists FROM pg_database WHERE datname = ?"
			checkAndCreate := func(q orm.Query) error {
				var rows []struct {
					Exists int `gorm:"column:exists"`
				}
				if err := q.Raw(existsSQL, tenant.Database).Scan(&rows); err != nil {
					return err
				}
				if len(rows) > 0 {
					return nil
				}
				_, err := q.Exec(createSQL)
				return err
			}
			if usePlatform {
				return checkAndCreate(appfacades.PlatformOrmQuery(nil))
			}
			return s.withMaintenanceOrmQuery(tenant, checkAndCreate)
		}
		if err := validateSQLIdent(tenant.Schema); err != nil {
			return err
		}
		schemaSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", quotePGIdent(tenant.Schema))
		platformDB := facades.Config().GetString("database.connections."+tenant.Driver+".database", "")
		if usePlatform && (tenant.Database == "" || tenant.Database == platformDB) {
			_, err := appfacades.PlatformOrmQuery(nil).Exec(schemaSQL)
			return err
		}
		// Schema on a dedicated / remote database: connect to that database then create schema.
		if err := s.EnsureRegistered(tenant); err != nil {
			return err
		}
		_, err := appfacades.Orm().Connection(tenant.ConnectionName).Query().Exec(schemaSQL)
		return err
	default:
		return apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}
}

// DropStorage drops the tenant database or schema created by CreateStorage (best-effort cleanup).
func (s *TenantConnectionService) DropStorage(tenant *models.Tenant) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	isolation, err := ResolveTenantIsolation(tenant.Driver, tenant.Isolation)
	if err != nil {
		return err
	}
	usePlatform := !TenantUsesCustomHost(tenant)
	driverName := NormalizeTenantDriver(tenant.Driver)
	s.Forget(tenant.ConnectionName)

	platformDB := facades.Config().GetString("database.connections."+driverName+".database", "")
	if platformDB == "" {
		platformDB = facades.Config().GetString("database.connections.mysql.database", "")
	}

	switch driverName {
	case models.TenantDriverMySQL:
		if err := validateSQLIdent(tenant.Database); err != nil {
			return err
		}
		if err := rejectDroppingPlatformDatabase(tenant.Database, platformDB); err != nil {
			return err
		}
		sql := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tenant.Database)
		if usePlatform {
			_, err := appfacades.PlatformOrmQuery(nil).Exec(sql)
			return err
		}
		return s.withMaintenanceOrmQuery(tenant, func(q orm.Query) error {
			_, err := q.Exec(sql)
			return err
		})
	case models.TenantDriverPostgres:
		if isolation == models.TenantIsolationDatabase {
			if err := validateSQLIdent(tenant.Database); err != nil {
				return err
			}
			if err := rejectDroppingPlatformDatabase(tenant.Database, platformDB); err != nil {
				return err
			}
			sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s", quotePGIdent(tenant.Database))
			if usePlatform {
				_, err := appfacades.PlatformOrmQuery(nil).Exec(sql)
				return err
			}
			return s.withMaintenanceOrmQuery(tenant, func(q orm.Query) error {
				_, err := q.Exec(sql)
				return err
			})
		}
		if err := validateSQLIdent(tenant.Schema); err != nil {
			return err
		}
		if err := rejectDroppingPlatformSchema(tenant.Schema); err != nil {
			return err
		}
		schemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quotePGIdent(tenant.Schema))
		if usePlatform && (tenant.Database == "" || tenant.Database == platformDB) {
			_, err := appfacades.PlatformOrmQuery(nil).Exec(schemaSQL)
			return err
		}
		if err := s.EnsureRegistered(tenant); err != nil {
			return err
		}
		_, err := appfacades.Orm().Connection(tenant.ConnectionName).Query().Exec(schemaSQL)
		return err
	default:
		return apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}
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

// FindTenantByIDOrCode 平台库查找租户
func (s *TenantConnectionService) FindTenantByIDOrCode(idOrCode string) (*models.Tenant, error) {
	var tenant models.Tenant
	q := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{})
	if id := cast.ToUint(idOrCode); id > 0 && fmt.Sprintf("%d", id) == idOrCode {
		if err := q.Where("id", id).First(&tenant); err != nil {
			return nil, apperrors.ErrRecordNotFound.WithError(err)
		}
		if tenant.ID == 0 {
			return nil, apperrors.ErrRecordNotFound
		}
		return &tenant, nil
	}
	if err := q.Where("code", strings.ToLower(strings.TrimSpace(idOrCode))).First(&tenant); err != nil {
		return nil, apperrors.ErrRecordNotFound.WithError(err)
	}
	if tenant.ID == 0 {
		return nil, apperrors.ErrRecordNotFound
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
	if !tenant.IsProvisionReady() {
		return apperrors.ErrTenantNotReady
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
		return ctx, apperrors.ErrTenantNotFound.WithError(err)
	}
	if tenant.Status != models.TenantStatusActive {
		return ctx, apperrors.ErrTenantDisabled
	}
	if !tenant.IsProvisionReady() {
		return ctx, apperrors.ErrTenantNotReady
	}
	if err := s.EnsureRegistered(tenant); err != nil {
		return ctx, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	return tenancyctx.WithTenant(ctx, tenant.ID, tenant.ConnectionName, tenant.Code), nil
}

// WithTenantConnection 在租户连接上串行执行（migrate / seed 共用）。
// 仅切换 Schema 连接与（为 Artisan seed/migrate 兼容）临时 database.default；
// 平台查询请始终走 PlatformOrmQuery（钉死 platform_connection，不受 default 翻转影响）。
func (s *TenantConnectionService) WithTenantConnection(tenant *models.Tenant, fn func() error) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if err := s.EnsureRegistered(tenant); err != nil {
		return err
	}
	migrateMu.Lock()
	defer migrateMu.Unlock()
	schemaMu := appfacades.SchemaConnLock()
	schemaMu.Lock()
	defer schemaMu.Unlock()

	// Pin platform connection before flipping default so first PlatformOrmQuery wins the pin.
	_ = appfacades.PlatformConnectionName()

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

// MigrateTenant 在租户连接上执行 migrate，成功后标记 provision_status=ready。
func (s *TenantConnectionService) MigrateTenant(tenant *models.Tenant) error {
	err := s.WithTenantConnection(tenant, func() error {
		return facades.Artisan().Call("migrate")
	})
	if err != nil {
		_ = s.SetProvisionStatus(tenant, models.TenantProvisionFailed)
		msg := err.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		_, _ = appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
			"last_migrate_error": msg,
		})
		tenant.LastMigrateError = msg
		return err
	}
	now := time.Now()
	if err := s.SetProvisionStatus(tenant, models.TenantProvisionReady); err != nil {
		return err
	}
	_, _ = appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
		"last_migrate_error": "",
		"migrated_at":        now,
	})
	tenant.LastMigrateError = ""
	tenant.MigratedAt = &now
	return nil
}

// Ping verifies the tenant database connection is reachable.
func (s *TenantConnectionService) Ping(tenant *models.Tenant, timeout time.Duration) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if err := s.EnsureRegistered(tenant); err != nil {
		return apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	db, err := appfacades.Orm().Connection(tenant.ConnectionName).DB()
	if err != nil {
		return apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	return nil
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
