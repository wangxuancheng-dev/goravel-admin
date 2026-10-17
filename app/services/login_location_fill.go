package services

import (
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/rbac"
	"goravel/app/utils"
)

const (
	loginLocationFillLimit = 30
	loginLocationScanRows  = 500
	// LoginLocationHTTPPause spaces free-tier ip-api lookups (~45/min).
	LoginLocationHTTPPause = 1400 * time.Millisecond
)

// LocationFillResult is the outcome of a batched empty-location backfill.
type LocationFillResult struct {
	Updated  int64 `json:"updated"`
	LookedUp int   `json:"looked_up"`
	Failed   int   `json:"failed"`
	HasMore  bool  `json:"has_more"`
}

type locationQuery func() orm.Query

func uniqueIPs(ips []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	seen := make(map[string]struct{}, limit)
	out := make([]string, 0, limit)
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func collectEmptyIPs(q orm.Query, scan, limit int) ([]string, error) {
	if scan <= 0 {
		scan = loginLocationScanRows
	}
	var ips []string
	err := q.Where("(location = '' OR location IS NULL)").
		Where("ip <> ?", "").
		Order("id desc").
		Limit(scan).
		Pluck("ip", &ips)
	if err != nil {
		return nil, err
	}
	return uniqueIPs(ips, limit), nil
}

func writeLocation(query locationQuery, ip, location string, alsoID uint) (int64, error) {
	apply := func(q orm.Query) orm.Query {
		q = q.Where("ip", ip)
		if alsoID > 0 {
			return q.Where("(id = ? OR location = '' OR location IS NULL)", alsoID)
		}
		return q.Where("(location = '' OR location IS NULL)")
	}
	n, err := apply(query()).Count()
	if err != nil || n == 0 {
		return n, err
	}
	_, err = apply(query()).Update("location", location)
	return n, err
}

func refreshLocationByIP(query locationQuery, ip string, alsoID uint) (string, int64, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "", 0, apperrors.ErrIPLocationUnavailable
	}
	location := utils.GetIPLocation(ip)
	if location == "" {
		return "", 0, apperrors.ErrIPLocationUnavailable
	}
	updated, err := writeLocation(query, ip, location, alsoID)
	if err != nil {
		return "", 0, apperrors.ErrUpdateFailed.WithError(err)
	}
	return location, updated, nil
}

func fillEmptyLocations(query locationQuery, limit, hardCap int, pause time.Duration) (LocationFillResult, error) {
	if hardCap <= 0 {
		hardCap = loginLocationFillLimit
	}
	if limit <= 0 || limit > hardCap {
		limit = hardCap
	}
	scan := loginLocationScanRows
	if limit > scan {
		scan = limit
	}
	ips, err := collectEmptyIPs(query(), scan, limit)
	if err != nil {
		return LocationFillResult{}, apperrors.ErrQueryFailed.WithError(err)
	}
	var result LocationFillResult
	for i, ip := range ips {
		if pause > 0 && i > 0 {
			time.Sleep(pause)
		}
		result.LookedUp++
		_, updated, err := refreshLocationByIP(query, ip, 0)
		if err != nil {
			result.Failed++
			continue
		}
		result.Updated += updated
	}
	var moreIPs []string
	if err := query().Where("(location = '' OR location IS NULL)").Where("ip <> ?", "").Limit(1).Pluck("ip", &moreIPs); err == nil {
		result.HasMore = len(moreIPs) > 0
	}
	return result, nil
}

func (s *LoginLogServiceImpl) loginLogQuery() orm.Query {
	q := appfacades.OrmQuery(s.ctx).Model(&models.LoginLog{})
	return rbac.ApplyDataScope(s.ctx, q, rbac.DataScopeApplyOpts{Mode: rbac.DataScopeModeAdmin, AdminColumn: "admin_id"})
}

// RefreshLocation looks up one login log IP and writes that location onto the row and other empty rows with the same IP.
func (s *LoginLogServiceImpl) RefreshLocation(id uint) (string, int64, error) {
	log, err := s.GetByID(id, false)
	if err != nil {
		return "", 0, err
	}
	return refreshLocationByIP(s.loginLogQuery, log.IP, id)
}

// FillEmptyLocations backfills up to loginLocationFillLimit distinct empty IPs.
func (s *LoginLogServiceImpl) FillEmptyLocations(limit int) (LocationFillResult, error) {
	return fillEmptyLocations(s.loginLogQuery, limit, loginLocationFillLimit, LoginLocationHTTPPause)
}

// FillEmptyLocationsBatch is the CLI backfill. pause is the gap between lookups.
func (s *LoginLogServiceImpl) FillEmptyLocationsBatch(limit int, pause time.Duration) (LocationFillResult, error) {
	if limit <= 0 {
		limit = 200
	}
	return fillEmptyLocations(s.loginLogQuery, limit, limit, pause)
}

func platformLoginLogQuery() orm.Query {
	return appfacades.PlatformOrmQuery(nil).Model(&models.PlatformLoginLog{})
}

// RefreshPlatformLoginLocation looks up one platform login log and fills empty rows with the same IP.
func RefreshPlatformLoginLocation(id uint) (string, int64, error) {
	row, err := GetPlatformLoginLogByID(id)
	if err != nil {
		return "", 0, err
	}
	return refreshLocationByIP(platformLoginLogQuery, row.IP, id)
}

// FillPlatformEmptyLoginLocations backfills empty platform login locations.
// Pass hardCap via limit>loginLocationFillLimit (CLI); HTTP callers should pass limit<=0.
func FillPlatformEmptyLoginLocations(limit int, pause time.Duration) (LocationFillResult, error) {
	hardCap := loginLocationFillLimit
	if limit > loginLocationFillLimit {
		hardCap = limit
	}
	return fillEmptyLocations(platformLoginLogQuery, limit, hardCap, pause)
}
