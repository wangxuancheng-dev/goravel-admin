package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	schemaContract "github.com/goravel/framework/contracts/database/schema"
	contractsseeder "github.com/goravel/framework/contracts/database/seeder"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/database/migration"
	"github.com/goravel/framework/facades"
	goravelmysql "github.com/goravel/mysql"
	goravelpostgres "github.com/goravel/postgres"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
	"goravel/app/tenantcache"
	"goravel/app/tenantdb"
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

type TenantConnectionService struct{}

func NewTenantConnectionService() *TenantConnectionService {
	return &TenantConnectionService{}
}

// EnsureRegistered makes the tenant's connection available (tenantdb.Default) and
// returns once it answers a ping. A registered connection built from the same
// host/database/credentials is reused without any process-wide lock; a new or changed
// one is built and pinged outside any lock, so a slow tenant database never stalls
// requests of other tenants.
func (s *TenantConnectionService) EnsureRegistered(tenant *models.Tenant) error {
	_, err := s.ensureConn(tenant)
	return err
}

func (s *TenantConnectionService) ensureConn(tenant *models.Tenant) (*tenantdb.Conn, error) {
	if tenant == nil {
		return nil, apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if tenant.ConnectionName == "" {
		tenant.ConnectionName = TenantConnectionName(tenant.ID)
	}
	configureTenantDB()
	spec, err := s.buildSpec(tenant)
	if err != nil {
		return nil, err
	}
	c, err := tenantdb.Default.Ensure(tenant.ConnectionName, spec)
	if err != nil {
		return nil, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	return c, nil
}

// Forget unregisters the tenant connection and closes its pool now (e.g. before
// DROP DATABASE). A later EnsureRegistered builds a new pool.
func (s *TenantConnectionService) Forget(connectionName string) {
	if connectionName == "" {
		return
	}
	tenantdb.Default.Forget(connectionName)
}

// ValidateTenantCredentials enforces dedicated DB users for remote hosts.
// Same-host (empty host) may reuse platform credentials when tenancy.allow_platform_db_credentials=true.
func ValidateTenantCredentials(host, username, password string) error {
	host = strings.TrimSpace(host)
	username = strings.TrimSpace(username)
	allowShared := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
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
	case models.TenantProvisionPending, models.TenantProvisionMigrating, models.TenantProvisionReady, models.TenantProvisionFailed:
	default:
		return apperrors.ErrInvalidArgument.WithMessage("invalid provision_status")
	}
	if _, err := appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"provision_status": status,
	}); err != nil {
		return err
	}
	tenant.ProvisionStatus = status
	return nil
}

// buildSpec resolves how to reach the tenant database (empty host / credentials fall
// back to the platform connection's).
func (s *TenantConnectionService) buildSpec(tenant *models.Tenant) (tenantdb.Spec, error) {
	driverName := NormalizeTenantDriver(tenant.Driver)
	if driverName == "" {
		driverName = facades.Config().GetString("database.default", "mysql")
	}
	isolation, err := ResolveTenantIsolation(driverName, tenant.Isolation)
	if err != nil {
		return tenantdb.Spec{}, err
	}

	host := tenant.Host
	if host == "" {
		host = facades.Config().GetString("database.connections."+driverName+".host", "127.0.0.1")
	}
	port := tenant.Port
	if port <= 0 {
		port = facades.Config().GetInt("database.connections."+driverName+".port", 0)
	}
	username, password, err := tenantDBAccount(tenant, driverName)
	if err != nil {
		return tenantdb.Spec{}, err
	}
	database := tenant.Database
	if database == "" {
		return tenantdb.Spec{}, apperrors.ErrInvalidArgument.WithMessage("tenant database name is required")
	}

	spec := tenantdb.Spec{
		Host:     host,
		Port:     port,
		Database: database,
		Username: username,
		Password: password,
		Pool:     tenantPoolLimits(),
	}
	switch driverName {
	case models.TenantDriverMySQL:
		spec.Driver = tenantdb.DriverMySQL
		spec.TLSMode = tenantMySQLTLSMode()
		return spec, nil
	case models.TenantDriverPostgres:
		schemaName := "public"
		if isolation == models.TenantIsolationSchema {
			schemaName = tenant.Schema
			if schemaName == "" {
				return tenantdb.Spec{}, apperrors.ErrInvalidArgument.WithMessage("postgres schema isolation requires schema name")
			}
		} else if tenant.Schema != "" {
			schemaName = tenant.Schema
		}
		spec.Driver = tenantdb.DriverPostgres
		spec.Schema = schemaName
		spec.SSLMode = tenantPostgresSSLMode()
		return spec, nil
	default:
		return tenantdb.Spec{}, apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}
}

