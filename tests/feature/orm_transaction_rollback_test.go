package feature_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

func configMarkerRow(key, value string) map[string]any {
	return map[string]any{
		"key":    key,
		"value":  value,
		"group":  "test",
		"type":   "string",
		"label":  "tx_test",
		"remark": "orm transaction rollback test",
	}
}

func countConfigKeyOn(q orm.Query, key string) (int64, error) {
	return q.Table("configs").Where("key", key).Count()
}

// TestOrmTransactionRollbackDefault covers platform/default connection rollback.
// Uses PlatformConnectionName explicitly so it stays valid even when TENANCY_DRIVER=database
// (does not toggle tenancy.driver mid-suite).
//
// Tenant-DB OrmTransaction is covered indirectly by isolation/migrate feature tests.
// A dedicated create+DropStorage case here repeatedly polluted CI (stale
// database.connections.tenant_* DSNs / pools) for later fleet tests.
func TestOrmTransactionRollbackDefault(t *testing.T) {
	platform := appfacades.BuildOrmConnection(appfacades.PlatformConnectionName())
	require.NotNil(t, platform)

	ctx := context.Background()
	pq := platform.WithContext(ctx).Query()
	if !appfacades.SchemaHasTable(ctx, "configs") {
		// SchemaHasTable follows tenant-aware schema; fall back to platform query probe.
		if _, err := pq.Table("configs").Limit(1).Count(); err != nil {
			t.Skip("configs table missing on platform connection")
		}
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	key := "tx_rb_def_" + suffix

	t.Cleanup(func() {
		_, _ = pq.Table("configs").Where("key", key).Delete(&models.Config{})
	})

	err := platform.WithContext(ctx).Transaction(func(tx orm.Query) error {
		if err := tx.Table("configs").Create(configMarkerRow(key, "should-roll-back")); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force rollback")

	n, err := countConfigKeyOn(pq, key)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "failed transaction must not leave a configs row")

	err = platform.WithContext(ctx).Transaction(func(tx orm.Query) error {
		return tx.Table("configs").Create(configMarkerRow(key, "committed"))
	})
	require.NoError(t, err)

	n, err = countConfigKeyOn(pq, key)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "successful transaction must commit")
}
