package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	appfacades "goravel/app/facades"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

// DemoActivityUpsert is the create/update payload for the schedule demo API.
type DemoActivityUpsert struct {
	Title         string `json:"title"`
	ScheduleType  string `json:"schedule_type"`
	StartAt       string `json:"start_at"`
	EndAt         string `json:"end_at"`
	DailyStart    string `json:"daily_start"`
	DailyEnd      string `json:"daily_end"`
	Weekdays      string `json:"weekdays"`
	MonthDays     string `json:"month_days"`
	MonthDayStart uint8  `json:"month_day_start"`
	MonthDayEnd   uint8  `json:"month_day_end"`
	YearStart     string `json:"year_start"`
	YearEnd       string `json:"year_end"`
	Timezone      string `json:"timezone"`
	Enabled       *bool  `json:"enabled"`
}

type DemoActivityService struct {
	ctx context.Context
}

func NewDemoActivityService(ctx context.Context) *DemoActivityService {
	return &DemoActivityService{ctx: ctx}
}

func (s *DemoActivityService) List(limit int) ([]models.DemoActivity, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	list, _, err := s.GetList(1, limit, "")
	return list, err
}

// GetList returns a page of demo activities for admin CRUD.
func (s *DemoActivityService) GetList(page, pageSize int, title string) ([]models.DemoActivity, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := appfacades.OrmQuery(s.ctx).Model(&models.DemoActivity{})
	title = strings.TrimSpace(title)
	if title != "" {
		query = query.Where("title LIKE ?", "%"+title+"%")
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	listQuery := appfacades.OrmQuery(s.ctx).Model(&models.DemoActivity{})
	if title != "" {
		listQuery = listQuery.Where("title LIKE ?", "%"+title+"%")
	}
	var list []models.DemoActivity
	err = listQuery.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list)
	return list, total, err
}

func (s *DemoActivityService) GetByID(id uint) (*models.DemoActivity, error) {
	var row models.DemoActivity
	if err := appfacades.OrmQuery(s.ctx).Model(&models.DemoActivity{}).Where("id", id).First(&row); err != nil || row.ID == 0 {
		return nil, apperrors.ErrDemoActivityNotFound
	}
	return &row, nil
}

func (s *DemoActivityService) Create(in DemoActivityUpsert) (*models.DemoActivity, error) {
	row, err := s.buildFromUpsert(in, nil)
	if err != nil {
		return nil, err
	}
	row.Status = DemoActivityDesiredStatus(row, time.Now().UTC())
	if err := appfacades.OrmQuery(s.ctx).Create(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *DemoActivityService) Update(id uint, in DemoActivityUpsert) (*models.DemoActivity, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	row, err := s.buildFromUpsert(in, existing)
	if err != nil {
		return nil, err
	}
	row.ID = existing.ID
	row.Status = DemoActivityDesiredStatus(row, time.Now().UTC())
	if _, err := appfacades.OrmQuery(s.ctx).Model(&models.DemoActivity{}).Where("id", id).Update(map[string]any{
		"title":           row.Title,
		"schedule_type":   row.ScheduleType,
		"start_at":        row.StartAt,
		"end_at":          row.EndAt,
		"daily_start":     row.DailyStart,
		"daily_end":       row.DailyEnd,
		"weekdays":        row.Weekdays,
		"month_days":      row.MonthDays,
		"month_day_start": row.MonthDayStart,
		"month_day_end":   row.MonthDayEnd,
		"year_start":      row.YearStart,
		"year_end":        row.YearEnd,
		"timezone":        row.Timezone,
		"enabled":         row.Enabled,
		"status":          row.Status,
	}); err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *DemoActivityService) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	_, err := appfacades.OrmQuery(s.ctx).Model(&models.DemoActivity{}).Where("id", id).Delete(&models.DemoActivity{})
	return err
}

// SyncStatuses flips status for enabled rows; used by activity:sync-status.
func (s *DemoActivityService) SyncStatuses() (updated int, err error) {
	var list []models.DemoActivity
	if err = appfacades.OrmQuery(s.ctx).Model(&models.DemoActivity{}).
		Where("enabled", true).
		Find(&list); err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	for i := range list {
		want := DemoActivityDesiredStatus(&list[i], now)
		if want == list[i].Status {
			continue
		}
		if _, uerr := appfacades.OrmQuery(s.ctx).Model(&models.DemoActivity{}).
			Where("id", list[i].ID).
			Where("status", list[i].Status).
			Update("status", want); uerr != nil {
			return updated, uerr
		}
		updated++
	}
	return updated, nil
}