func tenantPoolLimits() tenantdb.PoolLimits {
	idle := facades.Config().GetInt("tenancy.pool_max_idle_conns", 10)
	open := facades.Config().GetInt("tenancy.pool_max_open_conns", 20)
	if idle < 0 {
		idle = 0
	}
	if open < 1 {
		open = 1
	}
	return tenantdb.PoolLimits{
		MaxIdleConns:    idle,
		MaxOpenConns:    open,
		ConnMaxIdleTime: time.Duration(facades.Config().GetInt("tenancy.pool_conn_max_idletime", 300)) * time.Second,
		ConnMaxLifetime: time.Duration(facades.Config().GetInt("tenancy.pool_conn_max_lifetime", 1800)) * time.Second,
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
	username, password, err = tenantDBAccount(tenant, driverName)
	if err != nil {
		return "", "", 0, "", "", err
	}
	return driverName, host, port, username, password, nil
}

// tenantDBAccount returns the account a tenant connects with: its own, or — only on
// the platform's database server and only with TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS
// =true — the platform account. It is checked on every connection, not only when the
// tenant is created, so turning the setting off stops tenants without credentials of
// their own from running as the platform account (they fail with
// ErrTenantCredentialsRequired until credentials are set).
func tenantDBAccount(tenant *models.Tenant, driverName string) (username, password string, err error) {
	username = strings.TrimSpace(tenant.Username)
	if tenant.Password != "" {
		if password, err = RevealTenantPassword(tenant.Password); err != nil {
			return "", "", apperrors.ErrTenantConnectionFailed.WithError(err)
		}
	}
	if username != "" && password != "" {
		return username, password, nil
	}
	if TenantUsesCustomHost(tenant) || !facades.Config().GetBool("tenancy.allow_platform_db_credentials", false) {
		return "", "", apperrors.ErrTenantCredentialsRequired
	}
	if username == "" {
		username = facades.Config().GetString("database.connections."+driverName+".username", "")
	}
	if password == "" {
		password = facades.Config().GetString("database.connections."+driverName+".password", "")
	}
	return username, password, nil
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

	spec := tenantdb.Spec{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Pool:     tenantdb.PoolLimits{MaxIdleConns: 0, MaxOpenConns: 2},
	}
	switch driverName {
	case models.TenantDriverMySQL:
		spec.Driver, spec.Database = tenantdb.DriverMySQL, "mysql"
		spec.TLSMode = tenantMySQLTLSMode()
	case models.TenantDriverPostgres:
		spec.Driver, spec.Database, spec.Schema = tenantdb.DriverPostgres, "postgres", "public"
		spec.SSLMode = tenantPostgresSSLMode()
	default:
		return apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}

	// Short-lived and unregistered: closed as soon as fn returns.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := tenantdb.Open(ctx, maintName, spec)
	if err != nil {
		return apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	defer func() { _ = c.Close() }()
	return fn(c.Orm().Query())
}

// tenantQuery returns a query on the tenant's own database (registering it first).
func (s *TenantConnectionService) tenantQuery(tenant *models.Tenant) (orm.Query, error) {
	c, err := s.ensureConn(tenant)
	if err != nil {
		return nil, err
	}
	return c.Orm().Query(), nil
}

// CreateStorageWithOptions 在目标库所在主机上 CREATE DATABASE / SCHEMA。
// host 为空：在平台库所在实例执行；host 有值：用租户凭据连该主机的系统库执行。
// skipCreate=true 时假定库已由 DBA 建好，仅登记元数据。
func (s *TenantConnectionService) CreateStorageWithOptions(tenant *models.Tenant, skipCreate bool) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if skipCreate {
		return nil
	}
	// A dedicated account (TENANCY_DB_DEDICATED_USER) exists before its database
	// (PostgreSQL creates the database owned by it) and is granted it after (MySQL).
	if err := createDedicatedDBUser(tenant); err != nil {
		return err
	}
	if err := s.createStorage(tenant); err != nil {
		return err
	}
	return grantDedicatedDBUser(tenant)
}

// createStorage creates the tenant's database (or schema). An existing one is never
// adopted: it is usually what a former tenant with the same code left behind, and the
// new tenant would inherit its admins and data. ErrTenantStorageExists says so; a
// database prepared on purpose is taken over with skip_create instead.
func (s *TenantConnectionService) createStorage(tenant *models.Tenant) error {
	err := s.createStorageOnce(tenant)
	if err != nil && isAlreadyExistsErr(err) {
		return apperrors.ErrTenantStorageExists.WithError(err)
	}
	return err
}

// isAlreadyExistsErr matches MySQL 1007 (database exists) and PostgreSQL 42P04
// (duplicate_database) / 42P06 (duplicate_schema).
func isAlreadyExistsErr(err error) bool {
	if errors.Is(err, errStorageExists) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "1007") || strings.Contains(msg, "42P04") || strings.Contains(msg, "42P06")
}

