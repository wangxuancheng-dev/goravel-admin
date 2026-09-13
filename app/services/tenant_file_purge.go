package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
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

// PurgeTenantObjectStorage deletes tenants/{code}/ on the default filesystem disk (best-effort recursive).
func PurgeTenantObjectStorage(code string) error {
	prefix, err := TenantObjectStoragePrefix(code)
	if err != nil {
		return err
	}
	disk := facades.Config().GetString("filesystems.default", "local")
	storage, err := utils.StorageDisk(disk)
	if err != nil {
		return err
	}
	if err := storage.DeleteDirectory(prefix); err != nil {
		// Some drivers error when the directory is already missing; treat missing as success.
		if storage.Exists(prefix) {
			return apperrors.ErrDeleteFileFailed.WithError(fmt.Errorf("purge object storage %s: %w", prefix, err))
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
