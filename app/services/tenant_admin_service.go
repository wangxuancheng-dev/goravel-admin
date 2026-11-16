package services

import (
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenantstorage"
)

// errCrossDriverEndpoint rejects a tenant on another driver than the platform database
// without its own host and port (it cannot fall back to the platform server's).
var errCrossDriverEndpoint = apperrors.ErrInvalidArgument.WithMessage(
	"host and port are required when the tenant driver differs from the platform database driver")

// TenantCreateInput create tenant from API / CLI shared fields.
type TenantCreateInput struct {
	Code                string
	Name                string
	Driver              string
	Isolation           string
	Database            string
	Schema              string
	Host                string
	Port                int
	Username            string
	Password            string
	Migrate             bool
	SkipCreate          bool // remote DB already exists: skip CREATE DATABASE/SCHEMA
	StorageLimitBytes   *int64
	StorageMode         string
	StorageDriver       string
	StorageKey          string
	StorageSecret       string
	StorageRegion       string
	StorageBucket       string
	StorageURL          string
	StorageEndpoint     string
	StorageUsePathStyle *bool
	StorageSSL          *bool
}

// TenantUpdateInput updates connection metadata for an existing tenant.
type TenantUpdateInput struct {
	Name                *string
	Host                *string
	Port                *int
	Username            *string
	Password            *string
	Database            *string
	Schema              *string
	StorageLimitBytes   *int64
	StorageMode         *string
	StorageDriver       *string
	StorageKey          *string
	StorageSecret       *string
	StorageRegion       *string
	StorageBucket       *string
	StorageURL          *string
	StorageEndpoint     *string
	StorageUsePathStyle *bool
	StorageSSL          *bool
	ClearStorageSecret  *bool
}

type TenantAdminService struct {
	conn *TenantConnectionService
}

func NewTenantAdminService() *TenantAdminService {
	return &TenantAdminService{conn: NewTenantConnectionService()}
}

func (s *TenantAdminService) requireEnabled() error {
	if !tenancy.Enabled() {
		return apperrors.ErrTenancyDisabled
	}
	return nil
}

func (s *TenantAdminService) GetList(filters TenantAdminFilters, page, pageSize int) ([]models.Tenant, int64, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, 0, err
	}
	query := appfacades.PlatformQuery().Model(&models.Tenant{})
	trashed := strings.ToLower(strings.TrimSpace(filters.Trashed))
	switch trashed {
	case "only":
		query = query.WithTrashed().Where("deleted_at IS NOT NULL")
	case "with":
		query = query.WithTrashed()
	}
	if filters.Code != "" {
		query = query.Where("code like ?", "%"+filters.Code+"%")
	}
	if filters.Name != "" {
		query = query.Where("name like ?", "%"+filters.Name+"%")
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}
	if filters.ProvisionStatus != "" {
		query = query.Where("provision_status", filters.ProvisionStatus)
	}
	if op := NormalizeTenantLastOpFilter(filters.LastOp); op != "" {
		query = query.Where("last_op", op)
	}
	if st := NormalizeTenantLastOpStatusFilter(filters.LastOpStatus); st != "" {
		query = query.Where("last_op_status", st)
	}
	switch strings.ToLower(filters.Maintenance) {
	case "1", "true", "yes", "on":
		query = query.Where("maintenance", true)
	case "0", "false", "no", "off":
		query = query.Where("maintenance", false)
	}
	if hs := strings.ToLower(strings.TrimSpace(filters.HealthStatus)); hs != "" {
		query = query.Where("health_status", hs)
	}
	query = applyDomainFilters(query, filters.DomainHost, filters.DomainStatus)
	query = applySchemaStatusFilter(query, filters.SchemaStatus)
	query = applySubscriptionFilters(query, filters.PlanCode, filters.Expiring)
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var list []models.Tenant
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *TenantAdminService) GetByID(id uint) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var tenant models.Tenant
	if err := appfacades.PlatformQuery().Where("id", id).First(&tenant); err != nil {
		return nil, apperrors.ErrTenantNotFound.WithError(err)
	}
	// Goravel's First reports no error for a missing row and leaves dest empty.
	if tenant.ID == 0 {
		return nil, apperrors.ErrTenantNotFound
	}
	return &tenant, nil
}