var errStorageExists = errors.New("database already exists")

func (s *TenantConnectionService) createStorageOnce(tenant *models.Tenant) error {
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
			"CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
			tenant.Database,
		)
		if usePlatform {
			_, err := appfacades.PlatformQuery().Exec(sql)
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
					return errStorageExists
				}
				_, err := q.Exec(createSQL)
				return err
			}
			if usePlatform {
				// Created by the platform account but used by the tenant's: make the
				// tenant account the owner, or (PostgreSQL 15+) it cannot create tables
				// in the new database's public schema.
				if owner := pgTenantOwner(tenant); owner != "" {
					createSQL += " OWNER " + quotePGIdent(owner)
				}
				return checkAndCreate(appfacades.PlatformQuery())
			}
			return s.withMaintenanceOrmQuery(tenant, checkAndCreate)
		}
		if err := validateSQLIdent(tenant.Schema); err != nil {
			return err
		}
		schemaSQL := fmt.Sprintf("CREATE SCHEMA %s", quotePGIdent(tenant.Schema))
		platformDB := facades.Config().GetString("database.connections."+tenant.Driver+".database", "")
		if usePlatform && (tenant.Database == "" || tenant.Database == platformDB) {
			// A schema the platform account creates is only writable by its owner:
			// hand it to the tenant's account when that is a different one.
			if owner := pgTenantOwner(tenant); owner != "" {
				schemaSQL += " AUTHORIZATION " + quotePGIdent(owner)
			}
			_, err := appfacades.PlatformQuery().Exec(schemaSQL)
			return err
		}
		// Schema on a dedicated / remote database: connect to that database then create schema.
		q, err := s.tenantQuery(tenant)
		if err != nil {
			return err
		}
		_, err = q.Exec(schemaSQL)
		return err
	default:
		return apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}
}

// DropStorage drops the tenant database or schema created by CreateStorage (best-effort
// cleanup), then the tenant's dedicated DB account when it has one.
func (s *TenantConnectionService) DropStorage(tenant *models.Tenant) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if err := s.dropStorage(tenant); err != nil {
		return err
	}
	return dropDedicatedDBUser(tenant)
}

func (s *TenantConnectionService) dropStorage(tenant *models.Tenant) error {
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
			_, err := appfacades.PlatformQuery().Exec(sql)
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
			// Other app instances (and this one's retired pools) may still hold sessions
			// on the database, which makes a plain DROP DATABASE fail on PostgreSQL.
			// WITH (FORCE) terminates them (PostgreSQL 13+; every supported release).
			sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quotePGIdent(tenant.Database))
			drop := func(q orm.Query) error {
				_, err := q.Exec(sql)
				return err
			}
			if usePlatform {
				return drop(appfacades.PlatformQuery())
			}
			return s.withMaintenanceOrmQuery(tenant, drop)
		}
		if err := validateSQLIdent(tenant.Schema); err != nil {
			return err
		}
		if err := rejectDroppingPlatformSchema(tenant.Schema); err != nil {
			return err
		}
		schemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quotePGIdent(tenant.Schema))
		if usePlatform && (tenant.Database == "" || tenant.Database == platformDB) {
			_, err := appfacades.PlatformQuery().Exec(schemaSQL)
			return err
		}
		q, err := s.tenantQuery(tenant)
		if err != nil {
			return err
		}
		_, err = q.Exec(schemaSQL)
		// The connection was only opened to drop the schema; do not keep it registered.
		s.Forget(tenant.ConnectionName)
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

