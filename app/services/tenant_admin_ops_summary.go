package services

import (
	"strings"
	"time"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// TenantOpsSummary aggregates provision / op status counts for the platform console.
type TenantOpsSummary struct {
	Total                  int64            `json:"total"`
	ByProvision            map[string]int64 `json:"by_provision"`
	ByLastOpStatus         map[string]int64 `json:"by_last_op_status"`
	FailedProvision        int64            `json:"failed_provision"`
	FailedSeed             int64            `json:"failed_seed"`
	Busy                   int64            `json:"busy"`
	Deleted                int64            `json:"deleted"`
	FailedPurge            int64            `json:"failed_purge"`
	ExpectedMigrationCount int64            `json:"expected_migration_count"`
	SchemaAligned          int64            `json:"schema_aligned"`
	SchemaBehind           int64            `json:"schema_behind"`
	SchemaFailed           int64            `json:"schema_failed"`
	SchemaRunning          int64            `json:"schema_running"`
	SchemaUnknown          int64            `json:"schema_unknown"`
	BySchemaStatus         map[string]int64 `json:"by_schema_status"`
	Maintenance            int64            `json:"maintenance"`
	DomainUnbound          int64            `json:"domain_unbound"`
	DomainPending          int64            `json:"domain_pending"`
	DomainActive           int64            `json:"domain_active"`
	DomainVerifyFailed     int64            `json:"domain_verify_failed"`
	DomainDisabled         int64            `json:"domain_disabled"`
	HealthOK               int64            `json:"health_ok"`
	HealthWarn             int64            `json:"health_warn"`
	HealthFail             int64            `json:"health_fail"`
	HealthUnknown          int64            `json:"health_unknown"`
	// Subscription / plan ops (active tenants only; soft-deleted excluded).
	ByPlanCode     map[string]int64 `json:"by_plan_code"`
	NoSubscription int64            `json:"no_subscription"`
	Expiring7d     int64            `json:"expiring_7d"`
	Expiring30d    int64            `json:"expiring_30d"`
}

func (s *TenantAdminService) OpsSummary() (*TenantOpsSummary, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var tenants []models.Tenant
	if err := appfacades.PlatformQuery().Model(&models.Tenant{}).
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
		if strings.TrimSpace(t.LastOp) == models.TenantOpSeed && os == models.TenantOpStatusFailed {
			sum.FailedSeed++
		}
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

	deleted, _ := appfacades.PlatformQuery().Model(&models.Tenant{}).WithTrashed().Where("deleted_at IS NOT NULL").Count()
	sum.Deleted = deleted
	failedPurge, _ := appfacades.PlatformQuery().Model(&models.Tenant{}).WithTrashed().
		Where("deleted_at IS NOT NULL").
		Where("last_op", models.TenantOpPurge).
		Where("last_op_status", models.TenantOpStatusFailed).
		Count()
	sum.FailedPurge = failedPurge

	picks, planCodes, err := loadActiveSubscriptionPicks(ids)
	if err != nil {
		return nil, err
	}
	subStats := BuildTenantSubscriptionStats(ids, picks, planCodes, time.Now())
	sum.ByPlanCode = subStats.ByPlanCode
	sum.NoSubscription = subStats.NoSubscription
	sum.Expiring7d = subStats.Expiring7d
	sum.Expiring30d = subStats.Expiring30d
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
	query := appfacades.PlatformQuery().Model(&models.Tenant{})
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

