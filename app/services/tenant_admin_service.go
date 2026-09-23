package services

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenantstorage"
)

// TenantAdminFilters platform tenants list filters.
type TenantAdminFilters struct {
	Code            string
	Name            string
	Status          string
	ProvisionStatus string
	// SchemaStatus: aligned|behind|failed|running|unknown (computed vs current binary).
	SchemaStatus string
	// LastOp: migrate|seed|backup|restore|purge
	LastOp string
	// LastOpStatus: idle|queued|running|success|failed
	LastOpStatus string
	// Maintenance: "" = any; "1"/"true" = on; "0"/"false" = off.
	Maintenance string
	// Trashed: "" = exclude soft-deleted (default); "only" = recycle bin; "with" = include both.
	Trashed string
	// DomainHost: substring match on tenant_domains.host
	DomainHost string
	// DomainStatus: unbound|pending|active|verify_failed|disabled
	DomainStatus string
	// HealthStatus: ok|warn|fail|unknown
	HealthStatus string
}

func BuildTenantAdminFiltersFromHTTP(ctx http.Context) TenantAdminFilters {
	return TenantAdminFilters{
		Code:            strings.TrimSpace(ctx.Request().Query("code", "")),
		Name:            strings.TrimSpace(ctx.Request().Query("name", "")),
		Status:          strings.TrimSpace(ctx.Request().Query("status", "")),
		ProvisionStatus: strings.TrimSpace(ctx.Request().Query("provision_status", "")),
		SchemaStatus:    strings.TrimSpace(ctx.Request().Query("schema_status", "")),
		LastOp:          strings.TrimSpace(ctx.Request().Query("last_op", "")),
		LastOpStatus:    strings.TrimSpace(ctx.Request().Query("last_op_status", "")),
		Maintenance:     strings.TrimSpace(ctx.Request().Query("maintenance", "")),
		Trashed:         strings.TrimSpace(ctx.Request().Query("trashed", "")),
		DomainHost:      strings.TrimSpace(ctx.Request().Query("domain_host", "")),
		DomainStatus:    strings.TrimSpace(ctx.Request().Query("domain_status", "")),
		HealthStatus:    strings.TrimSpace(ctx.Request().Query("health_status", "")),
	}
}

