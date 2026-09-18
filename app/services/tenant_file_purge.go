package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenantstorage"
	"goravel/app/utils"
)

var tenantCodePathSafe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

// TenantObjectStoragePrefix returns the object-storage directory for a tenant code (no trailing slash).
func TenantObjectStoragePrefix(code string) (string, error) {
	code = strings.TrimSpace(code)
	if !tenantCodePathSafe.MatchString(code) {
		return "", apperrors.ErrInvalidArgument.WithMessage("invalid tenant code for storage purge")
	}
	return "tenants/" + code, nil
}

func purgePrefixOnDisk(disk, prefix string) error {
	storage, err := utils.StorageDisk(disk)
	if err != nil {
		return err
	}
	if err := storage.DeleteDirectory(prefix); err != nil {
		if storage.Exists(prefix) {
			return apperrors.ErrDeleteFileFailed.WithError(fmt.Errorf("purge object storage %s on %s: %w", prefix, disk, err))
		}
	}
	return nil
}

// PurgeTenantObjectStorage deletes tenants/{code}/ on the platform default disk and,
// when BYOB credentials exist, on the tenant custom disk as well (covers mode switches).
func PurgeTenantObjectStorage(code string) error {
	prefix, err := TenantObjectStoragePrefix(code)
	if err != nil {
		return err
	}
	disk := facades.Config().GetString("filesystems.default", "local")
	if err := purgePrefixOnDisk(disk, prefix); err != nil {
		return err
	}

	var tenant models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).WithTrashed().Where("code", code).First(&tenant); err == nil && tenant.ID > 0 {
		if tenantstorage.CredentialsReady(&tenant) {
			byob := tenantstorage.DiskName(tenant.ID)
			if err := purgePrefixOnDisk(byob, prefix); err != nil {
				return err
			}
		}
	}
	return nil
}

// PurgeTenantLocalBackups removes storage/backups/tenants/{code}/ SQL dumps on the app host.
func PurgeTenantLocalBackups(code string) error {
	if _, err := TenantObjectStoragePrefix(code); err != nil {
		return err
	}
	code = strings.TrimSpace(code)
	dir := filepath.Join("storage", "backups", "tenants", code)
	if err := os.RemoveAll(dir); err != nil {
		return apperrors.ErrDeleteFileFailed.WithError(fmt.Errorf("purge local backups %s: %w", dir, err))
	}
	return nil
}

// PurgeTenantFiles clears object-storage prefix and local SQL backup dumps for the tenant code.
func PurgeTenantFiles(code string) error {
	if err := PurgeTenantObjectStorage(code); err != nil {
		return err
	}
	return PurgeTenantLocalBackups(code)
}
