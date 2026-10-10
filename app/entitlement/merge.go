package entitlement

import (
	"strings"
	"time"
)

// MergeInput is the layered input for Resolve.
type MergeInput struct {
	TenantID uint
	PlanCode string
	Catalog  []FeatureMeta
	// Plan / Addon / Overrides are grants keyed by feature key (case-insensitive).
	Plan      map[string]Grant
	Addons    []map[string]Grant // each addon map; limits stack, bools OR
	Overrides map[string]Grant
	Now       time.Time
}

// Resolve merges plan < addons < overrides into an EffectiveSet.
// Catalog drives which keys exist; unknown grant keys are ignored.
func Resolve(in MergeInput) *EffectiveSet {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	out := NewEffectiveSet(in.TenantID)
	out.PlanCode = strings.TrimSpace(in.PlanCode)

	catalog := make(map[string]FeatureMeta, len(in.Catalog))
	for _, meta := range in.Catalog {
		key := normalizeKey(meta.Key)
		if key == "" || meta.Status == 0 {
			continue
		}
		meta.Key = key
		catalog[key] = meta
		out.FeatureChannels[key] = append([]string{}, meta.Channels...)
	}

	// 1) defaults from catalog
	for key, meta := range catalog {
		applyDefault(out, meta)
		_ = key
	}

	// 2) plan
	applyGrantMap(out, catalog, in.Plan, now, false)

	// 3) addons (stack limits, OR bools)
	for _, addon := range in.Addons {
		applyGrantMap(out, catalog, addon, now, true)
	}

	// 4) overrides (replace)
	applyGrantMap(out, catalog, in.Overrides, now, false)

	// always_on boolean features ignore plan/override
	for key, meta := range catalog {
		if meta.AlwaysOn && meta.Type == TypeBoolean {
			out.Features[key] = true
		}
	}

	// menu slug index
	for key, meta := range catalog {
		enabled := out.Features[key]
		if meta.Type != TypeBoolean {
			// non-boolean features do not gate menus by themselves
			continue
		}
		for _, slug := range meta.MenuSlugs {
			slug = strings.ToLower(strings.TrimSpace(slug))
			if slug == "" {
				continue
			}
			out.MenuSlugEnabled[slug] = enabled
		}
	}

	return out
}

func applyDefault(out *EffectiveSet, meta FeatureMeta) {
	switch meta.Type {
	case TypeBoolean:
		out.Features[meta.Key] = ParseBoolValue(meta.DefaultValue)
	case TypeLimit:
		out.Limits[meta.Key] = ParseLimitValue(meta.DefaultValue)
	case TypeConfig:
		out.Configs[meta.Key] = meta.DefaultValue
	default:
		out.Features[meta.Key] = ParseBoolValue(meta.DefaultValue)
	}
}

func applyGrantMap(out *EffectiveSet, catalog map[string]FeatureMeta, grants map[string]Grant, now time.Time, stackLimits bool) {
	if len(grants) == 0 {
		return
	}
	for rawKey, grant := range grants {
		key := normalizeKey(rawKey)
		if key == "" {
			key = normalizeKey(grant.Key)
		}
		meta, ok := catalog[key]
		if !ok {
			continue
		}
		if grant.ExpiresAt != nil && !grant.ExpiresAt.IsZero() && !grant.ExpiresAt.After(now) {
			continue
		}
		applyGrant(out, meta, grant.Value, stackLimits)
	}
}

func applyGrant(out *EffectiveSet, meta FeatureMeta, raw string, stackLimits bool) {
	switch meta.Type {
	case TypeBoolean:
		v := ParseBoolValue(raw)
		if stackLimits {
			// for bool, "stack" means OR
			out.Features[meta.Key] = out.Features[meta.Key] || v
		} else {
			out.Features[meta.Key] = v
		}
	case TypeLimit:
		v := ParseLimitValue(raw)
		if stackLimits {
			prev, ok := out.Limits[meta.Key]
			if !ok || prev < 0 || v < 0 {
				// any unlimited wins; else add
				if prev < 0 || v < 0 {
					out.Limits[meta.Key] = -1
				} else {
					out.Limits[meta.Key] = prev + v
				}
			} else {
				out.Limits[meta.Key] = prev + v
			}
		} else {
			out.Limits[meta.Key] = v
		}
	case TypeConfig:
		out.Configs[meta.Key] = raw
	default:
		out.Features[meta.Key] = ParseBoolValue(raw)
	}
}
