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
	Code   string
	Name   string
	Status string
}

func BuildTenantAdminFiltersFromHTTP(ctx http.Context) TenantAdminFilters {
	return TenantAdminFilters{
		Code:   strings.TrimSpace(ctx.Request().Query("code", "")),
		Name:   strings.TrimSpace(ctx.Request().Query("name", "")),
		Status: strings.TrimSpace(ctx.Request().Query("status", "")),
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
	Migrate   bool
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

	driverName := strings.TrimSpace(input.Driver)
	if driverName == "" {
		driverName = facades.Config().GetString("database.default", "mysql")
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

	tenant := models.Tenant{
		Code:           code,
		Name:           name,
		Status:         models.TenantStatusActive,
		Driver:         driverName,
		Isolation:      isolation,
		Database:       database,
		Schema:         schemaName,
		ConnectionName: fmt.Sprintf("tenant_pending_%s", code),
	}
	if err := appfacades.PlatformOrmQuery(nil).Create(&tenant); err != nil {
		return nil, err
	}
	tenant.ConnectionName = TenantConnectionName(tenant.ID)
	if _, err := appfacades.PlatformOrmQuery(nil).Model(&tenant).Update(map[string]any{
		"connection_name": tenant.ConnectionName,
	}); err != nil {
		return nil, err
	}
	if err := s.conn.CreateStorage(&tenant); err != nil {
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

// TenantToJSON hides password.
func TenantToJSON(t *models.Tenant) map[string]any {
	if t == nil {
		return nil
	}
	return map[string]any{
		"id":              t.ID,
		"code":            t.Code,
		"name":            t.Name,
		"status":          t.Status,
		"driver":          t.Driver,
		"isolation":       t.Isolation,
		"host":            t.Host,
		"port":            t.Port,
		"database":        t.Database,
		"schema":          t.Schema,
		"username":        t.Username,
		"connection_name": t.ConnectionName,
		"created_at":      t.CreatedAt,
		"updated_at":      t.UpdatedAt,
	}
}
