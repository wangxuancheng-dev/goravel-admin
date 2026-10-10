package entitlement

import (
	"testing"
	"time"
)

func TestResolvePlanAddonOverride(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	in := MergeInput{
		TenantID: 1,
		PlanCode: "pro",
		Catalog: []FeatureMeta{
			{Key: "module.guestbook", Type: TypeBoolean, DefaultValue: "false", Status: 1, MenuSlugs: []string{"guestbook"}},
			{Key: "quota.rows", Type: TypeLimit, DefaultValue: "0", Status: 1},
		},
		Plan: map[string]Grant{
			"module.guestbook": {Value: "true"},
			"quota.rows":       {Value: "10"},
		},
		Addons: []map[string]Grant{
			{"quota.rows": {Value: "5"}},
		},
		Overrides: map[string]Grant{
			"quota.rows": {Value: "100", ExpiresAt: &exp},
		},
	}
	set := Resolve(in)
	if !set.Can("module.guestbook") {
		t.Fatal("expected guestbook enabled by plan")
	}
	if set.Limits["quota.rows"] != 100 {
		t.Fatalf("override should replace stacked limit, got %d", set.Limits["quota.rows"])
	}
	if !set.MenuSlugEnabled["guestbook"] {
		t.Fatal("menu slug should be enabled")
	}
}

func TestResolveExpiredOverrideIgnored(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	in := MergeInput{
		TenantID: 1,
		Catalog: []FeatureMeta{
			{Key: "module.x", Type: TypeBoolean, DefaultValue: "false", Status: 1},
		},
		Plan: map[string]Grant{"module.x": {Value: "false"}},
		Overrides: map[string]Grant{
			"module.x": {Value: "true", ExpiresAt: &past},
		},
	}
	set := Resolve(in)
	if set.Can("module.x") {
		t.Fatal("expired override must not enable feature")
	}
}

func TestPresentFeatureOnOffIsGlobal(t *testing.T) {
	set := NewEffectiveSet(1)
	set.Features["module.x"] = true
	for _, ch := range []string{ChannelAdmin, ChannelPC, ChannelH5} {
		view := Present(set, ch, nil)
		if !view.Features["module.x"] {
			t.Fatalf("channel %s should see enabled feature", ch)
		}
	}
}

func TestModuleFeatureKey(t *testing.T) {
	if ModuleFeatureKey("Guest-Book") != "module.guest_book" {
		t.Fatalf("got %s", ModuleFeatureKey("Guest-Book"))
	}
	if ModuleRowsLimitKey("guestbook") != "quota.module.guestbook.rows" {
		t.Fatalf("got %s", ModuleRowsLimitKey("guestbook"))
	}
}

func TestResolveAlwaysOnIgnoresPlan(t *testing.T) {
	in := MergeInput{
		TenantID: 1,
		Catalog: []FeatureMeta{
			{Key: "module.internal", Type: TypeBoolean, DefaultValue: "false", Status: 1, AlwaysOn: true, MenuSlugs: []string{"internal"}},
		},
		Plan: map[string]Grant{"module.internal": {Value: "false"}},
	}
	set := Resolve(in)
	if !set.Can("module.internal") {
		t.Fatal("always_on must stay enabled against plan false")
	}
	if !set.MenuSlugEnabled["internal"] {
		t.Fatal("always_on menu should stay visible")
	}
}