// FindTenantByIDOrCode 平台库查找租户（未删除）。经 tenantcache 缓存：每个租户请求都会调用，
// 写租户行会立即失效（全部节点），返回的是调用方可修改的副本。
func (s *TenantConnectionService) FindTenantByIDOrCode(idOrCode string) (*models.Tenant, error) {
	var (
		tenant *models.Tenant
		err    error
	)
	if id := cast.ToUint(idOrCode); id > 0 && fmt.Sprintf("%d", id) == idOrCode {
		tenant, err = tenantcache.ByID(id)
	} else {
		tenant, err = tenantcache.ByCode(idOrCode)
	}
	if err != nil {
		if errors.Is(err, tenantcache.ErrNotFound) {
			return nil, apperrors.ErrTenantNotFound.WithError(err)
		}
		// The platform database could not answer: not "no such tenant" (404), which
		// would tell clients and probes the tenant is gone while the database is down.
		return nil, apperrors.ErrTenantLookupFailed.WithError(err)
	}
	return tenant, nil
}

// BindHTTP 解析租户、注册连接并写入 HTTP context。
// hint 可空；优先级：request Host (vanity > subdomain) > Origin host (split SPA/API) > ResolveHint。
func (s *TenantConnectionService) BindHTTP(ctx http.Context, hint string) error {
	if !tenancy.Enabled() {
		return nil
	}
	clientHint := strings.TrimSpace(hint)
	if clientHint == "" {
		clientHint = tenancy.ClientHint(ctx)
	}
	raw := ""
	var err error
	if ctx != nil {
		reqHost := tenancy.RequestHostFromContext(ctx)
		raw = tenancy.PickTenantHostCode(reqHost, ctx.Request().Header("Origin", ""), s.tenantCodeFromHost)
		// Host/Origin often yield a tenant code while X-Tenant-ID / WS tickets pass a numeric id.
		if raw != "" && clientHint != "" && !strings.EqualFold(clientHint, raw) {
			if !s.hintsReferToSameTenant(clientHint, raw) {
				return apperrors.ErrTenantHintConflict
			}
			// Prefer explicit client hint (header / ticket id) when both name the same tenant.
			raw = clientHint
		}
	}
	if raw == "" {
		raw, err = tenancy.ResolveHint(ctx, hint)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(raw) == "" {
		return apperrors.ErrTenantRequired
	}
	return s.bindHTTPTenant(ctx, raw)
}

// BindHTTPByTenantID binds from a trusted tenant id (e.g. one-time WS ticket).
// Skips Host/Origin hint picking so api.* + Origin tenant subdomain cannot false-conflict.
func (s *TenantConnectionService) BindHTTPByTenantID(ctx http.Context, tenantID uint) error {
	if !tenancy.Enabled() {
		return nil
	}
	if tenantID == 0 {
		return apperrors.ErrTenantRequired
	}
	return s.bindHTTPTenant(ctx, fmt.Sprintf("%d", tenantID))
}

func (s *TenantConnectionService) hintsReferToSameTenant(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if strings.EqualFold(a, b) {
		return true
	}
	ta, errA := s.FindTenantByIDOrCode(a)
	tb, errB := s.FindTenantByIDOrCode(b)
	return errA == nil && errB == nil && ta != nil && tb != nil && ta.ID == tb.ID
}

func (s *TenantConnectionService) bindHTTPTenant(ctx http.Context, raw string) error {
	tenant, err := s.FindTenantByIDOrCode(raw)
	if err != nil {
		return err // tenant_not_found or tenant_lookup_failed
	}
	if tenant.Status != models.TenantStatusActive {
		return apperrors.ErrTenantDisabled
	}
	if tenant.Maintenance {
		if msg := strings.TrimSpace(tenant.MaintenanceMessage); msg != "" {
			return apperrors.ErrTenantMaintenance.WithMessage(msg)
		}
		return apperrors.ErrTenantMaintenance
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

// tenantCodeFromHost resolves an active vanity domain, or a built-in subdomain label.
func (s *TenantConnectionService) tenantCodeFromHost(host string) string {
	host = tenancy.NormalizeHost(host)
	if host == "" {
		return ""
	}
	if code := NewTenantDomainService().ResolveActiveCode(host); code != "" {
		return code
	}
	if tenancy.Resolver() == "subdomain" {
		return tenancy.SubdomainHint(host)
	}
	return ""
}

// BindBackground 供队列任务切换到租户连接
func (s *TenantConnectionService) BindBackground(ctx context.Context, tenantID uint) (context.Context, error) {
	if !tenancy.Enabled() || tenantID == 0 {
		return ctx, nil
	}
	tenant, err := s.FindTenantByIDOrCode(fmt.Sprintf("%d", tenantID))
	if err != nil {
		return ctx, err // tenant_not_found or tenant_lookup_failed
	}
	if tenant.Status != models.TenantStatusActive {
		return ctx, apperrors.ErrTenantDisabled
	}
	if tenant.Maintenance {
		if msg := strings.TrimSpace(tenant.MaintenanceMessage); msg != "" {
			return ctx, apperrors.ErrTenantMaintenance.WithMessage(msg)
		}
		return ctx, apperrors.ErrTenantMaintenance
	}
	if !tenant.IsProvisionReady() {
		return ctx, apperrors.ErrTenantNotReady
	}
	if err := s.EnsureRegistered(tenant); err != nil {
		return ctx, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	return tenancyctx.WithTenant(ctx, tenant.ID, tenant.ConnectionName, tenant.Code), nil
}

// TenantContext returns a background ctx bound to tenant, for platform code that
// reads or writes one tenant's data (OrmQuery(ctx), tenancy.CacheKey(ctx, ...)).
// Unlike BindBackground it does not refuse disabled / maintenance / unready tenants:
// the platform inspects those too.
//
// Code that cannot take a ctx (written against facades.Orm() / facades.Schema())
// runs inside appfacades.WithSchemaContext(ctx, fn), which binds the tenant's Schema
// to the current goroutine only.
func (s *TenantConnectionService) TenantContext(tenant *models.Tenant) (context.Context, error) {
	if err := s.EnsureRegistered(tenant); err != nil {
		return nil, err
	}
	return tenancyctx.WithTenant(context.Background(), tenant.ID, tenant.ConnectionName, tenant.Code), nil
}

// SeedTenant runs db:seed on the tenant connection.
// seeders empty => all registered seeders; otherwise same as db:seed --seeder=Name.
// Uses per-goroutine Schema+Orm binds when routers are installed (parallel-safe with migrate).
// Marks last_op=seed + last_op_status so CLI seed-all failures are visible in the platform UI
// (same as MigrateTenant). Does not change provision_status (schema may already be ready).
func (s *TenantConnectionService) SeedTenant(tenant *models.Tenant, seeders ...string) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	runErr := s.seedTenant(tenant, seeders...)
	s.markSeedOpResult(tenant, runErr)
	return runErr
}

// markSeedOpResult persists last_op fields after SeedTenant (CLI + direct callers).
// Queue RunTenantOp may overwrite these with the enclosing op (e.g. migrate+seed).
func (s *TenantConnectionService) markSeedOpResult(tenant *models.Tenant, runErr error) {
	if tenant == nil || tenant.ID == 0 {
		return
	}
	now := time.Now()
	if runErr != nil {
		msg := runErr.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
			"last_op":         models.TenantOpSeed,
			"last_op_status":  models.TenantOpStatusFailed,
			"last_op_message": msg,
			"last_op_at":      now,
		})
		tenant.LastOp = models.TenantOpSeed
		tenant.LastOpStatus = models.TenantOpStatusFailed
		tenant.LastOpMessage = msg
		tenant.LastOpAt = &now
		return
	}
	_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"last_op":         models.TenantOpSeed,
		"last_op_status":  models.TenantOpStatusSuccess,
		"last_op_message": "seed ok",
		"last_op_at":      now,
	})
	tenant.LastOp = models.TenantOpSeed
	tenant.LastOpStatus = models.TenantOpStatusSuccess
	tenant.LastOpMessage = "seed ok"
	tenant.LastOpAt = &now
}

