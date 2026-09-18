package tenantstorage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/goravel/framework/contracts/filesystem"
	"github.com/goravel/framework/facades"
	fwfilesystem "github.com/goravel/framework/filesystem"
	cosfacades "github.com/goravel/cos/facades"
	miniofacades "github.com/goravel/minio/facades"
	ossfacades "github.com/goravel/oss/facades"
	s3facades "github.com/goravel/s3/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

const (
	ModeShared = models.TenantStorageModeShared
	ModeCustom = models.TenantStorageModeCustom

	diskPrefix = "tenant_byob_"
	secretPrefix = "enc:v1:"
)

var (
	registerMu sync.Mutex
	registered = map[uint]string{} // tenantID -> config fingerprint
)

// DiskName returns the runtime filesystem disk key for a tenant BYOB config.
func DiskName(tenantID uint) string {
	return fmt.Sprintf("%s%d", diskPrefix, tenantID)
}

// IsByobDisk reports whether disk is a per-tenant BYOB disk name.
func IsByobDisk(disk string) bool {
	return strings.HasPrefix(strings.TrimSpace(disk), diskPrefix)
}

// ParseTenantIDFromDisk extracts tenant id from tenant_byob_{id}.
func ParseTenantIDFromDisk(disk string) (uint, bool) {
	disk = strings.TrimSpace(disk)
	if !strings.HasPrefix(disk, diskPrefix) {
		return 0, false
	}
	raw := strings.TrimPrefix(disk, diskPrefix)
	id64, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id64 == 0 {
		return 0, false
	}
	return uint(id64), true
}

// NormalizeMode returns shared|custom.
func NormalizeMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == ModeCustom {
		return ModeCustom
	}
	return ModeShared
}

// NormalizeDriver returns s3|oss|cos|minio or empty.
func NormalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "s3", "oss", "cos", "minio":
		return strings.ToLower(strings.TrimSpace(driver))
	default:
		return ""
	}
}

// SealSecret encrypts a storage secret for platform DB storage.
func SealSecret(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if strings.HasPrefix(plain, secretPrefix) {
		return "", apperrors.ErrInvalidArgument.WithMessage("do not pass already-sealed storage secret")
	}
	enc, err := facades.Crypt().EncryptString(plain)
	if err != nil {
		return "", err
	}
	return secretPrefix + enc, nil
}

// RevealSecret decrypts a sealed storage secret.
func RevealSecret(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, secretPrefix) {
		return "", apperrors.ErrTenantStorageSecretCorrupt
	}
	plain, err := facades.Crypt().DecryptString(strings.TrimPrefix(stored, secretPrefix))
	if err != nil {
		return "", apperrors.ErrTenantStorageSecretCorrupt.WithError(err)
	}
	return plain, nil
}

// HasSecret reports whether a sealed secret is stored.
func HasSecret(stored string) bool {
	return strings.TrimSpace(stored) != ""
}

// LoadTenant loads platform tenant row by id (including soft-deleted for purge/read).
func LoadTenant(tenantID uint, withTrashed bool) (*models.Tenant, error) {
	var t models.Tenant
	q := appfacades.PlatformOrmQuery(nil)
	if withTrashed {
		q = q.WithTrashed()
	}
	if err := q.Where("id", tenantID).First(&t); err != nil {
		return nil, apperrors.ErrTenantNotFound.WithError(err)
	}
	return &t, nil
}

// CredentialsReady reports whether custom bucket fields are complete enough to register.
func CredentialsReady(t *models.Tenant) bool {
	if t == nil {
		return false
	}
	driver := NormalizeDriver(t.StorageDriver)
	if driver == "" || strings.TrimSpace(t.StorageBucket) == "" || strings.TrimSpace(t.StorageKey) == "" || !HasSecret(t.StorageSecret) {
		return false
	}
	switch driver {
	case "s3":
		return strings.TrimSpace(t.StorageRegion) != "" && strings.TrimSpace(t.StorageURL) != ""
	case "oss":
		return strings.TrimSpace(t.StorageURL) != "" && strings.TrimSpace(t.StorageEndpoint) != ""
	case "cos":
		return strings.TrimSpace(t.StorageURL) != ""
	case "minio":
		return strings.TrimSpace(t.StorageBucket) != "" && strings.TrimSpace(t.StorageURL) != "" && strings.TrimSpace(t.StorageEndpoint) != ""
	default:
		return false
	}
}

// ResolveWriteDisk returns the disk name for new uploads when tenancy is on.
// Falls back to empty string so callers use the platform default.
func ResolveWriteDisk(ctx context.Context) string {
	if !tenancy.Enabled() {
		return ""
	}
	id, ok := tenancyctx.IDFrom(ctx)
	if !ok || id == 0 {
		return ""
	}
	t, err := LoadTenant(id, false)
	if err != nil {
		return ""
	}
	if NormalizeMode(t.StorageMode) != ModeCustom {
		return ""
	}
	if err := EnsureRegistered(t); err != nil {
		facades.Log().Warningf("tenant BYOB register failed (fallback shared): tenant=%d err=%v", id, err)
		return ""
	}
	return DiskName(id)
}

