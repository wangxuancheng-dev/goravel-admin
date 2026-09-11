package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goravel/app/constants"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/utils"
)

type DashboardService interface {
	GetCountSnapshot() map[string]any
	GetUserAccessSource() []map[string]any
	GetWeeklyUserActivity() []map[string]any
	GetMonthlyOperations() []map[string]any
	GetRecentActivities() []map[string]any
	GetOnlineAdminCount() int64
	CollectDashboardData() map[string]any
}

type DashboardServiceImpl struct {
	ctx context.Context
}

func NewDashboardService(ctx context.Context) DashboardService {
	return &DashboardServiceImpl{ctx: ctx}
}

func (s *DashboardServiceImpl) GetCountSnapshot() map[string]any {
	countData := s.getEntityCounts()

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	todayVisits, _ := appfacades.OrmQuery(s.ctx).Model(&models.LoginLog{}).
		Where("created_at >= ?", todayStart).
		Where("created_at <= ?", todayEnd).
		Where("status", 1).
		Count()

	orderCountInYear, _ := NewOrderService(s.ctx).GetOrdersCountInYear()

	return map[string]any{
		"admin_count":         countData["admins"],
		"role_count":          countData["roles"],
		"menu_count":          countData["menus"],
		"today_visits":        todayVisits,
		"online_admins":       s.GetOnlineAdminCount(),
		"order_count_in_year": orderCountInYear,
	}
}

func (s *DashboardServiceImpl) GetUserAccessSource() []map[string]any {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	var loginLogs []models.LoginLog
	_ = appfacades.OrmQuery(s.ctx).Model(&models.LoginLog{}).
		Where("created_at >= ?", thirtyDaysAgo).
		Where("status", 1).
		Get(&loginLogs)

	deviceStats := make(map[string]int64)
	for _, log := range loginLogs {
		deviceStats[parseDeviceType(log.UserAgent)]++
	}

	return []map[string]any{
		{"name": "桌面端", "value": deviceStats["desktop"]},
		{"name": "移动端", "value": deviceStats["mobile"]},
		{"name": "平板端", "value": deviceStats["tablet"]},
		{"name": "其他", "value": deviceStats["other"]},
	}
}

func (s *DashboardServiceImpl) GetWeeklyUserActivity() []map[string]any {
	now := time.Now()
	weeklyData := make([]map[string]any, 7)

	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStr := utils.FormatDate(date)

		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		visitCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{}).
			Where("created_at >= ?", startOfDay).
			Where("created_at <= ?", endOfDay).
			Where("status", 1).
			Count()

		var uniqueAdmins []uint
		_ = appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{}).
			Where("created_at >= ?", startOfDay).
			Where("created_at <= ?", endOfDay).
			Where("status", 1).
			Select("DISTINCT admin_id").
			Pluck("admin_id", &uniqueAdmins)

		weeklyData[6-i] = map[string]any{
			"date":   dateStr,
			"visits": visitCount,
			"users":  int64(len(uniqueAdmins)),
		}
	}
	return weeklyData
}

func (s *DashboardServiceImpl) GetMonthlyOperations() []map[string]any {
	now := time.Now()
	monthlyData := make([]map[string]any, 12)

	for i := 11; i >= 0; i-- {
		date := now.AddDate(0, -i, 0)
		monthStr := date.Format("2006-01")

		startOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
		var endOfMonth time.Time
		if i == 0 {
			endOfMonth = now
		} else {
			endOfMonth = startOfMonth.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}

		operationCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{}).
			Where("created_at >= ?", startOfMonth).
			Where("created_at <= ?", endOfMonth).
			Where("status", 1).
			Count()

		monthlyData[11-i] = map[string]any{
			"month": monthStr,
			"count": operationCount,
		}
	}
	return monthlyData
}

func (s *DashboardServiceImpl) GetRecentActivities() []map[string]any {
	var logs []models.OperationLog
	_ = appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{}).
		With("Admin").
		Order("id desc").
		Limit(10).
		Get(&logs)

	activities := make([]map[string]any, 0, len(logs))
	for _, log := range logs {
		adminName := "未知用户"
		if log.Admin.ID > 0 {
			adminName = log.Admin.Nickname
			if adminName == "" {
				adminName = log.Admin.Username
			}
		}

		statusText := "成功"
		statusType := "success"
		if log.Status == 0 {
			statusText = "失败"
			statusType = "danger"
		}

		timeAgo := "未知"
		if log.CreatedAt != nil {
			timeStr := log.CreatedAt.ToDateTimeString()
			if t, err := utils.ParseDateTime(timeStr); err == nil {
				timeAgo = formatTimeAgo(t)
			}
		}

		activities = append(activities, map[string]any{
			"user":        adminName,
			"action":      log.Title,
			"time":        timeAgo,
			"status":      statusText,
			"type":        statusType,
			"avatarColor": avatarColor(adminName),
		})
	}
	return activities
}

func (s *DashboardServiceImpl) GetOnlineAdminCount() int64 {
	onlineThreshold := time.Now().Add(-constants.OnlineAdminThreshold)
	count, _ := appfacades.OrmQuery(s.ctx).Model(&models.PersonalAccessToken{}).
		Where("tokenable_type", "admin").
		Where("last_used_at IS NOT NULL").
		Where("last_used_at >= ?", onlineThreshold).
		Count()
	return count
}

func (s *DashboardServiceImpl) CollectDashboardData() map[string]any {
	return map[string]any{
		"count":                s.getEntityCounts(),
		"user_access_source":   s.GetUserAccessSource(),
		"weekly_user_activity": s.GetWeeklyUserActivity(),
		"monthly_sales":        s.GetMonthlyOperations(),
		"online_admin_count":   s.GetOnlineAdminCount(),
	}
}

func (s *DashboardServiceImpl) getEntityCounts() map[string]any {
	adminCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.Admin{}).Count()
	roleCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.Role{}).Count()
	permissionCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.Permission{}).Count()
	menuCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.Menu{}).Count()
	departmentCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.Department{}).Count()
	dictionaryCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.Dictionary{}).Count()
	configCount, _ := appfacades.OrmQuery(s.ctx).Model(&models.Config{}).Count()

	return map[string]any{
		"admins":       adminCount,
		"roles":        roleCount,
		"permissions":  permissionCount,
		"menus":        menuCount,
		"departments":  departmentCount,
		"dictionaries": dictionaryCount,
		"configs":      configCount,
	}
}

func parseDeviceType(userAgent string) string {
	if userAgent == "" {
		return "other"
	}
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "ipad") || (strings.Contains(ua, "tablet") && !strings.Contains(ua, "mobile")) {
		return "tablet"
	}
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	}
	return "desktop"
}

func formatTimeAgo(t time.Time) string {
	duration := time.Now().Sub(t)
	if duration < time.Minute {
		return "刚刚"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%d分钟前", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(duration.Hours()))
	}
	return fmt.Sprintf("%d天前", int(duration.Hours()/24))
}

func avatarColor(name string) string {
	colors := []string{"#409EFF", "#67C23A", "#E6A23C", "#F56C6C", "#909399", "#606266"}
	if name == "" {
		return colors[0]
	}
	hash := 0
	for _, char := range name {
		hash = hash*31 + int(char)
	}
	return colors[hash%len(colors)]
}