// NormalizeTenantLastOpFilter returns a known op name or empty (ignore unknown).
func NormalizeTenantLastOpFilter(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case models.TenantOpMigrate, models.TenantOpSeed, models.TenantOpBackup, models.TenantOpRestore, models.TenantOpPurge:
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

// NormalizeTenantLastOpStatusFilter returns a known status or empty (ignore unknown).
func NormalizeTenantLastOpStatusFilter(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case models.TenantOpStatusIdle, models.TenantOpStatusQueued, models.TenantOpStatusRunning,
		models.TenantOpStatusSuccess, models.TenantOpStatusFailed:
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

// TenantCreateInput create tenant from API / CLI shared fields.
type TenantCreateInput struct {
	Code              string
	Name              string
	Driver            string
	Isolation         string
	Database          string
	Schema            string
	Host              string
	Port              int
	Username          string
	Password          string
	Migrate           bool
	SkipCreate        bool // remote DB already exists: skip CREATE DATABASE/SCHEMA
	StorageLimitBytes *int64
	StorageMode       string
	StorageDriver     string
	StorageKey        string
	StorageSecret     string
	StorageRegion     string
	StorageBucket     string
	StorageURL        string
	StorageEndpoint   string
	StorageUsePathStyle *bool
	StorageSSL        *bool
}

// TenantUpdateInput updates connection metadata for an existing tenant.
type TenantUpdateInput struct {
	Name              *string
	Host              *string
	Port              *int
	Username          *string
	Password          *string
	Database          *string
	Schema            *string
	StorageLimitBytes *int64
	StorageMode       *string
	StorageDriver     *string
	StorageKey        *string
	StorageSecret     *string
	StorageRegion     *string
	StorageBucket     *string
	StorageURL        *string
	StorageEndpoint   *string
	StorageUsePathStyle *bool
	StorageSSL        *bool
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
	query := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{})
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

func applySchemaStatusFilter(query orm.Query, schemaStatus string) orm.Query {
	status := strings.ToLower(strings.TrimSpace(schemaStatus))
	if status == "" {
		return query
	}
	expected := ExpectedSchemaMigrationCount()
	switch status {
	case TenantSchemaRunning:
		return query.Where(
			"(provision_status = ? OR (last_op = ? AND last_op_status IN (?, ?)))",
			models.TenantProvisionMigrating,
			models.TenantOpMigrate,
			models.TenantOpStatusQueued,
			models.TenantOpStatusRunning,
		)
	case TenantSchemaFailed:
		return query.Where(
			"((last_migrate_error IS NOT NULL AND last_migrate_error != '') OR provision_status = ? OR (last_op = ? AND last_op_status = ?))",
			models.TenantProvisionFailed,
			models.TenantOpMigrate,
			models.TenantOpStatusFailed,
		)
	case TenantSchemaAligned:
		return query.Where("provision_status = ?", models.TenantProvisionReady).
			Where("schema_migration_count >= ?", expected).
			Where("(last_migrate_error IS NULL OR last_migrate_error = '')")
	case TenantSchemaUnknown:
		return query.Where("provision_status = ?", models.TenantProvisionReady).
			Where("schema_migration_count = ?", 0).
			Where("(last_migrate_error IS NULL OR last_migrate_error = '')")
	case TenantSchemaBehind:
		// Pending, or count below waterline, excluding failed/running/unknown-aligned cases.
		return query.Where(
			`(
				provision_status = ?
				OR (
					schema_migration_count < ?
					AND provision_status = ?
					AND schema_migration_count > 0
					AND (last_migrate_error IS NULL OR last_migrate_error = '')
				)
			)
			AND provision_status NOT IN (?, ?)
			AND NOT (last_op = ? AND last_op_status IN (?, ?))`,
			models.TenantProvisionPending,
			expected,
			models.TenantProvisionReady,
			models.TenantProvisionMigrating,
			models.TenantProvisionFailed,
			models.TenantOpMigrate,
			models.TenantOpStatusQueued,
			models.TenantOpStatusRunning,
		)
	default:
		return query
	}
}

func (s *TenantAdminService) GetByID(id uint) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var tenant models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Where("id", id).First(&tenant); err != nil {
		return nil, apperrors.ErrTenantNotFound.WithError(err)
	}
	return &tenant, nil
}

// GetByIDIncludingTrashed loads active or soft-deleted tenant metadata.
func (s *TenantAdminService) GetByIDIncludingTrashed(id uint) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var tenant models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).WithTrashed().Where("id", id).First(&tenant); err != nil {
		return nil, apperrors.ErrTenantNotFound.WithError(err)
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
	if err := appfacades.PlatformOrmQuery(nil).WithTrashed().Where("code", code).First(&existing); err == nil && existing.ID > 0 {
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

	if err := ValidateTenantCredentials(input.Host, input.Username, input.Password); err != nil {
		return nil, err
	}

	sealedPassword, err := SealTenantPassword(input.Password)
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
		Username:        strings.TrimSpace(input.Username),
		Password:        sealedPassword,
		ConnectionName:  fmt.Sprintf("tenant_pending_%s", code),
	}
	if input.StorageLimitBytes != nil && *input.StorageLimitBytes > 0 {
		tenant.StorageLimitBytes = *input.StorageLimitBytes
	}
	if err := applyTenantStorageOnCreate(&tenant, input); err != nil {
		return nil, err
	}
	if err := appfacades.PlatformOrmQuery(nil).Create(&tenant); err != nil {
		return nil, err
	}
	tenant.ConnectionName = TenantConnectionName(tenant.ID)
	if _, err := appfacades.PlatformOrmQuery(nil).Model(&tenant).Update(map[string]any{
		"connection_name": tenant.ConnectionName,
	}); err != nil {
		_, _ = appfacades.PlatformOrmQuery(nil).Delete(&tenant)
		return nil, err
	}
	if err := s.conn.CreateStorageWithOptions(&tenant, input.SkipCreate); err != nil {
		s.conn.Forget(tenant.ConnectionName)
		_, _ = appfacades.PlatformOrmQuery(nil).Delete(&tenant)
		return nil, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	if input.Migrate {
		if err := s.conn.MigrateTenant(&tenant); err != nil {
			return nil, err
		}
		if err := s.conn.SeedTenant(&tenant); err != nil {
			_ = s.conn.SetProvisionStatus(&tenant, models.TenantProvisionFailed)
			_, _ = appfacades.PlatformOrmQuery(nil).Model(&tenant).Update(map[string]any{
				"last_op":         models.TenantOpSeed,
				"last_op_status":  models.TenantOpStatusFailed,
				"last_op_message": err.Error(),
			})
			return nil, err
		}
	}
	return &tenant, nil
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
	if _, err := appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
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
	if len(msg) > 500 {
		msg = msg[:500]
	}
	if !enabled {
		msg = ""
	}
	if _, err := appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
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

	if _, err := appfacades.PlatformOrmQuery(nil).Model(tenant).Update(updates); err != nil {
		return nil, err
	}
	s.conn.Forget(tenant.ConnectionName)
	tenantstorage.Invalidate(tenant.ID)
	return tenant, nil
}

func (s *TenantAdminService) ListAll() ([]models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var list []models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).Order("id asc").Find(&list); err != nil {
		return nil, err
	}
	return list, nil
}