// CheckActive compares stored status vs live IsActive for demo clarity.
func (s *DemoActivityService) CheckActive(id uint) (map[string]any, error) {
	row, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	live := DemoActivityIsActive(row, now)
	desired := DemoActivityDesiredStatus(row, now)
	return map[string]any{
		"activity":       DemoActivityToJSON(row),
		"now_utc":        now.Format(time.RFC3339),
		"stored_status":  row.Status,
		"desired_status": desired,
		"is_active_live": live,
		"status_lag_hint": "stored_status is updated by activity:sync-status; is_active_live is wall-clock truth",
	}, nil
}

func (s *DemoActivityService) buildFromUpsert(in DemoActivityUpsert, existing *models.DemoActivity) (*models.DemoActivity, error) {
	row := &models.DemoActivity{
		Title:         strings.TrimSpace(in.Title),
		ScheduleType:  strings.TrimSpace(in.ScheduleType),
		DailyStart:    strings.TrimSpace(in.DailyStart),
		DailyEnd:      strings.TrimSpace(in.DailyEnd),
		Weekdays:      strings.TrimSpace(in.Weekdays),
		MonthDays:     strings.TrimSpace(in.MonthDays),
		MonthDayStart: in.MonthDayStart,
		MonthDayEnd:   in.MonthDayEnd,
		YearStart:     strings.TrimSpace(in.YearStart),
		YearEnd:       strings.TrimSpace(in.YearEnd),
		Timezone:      strings.TrimSpace(in.Timezone),
		Enabled:       true,
	}
	if existing != nil {
		row.Title = firstNonEmpty(row.Title, existing.Title)
		row.ScheduleType = firstNonEmpty(row.ScheduleType, existing.ScheduleType)
		row.DailyStart = coalesceString(in.DailyStart, existing.DailyStart)
		row.DailyEnd = coalesceString(in.DailyEnd, existing.DailyEnd)
		row.Weekdays = coalesceString(in.Weekdays, existing.Weekdays)
		row.MonthDays = coalesceString(in.MonthDays, existing.MonthDays)
		if in.MonthDayStart == 0 && in.MonthDayEnd == 0 && in.MonthDays == "" {
			row.MonthDayStart = existing.MonthDayStart
			row.MonthDayEnd = existing.MonthDayEnd
		}
		row.YearStart = coalesceString(in.YearStart, existing.YearStart)
		row.YearEnd = coalesceString(in.YearEnd, existing.YearEnd)
		row.Timezone = firstNonEmpty(row.Timezone, existing.Timezone)
		row.Enabled = existing.Enabled
		row.StartAt = existing.StartAt
		row.EndAt = existing.EndAt
	}
	if in.Enabled != nil {
		row.Enabled = *in.Enabled
	}
	if row.Title == "" {
		return nil, apperrors.ErrDemoActivityInvalid.WithMessage("title is required")
	}
	if row.ScheduleType == "" {
		row.ScheduleType = models.DemoActivityScheduleOnce
	}
	switch row.ScheduleType {
	case models.DemoActivityScheduleOnce,
		models.DemoActivityScheduleDaily,
		models.DemoActivityScheduleWeekly,
		models.DemoActivityScheduleMonthly,
		models.DemoActivityScheduleYearly:
	default:
		return nil, apperrors.ErrDemoActivityInvalid.WithMessage("schedule_type must be once|daily|weekly|monthly|yearly")
	}
	if row.Timezone == "" {
		row.Timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(row.Timezone); err != nil {
		return nil, apperrors.ErrDemoActivityInvalid.WithMessage("invalid timezone")
	}

	clearRecurring := func() {
		row.Weekdays = ""
		row.MonthDays = ""
		row.MonthDayStart = 0
		row.MonthDayEnd = 0
		row.YearStart = ""
		row.YearEnd = ""
	}

	switch row.ScheduleType {
	case models.DemoActivityScheduleOnce:
		startAt, err := parseOptionalRFC3339OrDatetime(in.StartAt)
		if err != nil {
			return nil, apperrors.ErrDemoActivityInvalid.WithMessage("invalid start_at")
		}
		endAt, err := parseOptionalRFC3339OrDatetime(in.EndAt)
		if err != nil {
			return nil, apperrors.ErrDemoActivityInvalid.WithMessage("invalid end_at")
		}
		if startAt == nil && existing != nil {
			startAt = existing.StartAt
		}
		if endAt == nil && existing != nil {
			endAt = existing.EndAt
		}
		if startAt == nil || endAt == nil {
			return nil, apperrors.ErrDemoActivityInvalid.WithMessage("once requires start_at and end_at")
		}
		if !endAt.After(*startAt) {
			return nil, apperrors.ErrDemoActivityInvalid.WithMessage("end_at must be after start_at")
		}
		row.StartAt = startAt
		row.EndAt = endAt
		row.DailyStart = ""
		row.DailyEnd = ""
		clearRecurring()

	case models.DemoActivityScheduleDaily:
		if err := requireDailyWindow(row.DailyStart, row.DailyEnd); err != nil {
			return nil, err
		}
		row.StartAt = nil
		row.EndAt = nil
		clearRecurring()

	case models.DemoActivityScheduleWeekly:
		days, err := parseIntCSV(row.Weekdays, 1, 7)
		if err != nil || len(days) == 0 {
			return nil, apperrors.ErrDemoActivityInvalid.WithMessage("weekly requires weekdays csv 1-7 (Mon-Sun)")
		}
		if err := requireDailyWindow(row.DailyStart, row.DailyEnd); err != nil {
			return nil, err
		}
		row.Weekdays = joinIntCSV(days)
		row.StartAt = nil
		row.EndAt = nil
		row.MonthDays = ""
		row.MonthDayStart = 0
		row.MonthDayEnd = 0
		row.YearStart = ""
		row.YearEnd = ""

	case models.DemoActivityScheduleMonthly:
		row.StartAt = nil
		row.EndAt = nil
		row.Weekdays = ""
		row.YearStart = ""
		row.YearEnd = ""
		if row.MonthDays != "" {
			days, err := parseIntCSV(row.MonthDays, 1, 31)
			if err != nil || len(days) == 0 {
				return nil, apperrors.ErrDemoActivityInvalid.WithMessage("month_days must be csv 1-31")
			}
			row.MonthDays = joinIntCSV(days)
			row.MonthDayStart = 0
			row.MonthDayEnd = 0
			if err := requireDailyWindow(row.DailyStart, row.DailyEnd); err != nil {
				return nil, err
			}
		} else if row.MonthDayStart >= 1 && row.MonthDayEnd >= 1 && row.MonthDayStart <= 31 && row.MonthDayEnd <= 31 {
			// range mode; daily window optional (empty = whole day)
			if row.DailyStart != "" || row.DailyEnd != "" {
				if err := requireDailyWindow(row.DailyStart, row.DailyEnd); err != nil {
					return nil, err
				}
			}
			row.MonthDays = ""
		} else {
			return nil, apperrors.ErrDemoActivityInvalid.WithMessage("monthly requires month_days or month_day_start/end")
		}

	case models.DemoActivityScheduleYearly:
		if !validMMDD(row.YearStart) || !validMMDD(row.YearEnd) {
			return nil, apperrors.ErrDemoActivityInvalid.WithMessage("yearly requires year_start/year_end as MM-DD")
		}
		if row.DailyStart != "" || row.DailyEnd != "" {
			if err := requireDailyWindow(row.DailyStart, row.DailyEnd); err != nil {
				return nil, err
			}
		}
		row.StartAt = nil
		row.EndAt = nil
		row.Weekdays = ""
		row.MonthDays = ""
		row.MonthDayStart = 0
		row.MonthDayEnd = 0
	}
	return row, nil
}

func requireDailyWindow(start, end string) error {
	if start == "" || end == "" {
		return apperrors.ErrDemoActivityInvalid.WithMessage("daily_start and daily_end (HH:MM) are required")
	}
	if !validHHMM(start) || !validHHMM(end) {
		return apperrors.ErrDemoActivityInvalid.WithMessage("daily_start/daily_end must be HH:MM")
	}
	return nil
}

func DemoActivityDesiredStatus(a *models.DemoActivity, nowUTC time.Time) uint8 {
	if a == nil || !a.Enabled {
		return models.DemoActivityStatusEnded
	}
	loc, err := time.LoadLocation(firstNonEmpty(a.Timezone, "Asia/Shanghai"))
	if err != nil {
		loc = time.UTC
	}
	now := nowUTC.In(loc)

	switch a.ScheduleType {
	case models.DemoActivityScheduleOnce:
		if a.StartAt == nil || a.EndAt == nil {
			return models.DemoActivityStatusPending
		}
		if nowUTC.Before(a.StartAt.UTC()) {
			return models.DemoActivityStatusPending
		}
		if nowUTC.Before(a.EndAt.UTC()) {
			return models.DemoActivityStatusRunning
		}
		return models.DemoActivityStatusEnded

	case models.DemoActivityScheduleDaily:
		if inLocalTimeWindow(now, a.DailyStart, a.DailyEnd) {
			return models.DemoActivityStatusRunning
		}
		return models.DemoActivityStatusPending

	case models.DemoActivityScheduleWeekly:
		if !csvContainsInt(a.Weekdays, isoWeekday(now)) {
			return models.DemoActivityStatusPending
		}
		if inLocalTimeWindow(now, a.DailyStart, a.DailyEnd) {
			return models.DemoActivityStatusRunning
		}
		return models.DemoActivityStatusPending

	case models.DemoActivityScheduleMonthly:
		day := now.Day()
		match := false
		if a.MonthDays != "" {
			match = csvContainsInt(a.MonthDays, day)
		} else {
			match = inMonthDayRange(day, int(a.MonthDayStart), int(a.MonthDayEnd))
		}
		if !match {
			return models.DemoActivityStatusPending
		}
		if a.DailyStart == "" && a.DailyEnd == "" {
			return models.DemoActivityStatusRunning
		}
		if inLocalTimeWindow(now, a.DailyStart, a.DailyEnd) {
			return models.DemoActivityStatusRunning
		}
		return models.DemoActivityStatusPending

	case models.DemoActivityScheduleYearly:
		if !inYearMDRange(now, a.YearStart, a.YearEnd) {
			return models.DemoActivityStatusPending
		}
		if a.DailyStart == "" && a.DailyEnd == "" {
			return models.DemoActivityStatusRunning
		}
		if inLocalTimeWindow(now, a.DailyStart, a.DailyEnd) {
			return models.DemoActivityStatusRunning
		}
		return models.DemoActivityStatusPending

	default:
		return models.DemoActivityStatusPending
	}
}

func DemoActivityIsActive(a *models.DemoActivity, nowUTC time.Time) bool {
	return DemoActivityDesiredStatus(a, nowUTC) == models.DemoActivityStatusRunning
}

func DemoActivityToJSON(a *models.DemoActivity) map[string]any {
	if a == nil {
		return nil
	}
	out := map[string]any{
		"id":              a.ID,
		"title":           a.Title,
		"schedule_type":   a.ScheduleType,
		"daily_start":     a.DailyStart,
		"daily_end":       a.DailyEnd,
		"weekdays":        a.Weekdays,
		"month_days":      a.MonthDays,
		"month_day_start": a.MonthDayStart,
		"month_day_end":   a.MonthDayEnd,
		"year_start":      a.YearStart,
		"year_end":        a.YearEnd,
		"timezone":        a.Timezone,
		"status":          a.Status,
		"enabled":         a.Enabled,
		"created_at":      a.CreatedAt,
		"updated_at":      a.UpdatedAt,
	}
	if a.StartAt != nil {
		out["start_at"] = a.StartAt.UTC().Format(time.RFC3339)
	} else {
		out["start_at"] = nil
	}
	if a.EndAt != nil {
		out["end_at"] = a.EndAt.UTC().Format(time.RFC3339)
	} else {
		out["end_at"] = nil
	}
	return out
}

func inLocalTimeWindow(now time.Time, start, end string) bool {
	if start == "" || end == "" || start == end {
		return false
	}
	tod := now.Format("15:04")
	if start < end {
		return tod >= start && tod < end
	}
	return tod >= start || tod < end
}

func isoWeekday(t time.Time) int {
	// Monday=1 ... Sunday=7
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

func inMonthDayRange(day, start, end int) bool {
	if start < 1 || end < 1 {
		return false
	}
	if start <= end {
		return day >= start && day <= end
	}
	// wrap within month rare; treat as start..31 or 1..end
	return day >= start || day <= end
}

func inYearMDRange(now time.Time, startMD, endMD string) bool {
	cur := now.Format("01-02")
	if startMD == "" || endMD == "" {
		return false
	}
	if startMD <= endMD {
		return cur >= startMD && cur <= endMD
	}
	// wraps year e.g. 12-20 .. 01-10
	return cur >= startMD || cur <= endMD
}

func validMMDD(v string) bool {
	_, err := time.Parse("01-02", v)
	return err == nil
}

func parseIntCSV(raw string, min, max int) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty")
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	seen := map[int]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil {
			return nil, err
		}
		if n < min || n > max {
			return nil, fmt.Errorf("out of range")
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out, nil
}

func joinIntCSV(vals []int) string {
	parts := make([]string, 0, len(vals))
	for _, v := range vals {
		parts = append(parts, fmt.Sprintf("%d", v))
	}
	return strings.Join(parts, ",")
}

func csvContainsInt(raw string, want int) bool {
	vals, err := parseIntCSV(raw, 1, 31)
	if err != nil {
		return false
	}
	for _, v := range vals {
		if v == want {
			return true
		}
	}
	return false
}

func parseOptionalRFC3339OrDatetime(v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, v, time.UTC)
		if err == nil {
			return &t, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("parse time: %w", lastErr)
}

func validHHMM(v string) bool {
	_, err := time.Parse("15:04", v)
	return err == nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func coalesceString(input, fallback string) string {
	if input == "" {
		return fallback
	}
	return strings.TrimSpace(input)
}
