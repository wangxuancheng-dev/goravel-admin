package services

import (
	"context"
	"database/sql"

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
}

// LoadPlatformTenantStorageLimit loads storage quota from the landlord tenants row.
func LoadPlatformTenantStorageLimit(tenantID uint) (int64, error) {
	var t models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Where("id", tenantID).First(&t); err != nil {
		return 0, apperrors.ErrTenantNotFound
	}
	return t.StorageLimitBytes, nil
}

// SumTenantAttachmentBytes sums attachments.size on the current (tenant) ORM connection.
// Empty table / no matching rows: SUM returns NULL — treat as 0 (do not scan into bare int64).
func SumTenantAttachmentBytes(ctx context.Context) (int64, error) {
	var total sql.NullInt64
	err := appfacades.OrmQuery(ctx).Model(&models.Attachment{}).Where("status", 1).Sum("size", &total)
	if err != nil {
		return 0, err
	}
	if !total.Valid || total.Int64 < 0 {
		return 0, nil
	}
	return total.Int64, nil
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
	storageLimit, err := LoadPlatformTenantStorageLimit(id)
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

// BuildTenantQuotaSnapshot fills used/limit for platform UI.
func BuildTenantQuotaSnapshot(tenant *models.Tenant) TenantQuotaSnapshot {
	out := TenantQuotaSnapshot{
		StorageLimitBytes: tenant.StorageLimitBytes,
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
