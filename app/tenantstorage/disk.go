package tenantstorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cosdriver "github.com/goravel/cos"
	contractsconfig "github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/filesystem"
	"github.com/goravel/framework/facades"
	fwfilesystem "github.com/goravel/framework/filesystem"
	miniodriver "github.com/goravel/minio"
	ossdriver "github.com/goravel/oss"
	s3driver "github.com/goravel/s3"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/support/configoverlay"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
	"goravel/app/tenantcache"
)

const (
	ModeShared = models.TenantStorageModeShared
	ModeCustom = models.TenantStorageModeCustom

	diskPrefix   = "tenant_byob_"
	secretPrefix = "enc:v1:"
)

// diskEntry is one tenant's storage driver, built from its own config overlay (the
// credentials are never written to the global config) and reused across requests
// until the credentials change.
type diskEntry struct {
	fingerprint string
	cfg         contractsconfig.Config
	driver      filesystem.Driver
	lastUsed    atomic.Int64 // unix nanos
}

var (
	disksMu   sync.RWMutex
	disks     = map[uint]*diskEntry{} // tenantID -> driver
	lastSweep atomic.Int64
)

// Drivers of tenants that stopped using their bucket are dropped like idle tenant DB
// connections (tenancy.registered_idle_ttl / tenancy.registered_max). They hold only
// an API client, so dropping the reference is enough; the next use rebuilds it.
const sweepEvery = time.Minute