// GetByIDIncludingTrashed loads active or soft-deleted tenant metadata.
func (s *TenantAdminService) GetByIDIncludingTrashed(id uint) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var tenant models.Tenant
	if err := appfacades.PlatformQuery().WithTrashed().Where("id", id).First(&tenant); err != nil {
		return nil, apperrors.ErrTenantNotFound.WithError(err)
	}
	if tenant.ID == 0 { // missing row: Goravel's First reports no error
		return nil, apperrors.ErrTenantNotFound
	}
	return &tenant, nil
}

// TenantIsTrashed reports whether soft-delete is set.
func TenantIsTrashed(t *models.Tenant) bool {
	return t != nil && t.DeletedAt.Valid
}

func (s *TenantAdminService) Create(input TenantCreateInput) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	code, err := NormalizeTenantCode(input.Code)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperrors.ErrInvalidArgument.WithMessage("name is required")
	}

	var existing models.Tenant
	if appfacades.PlatformQuery().WithTrashed().Where("code", code).First(&existing) == nil && existing.ID > 0 {
		if TenantIsTrashed(&existing) {
			return nil, apperrors.ErrTenantCodeInRecycle
		}
		return nil, apperrors.ErrTenantExists
	}

	driverName := NormalizeTenantDriver(input.Driver)
	if driverName == "" {
		driverName = NormalizeTenantDriver(facades.Config().GetString("database.default", "mysql"))
	}
	isolation, err := ResolveTenantIsolation(driverName, input.Isolation)
	if err != nil {
		return nil, err
	}
	// An empty host / port means "the platform database server's", which only works
	// for the platform's own driver: a postgres tenant cannot use a MySQL server's.
	if !samePlatformDriver(driverName) && (strings.TrimSpace(input.Host) == "" || input.Port <= 0) {
		return nil, errCrossDriverEndpoint
	}

	database := strings.TrimSpace(input.Database)
	if database == "" {
		if isolation == models.TenantIsolationSchema {
			database = facades.Config().GetString("database.connections."+driverName+".database", "goravel")
		} else {
			database = DefaultTenantDatabaseName(code)
		}
	}
	schemaName := strings.TrimSpace(input.Schema)
	if isolation == models.TenantIsolationSchema && schemaName == "" {
		schemaName = DefaultTenantSchemaName(code)
	}

	username, password := strings.TrimSpace(input.Username), input.Password
	if username != "" && username == DedicatedTenantDBUsername(code) {
		// Reserved: an account with this name is created and dropped with the tenant.
		return nil, apperrors.ErrInvalidArgument.WithMessage("username " + username + " is reserved for the tenant's dedicated database account")
	}
	if DedicatedTenantDBUserEnabled() && !input.SkipCreate && strings.TrimSpace(input.Host) == "" &&
		username == "" && strings.TrimSpace(password) == "" {
		// An account of its own (created with the storage): see tenant_db_user.go.
		username = DedicatedTenantDBUsername(code)
		if password, err = generateDBPassword(); err != nil {
			return nil, apperrors.ErrPasswordEncryptFailed.WithError(err)
		}
	}
	if err = ValidateTenantCredentials(input.Host, username, password); err != nil {
		return nil, err
	}

	sealedPassword, err := SealTenantPassword(password)
	if err != nil {
		return nil, apperrors.ErrPasswordEncryptFailed.WithError(err)
	}

	tenant := models.Tenant{
		Code:            code,
		Name:            name,
		Status:          models.TenantStatusActive,
		ProvisionStatus: models.TenantProvisionPending,
		Driver:          driverName,
		Isolation:       isolation,
		Host:            strings.TrimSpace(input.Host),
		Port:            input.Port,
		Database:        database,
		Schema:          schemaName,
		Username:        username,
		Password:        sealedPassword,
		ConnectionName:  pendingConnectionName(code),
	}
	if input.StorageLimitBytes != nil && *input.StorageLimitBytes > 0 {
		tenant.StorageLimitBytes = *input.StorageLimitBytes
	}
	if err := applyTenantStorageOnCreate(&tenant, input); err != nil {
		return nil, err
	}
	if err := appfacades.PlatformQuery().Create(&tenant); err != nil {
		return nil, err
	}
	tenant.ConnectionName = TenantConnectionName(tenant.ID)
	// Best effort: indexes a former tenant with this code left behind (deleted before
	// indexes were dropped with tenants) must not serve the new tenant.
	_ = dropTenantSearchIndexes(&tenant)
	if _, err := appfacades.PlatformQuery().Model(&tenant).Update(map[string]any{
		"connection_name": tenant.ConnectionName,
	}); err != nil {
		s.discardFailedCreate(&tenant, false)
		return nil, err
	}
	if err := s.conn.CreateStorageWithOptions(&tenant, input.SkipCreate); err != nil {
		if be, ok := apperrors.GetBusinessError(err); ok && be.Code == apperrors.ErrTenantStorageExists.Code {
			// Not ours: the existing database stays untouched.
			s.discardFailedCreate(&tenant, false)
			return nil, err
		}
		// Anything under this name was created by this attempt (an existing database is
		// refused above), so it goes too; skip_create storage belongs to the DBA.
		s.discardFailedCreate(&tenant, !input.SkipCreate)
		return nil, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	if input.Migrate {
		if err := s.conn.MigrateTenant(&tenant); err != nil {
			return nil, err
		}
		if err := s.conn.SeedTenant(&tenant); err != nil {
			_ = s.conn.SetProvisionStatus(&tenant, models.TenantProvisionFailed)
			_, _ = appfacades.PlatformQuery().Model(&tenant).Update(map[string]any{
				"last_op":         models.TenantOpSeed,
				"last_op_status":  models.TenantOpStatusFailed,
				"last_op_message": err.Error(),
			})
			return nil, err
		}
	}
	return &tenant, nil
}

