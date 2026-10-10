package entitlement

import (
	"strconv"
	"strings"
	"time"
)

// Feature types stored in platform_features.type.
const (
	TypeBoolean = "boolean"
	TypeLimit   = "limit"
	TypeConfig  = "config"
)

// Subscription statuses.
const (
	SubPending   = "pending"
	SubActive    = "active"
	SubSuspended = "suspended"
	SubCanceled  = "canceled"
	SubExpired   = "expired"
)

// FeatureMeta is catalog metadata used while resolving and presenting.
type FeatureMeta struct {
	Key          string
	Type         string
	DefaultValue string
	MenuSlugs    []string
	Channels     []string
	AlwaysOn     bool // ignore plan/override; always enabled
	Status       uint8
}

// Grant is one layer contribution for a feature key.
type Grant struct {
	Key       string
	Value     string
	Source    string
	ExpiresAt *time.Time
}

// LimitView is the numeric quota view exposed to APIs/UI.
type LimitView struct {
	Limit     int64 `json:"limit"`
	Used      int64 `json:"used"`
	Remaining int64 `json:"remaining"`
	Unlimited bool  `json:"unlimited"`
}

// EffectiveSet is the resolved entitlement snapshot for one tenant.
type EffectiveSet struct {
	TenantID  uint              `json:"tenant_id"`
	Version   int64             `json:"version"`
	PlanCode  string            `json:"plan_code,omitempty"`
	Features  map[string]bool   `json:"features"`
	Limits    map[string]int64  `json:"limits"` // -1 = unlimited
	Configs   map[string]string `json:"configs"`
	// MenuSlugEnabled maps menu slug -> whether its owning feature is on.
	// Only slugs belonging to entitlement-managed features appear here.
	MenuSlugEnabled map[string]bool `json:"menu_slug_enabled"`
	// FeatureChannels maps feature key -> allowed channels.
	FeatureChannels map[string][]string `json:"feature_channels"`
}

// NewEffectiveSet returns an empty set ready for merge.
func NewEffectiveSet(tenantID uint) *EffectiveSet {
	return &EffectiveSet{
		TenantID:        tenantID,
		Features:        map[string]bool{},
		Limits:          map[string]int64{},
		Configs:         map[string]string{},
		MenuSlugEnabled: map[string]bool{},
		FeatureChannels: map[string][]string{},
	}
}

// Can reports whether a boolean feature is enabled.
func (s *EffectiveSet) Can(featureKey string) bool {
	if s == nil {
		return false
	}
	return s.Features[normalizeKey(featureKey)]
}

// CanOnChannel is the same as Can (feature on/off is global across clients).
func (s *EffectiveSet) CanOnChannel(featureKey, channel string) bool {
	_ = channel
	return s.Can(featureKey)
}

// Limit returns limit value; unlimited=true when limit is -1 or missing as unlimited.
func (s *EffectiveSet) Limit(limitKey string) (value int64, unlimited bool, ok bool) {
	if s == nil {
		return 0, false, false
	}
	v, exists := s.Limits[normalizeKey(limitKey)]
	if !exists {
		return 0, false, false
	}
	if v < 0 {
		return 0, true, true
	}
	return v, false, true
}

// DisabledMenuSlugs returns slugs that should be hidden for entitlement features.
// Slugs not managed by entitlements are omitted (legacy menus untouched).
func (s *EffectiveSet) DisabledMenuSlugs() map[string]bool {
	out := map[string]bool{}
	if s == nil {
		return out
	}
	for slug, enabled := range s.MenuSlugEnabled {
		if !enabled {
			out[strings.ToLower(strings.TrimSpace(slug))] = true
		}
	}
	return out
}

// ParseBoolValue accepts true/false/1/0/yes/no.
func ParseBoolValue(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

// ParseLimitValue parses a limit; empty or "unlimited" => -1.
func ParseLimitValue(raw string) int64 {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || raw == "unlimited" || raw == "null" || raw == "-1" {
		return -1
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// FormatBoolValue formats bool for storage.
func FormatBoolValue(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
