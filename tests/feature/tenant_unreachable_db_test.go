package feature_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"goravel/app/tenantdb"
)

// An unreachable tenant database host must fail within the caller's timeout. The
// mysql driver's version probe has no timeout of its own, so the ping has to come
// first; otherwise every request of that tenant waits for the OS TCP timeout.
func TestTenantConnectionToDeadHostFailsWithinTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	c, err := tenantdb.Open(ctx, "tenant_unreachable_probe", tenantdb.Spec{
		Driver:   tenantdb.DriverMySQL,
		Host:     "10.255.255.1", // non-routable: connects hang until the timeout
		Port:     3306,
		Database: "tenant_unreachable",
		Username: "u",
		Password: "p",
	})
	elapsed := time.Since(start)
	if c != nil {
		_ = c.Close()
	}
	require.Error(t, err)
	require.Less(t, elapsed, 5*time.Second, "building the connection must honour the 2s timeout, took %s", elapsed)
}
