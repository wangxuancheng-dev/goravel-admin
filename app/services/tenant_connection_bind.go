package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

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