// OpenDisk opens a filesystem driver. For tenant_byob_* it loads/registers tenant credentials.
func OpenDisk(disk string) (filesystem.Driver, error) {
	disk = strings.TrimSpace(disk)
	if disk == "" {
		disk = facades.Config().GetString("filesystems.default", "local")
	}
	if IsByobDisk(disk) {
		id, ok := ParseTenantIDFromDisk(disk)
		if !ok {
			return nil, apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": disk})
		}
		t, err := LoadTenant(id, true)
		if err != nil {
			return nil, err
		}
		if err := EnsureRegistered(t); err != nil {
			return nil, err
		}
		disk = DiskName(id)
	}
	driver, err := fwfilesystem.NewDriver(facades.Config(), disk)
	if err != nil {
		return nil, apperrors.ErrStorageDiskNotConfigured.WithParams(map[string]any{"disk": disk}).WithError(err)
	}
	return driver, nil
}

// EnsureRegistered writes filesystem disk config for the tenant BYOB credentials.
func EnsureRegistered(t *models.Tenant) error {
	if t == nil || t.ID == 0 {
		return apperrors.ErrInvalidArgument
	}
	if !CredentialsReady(t) {
		return apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": DiskName(t.ID)})
	}
	secret, err := RevealSecret(t.StorageSecret)
	if err != nil {
		return err
	}
	driver := NormalizeDriver(t.StorageDriver)
	name := DiskName(t.ID)
	fp := fingerprint(t, secret)

	registerMu.Lock()
	defer registerMu.Unlock()
	if registered[t.ID] == fp {
		return nil
	}

	cfg := buildDiskConfig(name, driver, t, secret)
	facades.Config().Add("filesystems.disks."+name, cfg)
	registered[t.ID] = fp
	return nil
}

func fingerprint(t *models.Tenant, secret string) string {
	return strings.Join([]string{
		NormalizeDriver(t.StorageDriver),
		t.StorageKey,
		secret,
		t.StorageRegion,
		t.StorageBucket,
		t.StorageURL,
		t.StorageEndpoint,
		strconv.FormatBool(t.StorageUsePathStyle),
		strconv.FormatBool(t.StorageSSL),
	}, "|")
}

func buildDiskConfig(name, driver string, t *models.Tenant, secret string) map[string]any {
	switch driver {
	case "s3":
		return map[string]any{
			"driver":         "custom",
			"key":            t.StorageKey,
			"secret":         secret,
			"region":         t.StorageRegion,
			"bucket":         t.StorageBucket,
			"url":            t.StorageURL,
			"endpoint":       t.StorageEndpoint,
			"use_path_style": t.StorageUsePathStyle,
			"via": func() (filesystem.Driver, error) {
				return s3facades.S3(name)
			},
		}
	case "oss":
		return map[string]any{
			"driver":   "custom",
			"key":      t.StorageKey,
			"secret":   secret,
			"bucket":   t.StorageBucket,
			"url":      t.StorageURL,
			"endpoint": t.StorageEndpoint,
			"via": func() (filesystem.Driver, error) {
				return ossfacades.Oss(name)
			},
		}
	case "cos":
		return map[string]any{
			"driver": "custom",
			"key":    t.StorageKey,
			"secret": secret,
			"url":    t.StorageURL,
			"via": func() (filesystem.Driver, error) {
				return cosfacades.Cos(name)
			},
		}
	case "minio":
		return map[string]any{
			"driver":   "custom",
			"key":      t.StorageKey,
			"secret":   secret,
			"region":   t.StorageRegion,
			"bucket":   t.StorageBucket,
			"url":      t.StorageURL,
			"endpoint": t.StorageEndpoint,
			"ssl":      t.StorageSSL,
			"via": func() (filesystem.Driver, error) {
				return miniofacades.Minio(name)
			},
		}
	default:
		return map[string]any{"driver": "local"}
	}
}

// PublicJSON returns safe storage fields for platform API (never secret).
func PublicJSON(t *models.Tenant) map[string]any {
	if t == nil {
		return map[string]any{}
	}
	mode := NormalizeMode(t.StorageMode)
	return map[string]any{
		"storage_mode":            mode,
		"storage_driver":          NormalizeDriver(t.StorageDriver),
		"storage_key":             t.StorageKey,
		"storage_has_secret":      HasSecret(t.StorageSecret),
		"storage_region":          t.StorageRegion,
		"storage_bucket":          t.StorageBucket,
		"storage_url":             t.StorageURL,
		"storage_endpoint":        t.StorageEndpoint,
		"storage_use_path_style":  t.StorageUsePathStyle,
		"storage_ssl":             t.StorageSSL,
		"storage_write_disk":      writeDiskHint(t),
		"storage_credentials_ok":  CredentialsReady(t),
	}
}

func writeDiskHint(t *models.Tenant) string {
	if NormalizeMode(t.StorageMode) == ModeCustom && CredentialsReady(t) {
		return DiskName(t.ID)
	}
	return "shared"
}

// Invalidate clears cached registration so next open reloads credentials.
func Invalidate(tenantID uint) {
	registerMu.Lock()
	delete(registered, tenantID)
	registerMu.Unlock()
}
