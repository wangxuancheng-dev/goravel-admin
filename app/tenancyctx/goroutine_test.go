package tenancyctx

import (
	"context"
	"testing"
	"time"
)

func TestGoKeepsTenantDropsCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(WithTenant(context.Background(), 7, "tenant_7", "acme"))
	cancel() // the request has ended

	got := make(chan context.Context, 1)
	Go(parent, "test", func(ctx context.Context) { got <- ctx })

	select {
	case ctx := <-got:
		if err := ctx.Err(); err != nil {
			t.Fatalf("detached ctx is cancelled: %v", err)
		}
		if id, ok := IDFrom(ctx); !ok || id != 7 {
			t.Fatalf("tenant id = %d %v", id, ok)
		}
		if conn, ok := ConnectionFrom(ctx); !ok || conn != "tenant_7" {
			t.Fatalf("connection = %q %v", conn, ok)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fn did not run")
	}
}

func TestGoRecoversPanic(t *testing.T) {
	done := make(chan struct{})
	Go(context.Background(), "test", func(context.Context) {
		defer close(done)
		panic("boom")
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("fn did not run")
	}
}
