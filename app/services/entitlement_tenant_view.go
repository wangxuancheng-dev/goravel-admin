package services

import (
	"strings"
	"time"

	"goravel/app/entitlement"
	"goravel/app/models"
)

// TenantPlanBrief is the plan summary shown to tenant admins.
type TenantPlanBrief struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TenantSubscriptionBrief is the subscription window shown to tenant admins.
type TenantSubscriptionBrief struct {
	Status       string     `json:"status"`
	BillingCycle string     `json:"billing_cycle"`
	StartsAt     *time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	TrialEndsAt  *time.Time `json:"trial_ends_at"`
}

// TenantFeatureBrief is one boolean feature (module or capability).
type TenantFeatureBrief struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Kind     string `json:"kind"` // module|capability|other
	Enabled  bool   `json:"enabled"`
	AlwaysOn bool   `json:"always_on"`
}

// TenantLimitBrief is one quota with current usage.
type TenantLimitBrief struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Limit     int64  `json:"limit"`
	Used      int64  `json:"used"`
	Remaining int64  `json:"remaining"`
	Unlimited bool   `json:"unlimited"`
}

// TenantPlanSummary is the read-only "my plan" payload for tenant admins.
// It intentionally omits platform-only data (overrides, audit reasons, prices).
type TenantPlanSummary struct {
	Version      int64                    `json:"version"`
	Plan         *TenantPlanBrief         `json:"plan"`
	Subscription *TenantSubscriptionBrief `json:"subscription"`
	Features     []TenantFeatureBrief     `json:"features"`
	Limits       []TenantLimitBrief       `json:"limits"`
}

// TenantPlanSummaryFor builds the tenant-facing plan summary.
func (s *EntitlementService) TenantPlanSummaryFor(tenantID uint) (*TenantPlanSummary, error) {
	set, err := s.LoadEffective(tenantID)
	if err != nil {
		return nil, err
	}
	view, err := s.ClientViewForTenant(tenantID, entitlement.ChannelAdmin)
	if err != nil {
		return nil, err
	}

	out := &TenantPlanSummary{
		Version:  view.Version,
		Features: []TenantFeatureBrief{},
		Limits:   []TenantLimitBrief{},
	}

	var sub models.TenantSubscription
	planID := uint(0)
	if err := s.q().Where("tenant_id", tenantID).Where("status", entitlement.SubActive).
		Order("id desc").First(&sub); err == nil && sub.ID > 0 {
		planID = sub.PlanID
		out.Subscription = &TenantSubscriptionBrief{
			Status:       sub.Status,
			BillingCycle: sub.BillingCycle,
			StartsAt:     sub.StartsAt,
			EndsAt:       sub.EndsAt,
			TrialEndsAt:  sub.TrialEndsAt,
		}
	}

	var plan models.PlatformPlan
	if planID > 0 {
		_ = s.q().Where("id", planID).First(&plan)
	} else if code := strings.TrimSpace(set.PlanCode); code != "" {
		_ = s.q().Where("code", code).First(&plan)
	}
	if plan.ID > 0 {
		out.Plan = &TenantPlanBrief{Code: plan.Code, Name: plan.Name, Description: plan.Description}
	}

	features, err := s.ListFeatures()
	if err != nil {
		return nil, err
	}
	for _, f := range features {
		if f.Status == 0 {
			continue
		}
		switch f.Type {
		case entitlement.TypeBoolean:
			kind := "other"
			if entitlement.IsModuleFeatureKey(f.Key) {
				kind = "module"
				if strings.Count(f.Key, ".") >= 2 {
					kind = "capability"
				}
			}
			out.Features = append(out.Features, TenantFeatureBrief{
				Key:      f.Key,
				Name:     f.Name,
				Kind:     kind,
				Enabled:  set.Can(f.Key),
				AlwaysOn: f.AlwaysOn,
			})
		case entitlement.TypeLimit:
			lv, ok := view.Limits[f.Key]
			if !ok {
				continue
			}
			out.Limits = append(out.Limits, TenantLimitBrief{
				Key:       f.Key,
				Name:      f.Name,
				Limit:     lv.Limit,
				Used:      lv.Used,
				Remaining: lv.Remaining,
				Unlimited: lv.Unlimited,
			})
		}
	}
	return out, nil
}