// seedTenant runs the seeders on this goroutine with a fresh tenant Schema bound, so
// seeders' facades.Orm() / facades.Schema() hit the tenant database.
func (s *TenantConnectionService) seedTenant(tenant *models.Tenant, seeders ...string) error {
	sch, err := s.openTenantSchema(tenant)
	if err != nil {
		return err
	}
	return appfacades.WithBoundTenantSchema(sch, func() error {
		// Call seeders on this goroutine. Artisan.Call runs Handle in a NEW goroutine
		// (framework console), which would not see the binding.
		if err := runTenantSeeders(seeders...); err != nil {
			return err
		}
		n, err := sch.Orm().Query().Table("admins").Count()
		if err != nil {
			return fmt.Errorf("seed verify failed: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("seed failed: admins empty on %s", sch.Orm().DatabaseName())
		}
		return nil
	})
}

// runTenantSeeders runs registered seeders on the current goroutine (same as
// db:seed --force [--seeder=...], without Artisan's extra Handle goroutine).
func runTenantSeeders(names ...string) error {
	facade := facades.Seeder()
	if facade == nil {
		return fmt.Errorf("seeder facade unavailable")
	}
	var list []contractsseeder.Seeder
	trimmed := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			trimmed = append(trimmed, name)
		}
	}
	if len(trimmed) == 0 {
		list = facade.GetSeeders()
	} else {
		list = make([]contractsseeder.Seeder, 0, len(trimmed))
		for _, name := range trimmed {
			item := facade.GetSeeder(name)
			if item == nil {
				return fmt.Errorf("seeder not found: %s", name)
			}
			list = append(list, item)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return facade.Call(list)
}

// MigrateTenant runs migrations on the tenant connection and marks provision_status=ready.
// Uses a per-goroutine Schema bind (no global database.default flip) so tenant:migrate-all
// concurrency and long-running workers can run true parallel DDL.
func (s *TenantConnectionService) MigrateTenant(tenant *models.Tenant) error {
	var migCount int64
	err := s.migrateTenant(tenant, &migCount)
	now := time.Now()
	if err != nil {
		_ = s.SetProvisionStatus(tenant, models.TenantProvisionFailed)
		msg := err.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
			"last_migrate_error": msg,
			"last_op":            models.TenantOpMigrate,
			"last_op_status":     models.TenantOpStatusFailed,
			"last_op_message":    msg,
			"last_op_at":         now,
		})
		tenant.LastMigrateError = msg
		tenant.LastOp = models.TenantOpMigrate
		tenant.LastOpStatus = models.TenantOpStatusFailed
		tenant.LastOpMessage = msg
		tenant.LastOpAt = &now
		return err
	}
	if err := s.SetProvisionStatus(tenant, models.TenantProvisionReady); err != nil {
		return err
	}
	_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"last_migrate_error":     "",
		"migrated_at":            now,
		"schema_migration_count": migCount,
		"last_op":                models.TenantOpMigrate,
		"last_op_status":         models.TenantOpStatusSuccess,
		"last_op_message":        "migrate ok",
		"last_op_at":             now,
	})
	tenant.LastMigrateError = ""
	tenant.MigratedAt = &now
	tenant.SchemaMigrationCount = migCount
	tenant.LastOp = models.TenantOpMigrate
	tenant.LastOpStatus = models.TenantOpStatusSuccess
	tenant.LastOpMessage = "migrate ok"
	tenant.LastOpAt = &now
	return nil
}

