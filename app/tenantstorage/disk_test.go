package tenantstorage

import (
	"sort"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"

	"goravel/app/models"
)

func TestDiskNameAndParse(t *testing.T) {
	assert.Equal(t, "tenant_byob_12", DiskName(12))
	id, ok := ParseTenantIDFromDisk("tenant_byob_12")
	assert.True(t, ok)
	assert.Equal(t, uint(12), id)
	assert.True(t, IsByobDisk("tenant_byob_1"))
	assert.False(t, IsByobDisk("s3"))
}

func TestNormalizeModeAndDriver(t *testing.T) {
	assert.Equal(t, ModeShared, NormalizeMode(""))
	assert.Equal(t, ModeCustom, NormalizeMode("custom"))
	assert.Equal(t, "s3", NormalizeDriver("S3"))
	assert.Equal(t, "", NormalizeDriver("local"))
}

func TestCredentialsReady(t *testing.T) {
	tnt := &models.Tenant{
		StorageDriver: "s3",
		StorageKey:    "ak",
		StorageSecret: "enc:v1:x",
		StorageRegion: "us-east-1",
		StorageBucket: "b",
		StorageURL:    "https://example.com",
	}
	assert.True(t, CredentialsReady(tnt))
	tnt.StorageBucket = ""
	assert.False(t, CredentialsReady(tnt))
}

func seedDrivers(t *testing.T, ages map[uint]time.Duration) {
	t.Helper()
	disksMu.Lock()
	disks = map[uint]*diskEntry{}
	now := time.Now()
	for id, age := range ages {
		e := &diskEntry{}
		e.lastUsed.Store(now.Add(-age).UnixNano())
		disks[id] = e
	}
	disksMu.Unlock()
	lastSweep.Store(0)
	t.Cleanup(func() {
		disksMu.Lock()
		disks = map[uint]*diskEntry{}
		disksMu.Unlock()
	})
}

func setTenancyLimits(t *testing.T, idleSec, maxN int) {
	t.Helper()
	prevTTL := facades.Config().GetInt("tenancy.registered_idle_ttl", 900)
	prevMax := facades.Config().GetInt("tenancy.registered_max", 0)
	facades.Config().Add("tenancy.registered_idle_ttl", idleSec)
	facades.Config().Add("tenancy.registered_max", maxN)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.registered_idle_ttl", prevTTL)
		facades.Config().Add("tenancy.registered_max", prevMax)
	})
}

func keptIDs() []uint {
	disksMu.RLock()
	defer disksMu.RUnlock()
	ids := make([]uint, 0, len(disks))
	for id := range disks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func TestIdleDriversAreDropped(t *testing.T) {
	setTenancyLimits(t, 60, 0)
	seedDrivers(t, map[uint]time.Duration{1: 2 * time.Minute, 2: 10 * time.Second})
	sweepIdle(time.Now())
	assert.Equal(t, []uint{2}, keptIDs(), "a driver idle past the TTL must be dropped")
}

func TestDriversOverMaxDropLeastRecentlyUsed(t *testing.T) {
	setTenancyLimits(t, 0, 2)
	seedDrivers(t, map[uint]time.Duration{1: 3 * time.Minute, 2: 2 * time.Minute, 3: time.Minute})
	sweepIdle(time.Now())
	assert.Equal(t, []uint{2, 3}, keptIDs(), "over the cap the least recently used must go")
}

func TestSweepRunsAtMostOncePerInterval(t *testing.T) {
	setTenancyLimits(t, 60, 0)
	seedDrivers(t, map[uint]time.Duration{1: 2 * time.Minute})
	now := time.Now()
	lastSweep.Store(now.UnixNano())
	sweepIdle(now.Add(10 * time.Second))
	assert.Equal(t, []uint{1}, keptIDs(), "no sweep within the interval")
	sweepIdle(now.Add(sweepEvery + time.Second))
	assert.Empty(t, keptIDs(), "sweeps again after the interval")
}
