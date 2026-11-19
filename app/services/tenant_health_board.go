package services

import (
	"sort"
	"strings"
	"time"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// TenantHealthStaleAfter marks a health snapshot as stale (inspect runs hourly).
const TenantHealthStaleAfter = 2 * time.Hour

// tenantHealthBoardMaxItems caps abnormal rows returned to the console.
const tenantHealthBoardMaxItems = 200

// TenantHealthBoardItem is one warn/fail tenant on the health board.
type TenantHealthBoardItem struct {
	ID              uint       `json:"id"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	HealthStatus    string     `json:"health_status"`
	Issues          []string   `json:"issues"`
	LastPingOK      bool       `json:"last_ping_ok"`
	LastPingMs      int64      `json:"last_ping_ms"`
	HealthCheckedAt *time.Time `json:"health_checked_at"`
	Maintenance     bool       `json:"maintenance"`
	Stale           bool       `json:"stale"`
}

// TenantHealthBoard aggregates the latest stored health snapshots of active tenants.
type TenantHealthBoard struct {
	Checked       int64                   `json:"checked"`
	Stale         int64                   `json:"stale"`
	NeverChecked  int64                   `json:"never_checked"`
	LastCheckedAt *time.Time              `json:"last_checked_at"`
	StaleAfterSec int64                   `json:"stale_after_sec"`
	ByStatus      map[string]int64        `json:"by_status"`
	ByIssue       map[string]int64        `json:"by_issue"`
	Abnormal      int64                   `json:"abnormal"`
	Truncated     bool                    `json:"truncated"`
	Items         []TenantHealthBoardItem `json:"items"`
}

// HealthBoard reads stored inspect results; it does not ping tenants.
func (s *TenantAdminService) HealthBoard() (*TenantHealthBoard, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var tenants []models.Tenant
	if err := appfacades.PlatformQuery().Model(&models.Tenant{}).
		Select("id", "code", "name", "maintenance", "health_status", "health_issues", "health_checked_at", "last_ping_ok", "last_ping_ms").
		Where("status", models.TenantStatusActive).
		Get(&tenants); err != nil {
		return nil, err
	}
	return BuildTenantHealthBoard(tenants, time.Now()), nil
}

// BuildTenantHealthBoard is the pure aggregation behind HealthBoard.
func BuildTenantHealthBoard(tenants []models.Tenant, now time.Time) *TenantHealthBoard {
	board := &TenantHealthBoard{
		StaleAfterSec: int64(TenantHealthStaleAfter / time.Second),
		ByStatus: map[string]int64{
			TenantHealthOK:      0,
			TenantHealthWarn:    0,
			TenantHealthFail:    0,
			TenantHealthUnknown: 0,
		},
		ByIssue: map[string]int64{},
		Items:   []TenantHealthBoardItem{},
	}
	staleBefore := now.Add(-TenantHealthStaleAfter)
	for i := range tenants {
		t := &tenants[i]
		status := strings.ToLower(strings.TrimSpace(t.HealthStatus))
		switch status {
		case TenantHealthOK, TenantHealthWarn, TenantHealthFail:
		default:
			status = TenantHealthUnknown
		}
		board.ByStatus[status]++

		stale := false
		if t.HealthCheckedAt == nil {
			board.NeverChecked++
			stale = true
		} else {
			board.Checked++
			if board.LastCheckedAt == nil || t.HealthCheckedAt.After(*board.LastCheckedAt) {
				checked := *t.HealthCheckedAt
				board.LastCheckedAt = &checked
			}
			stale = t.HealthCheckedAt.Before(staleBefore)
		}
		if stale {
			board.Stale++
		}

		raw := parseHealthIssues(t.HealthIssues)
		unique := make([]string, 0, len(raw))
		seen := make(map[string]bool, len(raw))
		for _, issue := range raw {
			issue = strings.TrimSpace(issue)
			if issue == "" || seen[issue] {
				continue
			}
			seen[issue] = true
			unique = append(unique, issue)
			board.ByIssue[issue]++
		}

		if status != TenantHealthWarn && status != TenantHealthFail {
			continue
		}
		board.Abnormal++
		board.Items = append(board.Items, TenantHealthBoardItem{
			ID:              t.ID,
			Code:            t.Code,
			Name:            t.Name,
			HealthStatus:    status,
			Issues:          unique,
			LastPingOK:      t.LastPingOK,
			LastPingMs:      t.LastPingMs,
			HealthCheckedAt: t.HealthCheckedAt,
			Maintenance:     t.Maintenance,
			Stale:           stale,
		})
	}

	// fail before warn, then more issues first, then code for a stable order.
	sort.SliceStable(board.Items, func(i, j int) bool {
		a, b := board.Items[i], board.Items[j]
		if a.HealthStatus != b.HealthStatus {
			return a.HealthStatus == TenantHealthFail
		}
		if len(a.Issues) != len(b.Issues) {
			return len(a.Issues) > len(b.Issues)
		}
		return a.Code < b.Code
	})
	if len(board.Items) > tenantHealthBoardMaxItems {
		board.Items = board.Items[:tenantHealthBoardMaxItems]
		board.Truncated = true
	}
	return board
}
