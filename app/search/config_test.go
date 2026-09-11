package search

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"goravel/app/tenancyctx"
)

func TestOrdersIndexShortNameFor_tenantCode(t *testing.T) {
	ctx := tenancyctx.WithTenant(context.Background(), 9, "tenant_9", "Acme-Co")
	// tenancy.Enabled() depends on config; when off, should equal base
	got := OrdersIndexShortNameFor(ctx)
	base := OrdersIndexShortName()
	if got != base && got != "acme_co_"+base {
		// Either tenancy off (base) or on with sanitized code
		assert.True(t, got == base || got == "acme_co_"+base || got == "t9_"+base, "got=%s", got)
	}
}

func TestIsOrdersIndexShortName(t *testing.T) {
	base := OrdersIndexShortName()
	assert.True(t, IsOrdersIndexShortName(base))
	assert.True(t, IsOrdersIndexShortName("orders"))
	assert.True(t, IsOrdersIndexShortName("acme_"+base))
	assert.False(t, IsOrdersIndexShortName("products"))
}

func TestSanitizeIndexSegment(t *testing.T) {
	assert.Equal(t, "acme_co", sanitizeIndexSegment("Acme-Co"))
	assert.Equal(t, "t1", sanitizeIndexSegment("t1"))
}