// discardFailedCreate undoes a Create that failed before the tenant was usable: the row
// is removed for good (a soft delete would keep the code in the recycle bin and block
// the retry), with the database / schema this attempt created when dropStorage, and the
// dedicated DB account.
func (s *TenantAdminService) discardFailedCreate(tenant *models.Tenant, dropStorage bool) {
	s.conn.Forget(tenant.ConnectionName)
	if dropStorage {
		if err := s.conn.DropStorage(tenant); err != nil {
			facades.Log().Warningf("tenant create %s failed; dropping its storage: %v", tenant.Code, err)
		}
	} else {
		_ = dropDedicatedDBUser(tenant)
	}
	_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tenant.ID).ForceDelete(&models.Tenant{})
}

func (s *TenantAdminService) SetStatus(id uint, status uint8) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	tenant, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if status != models.TenantStatusActive && status != models.TenantStatusDisabled {
		return nil, apperrors.ErrInvalidArgument.WithMessage("status must be 0 or 1")
	}
	if status == models.TenantStatusActive && !tenant.IsProvisionReady() {
		return nil, apperrors.ErrTenantNotReady
	}
	if _, err := appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"status": status,
	}); err != nil {
		return nil, err
	}
	tenant.Status = status
	if status == models.TenantStatusDisabled {
		s.conn.Forget(tenant.ConnectionName)
	}
	return tenant, nil
}

// SetMaintenance toggles per-tenant maintenance mode (blocks business BindHTTP/BindBackground).
func (s *TenantAdminService) SetMaintenance(id uint, enabled bool, message string) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	tenant, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	msg := strings.TrimSpace(message)
	// maintenance_message is VARCHAR(500): cut by characters so multi-byte text stays valid UTF-8.
	if r := []rune(msg); len(r) > 500 {
		msg = string(r[:500])
	}
	if !enabled {
		msg = ""
	}
	if _, err := appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"maintenance":         enabled,
		"maintenance_message": msg,
	}); err != nil {
		return nil, err
	}
	tenant.Maintenance = enabled
	tenant.MaintenanceMessage = msg
	if enabled {
		s.conn.Forget(tenant.ConnectionName)
	}
	return tenant, nil
}

