package services

import (
	"strings"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/tenantstorage"
)

func applyTenantStorageOnCreate(tenant *models.Tenant, input TenantCreateInput) error {
	mode := tenantstorage.NormalizeMode(input.StorageMode)
	tenant.StorageMode = mode
	if mode != tenantstorage.ModeCustom {
		return nil
	}
	driver := tenantstorage.NormalizeDriver(input.StorageDriver)
	if driver == "" {
		return apperrors.ErrTenantStorageInvalidDriver
	}
	tenant.StorageDriver = driver
	tenant.StorageKey = strings.TrimSpace(input.StorageKey)
	tenant.StorageRegion = strings.TrimSpace(input.StorageRegion)
	tenant.StorageBucket = strings.TrimSpace(input.StorageBucket)
	tenant.StorageURL = strings.TrimSpace(input.StorageURL)
	tenant.StorageEndpoint = strings.TrimSpace(input.StorageEndpoint)
	if input.StorageUsePathStyle != nil {
		tenant.StorageUsePathStyle = *input.StorageUsePathStyle
	}
	if input.StorageSSL != nil {
		tenant.StorageSSL = *input.StorageSSL
	} else {
		tenant.StorageSSL = true
	}
	if strings.TrimSpace(input.StorageSecret) == "" {
		return apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": "custom"})
	}
	sealed, err := tenantstorage.SealSecret(input.StorageSecret)
	if err != nil {
		return apperrors.ErrPasswordEncryptFailed.WithError(err)
	}
	tenant.StorageSecret = sealed
	if !tenantstorage.CredentialsReady(tenant) {
		return apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": "custom"})
	}
	return nil
}

func mergeTenantStorageUpdates(tenant *models.Tenant, input TenantUpdateInput, updates map[string]any) error {
	storageTouched := input.StorageMode != nil ||
		input.StorageDriver != nil ||
		input.StorageKey != nil ||
		input.StorageSecret != nil ||
		input.StorageRegion != nil ||
		input.StorageBucket != nil ||
		input.StorageURL != nil ||
		input.StorageEndpoint != nil ||
		input.StorageUsePathStyle != nil ||
		input.StorageSSL != nil ||
		(input.ClearStorageSecret != nil && *input.ClearStorageSecret)
	if !storageTouched {
		return nil
	}

	mode := tenantstorage.NormalizeMode(tenant.StorageMode)
	if input.StorageMode != nil {
		mode = tenantstorage.NormalizeMode(*input.StorageMode)
		updates["storage_mode"] = mode
		tenant.StorageMode = mode
	}

	if input.StorageDriver != nil {
		driver := tenantstorage.NormalizeDriver(*input.StorageDriver)
		if strings.TrimSpace(*input.StorageDriver) != "" && driver == "" {
			return apperrors.ErrTenantStorageInvalidDriver
		}
		updates["storage_driver"] = driver
		tenant.StorageDriver = driver
	}
	if input.StorageKey != nil {
		key := strings.TrimSpace(*input.StorageKey)
		updates["storage_key"] = key
		tenant.StorageKey = key
	}
	if input.StorageRegion != nil {
		region := strings.TrimSpace(*input.StorageRegion)
		updates["storage_region"] = region
		tenant.StorageRegion = region
	}
	if input.StorageBucket != nil {
		bucket := strings.TrimSpace(*input.StorageBucket)
		updates["storage_bucket"] = bucket
		tenant.StorageBucket = bucket
	}
	if input.StorageURL != nil {
		u := strings.TrimSpace(*input.StorageURL)
		updates["storage_url"] = u
		tenant.StorageURL = u
	}
	if input.StorageEndpoint != nil {
		ep := strings.TrimSpace(*input.StorageEndpoint)
		updates["storage_endpoint"] = ep
		tenant.StorageEndpoint = ep
	}
	if input.StorageUsePathStyle != nil {
		updates["storage_use_path_style"] = *input.StorageUsePathStyle
		tenant.StorageUsePathStyle = *input.StorageUsePathStyle
	}
	if input.StorageSSL != nil {
		updates["storage_ssl"] = *input.StorageSSL
		tenant.StorageSSL = *input.StorageSSL
	}
	if input.ClearStorageSecret != nil && *input.ClearStorageSecret {
		updates["storage_secret"] = ""
		tenant.StorageSecret = ""
	} else if input.StorageSecret != nil && strings.TrimSpace(*input.StorageSecret) != "" {
		sealed, err := tenantstorage.SealSecret(*input.StorageSecret)
		if err != nil {
			return apperrors.ErrPasswordEncryptFailed.WithError(err)
		}
		updates["storage_secret"] = sealed
		tenant.StorageSecret = sealed
	}

	// Switching to custom requires complete credentials (keep existing secret if not rotated).
	if mode == tenantstorage.ModeCustom && !tenantstorage.CredentialsReady(tenant) {
		return apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": "custom"})
	}
	return nil
}
