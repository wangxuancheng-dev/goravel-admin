package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// TenantWork runs under the tenant DB (or single default DB when tenancy is off / tenant is nil).
// ctx carries tenant routing keys when tenancy is on.
type TenantWork func(tenant *models.Tenant, ctx context.Context) error

// TenantScopeOptions controls fleet iteration for ops / scheduled commands.
// Small fleets (at or below TENANCY_SCOPE_AUTO_BATCH_AT) still run all ready tenants in one pass.
// Larger fleets auto-page; set TENANCY_SCOPE_BATCH to force a fixed page size.
type TenantScopeOptions struct {
	Hint string
	// Limit max tenants this call; 0 = resolve from config / auto. Use -1 for unlimited.
	Limit int
	// AfterID exclusive cursor (id > AfterID). Ignored when Limit resolves to unlimited.
	AfterID uint
	// Rotate persists the next AfterID in cache for scheduled sweeps.
	Rotate bool
	// RotateKey distinguishes cursors (default "default").
	RotateKey string
}

const tenantScopeCursorCacheTTL = 48 * time.Hour

// ListActiveTenants returns platform tenants with status=active (any provision_status).
// Prefer ListReadyActiveTenants for maintenance that assumes a migrated schema.
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

// ListReadyActiveTenants returns enabled tenants that are provision_status=ready.
func (s *TenantConnectionService) ListReadyActiveTenants() ([]models.Tenant, error) {
	tenants, err := s.ListActiveTenants()
	if err != nil {
		return nil, err
	}
	return filterReadyTenants(tenants), nil
}

