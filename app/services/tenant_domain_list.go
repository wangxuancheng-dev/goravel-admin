package services

import (
	"encoding/json"
	"strings"

	"github.com/goravel/framework/contracts/database/orm"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

const (
	TenantDomainStatusUnbound      = "unbound"
	TenantDomainStatusPendingView  = "pending"
	TenantDomainStatusActiveView   = "active"
	TenantDomainStatusVerifyFailed = "verify_failed"
	TenantDomainStatusDisabledView = "disabled"
)

// TenantDomainListMeta is list-row summary for vanity domains.
type TenantDomainListMeta struct {
	Status       string   `json:"domain_status"`
	PrimaryHost  string   `json:"domain_primary_host"`
	ActiveHosts  []string `json:"domain_active_hosts"`
	DomainCount  int      `json:"domain_count"`
	PendingCount int      `json:"domain_pending_count"`
	FailedCount  int      `json:"domain_failed_count"`
}

func (m TenantDomainListMeta) empty() TenantDomainListMeta {
	return TenantDomainListMeta{
		Status:      TenantDomainStatusUnbound,
		ActiveHosts: []string{},
	}
}

// LoadTenantDomainListMeta returns domain summary keyed by tenant id.
func LoadTenantDomainListMeta(tenantIDs []uint) map[uint]TenantDomainListMeta {
	out := make(map[uint]TenantDomainListMeta, len(tenantIDs))
	for _, id := range tenantIDs {
		out[id] = TenantDomainListMeta{}.empty()
	}
	if len(tenantIDs) == 0 {
		return out
	}
	var rows []models.TenantDomain
	_ = appfacades.PlatformOrmQuery(nil).
		Where("tenant_id IN ?", tenantIDs).
		Order("is_primary desc, id asc").
		Find(&rows)

	type agg struct {
		meta         TenantDomainListMeta
		hasActive    bool
		hasPending   bool
		hasFailed    bool
		hasDisabled  bool
		primaryHost  string
		activeHosts  []string
	}
	by := map[uint]*agg{}
	for i := range rows {
		r := &rows[i]
		a := by[r.TenantID]
		if a == nil {
			a = &agg{activeHosts: []string{}}
			by[r.TenantID] = a
		}
		a.meta.DomainCount++
		switch r.Status {
		case models.TenantDomainStatusActive:
			a.hasActive = true
			a.activeHosts = append(a.activeHosts, r.Host)
			if r.IsPrimary && a.primaryHost == "" {
				a.primaryHost = r.Host
			}
		case models.TenantDomainStatusPending, models.TenantDomainStatusVerified:
			a.hasPending = true
			a.meta.PendingCount++
			if strings.TrimSpace(r.LastCheckError) != "" {
				a.hasFailed = true
				a.meta.FailedCount++
			}
		case models.TenantDomainStatusDisabled:
			a.hasDisabled = true
		}
		if strings.TrimSpace(r.LastCheckError) != "" && r.Status != models.TenantDomainStatusActive {
			a.hasFailed = true
		}
	}
	for id, a := range by {
		status := TenantDomainStatusUnbound
		switch {
		case a.hasActive:
			status = TenantDomainStatusActiveView
		case a.hasFailed:
			status = TenantDomainStatusVerifyFailed
		case a.hasPending:
			status = TenantDomainStatusPendingView
		case a.hasDisabled:
			status = TenantDomainStatusDisabledView
		}
		primary := a.primaryHost
		if primary == "" && len(a.activeHosts) > 0 {
			primary = a.activeHosts[0]
		}
		out[id] = TenantDomainListMeta{
			Status:       status,
			PrimaryHost:  primary,
			ActiveHosts:  a.activeHosts,
			DomainCount:  a.meta.DomainCount,
			PendingCount: a.meta.PendingCount,
			FailedCount:  a.meta.FailedCount,
		}
	}
	return out
}

func applyDomainFilters(query orm.Query, domainHost, domainStatus string) orm.Query {
	domainHost = strings.TrimSpace(domainHost)
	domainStatus = strings.ToLower(strings.TrimSpace(domainStatus))
	if domainHost != "" {
		query = query.Where(
			"id IN (SELECT tenant_id FROM tenant_domains WHERE deleted_at IS NULL AND host LIKE ?)",
			"%"+domainHost+"%",
		)
	}
	switch domainStatus {
	case TenantDomainStatusActiveView:
		query = query.Where(
			"id IN (SELECT tenant_id FROM tenant_domains WHERE deleted_at IS NULL AND status = ?)",
			models.TenantDomainStatusActive,
		)
	case TenantDomainStatusPendingView:
		query = query.Where(
			`id IN (
				SELECT tenant_id FROM tenant_domains
				WHERE deleted_at IS NULL AND status IN (?, ?)
			) AND id NOT IN (
				SELECT tenant_id FROM tenant_domains
				WHERE deleted_at IS NULL AND status = ?
			)`,
			models.TenantDomainStatusPending, models.TenantDomainStatusVerified, models.TenantDomainStatusActive,
		)
	case TenantDomainStatusVerifyFailed:
		query = query.Where(
			`id IN (
				SELECT tenant_id FROM tenant_domains
				WHERE deleted_at IS NULL AND last_check_error IS NOT NULL AND last_check_error <> ''
				  AND status <> ?
			)`,
			models.TenantDomainStatusActive,
		)
	case TenantDomainStatusDisabledView:
		query = query.Where(
			`id IN (SELECT tenant_id FROM tenant_domains WHERE deleted_at IS NULL AND status = ?)
			 AND id NOT IN (SELECT tenant_id FROM tenant_domains WHERE deleted_at IS NULL AND status IN (?, ?, ?))`,
			models.TenantDomainStatusDisabled,
			models.TenantDomainStatusActive, models.TenantDomainStatusPending, models.TenantDomainStatusVerified,
		)
	case TenantDomainStatusUnbound, "none":
		query = query.Where(
			"id NOT IN (SELECT tenant_id FROM tenant_domains WHERE deleted_at IS NULL)",
		)
	}
	return query
}

func parseHealthIssues(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []string{}
	}
	return list
}

func encodeHealthIssues(issues []string) string {
	if len(issues) == 0 {
		return "[]"
	}
	b, err := json.Marshal(issues)
	if err != nil {
		return "[]"
	}
	return string(b)
}