func (s *TenantAdminService) UpdateConnection(id uint, input TenantUpdateInput) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	tenant, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, apperrors.ErrInvalidArgument.WithMessage("name is required")
		}
		updates["name"] = name
		tenant.Name = name
	}
	if input.Host != nil {
		updates["host"] = strings.TrimSpace(*input.Host)
		tenant.Host = strings.TrimSpace(*input.Host)
	}
	if input.Port != nil {
		updates["port"] = *input.Port
		tenant.Port = *input.Port
	}
	if input.Username != nil {
		updates["username"] = strings.TrimSpace(*input.Username)
		tenant.Username = strings.TrimSpace(*input.Username)
	}
	if input.Password != nil {
		sealed, err := SealTenantPassword(*input.Password)
		if err != nil {
			return nil, apperrors.ErrPasswordEncryptFailed.WithError(err)
		}
		updates["password"] = sealed
		tenant.Password = sealed
	}
	if input.Database != nil {
		dbName := strings.TrimSpace(*input.Database)
		if dbName == "" {
			return nil, apperrors.ErrInvalidArgument.WithMessage("database is required")
		}
		updates["database"] = dbName
		tenant.Database = dbName
	}
	if input.Schema != nil {
		updates["schema"] = strings.TrimSpace(*input.Schema)
		tenant.Schema = strings.TrimSpace(*input.Schema)
	}
	if input.StorageLimitBytes != nil {
		limit := *input.StorageLimitBytes
		if limit < 0 {
			limit = 0
		}
		updates["storage_limit_bytes"] = limit
		tenant.StorageLimitBytes = limit
	}
	if err := mergeTenantStorageUpdates(tenant, input, updates); err != nil {
		return nil, err
	}
	if len(updates) == 0 {
		return tenant, nil
	}

	if (input.Host != nil || input.Port != nil) && (tenant.Host == "" || tenant.Port <= 0) &&
		!samePlatformDriver(NormalizeTenantDriver(tenant.Driver)) {
		return nil, errCrossDriverEndpoint
	}

	checkHost := tenant.Host
	checkUser := tenant.Username
	checkPass := ""
	if TenantHasPassword(tenant.Password) {
		checkPass = "set"
	}
	if input.Password != nil {
		checkPass = *input.Password
	}
	if err := ValidateTenantCredentials(checkHost, checkUser, checkPass); err != nil {
		return nil, err
	}

	// Dedicated accounts: ALTER USER before updating the row, then Forget the pool, so
	// no request rebuilds a connection with a password the server does not have yet.
	syncDedicatedPassword := input.Password != nil && usesDedicatedDBUser(tenant)
	if syncDedicatedPassword {
		if !isAlnum(*input.Password) {
			return nil, apperrors.ErrInvalidArgument.WithMessage("dedicated tenant DB password must be alphanumeric")
		}
		if err := createDedicatedDBUser(tenant); err != nil {
			return nil, apperrors.ErrTenantConnectionFailed.WithError(err)
		}
		s.conn.Forget(tenant.ConnectionName)
	}

	if _, err := appfacades.PlatformQuery().Model(tenant).Update(updates); err != nil {
		return nil, err
	}
	if !syncDedicatedPassword {
		s.conn.Forget(tenant.ConnectionName)
	}
	tenantstorage.Invalidate(tenant.ID)
	return tenant, nil
}

// EnsureDedicatedDBUser creates or rotates a same-host dedicated DB account for an
// existing tenant (backfill for tenants created before TENANCY_DB_DEDICATED_USER, or
// password rotation with opts.RotatePassword). Does not require the config flag: the
// CLI / operator opts in.
func (s *TenantAdminService) EnsureDedicatedDBUser(id uint, opts DedicatedDBUserEnsureOpts) (DedicatedDBUserEnsureResult, error) {
	out := DedicatedDBUserEnsureResult{}
	if err := s.requireEnabled(); err != nil {
		return out, err
	}
	tenant, err := s.GetByID(id)
	if err != nil {
		return out, err
	}
	out.TenantID = tenant.ID
	out.Code = tenant.Code
	action, detail := ClassifyDedicatedDBUserEnsure(tenant, opts)
	out.Action = action
	out.Detail = detail
	if action == DedicatedDBUserActionSkipped || opts.DryRun {
		return out, nil
	}

	password, err := generateDBPassword()
	if err != nil {
		return out, apperrors.ErrPasswordEncryptFailed.WithError(err)
	}
	sealed, err := SealTenantPassword(password)
	if err != nil {
		return out, apperrors.ErrPasswordEncryptFailed.WithError(err)
	}
	wantUser := DedicatedTenantDBUsername(tenant.Code)
	tenant.Username = wantUser
	tenant.Password = sealed

	if err := createDedicatedDBUser(tenant); err != nil {
		return out, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	if err := grantDedicatedDBUser(tenant); err != nil {
		return out, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	if err := reclaimDedicatedDBOwnership(tenant); err != nil {
		return out, apperrors.ErrTenantConnectionFailed.WithError(err)
	}

	// ALTER / CREATE before the row update, then Forget, so rebuilds use the new secret.
	s.conn.Forget(tenant.ConnectionName)
	if _, err := appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"username": wantUser,
		"password": sealed,
	}); err != nil {
		return out, err
	}
	if err := s.conn.Ping(tenant, 5*time.Second); err != nil {
		return out, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	return out, nil
}

func (s *TenantAdminService) ListAll() ([]models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var list []models.Tenant
	if err := appfacades.PlatformQuery().Model(&models.Tenant{}).Order("id asc").Find(&list); err != nil {
		return nil, err
	}
	return list, nil
}
