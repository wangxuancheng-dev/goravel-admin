package desensitize

import (
	"context"
	"testing"
)

func TestBindViewerCachesBypass(t *testing.T) {
	ctx := BindViewer(context.Background())
	cached, ok := ctx.Value(viewerBypassCacheKey{}).(bool)
	if !ok {
		t.Fatal("expected bypass cache on context")
	}
	// No admin in context => no bypass
	if cached {
		t.Fatal("expected bypass=false without admin")
	}
	p := Policy{Enabled: true, Modes: map[string]bool{ModeList: true}, BypassRoleSlugs: []string{"super-admin"}}
	if p.ViewerBypasses(ctx) {
		t.Fatal("cached bypass should be false")
	}
	// Second BindViewer should keep same cache
	ctx2 := BindViewer(ctx)
	if ctx2.Value(viewerBypassCacheKey{}) != cached {
		t.Fatal("BindViewer should keep existing cache")
	}
}

func TestShouldMaskRespectsCache(t *testing.T) {
	ctx := context.WithValue(context.Background(), viewerBypassCacheKey{}, true)
	p := Policy{
		Enabled:         true,
		Modes:           map[string]bool{ModeList: true},
		BypassRoleSlugs: []string{"super-admin"},
		Modules:         map[string]map[string]string{"user": {"phone": StrategyPhone}},
	}
	if p.ShouldMask(ctx, ModeList) {
		t.Fatal("bypass cache true should not mask")
	}
	got := p.ApplyString(ctx, "user", "phone", ModeList, "13800138000")
	if got != "13800138000" {
		t.Fatalf("got %q", got)
	}
}
