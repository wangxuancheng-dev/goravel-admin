package services

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	contractsfilesystem "github.com/goravel/framework/contracts/filesystem"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/filesystem"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/utils"
)

// Where tenant backups are kept.
//
// By default in storage/backups/tenants/{code}/ on the host that ran the backup. With
// TENANT_BACKUP_DISK set to a disk of config/filesystems.go (s3, oss, cos, minio, ...)
// each finished dump is uploaded to tenant-backups/{code}/ on that disk and removed
// locally, so every app instance lists, downloads and restores the same files and
// they outlive the host. The framework disk drivers hold a whole file in memory while
// uploading or downloading it; for very large tenant databases prefer a shared volume
// mounted at storage/backups.

// TenantBackupDisk returns the filesystem disk tenant backups are kept on, or "" to
// keep them on the local host.
func TenantBackupDisk() string {
	return strings.TrimSpace(facades.Config().GetString("tenancy.backup_disk", ""))
}

// TenantBackupLocation describes where a tenant's backups are kept, for display.
func TenantBackupLocation(code string) string {
	if disk := TenantBackupDisk(); disk != "" {
		return disk + ":" + tenantBackupDiskDir(code)
	}
	return TenantBackupDir(code)
}

func tenantBackupDiskDir(code string) string {
	return "tenant-backups/" + strings.TrimSpace(code)
}

// tenantBackupStorage opens the backup disk. Callers check TenantBackupDisk() first;
// an empty name must not fall through to StorageDisk's default disk.
func tenantBackupStorage() (contractsfilesystem.Driver, error) {
	disk := TenantBackupDisk()
	if disk == "" {
		return nil, apperrors.ErrInvalidArgument.WithMessage("no tenant backup disk configured (TENANT_BACKUP_DISK)")
	}
	return utils.StorageDisk(disk)
}

// validTenantBackupName accepts a bare "<stamp>.sql" file name.
func validTenantBackupName(name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." || strings.Contains(name, "..") ||
		!strings.HasSuffix(strings.ToLower(name), ".sql") {
		return "", apperrors.ErrInvalidArgument.WithMessage("invalid backup name")
	}
	return name, nil
}

// storeTenantBackup moves a finished local dump to its final place and returns the
// path recorded as the tenant's last backup.
func storeTenantBackup(code, localFile string) (string, error) {
	if TenantBackupDisk() == "" {
		return localFile, nil
	}
	storage, err := tenantBackupStorage()
	if err != nil {
		return "", err
	}
	file, err := filesystem.NewFile(localFile)
	if err != nil {
		return "", err
	}
	dir, name := tenantBackupDiskDir(code), filepath.Base(localFile)
	if _, err := storage.PutFileAs(dir, file, name); err != nil {
		return "", apperrors.ErrSaveFileFailed.WithError(err)
	}
	_ = os.Remove(localFile)
	// The key as object storage spells it (a local driver returns OS separators).
	return dir + "/" + name, nil
}

// listDiskTenantBackups lists the .sql dumps of a tenant on the backup disk.
func listDiskTenantBackups(code string) ([]TenantBackupFile, error) {
	storage, err := tenantBackupStorage()
	if err != nil {
		return nil, err
	}
	dir := tenantBackupDiskDir(code)
	if !storage.Exists(dir) {
		return []TenantBackupFile{}, nil
	}
	entries, err := storage.Files(dir)
	if err != nil {
		return nil, err
	}
	files := make([]TenantBackupFile, 0, len(entries))
	for _, entry := range entries {
		name := path.Base(entry) // drivers return bare names or full keys
		if !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}
		key := dir + "/" + name
		file := TenantBackupFile{Name: name, Path: key}
		if size, err := storage.Size(key); err == nil {
			file.Size = size
		}
		if mod, err := storage.LastModified(key); err == nil {
			file.ModTime = mod.Format("2006-01-02 15:04:05")
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name > files[j].Name })
	return files, nil
}

// pruneDiskTenantBackups keeps the newest keep dumps on the backup disk.
func pruneDiskTenantBackups(code string, keep int) (int, error) {
	if keep <= 0 {
		return 0, nil
	}
	files, err := listDiskTenantBackups(code)
	if err != nil || len(files) <= keep {
		return 0, err
	}
	storage, err := tenantBackupStorage()
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, f := range files[keep:] {
		if err := storage.Delete(f.Path); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// ReadTenantBackup returns the content of a backup kept on the backup disk (for the
// download endpoint; local backups are streamed from their path instead).
func ReadTenantBackup(tenant *models.Tenant, name string) ([]byte, error) {
	if tenant == nil || tenant.Code == "" {
		return nil, apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	name, err := validTenantBackupName(name)
	if err != nil {
		return nil, err
	}
	storage, err := tenantBackupStorage()
	if err != nil {
		return nil, err
	}
	key := tenantBackupDiskDir(tenant.Code) + "/" + name
	if !storage.Exists(key) {
		return nil, apperrors.ErrInvalidArgument.WithMessage("backup file not found")
	}
	return storage.GetBytes(key)
}

// FetchTenantBackup returns a local path of the backup name for a restore. A backup
// on the backup disk is downloaded to a private temp file that cleanup removes.
func FetchTenantBackup(tenant *models.Tenant, name string) (string, func(), error) {
	noop := func() {}
	if TenantBackupDisk() == "" {
		p, err := ResolveTenantBackupAbsPath(tenant, name)
		return p, noop, err
	}
	data, err := ReadTenantBackup(tenant, name)
	if err != nil {
		return "", noop, err
	}
	tmp, err := os.CreateTemp("", "tenant-restore-*.sql") // 0600
	if err != nil {
		return "", noop, err
	}
	cleanup := func() { _ = os.Remove(tmp.Name()) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return "", noop, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", noop, err
	}
	return tmp.Name(), cleanup, nil
}

// purgeDiskTenantBackups deletes tenant-backups/{code}/ on the backup disk.
func purgeDiskTenantBackups(code string) error {
	if TenantBackupDisk() == "" {
		return nil
	}
	return purgePrefixOnDisk(TenantBackupDisk(), tenantBackupDiskDir(code))
}
