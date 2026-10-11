package services

import (
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"

	"goravel/app/entitlement"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

const (
	// PlanCodeNoneFilter matches tenants with no active subscription.
	PlanCodeNoneFilter = "_none"
)

// TenantSubscriptionListMeta is the active-plan snapshot for one tenant (list/detail).
type TenantSubscriptionListMeta struct {
	PlanCode    string     `json:"plan_code"`
	PlanName    string     `json:"plan_name"`
	EndsAt      *time.Time `json:"ends_at"`
	TrialEndsAt *time.Time `json:"trial_ends_at"`
	HasActive   bool       `json:"has_active"`
}

// TenantSubscriptionStats is plan distribution + upcoming expiry counts (landlord).
type TenantSubscriptionStats struct {
	ByPlanCode     map[string]int64 `json:"by_plan_code"`
	NoSubscription int64            `json:"no_subscription"`
	Expiring7d     int64            `json:"expiring_7d"`
	Expiring30d    int64            `json:"expiring_30d"`
}

type activeSubPick struct {
	ID          uint
	PlanCode    string
	PlanName    string
	EndsAt      *time.Time
	TrialEndsAt *time.Time
}

// BuildTenantSubscriptionStats aggregates active subscriptions for the given tenants.
// When tenantIDs is empty, returns zeroed stats (still seeds known plan codes).
func BuildTenantSubscriptionStats(tenantIDs []uint, picks map[uint]activeSubPick, planCodes []string, now time.Time) TenantSubscriptionStats {
	out := TenantSubscriptionStats{
		ByPlanCode: map[string]int64{},
	}
	for _, code := range planCodes {
		code = strings.ToLower(strings.TrimSpace(code))
		if code != "" {
			out.ByPlanCode[code] = 0
		}
	}
	if now.IsZero() {
		now = time.Now()
	}
	end7 := now.Add(7 * 24 * time.Hour)
	end30 := now.Add(30 * 24 * time.Hour)

	for _, id := range tenantIDs {
		pick, ok := picks[id]
		if !ok || !pick.HasActive() {
			out.NoSubscription++
			continue
		}
		code := strings.ToLower(strings.TrimSpace(pick.PlanCode))
		if code == "" {
			code = "_unknown"
		}
		out.ByPlanCode[code]++
		if pick.EndsAt == nil || pick.EndsAt.IsZero() || !pick.EndsAt.After(now) {
			continue
		}
		if !pick.EndsAt.After(end7) {
			out.Expiring7d++
		}
		if !pick.EndsAt.After(end30) {
			out.Expiring30d++
		}
	}
	return out
}

func (p activeSubPick) HasActive() bool {
	return p.ID > 0
}

// loadActiveSubscriptionPicks returns the latest active subscription per tenant.
func loadActiveSubscriptionPicks(tenantIDs []uint) (map[uint]activeSubPick, []string, error) {
	out := map[uint]activeSubPick{}
	planCodes := []string{}
	q := appfacades.PlatformOrmQuery(nil)
	if q == nil {
		return out, planCodes, nil
	}

	var plans []models.PlatformPlan
	_ = q.Model(&models.PlatformPlan{}).Where("status", 1).Order("sort asc").Order("id asc").Find(&plans)
	planByID := map[uint]models.PlatformPlan{}
	for _, p := range plans {
		planByID[p.ID] = p
		planCodes = append(planCodes, p.Code)
	}

	if len(tenantIDs) == 0 {
		return out, planCodes, nil
	}

	var subs []models.TenantSubscription
	if err := q.Model(&models.TenantSubscription{}).
		Where("status", entitlement.SubActive).
		Where("tenant_id IN ?", tenantIDs).
		Order("id asc").
		Find(&subs); err != nil {
		return out, planCodes, err
	}
	for _, sub := range subs {
		plan := planByID[sub.PlanID]
		out[sub.TenantID] = activeSubPick{
			ID:          sub.ID,
			PlanCode:    plan.Code,
			PlanName:    plan.Name,
			EndsAt:      sub.EndsAt,
			TrialEndsAt: sub.TrialEndsAt,
		}
	}
	return out, planCodes, nil
}

// LoadTenantSubscriptionListMeta loads active plan meta for tenant list rows.
func LoadTenantSubscriptionListMeta(tenantIDs []uint) map[uint]TenantSubscriptionListMeta {
	out := map[uint]TenantSubscriptionListMeta{}
	picks, _, err := loadActiveSubscriptionPicks(tenantIDs)
	if err != nil {
		return out
	}
	for _, id := range tenantIDs {
		pick, ok := picks[id]
		if !ok || !pick.HasActive() {
			out[id] = TenantSubscriptionListMeta{}
			continue
		}
		out[id] = TenantSubscriptionListMeta{
			PlanCode:    pick.PlanCode,
			PlanName:    pick.PlanName,
			EndsAt:      pick.EndsAt,
			TrialEndsAt: pick.TrialEndsAt,
			HasActive:   true,
		}
	}
	return out
}

func applySubscriptionFilters(query orm.Query, planCode, expiring string) orm.Query {
	planCode = strings.ToLower(strings.TrimSpace(planCode))
	expiring = strings.ToLower(strings.TrimSpace(expiring))
	now := time.Now()

	switch planCode {
	case "":
		// no plan filter
	case PlanCodeNoneFilter, "none", "unsubscribed":
		query = query.Where(
			"id NOT IN (SELECT tenant_id FROM tenant_subscriptions WHERE status = ?)",
			entitlement.SubActive,
		)
	default:
		query = query.Where(
			`id IN (
				SELECT ts.tenant_id FROM tenant_subscriptions ts
				INNER JOIN platform_plans pp ON pp.id = ts.plan_id
				WHERE ts.status = ? AND pp.code = ?
			)`,
			entitlement.SubActive, planCode,
		)
	}

	var until time.Time
	switch expiring {
	case "7", "7d", "expiring_7d":
		until = now.Add(7 * 24 * time.Hour)
	case "30", "30d", "expiring_30d":
		until = now.Add(30 * 24 * time.Hour)
	default:
		return query
	}
	return query.Where(
		`id IN (
			SELECT tenant_id FROM tenant_subscriptions
			WHERE status = ? AND ends_at IS NOT NULL AND ends_at > ? AND ends_at <= ?
		)`,
		entitlement.SubActive, now, until,
	)
}
