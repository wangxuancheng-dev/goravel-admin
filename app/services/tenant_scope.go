package services

import (
	"context"
	"fmt"
	"strings"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// TenantWork runs under the tenant DB (or single default DB when tenancy is off / tenant is nil).
// ctx carries tenant routing keys when tenancy is on.
type TenantWork func(tenant *models.Tenant, ctx context.Context) error

// ListActiveTenants returns platform tenants with status=active.
func (s *TenantConnectionService) ListActiveTenants() ([]models.Tenant, error) {
	var tenants []models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).
		Where("status", models.TenantStatusActive).
		Order("id asc").
		Get(&tenants); err != nil {
		return nil, err
	}
	return tenants, nil
}

// RunTenantScope runs work once on the default DB when tenancy is off.
// When tenancy is on: --tenant hint runs one active tenant; empty hint iterates all active tenants.
func (s *TenantConnectionService) RunTenantScope(hint string, work TenantWork) error {
	if work == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant work is nil")
	}
	if !tenancy.Enabled() {
		return work(nil, context.Background())
	}

	hint = strings.TrimSpace(hint)
	if hint != "" {
		tenant, err := s.FindTenantByIDOrCode(hint)
		if err != nil {
			return apperrors.ErrTenantNotFound.WithError(err)
		}
		if tenant.Status != models.TenantStatusActive {
			return apperrors.ErrTenantDisabled
		}
		return s.WithTenantConnection(tenant, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tenant.ID, tenant.ConnectionName, tenant.Code)
			return work(tenant, bound)
		})
	}

	tenants, err := s.ListActiveTenants()
	if err != nil {
		return err
	}
	if len(tenants) == 0 {
		return nil
	}

	var (
		failed   int
		firstErr error
	)
	for i := range tenants {
		tenant := &tenants[i]
		err := s.WithTenantConnection(tenant, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tenant.ID, tenant.ConnectionName, tenant.Code)
			return work(tenant, bound)
		})
		if err != nil {
			failed++
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d tenant op(s) failed: %w", failed, firstErr)
	}
	return nil
}