// TenantOpsSummary aggregates provision / op status counts for the platform console.
type TenantOpsSummary struct {
	Total                  int64             `json:"total"`
	ByProvision            map[string]int64  `json:"by_provision"`
	ByLastOpStatus         map[string]int64  `json:"by_last_op_status"`
	FailedProvision        int64             `json:"failed_provision"`
	Busy                   int64             `json:"busy"`
	Deleted                int64             `json:"deleted"`
	FailedPurge            int64             `json:"failed_purge"`
	ExpectedMigrationCount int64             `json:"expected_migration_count"`
	SchemaAligned          int64             `json:"schema_aligned"`
	SchemaBehind           int64             `json:"schema_behind"`
	SchemaFailed           int64             `json:"schema_failed"`
	SchemaRunning          int64             `json:"schema_running"`
	SchemaUnknown          int64             `json:"schema_unknown"`
	BySchemaStatus         map[string]int64  `json:"by_schema_status"`
	Maintenance            int64             `json:"maintenance"`
	DomainUnbound          int64             `json:"domain_unbound"`
	DomainPending          int64             `json:"domain_pending"`
	DomainActive           int64             `json:"domain_active"`
	DomainVerifyFailed     int64             `json:"domain_verify_failed"`
	DomainDisabled         int64             `json:"domain_disabled"`
	HealthOK               int64             `json:"health_ok"`
	HealthWarn             int64             `json:"health_warn"`
	HealthFail             int64             `json:"health_fail"`
	HealthUnknown          int64             `json:"health_unknown"`
}

func (s *TenantAdminService) OpsSummary() (*TenantOpsSummary, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var tenants []models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).
		Select("id", "provision_status", "last_op", "last_op_status", "last_migrate_error", "schema_migration_count", "maintenance", "health_status").
		Get(&tenants); err != nil {
		return nil, err
	}
	sum := &TenantOpsSummary{
		Total:          int64(len(tenants)),
		ByProvision:    map[string]int64{},
		ByLastOpStatus: map[string]int64{},
	}
	ids := make([]uint, 0, len(tenants))
	for i := range tenants {
		t := &tenants[i]
		ids = append(ids, t.ID)
		ps := strings.TrimSpace(t.ProvisionStatus)
		if ps == "" {
			ps = models.TenantProvisionPending
		}
		sum.ByProvision[ps]++
		if ps == models.TenantProvisionFailed {
			sum.FailedProvision++
		}
		os := strings.TrimSpace(t.LastOpStatus)
		if os == "" {
			os = models.TenantOpStatusIdle
		}
		sum.ByLastOpStatus[os]++
		if ps == models.TenantProvisionMigrating || os == models.TenantOpStatusQueued || os == models.TenantOpStatusRunning {
			sum.Busy++
		}
		if t.Maintenance {
			sum.Maintenance++
		}
		switch strings.ToLower(strings.TrimSpace(t.HealthStatus)) {
		case "ok":
			sum.HealthOK++
		case "warn":
			sum.HealthWarn++
		case "fail":
			sum.HealthFail++
		default:
			sum.HealthUnknown++
		}
	}
	schema := CountSchemaStatuses(tenants)
	sum.ExpectedMigrationCount = schema.Expected
	sum.SchemaAligned = schema.Aligned
	sum.SchemaBehind = schema.Behind
	sum.SchemaFailed = schema.Failed
	sum.SchemaRunning = schema.Running
	sum.SchemaUnknown = schema.Unknown
	sum.BySchemaStatus = schema.ByStatus

	domainMeta := LoadTenantDomainListMeta(ids)
	for _, m := range domainMeta {
		switch m.Status {
		case TenantDomainStatusActiveView:
			sum.DomainActive++
		case TenantDomainStatusPendingView:
			sum.DomainPending++
		case TenantDomainStatusVerifyFailed:
			sum.DomainVerifyFailed++
		case TenantDomainStatusDisabledView:
			sum.DomainDisabled++
		default:
			sum.DomainUnbound++
		}
	}

	deleted, _ := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).WithTrashed().Where("deleted_at IS NOT NULL").Count()
	sum.Deleted = deleted
	failedPurge, _ := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).WithTrashed().
		Where("deleted_at IS NOT NULL").
		Where("last_op", models.TenantOpPurge).
		Where("last_op_status", models.TenantOpStatusFailed).
		Count()
	sum.FailedPurge = failedPurge
	return sum, nil
}

