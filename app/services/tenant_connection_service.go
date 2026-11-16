package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenantcache"
	"goravel/app/tenantdb"
)

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