// RollbackTenant rolls back migrations on the tenant connection.
// step>0: last N migration files; migBatch>0: that migrations.batch; both 0: last batch (GetLast).
// Uses BindTenantSchema so it can run in parallel with TENANCY_MIGRATE_CONCURRENCY.
func (s *TenantConnectionService) RollbackTenant(tenant *models.Tenant, step, migBatch int) error {
	if step < 0 {
		step = 0
	}
	if migBatch < 0 {
		migBatch = 0
	}
	var migCount int64
	err := s.rollbackTenant(tenant, step, migBatch, &migCount)
	now := time.Now()
	if err != nil {
		msg := err.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
			"last_op":         models.TenantOpRollback,
			"last_op_status":  models.TenantOpStatusFailed,
			"last_op_message": msg,
			"last_op_at":      now,
		})
		tenant.LastOp = models.TenantOpRollback
		tenant.LastOpStatus = models.TenantOpStatusFailed
		tenant.LastOpMessage = msg
		tenant.LastOpAt = &now
		return err
	}

	provision := models.TenantProvisionReady
	// Bind briefly to check admins still exists after rollback.
	if checkErr := s.withTenantHasAdmins(tenant); checkErr != nil {
		provision = models.TenantProvisionPending
	}
	_ = s.SetProvisionStatus(tenant, provision)
	msg := fmt.Sprintf("rollback ok (step=%d batch=%d)", step, migBatch)
	_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"schema_migration_count": migCount,
		"last_op":                models.TenantOpRollback,
		"last_op_status":         models.TenantOpStatusSuccess,
		"last_op_message":        msg,
		"last_op_at":             now,
		"last_migrate_error":     "",
	})
	tenant.SchemaMigrationCount = migCount
	tenant.LastOp = models.TenantOpRollback
	tenant.LastOpStatus = models.TenantOpStatusSuccess
	tenant.LastOpMessage = msg
	tenant.LastOpAt = &now
	tenant.LastMigrateError = ""
	return nil
}