func idleTTL() time.Duration {
	sec := facades.Config().GetInt("tenancy.registered_idle_ttl", 900)
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

// sweepIdle drops drivers unused for idleTTL and, over tenancy.registered_max, the
// least recently used ones. Runs at most once per sweepEvery.
func sweepIdle(now time.Time) {
	last := lastSweep.Load()
	if now.UnixNano()-last < int64(sweepEvery) || !lastSweep.CompareAndSwap(last, now.UnixNano()) {
		return
	}
	ttl := idleTTL()
	maxN := facades.Config().GetInt("tenancy.registered_max", 0)

	disksMu.Lock()
	defer disksMu.Unlock()
	if ttl > 0 {
		for id, e := range disks {
			if now.Sub(time.Unix(0, e.lastUsed.Load())) >= ttl {
				delete(disks, id)
			}
		}
	}
	for maxN > 0 && len(disks) > maxN {
		var oldestID uint
		var oldest int64
		for id, e := range disks {
			if at := e.lastUsed.Load(); oldestID == 0 || at < oldest {
				oldestID, oldest = id, at
			}
		}
		delete(disks, oldestID)
	}
}

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

// LoadTenant loads platform tenant row by id (cached); withTrashed also finds
// soft-deleted tenants (purge / reading their files), which are never cached.
func LoadTenant(tenantID uint, withTrashed bool) (*models.Tenant, error) {
	t, err := tenantcache.ByID(tenantID)
	if err == nil {
		return t, nil
	}
	if !withTrashed {
		return nil, apperrors.ErrTenantNotFound.WithError(err)
	}
	var trashed models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).WithTrashed().Where("id", tenantID).First(&trashed); err != nil {
		return nil, apperrors.ErrTenantNotFound.WithError(err)
	}
	if trashed.ID == 0 {
		return nil, apperrors.ErrTenantNotFound
	}
	return &trashed, nil
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

// OpenDisk opens a filesystem driver. tenant_byob_* disks return the tenant's cached
// driver (built from its credentials, never from the global config).
func OpenDisk(disk string) (filesystem.Driver, error) {
	disk = strings.TrimSpace(disk)
	if disk == "" {
		disk = facades.Config().GetString("filesystems.default", "local")
	}
	if IsByobDisk(disk) {
		e, err := byobEntry(disk)
		if err != nil {
			return nil, err
		}
		return e.driver, nil
	}
	driver, err := fwfilesystem.NewDriver(facades.Config(), disk)
	if err != nil {
		return nil, apperrors.ErrStorageDiskNotConfigured.WithParams(map[string]any{"disk": disk}).WithError(err)
	}
	return driver, nil
}

// DiskConfig returns the config that holds disk's settings
// (filesystems.disks.<disk>.*): the tenant's own for tenant_byob_* disks, the global
// config otherwise. Use it to read a disk's credentials (e.g. streaming uploads).
func DiskConfig(disk string) (contractsconfig.Config, error) {
	if IsByobDisk(disk) {
		e, err := byobEntry(disk)
		if err != nil {
			return nil, err
		}
		return e.cfg, nil
	}
	return facades.Config(), nil
}

func byobEntry(disk string) (*diskEntry, error) {
	id, ok := ParseTenantIDFromDisk(disk)
	if !ok {
		return nil, apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": disk})
	}
	t, err := LoadTenant(id, true)
	if err != nil {
		return nil, err
	}
	return entryFor(t)
}

// EnsureRegistered builds (or reuses) the tenant's BYOB storage driver.
func EnsureRegistered(t *models.Tenant) error {
	_, err := entryFor(t)
	return err
}

func entryFor(t *models.Tenant) (*diskEntry, error) {
	if t == nil || t.ID == 0 {
		return nil, apperrors.ErrInvalidArgument
	}
	if !CredentialsReady(t) {
		return nil, apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": DiskName(t.ID)})
	}
	secret, err := RevealSecret(t.StorageSecret)
	if err != nil {
		return nil, err
	}
	fp := fingerprint(t, secret)
	now := time.Now()
	sweepIdle(now)

	disksMu.RLock()
	e := disks[t.ID]
	disksMu.RUnlock()
	if e != nil && e.fingerprint == fp {
		e.lastUsed.Store(now.UnixNano())
		return e, nil
	}

	// Building only creates an API client (no network), so a duplicate build by a
	// concurrent first request is harmless; the last one stored wins.
	e, err = buildEntry(t, secret, fp)
	if err != nil {
		return nil, err
	}
	e.lastUsed.Store(now.UnixNano())
	disksMu.Lock()
	disks[t.ID] = e
	disksMu.Unlock()
	return e, nil
}

// CachedDriverCount returns how many tenant BYOB drivers are kept (tests, metrics).
func CachedDriverCount() int {
	disksMu.RLock()
	defer disksMu.RUnlock()
	return len(disks)
}

// fingerprint identifies the credentials a driver was built from (hashed, so the
// secret is not kept in memory in plain text).
func fingerprint(t *models.Tenant, secret string) string {
	h := sha256.New()
	for _, part := range []string{
		NormalizeDriver(t.StorageDriver),
		t.StorageKey,
		secret,
		t.StorageRegion,
		t.StorageBucket,
		t.StorageURL,
		t.StorageEndpoint,
		strconv.FormatBool(t.StorageUsePathStyle),
		strconv.FormatBool(t.StorageSSL),
	} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func buildEntry(t *models.Tenant, secret, fp string) (*diskEntry, error) {
	name := DiskName(t.ID)
	values := diskValues(NormalizeDriver(t.StorageDriver), t, secret)
	cfg := configoverlay.New(facades.Config(), "filesystems.disks."+name, values)

	ctx := context.Background()
	var (
		driver filesystem.Driver
		err    error
	)
	switch NormalizeDriver(t.StorageDriver) {
	case "s3":
		driver, err = s3driver.NewS3(ctx, cfg, name)
	case "oss":
		driver, err = ossdriver.NewOss(ctx, cfg, name)
	case "cos":
		driver, err = cosdriver.NewCos(ctx, cfg, name)
	case "minio":
		driver, err = miniodriver.NewMinio(ctx, cfg, name)
	default:
		return nil, apperrors.ErrTenantStorageNotConfigured.WithParams(map[string]any{"disk": name})
	}
	if err != nil {
		return nil, apperrors.ErrStorageDiskNotConfigured.WithParams(map[string]any{"disk": name}).WithError(err)
	}
	return &diskEntry{fingerprint: fp, cfg: cfg, driver: driver}, nil
}

// diskValues are the filesystems.disks.<name>.* keys the goravel s3/oss/cos/minio
// drivers read.
func diskValues(driver string, t *models.Tenant, secret string) map[string]any {
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
		}
	case "oss":
		return map[string]any{
			"driver":   "custom",
			"key":      t.StorageKey,
			"secret":   secret,
			"bucket":   t.StorageBucket,
			"url":      t.StorageURL,
			"endpoint": t.StorageEndpoint,
		}
	case "cos":
		return map[string]any{
			"driver": "custom",
			"key":    t.StorageKey,
			"secret": secret,
			"url":    t.StorageURL,
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
		"storage_mode":           mode,
		"storage_driver":         NormalizeDriver(t.StorageDriver),
		"storage_key":            t.StorageKey,
		"storage_has_secret":     HasSecret(t.StorageSecret),
		"storage_region":         t.StorageRegion,
		"storage_bucket":         t.StorageBucket,
		"storage_url":            t.StorageURL,
		"storage_endpoint":       t.StorageEndpoint,
		"storage_use_path_style": t.StorageUsePathStyle,
		"storage_ssl":            t.StorageSSL,
		"storage_write_disk":     writeDiskHint(t),
		"storage_credentials_ok": CredentialsReady(t),
	}
}

func writeDiskHint(t *models.Tenant) string {
	if NormalizeMode(t.StorageMode) == ModeCustom && CredentialsReady(t) {
		return DiskName(t.ID)
	}
	return "shared"
}

// Invalidate drops the tenant's cached driver so the next open rebuilds it.
// (Changed credentials are also picked up on their own: the fingerprint differs.)
func Invalidate(tenantID uint) {
	disksMu.Lock()
	delete(disks, tenantID)
	disksMu.Unlock()
}
