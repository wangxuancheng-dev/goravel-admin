package tenantstorage

import (
	"testing"

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
