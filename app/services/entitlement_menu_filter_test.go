package services

import (
	"testing"

	"goravel/app/entitlement"
	"goravel/app/models"
)

func TestFilterTreeByDisabledRemovesEntitlementMenus(t *testing.T) {
	disabled := map[string]bool{"guestbook": true}
	menus := []models.Menu{
		{
			Slug: "system",
			Children: []models.Menu{
				{Slug: "guestbook"},
				{Slug: "role"},
			},
		},
	}
	out := filterTreeByDisabledForTest(menus, disabled)
	if len(out) != 1 || out[0].Slug != "system" {
		t.Fatalf("expected system root kept, got %+v", out)
	}
	if len(out[0].Children) != 1 || out[0].Children[0].Slug != "role" {
		t.Fatalf("expected guestbook child removed, got %+v", out[0].Children)
	}
}

func TestEnrichLogicAlwaysOnAndDefaultDeny(t *testing.T) {
	set := entitlement.NewEffectiveSet(1)
	set.Features["module.paid"] = true
	catalog := []entitlement.FeatureMeta{
		{Key: "module.paid", Type: entitlement.TypeBoolean, AlwaysOn: false, Status: 1, MenuSlugs: []string{"paid"}},
		{Key: "module.free", Type: entitlement.TypeBoolean, AlwaysOn: true, Status: 1, MenuSlugs: []string{"free"}},
		{Key: "module.new", Type: entitlement.TypeBoolean, DefaultValue: "false", Status: 1, MenuSlugs: []string{"new"}},
	}
	for _, f := range catalog {
		if f.AlwaysOn {
			set.Features[f.Key] = true
		} else if _, ok := set.Features[f.Key]; !ok {
			set.Features[f.Key] = entitlement.ParseBoolValue(f.DefaultValue)
		}
		for _, slug := range f.MenuSlugs {
			set.MenuSlugEnabled[slug] = set.Features[f.Key]
		}
	}
	if !set.Can("module.free") || !set.Can("module.paid") || set.Can("module.new") {
		t.Fatalf("features mismatch: %+v", set.Features)
	}
	if set.MenuSlugEnabled["new"] {
		t.Fatal("new module menu should be disabled by default")
	}
}

func filterTreeByDisabledForTest(menus []models.Menu, disabled map[string]bool) []models.Menu {
	filtered := make([]models.Menu, 0, len(menus))
	for _, menu := range menus {
		if disabled[menu.Slug] {
			continue
		}
		if len(menu.Children) > 0 {
			menu.Children = filterTreeByDisabledForTest(menu.Children, disabled)
		}
		filtered = append(filtered, menu)
	}
	return filtered
}
