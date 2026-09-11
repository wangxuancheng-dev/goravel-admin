package tenancyctx

import (
	"context"
	"testing"
)

func TestConnectionFrom(t *testing.T) {
	ctx := context.WithValue(context.Background(), KeyTenantConnection, "tenant_9")
	name, ok := ConnectionFrom(ctx)
	if !ok || name != "tenant_9" {
		t.Fatalf("got %q %v", name, ok)
	}
}

func TestIDFrom(t *testing.T) {
	ctx := context.WithValue(context.Background(), KeyTenantID, uint(7))
	id, ok := IDFrom(ctx)
	if !ok || id != 7 {
		t.Fatalf("got %d %v", id, ok)
	}
}