// ListForBatchMigrate returns tenants matching ids and/or provision_status (max limit).
func (s *TenantAdminService) ListForBatchMigrate(ids []uint, provisionStatus string, limit int) ([]models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	query := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{})
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	if strings.TrimSpace(provisionStatus) != "" {
		query = query.Where("provision_status", strings.TrimSpace(provisionStatus))
	}
	var list []models.Tenant
	if err := query.Order("id asc").Limit(limit).Find(&list); err != nil {
		return nil, err
	}
	return list, nil
}

// TenantToJSON hides password; has_password indicates a stored credential exists.
func TenantToJSON(t *models.Tenant) map[string]any {
	return TenantToJSONWithDomain(t, TenantDomainListMeta{}.empty())
}

// TenantToJSONWithDomain includes vanity-domain list summary.
func TenantToJSONWithDomain(t *models.Tenant, domain TenantDomainListMeta) map[string]any {
	if t == nil {
		return nil
	}
	var deletedAt any
	if t.DeletedAt.Valid {
		deletedAt = t.DeletedAt.Time
	}
	expected := ExpectedSchemaMigrationCount()
	if domain.ActiveHosts == nil {
		domain.ActiveHosts = []string{}
	}
	if domain.Status == "" {
		domain.Status = TenantDomainStatusUnbound
	}
	out := map[string]any{
		"id":                       t.ID,
		"code":                     t.Code,
		"name":                     t.Name,
		"status":                   t.Status,
		"provision_status":         t.ProvisionStatus,
		"driver":                   t.Driver,
		"isolation":                t.Isolation,
		"host":                     t.Host,
		"port":                     t.Port,
		"database":                 t.Database,
		"schema":                   t.Schema,
		"username":                 t.Username,
		"has_password":             TenantHasPassword(t.Password),
		"connection_name":          t.ConnectionName,
		"last_migrate_error":       t.LastMigrateError,
		"migrated_at":              t.MigratedAt,
		"schema_migration_count":   t.SchemaMigrationCount,
		"expected_migration_count": expected,
		"schema_status":            ResolveTenantSchemaStatus(t, expected),
		"maintenance":              t.Maintenance,
		"maintenance_message":      t.MaintenanceMessage,
		"last_op":                  t.LastOp,
		"last_op_status":           t.LastOpStatus,
		"last_op_message":          t.LastOpMessage,
		"last_op_at":               t.LastOpAt,
		"last_backup_path":         t.LastBackupPath,
		"backup_dir":               TenantBackupDir(t.Code),
		"storage_limit_bytes":      t.StorageLimitBytes,
		"health_status":            strings.TrimSpace(t.HealthStatus),
		"health_checked_at":        t.HealthCheckedAt,
		"health_issues":            parseHealthIssues(t.HealthIssues),
		"last_ping_ok":             t.LastPingOK,
		"last_ping_ms":             t.LastPingMs,
		"domain_status":            domain.Status,
		"domain_primary_host":      domain.PrimaryHost,
		"domain_active_hosts":      domain.ActiveHosts,
		"domain_count":             domain.DomainCount,
		"deleted_at":               deletedAt,
		"trashed":                  TenantIsTrashed(t),
		"created_at":               t.CreatedAt,
		"updated_at":               t.UpdatedAt,
	}
	for k, v := range tenantstorage.PublicJSON(t) {
		out[k] = v
	}
	return out
}

// TenantsToJSONList enriches rows with domain meta in one query.
func TenantsToJSONList(list []models.Tenant) []map[string]any {
	ids := make([]uint, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
	}
	meta := LoadTenantDomainListMeta(ids)
	rows := make([]map[string]any, 0, len(list))
	for i := range list {
		rows = append(rows, TenantToJSONWithDomain(&list[i], meta[list[i].ID]))
	}
	return rows
}
