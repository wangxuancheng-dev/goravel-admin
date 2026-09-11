package services

import (
	"testing"
	"time"

	"github.com/goravel/framework/facades"
)

func TestAdminTokenExpiresAtRespectsTTL(t *testing.T) {
	original := facades.Config().GetInt("jwt.ttl", 60)
	t.Cleanup(func() {
		facades.Config().Add("jwt.ttl", original)
	})

	facades.Config().Add("jwt.ttl", 30)
	exp := AdminTokenExpiresAt()
	if exp == nil {
		t.Fatal("expected non-nil expiry when ttl > 0")
	}
	want := time.Now().Add(30 * time.Minute)
	if exp.Before(want.Add(-2*time.Minute)) || exp.After(want.Add(2*time.Minute)) {
		t.Fatalf("expiry out of range: got %v want ~%v", exp, want)
	}

	facades.Config().Add("jwt.ttl", 0)
	if AdminTokenExpiresAt() != nil {
		t.Fatal("ttl<=0 should mean never expire (nil)")
	}
}
