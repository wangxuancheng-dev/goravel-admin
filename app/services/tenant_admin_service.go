package services

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
)

// TenantAdminFilters platform tenants list filters.
type TenantAdminFilters struct {
	Code            string
	Name            string
	Status          string
	ProvisionStatus string
}

func BuildTenantAdminFiltersFromHTTP(ctx http.Context) TenantAdminFilters {
	return TenantAdminFilters{
		Code:            strings.TrimSpace(ctx.Request().Query("code", "")),
		Name:            strings.TrimSpace(ctx.Request().Query("name", "")),
		Status:          strings.TrimSpace(ctx.Request().Query("status", "")),
		ProvisionStatus: strings.TrimSpace(ctx.Request().Query("provision_status", "")),
	}
}

// TenantCreateInput create tenant from API / CLI shared fields.
type TenantCreateInput struct {
	Code      string
	Name      string
	Driver    string
	Isolation string
	Database  string
	Schema    string
	Host      string
	Port      int
	Username  string
	Password  string
	Migrate   bool
	SkipCreate bool // 远程库已存在时跳过 CREATE DATABASE/SCHEMA
}

// TenantUpdateInput updates connection metadata for an existing tenant.
type TenantUpdateInput struct {
	Name     *string
	Host     *string
	Port     *int
	Username *string
	Password *string
	Database *string
	Schema   *string
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
	if err := appfacades.PlatformOrmQuery(nil).Where("id", id).First(&tenant); err != nil {
		return nil, apperrors.ErrTenantNotFound.WithError(err)
	}
	return &tenant, nil
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
	if err := appfacades.PlatformOrmQuery(nil).Where("code", code).First(&existing); err == nil && existing.ID > 0 {
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
	Total           int64            `json:"total"`
	ByProvision     map[string]int64 `json:"by_provision"`
	ByLastOpStatus  map[string]int64 `json:"by_last_op_status"`
	FailedProvision int64            `json:"failed_provision"`
	Busy            int64            `json:"busy"`
}

func (s *TenantAdminService) OpsSummary() (*TenantOpsSummary, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var tenants []models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).
		Select("id", "provision_status", "last_op_status").
		Get(&tenants); err != nil {
		return nil, err
	}
	sum := &TenantOpsSummary{
		Total:          int64(len(tenants)),
		ByProvision:    map[string]int64{},
		ByLastOpStatus: map[string]int64{},
	}
	for i := range tenants {
		t := &tenants[i]
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
	}
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
	if t == nil {
		return nil
	}
	return map[string]any{
		"id":                 t.ID,
		"code":               t.Code,
		"name":               t.Name,
		"status":             t.Status,
		"provision_status":   t.ProvisionStatus,
		"driver":             t.Driver,
		"isolation":          t.Isolation,
		"host":               t.Host,
		"port":               t.Port,
		"database":           t.Database,
		"schema":             t.Schema,
		"username":           t.Username,
		"has_password":       TenantHasPassword(t.Password),
		"connection_name":    t.ConnectionName,
		"last_migrate_error": t.LastMigrateError,
		"migrated_at":        t.MigratedAt,
		"last_op":            t.LastOp,
		"last_op_status":     t.LastOpStatus,
		"last_op_message":    t.LastOpMessage,
		"last_op_at":         t.LastOpAt,
		"last_backup_path":   t.LastBackupPath,
		"created_at":         t.CreatedAt,
		"updated_at":         t.UpdatedAt,
	}
}
