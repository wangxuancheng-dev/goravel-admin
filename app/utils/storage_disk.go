package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/filesystem"
	"github.com/goravel/framework/facades"
	fwfilesystem "github.com/goravel/framework/filesystem"

	apperrors "goravel/app/errors"
	"goravel/app/tenancy"
	"goravel/app/tenantstorage"
)

// ResolveFileDisk returns the active filesystem disk name for new writes.
// Multi-tenant: optional per-tenant BYOB (storage_mode=custom); otherwise platform default.
// Single-tenant: admin configs.file_disk (then storage_disk / export_disk / filesystems.default).
func ResolveFileDisk(ctx context.Context) string {
	if tenancy.Enabled() {
		if byob := tenantstorage.ResolveWriteDisk(ctx); byob != "" {
			return byob
		}
		disk := strings.TrimSpace(facades.Config().GetString("filesystems.default", ""))
		if disk != "" {
			return disk
		}
		return "local"
	}
	disk := GetConfigValue(ctx, "storage", "file_disk", "")
	if disk == "" {
		disk = GetConfigValue(ctx, "storage", "storage_disk", "")
	}
	if disk == "" {
		disk = GetConfigValue(ctx, "storage", "export_disk", "")
	}
	if disk == "" {
		disk = strings.TrimSpace(facades.Config().GetString("filesystems.default", ""))
	}
	if disk == "" {
		return "local"
	}
	return disk
}

// ValidateFilesystemDisk 检查云存储磁盘的必填配置是否已写入 .env / config。
// 未配置时返回业务错误，避免 Storage().Disk() 内部 panic。
func ValidateFilesystemDisk(disk string) error {
	if tenantstorage.IsByobDisk(disk) {
		id, ok := tenantstorage.ParseTenantIDFromDisk(disk)
		if !ok {
			return storageDiskNotConfigured(disk)
		}
		t, err := tenantstorage.LoadTenant(id, true)
		if err != nil {
			return err
		}
		if !tenantstorage.CredentialsReady(t) {
			return apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": disk})
		}
		return nil
	}
	switch disk {
	case "", "local", "public":
		return nil
	case "s3":
		return requireDiskFields(disk, "key", "secret", "region", "bucket", "url")
	case "oss":
		return requireDiskFields(disk, "key", "secret", "bucket", "url", "endpoint")
	case "cos":
		return requireDiskFields(disk, "key", "secret", "url")
	case "minio":
		return requireDiskFields(disk, "key", "secret", "bucket", "url", "endpoint")
	default:
		return storageDiskNotConfigured(disk)
	}
}

func requireDiskFields(disk string, fields ...string) error {
	cfg := facades.Config()
	for _, field := range fields {
		if cfg.GetString(fmt.Sprintf("filesystems.disks.%s.%s", disk, field)) == "" {
			return storageDiskNotConfigured(disk)
		}
	}
	return nil
}

func storageDiskNotConfigured(disk string) *apperrors.BusinessError {
	return apperrors.NewBusinessError(
		apperrors.ErrStorageDiskNotConfigured.Code,
		apperrors.ErrStorageDiskNotConfigured.Message,
	).WithParams(map[string]any{"disk": disk})
}

// StorageDisk 安全获取磁盘驱动：先校验配置，再用 NewDriver，避免 Disk() panic。
// tenant_byob_{id} disks are registered from platform tenant credentials.
func StorageDisk(disk string) (filesystem.Driver, error) {
	if disk == "" {
		disk = facades.Config().GetString("filesystems.default", "local")
	}
	if tenantstorage.IsByobDisk(disk) {
		return tenantstorage.OpenDisk(disk)
	}
	if err := ValidateFilesystemDisk(disk); err != nil {
		return nil, err
	}
	driver, err := fwfilesystem.NewDriver(facades.Config(), disk)
	if err != nil {
		return nil, storageDiskNotConfigured(disk).WithError(err)
	}
	return driver, nil
}

// IsLocalFilesystemDisk reports disks that support local chunk merge paths.
func IsLocalFilesystemDisk(disk string) bool {
	disk = strings.TrimSpace(disk)
	return disk == "local" || disk == "public"
}