// CountReadyActiveTenants counts enabled+ready tenants (cheap fleet-size probe).
func (s *TenantConnectionService) CountReadyActiveTenants() (int64, error) {
	q := readyActiveTenantQuery()
	if q == nil {
		return 0, nil
	}
	n, err := q.Count()
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ListReadyActiveTenantsPage returns ready+active tenants with id > afterID, ordered by id.
func (s *TenantConnectionService) ListReadyActiveTenantsPage(afterID uint, limit int) ([]models.Tenant, error) {
	if limit < 1 {
		return nil, nil
	}
	q := readyActiveTenantQuery()
	if q == nil {
		return nil, nil
	}
	if afterID > 0 {
		q = q.Where("id > ?", afterID)
	}
	var tenants []models.Tenant
	if err := q.Order("id asc").Limit(limit).Get(&tenants); err != nil {
		return nil, err
	}
	return filterReadyTenants(tenants), nil
}

func readyActiveTenantQuery() orm.Query {
	q := appfacades.PlatformOrmQuery(nil)
	if q == nil {
		return nil
	}
	// Legacy empty provision_status is treated as ready (see Tenant.IsProvisionReady).
	return q.Model(&models.Tenant{}).
		Where("status", models.TenantStatusActive).
		Where("(provision_status = ? OR provision_status = ? OR provision_status IS NULL)",
			models.TenantProvisionReady, "")
}

func filterReadyTenants(tenants []models.Tenant) []models.Tenant {
	out := make([]models.Tenant, 0, len(tenants))
	for i := range tenants {
		if tenants[i].IsProvisionReady() {
			out = append(out, tenants[i])
		}
	}
	return out
}

// ResolveScopeBatch picks page size for a fleet: explicit limit, TENANCY_SCOPE_BATCH,
// or auto-batch when fleetSize > TENANCY_SCOPE_AUTO_BATCH_AT. 0 = unlimited.
func ResolveScopeBatch(explicitLimit int, fleetSize int) int {
	if explicitLimit < 0 {
		return 0
	}
	if explicitLimit > 0 {
		return explicitLimit
	}
	batch := facades.Config().GetInt("tenancy.scope_batch", 0)
	if batch > 0 {
		return batch
	}
	autoAt := facades.Config().GetInt("tenancy.scope_auto_batch_at", 200)
	autoBatch := facades.Config().GetInt("tenancy.scope_auto_batch", 100)
	if autoAt > 0 && fleetSize > autoAt {
		if autoBatch < 1 {
			autoBatch = 100
		}
		return autoBatch
	}
	return 0
}

func tenantScopeCursorKey(rotateKey string) string {
	rotateKey = strings.TrimSpace(rotateKey)
	if rotateKey == "" {
		rotateKey = "default"
	}
	return "tenancy:scope_cursor:" + rotateKey
}

func loadScopeCursor(rotateKey string) uint {
	raw := facades.Cache().Get(tenantScopeCursorKey(rotateKey), uint(0))
	switch v := raw.(type) {
	case uint:
		return v
	case int:
		if v > 0 {
			return uint(v)
		}
	case int64:
		if v > 0 {
			return uint(v)
		}
	case float64:
		if v > 0 {
			return uint(v)
		}
	case string:
		n, _ := strconv.ParseUint(v, 10, 64)
		return uint(n)
	}
	return 0
}

func storeScopeCursor(rotateKey string, afterID uint) {
	_ = facades.Cache().Put(tenantScopeCursorKey(rotateKey), afterID, tenantScopeCursorCacheTTL)
}

// RunTenantScope runs work once on the default DB when tenancy is off.
// When tenancy is on: hint runs one active+ready tenant; empty hint iterates ready active tenants.
// (migrate-all / seed-all do not use this helper — they may target pending tenants.)
func (s *TenantConnectionService) RunTenantScope(hint string, work TenantWork) error {
	return s.RunTenantScopeWithOptions(TenantScopeOptions{Hint: hint}, work)
}

// RunTenantScopeWithOptions is RunTenantScope with paging / rotate cursor support.
func (s *TenantConnectionService) RunTenantScopeWithOptions(opts TenantScopeOptions, work TenantWork) error {
	if work == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant work is nil")
	}
	if !tenancy.Enabled() {
		return work(nil, context.Background())
	}

	hint := strings.TrimSpace(opts.Hint)
	if hint != "" {
		tenant, err := s.FindTenantByIDOrCode(hint)
		if err != nil {
			return apperrors.ErrTenantNotFound.WithError(err)
		}
		if tenant.Status != models.TenantStatusActive {
			return apperrors.ErrTenantDisabled
		}
		if !tenant.IsProvisionReady() {
			return apperrors.ErrTenantNotReady
		}
		return s.WithTenantConnection(tenant, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tenant.ID, tenant.ConnectionName, tenant.Code)
			return work(tenant, bound)
		})
	}

	fleetSize := 0
	if n, err := s.CountReadyActiveTenants(); err == nil {
		fleetSize = int(n)
	}
	limit := ResolveScopeBatch(opts.Limit, fleetSize)

	if limit <= 0 {
		tenants, err := s.ListReadyActiveTenants()
		if err != nil {
			return err
		}
		return s.runTenantWorkList(tenants, work)
	}

	afterID := opts.AfterID
	if opts.Rotate && afterID == 0 {
		afterID = loadScopeCursor(opts.RotateKey)
	}

	// Scheduled rotate: one page per invocation. Manual/daily: drain all pages.
	if !opts.Rotate {
		var (
			failed   int
			firstErr error
		)
		for {
			tenants, err := s.ListReadyActiveTenantsPage(afterID, limit)
			if err != nil {
				return err
			}
			if len(tenants) == 0 {
				break
			}
			if err := s.runTenantWorkList(tenants, work); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
			}
			if len(tenants) < limit {
				break
			}
			afterID = tenants[len(tenants)-1].ID
		}
		if failed > 0 {
			return fmt.Errorf("%d tenant page(s) had failures: %w", failed, firstErr)
		}
		return nil
	}

	tenants, err := s.ListReadyActiveTenantsPage(afterID, limit)
	if err != nil {
		return err
	}
	// Wrap once when cursor is past the end so remainder + new tenants are covered.
	if len(tenants) == 0 && afterID > 0 {
		tenants, err = s.ListReadyActiveTenantsPage(0, limit)
		if err != nil {
			return err
		}
	}

	runErr := s.runTenantWorkList(tenants, work)

	next := uint(0)
	if len(tenants) >= limit && limit > 0 {
		next = tenants[len(tenants)-1].ID
	}
	storeScopeCursor(opts.RotateKey, next)

	return runErr
}

func (s *TenantConnectionService) runTenantWorkList(tenants []models.Tenant, work TenantWork) error {
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
