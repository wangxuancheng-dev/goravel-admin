package services

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// TenantQuotaSnapshot is used by platform overview / JSON.
type TenantQuotaSnapshot struct {
	StorageLimitBytes int64 `json:"storage_limit_bytes"`
	StorageUsedBytes  int64 `json:"storage_used_bytes"`
	TrafficLimitBytes int64 `json:"traffic_limit_bytes"`
	TrafficUsedBytes  int64 `json:"traffic_used_bytes"`
	TrafficMonth      string `json:"traffic_month"`
}

func trafficMonthKey() string {
	return time.Now().UTC().Format("200601")
}

func tenantTrafficCacheKey(tenantID uint) string {
	return fmt.Sprintf("t%d:traffic:%s", tenantID, trafficMonthKey())
}

// LoadPlatformTenantQuota loads quota limits from the landlord tenants row.
func LoadPlatformTenantQuota(tenantID uint) (storageLimit, trafficLimit int64, err error) {
	var t models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Where("id", tenantID).First(&t); err != nil {
		return 0, 0, apperrors.ErrTenantNotFound
	}
	return t.StorageLimitBytes, t.TrafficLimitBytes, nil
}

// SumTenantAttachmentBytes sums attachments.size on the current (tenant) ORM connection.
func SumTenantAttachmentBytes(ctx context.Context) (int64, error) {
	var total int64
	err := appfacades.OrmQuery(ctx).Model(&models.Attachment{}).Where("status", 1).Sum("size", &total)
	if err != nil {
		return 0, err
	}
	if total < 0 {
		total = 0
	}
	return total, nil
}

// GetTenantTrafficUsed returns this UTC month's counted transfer bytes from cache (0 if unset).
func GetTenantTrafficUsed(tenantID uint) int64 {
	if tenantID == 0 {
		return 0
	}
	var used int64
	_ = facades.Cache().Get(tenantTrafficCacheKey(tenantID), &used)
	if used < 0 {
		return 0
	}
	return used
}

// AddTenantTraffic increments monthly transfer usage (upload/download counted by the app).
func AddTenantTraffic(tenantID uint, bytes int64) {
	if tenantID == 0 || bytes <= 0 || !tenancy.Enabled() {
		return
	}
	key := tenantTrafficCacheKey(tenantID)
	used := GetTenantTrafficUsed(tenantID) + bytes
	// Keep until ~40 days so month rollover does not need an explicit reset job.
	_ = facades.Cache().Put(key, used, 40*24*time.Hour)
}

// EnsureTenantStorageQuota fails when used+additional would exceed storage_limit_bytes (0=unlimited).
func EnsureTenantStorageQuota(ctx context.Context, additionalBytes int64) error {
	if !tenancy.Enabled() || additionalBytes < 0 {
		return nil
	}
	id, ok := tenancyctx.IDFrom(ctx)
	if !ok || id == 0 {
		return nil
	}
	storageLimit, _, err := LoadPlatformTenantQuota(id)
	if err != nil || storageLimit <= 0 {
		return nil
	}
	used, err := SumTenantAttachmentBytes(ctx)
	if err != nil {
		return err
	}
	if used+additionalBytes > storageLimit {
		return apperrors.ErrTenantStorageQuotaExceeded.WithParams(map[string]any{
			"used":  used,
			"limit": storageLimit,
		})
	}
	return nil
}

// EnsureTenantTrafficQuota fails when used+additional would exceed traffic_limit_bytes (0=unlimited).
func EnsureTenantTrafficQuota(ctx context.Context, additionalBytes int64) error {
	if !tenancy.Enabled() || additionalBytes <= 0 {
		return nil
	}
	id, ok := tenancyctx.IDFrom(ctx)
	if !ok || id == 0 {
		return nil
	}
	_, trafficLimit, err := LoadPlatformTenantQuota(id)
	if err != nil || trafficLimit <= 0 {
		return nil
	}
	used := GetTenantTrafficUsed(id)
	if used+additionalBytes > trafficLimit {
		return apperrors.ErrTenantTrafficQuotaExceeded.WithParams(map[string]any{
			"used":  used,
			"limit": trafficLimit,
		})
	}
	return nil
}

// RecordTenantTraffic checks quota then adds bytes (upload or app-proxied download).
func RecordTenantTraffic(ctx context.Context, bytes int64) error {
	if err := EnsureTenantTrafficQuota(ctx, bytes); err != nil {
		return err
	}
	if id, ok := tenancyctx.IDFrom(ctx); ok {
		AddTenantTraffic(id, bytes)
	}
	return nil
}

// BuildTenantQuotaSnapshot fills used/limit for platform UI.
func BuildTenantQuotaSnapshot(tenant *models.Tenant) TenantQuotaSnapshot {
	out := TenantQuotaSnapshot{
		StorageLimitBytes: tenant.StorageLimitBytes,
		TrafficLimitBytes: tenant.TrafficLimitBytes,
		TrafficMonth:      trafficMonthKey(),
		TrafficUsedBytes:  GetTenantTrafficUsed(tenant.ID),
	}
	_ = NewTenantConnectionService().WithTenantConnection(tenant, func() error {
		used, err := SumTenantAttachmentBytes(context.Background())
		if err != nil {
			return err
		}
		out.StorageUsedBytes = used
		return nil
	})
	return out
}