func (s *TenantConnectionService) withTenantHasAdmins(tenant *models.Tenant) error {
	sch, err := s.openTenantSchema(tenant)
	if err != nil {
		return err
	}
	if !sch.HasTable("admins") {
		return fmt.Errorf("admins missing")
	}
	return nil
}

// rollbackTenant rolls back on a fresh tenant Schema bound to this goroutine (so
// migrations' Down() via facades.Schema() hit the tenant); safe to run in parallel.
func (s *TenantConnectionService) rollbackTenant(tenant *models.Tenant, step, migBatch int, migCount *int64) error {
	sch, err := s.openTenantSchema(tenant)
	if err != nil {
		return err
	}
	return appfacades.WithBoundTenantSchema(sch, func() error {
		migrator := migration.NewMigrator(facades.Artisan(), sch, "migrations")
		if err := migrator.Rollback(step, migBatch); err != nil {
			return err
		}
		return countTenantMigrations(sch, migCount)
	})
}

// migrateTenant migrates on a fresh tenant Schema bound to this goroutine (so
// migrations' Up() via facades.Schema() hit the tenant); safe to run in parallel.
func (s *TenantConnectionService) migrateTenant(tenant *models.Tenant, migCount *int64) error {
	sch, err := s.openTenantSchema(tenant)
	if err != nil {
		return err
	}
	return appfacades.WithBoundTenantSchema(sch, func() error {
		// Call Migrator.Run directly: Artisan migrate returns nil even on failure.
		migrator := migration.NewMigrator(facades.Artisan(), sch, "migrations")
		if err := migrator.Run(); err != nil {
			return err
		}
		if !sch.HasTable("admins") {
			return fmt.Errorf("migrate failed: admins table missing on %s", sch.Orm().DatabaseName())
		}
		return countTenantMigrations(sch, migCount)
	})
}

func countTenantMigrations(sch schemaContract.Schema, migCount *int64) error {
	if !sch.HasTable("migrations") {
		return nil
	}
	n, err := sch.Orm().Query().Table("migrations").Count()
	if err != nil {
		return err
	}
	*migCount = n
	return nil
}

// openTenantSchema registers the tenant connection and returns a fresh Schema on it.
// Each caller gets its own instance: the migrator mutates the Schema it runs on.
func (s *TenantConnectionService) openTenantSchema(tenant *models.Tenant) (schemaContract.Schema, error) {
	c, err := s.ensureConn(tenant)
	if err != nil {
		return nil, err
	}
	sch, err := c.NewSchema()
	if err != nil {
		return nil, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	return sch, nil
}

// Ping verifies the tenant database connection is reachable. A failing connection is
// unregistered so the next use rebuilds it.
func (s *TenantConnectionService) Ping(tenant *models.Tenant, timeout time.Duration) error {
	c, err := s.ensureConn(tenant)
	if err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		s.Forget(tenant.ConnectionName)
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
