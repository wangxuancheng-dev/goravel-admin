package facades

import "testing"

func TestTenantSchemaBindRoundTrip(t *testing.T) {
	t.Parallel()
	UnbindTenantSchema()
	if BoundTenantSchema() != nil {
		t.Fatal("expected no bind")
	}
	// nil bind is no-op
	BindTenantSchema(nil)
	if BoundTenantSchema() != nil {
		t.Fatal("nil should not bind")
	}
}
