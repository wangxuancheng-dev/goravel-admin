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
)

// ResolveFileDisk returns the active filesystem disk name.
// When TENANCY_DRIVER=database, tenants share the platform default (FILESYSTEM_DISK / filesystems.default);
// per-tenant configs.file_disk is ignored so purge/quota stay consistent.
func ResolveFileDisk(ctx context.Context) string {
	if tenancy.Enabled() {
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
func StorageDisk(disk string) (filesystem.Driver, error) {
	if disk == "" {
		disk = facades.Config().GetString("filesystems.default", "local")
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
