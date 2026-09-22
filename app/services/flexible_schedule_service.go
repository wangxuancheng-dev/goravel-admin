package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/goravel/framework/facades"
	"github.com/robfig/cron/v3"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/utils/errorlog"
)

const (
	FlexibleHandlerTenantBackup    = "tenant_backup"
	FlexibleHandlerScheduleTestLog = "schedule_test_log"

	flexibleScheduleOutputMax = 16 * 1024
	flexibleScheduleLockTTL   = 30 * time.Minute
)

// flexibleSoftSkip marks an intentional no-op (not a failure).
type flexibleSoftSkip struct{ msg string }

func (e *flexibleSoftSkip) Error() string { return e.msg }

func softSkip(msg string) error { return &flexibleSoftSkip{msg: msg} }

func isSoftSkip(err error) bool {
	_, ok := err.(*flexibleSoftSkip)
	return ok
}

// FlexibleHandlerMeta describes a whitelist handler for the admin UI.
type FlexibleHandlerMeta struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TenantAware bool   `json:"tenant_aware"`
}

var flexibleCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// ListFlexibleHandlers returns the code whitelist (not user-extensible at runtime).
func ListFlexibleHandlers() []FlexibleHandlerMeta {
	return []FlexibleHandlerMeta{
		{
			Key:         FlexibleHandlerTenantBackup,
			Name:        "Tenant backup",
			Description: "Runs tenant:backup (tenant_id>0) or tenant:backup-all (tenant_id=0). Honors TENANT_BACKUP_SCHEDULE_ENABLED.",
			TenantAware: true,
		},
		{
			Key:         FlexibleHandlerScheduleTestLog,
			Name:        "Schedule test log",
			Description: "Writes a heartbeat via app:schedule-test-log (safe for cron expression tests).",
			TenantAware: false,
		},
	}
}

func flexibleHandlerAllowed(handler string) bool {
	for _, h := range ListFlexibleHandlers() {
		if h.Key == handler {
			return true
		}
	}
	return false
}

// FlexibleScheduleService manages landlord flexible_schedules rows.
type FlexibleScheduleService struct {
	ctx context.Context
}

func NewFlexibleScheduleService(ctx context.Context) *FlexibleScheduleService {
	return &FlexibleScheduleService{ctx: ctx}
}

// EnsureDefaults inserts seed rows when the table is empty.
func (s *FlexibleScheduleService) EnsureDefaults() {
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return
	}
	n, err := q.Model(&models.FlexibleSchedule{}).Count()
	if err != nil {
		return
	}
	if n > 0 {
		return
	}
	at := strings.TrimSpace(facades.Config().GetString("tenancy.backup_schedule_at", "20:00"))
	row := models.FlexibleSchedule{
		Name:     "Tenant backup",
		Handler:  FlexibleHandlerTenantBackup,
		CronExpr: dailyAtToCron(at),
		Timezone: "UTC",
		TenantID: 0,
		Enabled:  true,
	}
	if err := q.Create(&row); err != nil {
		errorlog.Record(s.ctx, "flexible_schedule", "seed default failed", map[string]any{
			"error": err.Error(),
		}, "flexible schedule seed failed: %v", err)
	}
}

func dailyAtToCron(hhmm string) string {
	parts := strings.Split(strings.TrimSpace(hhmm), ":")
	if len(parts) != 2 {
		return "0 20 * * *"
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return "0 20 * * *"
	}
	return fmt.Sprintf("%d %d * * *", m, h)
}

// ValidateCronExpr checks a 5-field cron (or robfig descriptor).
func ValidateCronExpr(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return apperrors.ErrFlexibleScheduleCronInvalid
	}
	if _, err := flexibleCronParser.Parse(expr); err != nil {
		return apperrors.ErrFlexibleScheduleCronInvalid.WithError(err)
	}
	return nil
}

// PreviewNextRuns returns upcoming fire times in the given timezone (YYYY-MM-DD HH:mm:ss).
func PreviewNextRuns(expr, timezone string, from time.Time, count int) ([]string, error) {
	if err := ValidateCronExpr(expr); err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 5
	}
	if count > 20 {
		count = 20
	}
	loc := loadFlexibleTZ(timezone)
	sched, err := flexibleCronParser.Parse(strings.TrimSpace(expr))
	if err != nil {
		return nil, apperrors.ErrFlexibleScheduleCronInvalid.WithError(err)
	}
	t := from.In(loc)
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		t = sched.Next(t)
		out = append(out, t.Format("2006-01-02 15:04:05"))
	}
	return out, nil
}

func loadFlexibleTZ(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "UTC"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// CronMatchesMinute reports whether expr should fire in the minute of now (in timezone).
func CronMatchesMinute(expr, timezone string, now time.Time) (slot string, ok bool) {
	expr = strings.TrimSpace(expr)
	loc := loadFlexibleTZ(timezone)
	nowLocal := now.In(loc)
	sched, err := flexibleCronParser.Parse(expr)
	if err != nil {
		return "", false
	}
	minuteStart := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), nowLocal.Hour(), nowLocal.Minute(), 0, 0, loc)
	next := sched.Next(minuteStart.Add(-time.Second))
	if next.Before(minuteStart) || !next.Before(minuteStart.Add(time.Minute)) {
		return "", false
	}
	return minuteStart.Format("2006-01-02T15:04"), true
}

// List returns all flexible schedules (landlord).
func (s *FlexibleScheduleService) List() ([]models.FlexibleSchedule, error) {
	s.EnsureDefaults()
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return nil, apperrors.ErrParamsError
	}
	var rows []models.FlexibleSchedule
	if err := q.Model(&models.FlexibleSchedule{}).Order("id asc").Find(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// GetByID loads one row.
func (s *FlexibleScheduleService) GetByID(id uint) (*models.FlexibleSchedule, error) {
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return nil, apperrors.ErrParamsError
	}
	var row models.FlexibleSchedule
	if err := q.Where("id", id).First(&row); err != nil || row.ID == 0 {
		return nil, apperrors.ErrFlexibleScheduleNotFound
	}
	return &row, nil
}

// FlexibleScheduleInput is create/update payload.
type FlexibleScheduleInput struct {
	Name     string
	Handler  string
	CronExpr string
	Timezone string
	TenantID uint
	Enabled  *bool
}

func (s *FlexibleScheduleService) Create(in FlexibleScheduleInput) (*models.FlexibleSchedule, error) {
	s.EnsureDefaults()
	in.Handler = strings.TrimSpace(in.Handler)
	in.Name = strings.TrimSpace(in.Name)
	in.CronExpr = strings.TrimSpace(in.CronExpr)
	in.Timezone = strings.TrimSpace(in.Timezone)
	if in.Name == "" {
		in.Name = in.Handler
	}
	if !flexibleHandlerAllowed(in.Handler) {
		return nil, apperrors.ErrFlexibleScheduleHandlerInvalid
	}
	if err := ValidateCronExpr(in.CronExpr); err != nil {
		return nil, err
	}
	if in.Timezone == "" {
		in.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return nil, apperrors.ErrFlexibleScheduleTimezoneInvalid
	}
	if err := s.validateTenantID(in.TenantID); err != nil {
		return nil, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	row := models.FlexibleSchedule{
		Name:     in.Name,
		Handler:  in.Handler,
		CronExpr: in.CronExpr,
		Timezone: in.Timezone,
		TenantID: in.TenantID,
		Enabled:  enabled,
	}
	if err := appfacades.PlatformOrmQuery(s.ctx).Create(&row); err != nil {
		return nil, err
	}
	return &row, nil
}

// Update updates fields; setTenant controls whether tenant_id is written (including 0).
func (s *FlexibleScheduleService) Update(id uint, in FlexibleScheduleInput, setTenant bool) (*models.FlexibleSchedule, error) {
	row, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if name := strings.TrimSpace(in.Name); name != "" {
		updates["name"] = name
	}
	if h := strings.TrimSpace(in.Handler); h != "" {
		if !flexibleHandlerAllowed(h) {
			return nil, apperrors.ErrFlexibleScheduleHandlerInvalid
		}
		updates["handler"] = h
	}
	if expr := strings.TrimSpace(in.CronExpr); expr != "" {
		if err := ValidateCronExpr(expr); err != nil {
			return nil, err
		}
		updates["cron_expr"] = expr
	}
	if tz := strings.TrimSpace(in.Timezone); tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			return nil, apperrors.ErrFlexibleScheduleTimezoneInvalid
		}
		updates["timezone"] = tz
	}
	if in.Enabled != nil {
		updates["enabled"] = *in.Enabled
	}
	if setTenant {
		if err := s.validateTenantID(in.TenantID); err != nil {
			return nil, err
		}
		updates["tenant_id"] = in.TenantID
	}
	if len(updates) == 0 {
		return row, nil
	}
	if _, err := appfacades.PlatformOrmQuery(s.ctx).Model(row).Where("id", id).Update(updates); err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *FlexibleScheduleService) validateTenantID(tenantID uint) error {
	if tenantID == 0 {
		return nil
	}
	if !tenancy.Enabled() {
		return apperrors.ErrFlexibleScheduleTenantInvalid
	}
	var t models.Tenant
	if err := appfacades.PlatformOrmQuery(s.ctx).Where("id", tenantID).First(&t); err != nil || t.ID == 0 {
		return apperrors.ErrFlexibleScheduleTenantInvalid
	}
	return nil
}

func (s *FlexibleScheduleService) Delete(id uint) error {
	row, err := s.GetByID(id)
	if err != nil {
		return err
	}
	_, err = appfacades.PlatformOrmQuery(s.ctx).Delete(row)
	return err
}

// RunNow executes a row immediately (respects enabled unless force).
func (s *FlexibleScheduleService) RunNow(id uint, force bool) (*models.FlexibleSchedule, error) {
	row, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !force && !row.Enabled {
		return nil, apperrors.ErrFlexibleScheduleDisabled
	}
	runErr := s.executeRow(row, "")
	updated, getErr := s.GetByID(id)
	if getErr != nil {
		return nil, getErr
	}
	if runErr != nil {
		return updated, apperrors.WrapError(runErr, apperrors.ErrScheduleRunFailed.Code, apperrors.ErrScheduleRunFailed.Message)
	}
	return updated, nil
}

// TickDue runs all enabled rows whose cron matches the current minute.
func (s *FlexibleScheduleService) TickDue(now time.Time) (ran int, err error) {
	s.EnsureDefaults()
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return 0, nil
	}
	var rows []models.FlexibleSchedule
	if err := q.Model(&models.FlexibleSchedule{}).Where("enabled", true).Find(&rows); err != nil {
		// Table may not be migrated yet on landlord DB.
		return 0, nil
	}
	for i := range rows {
		row := &rows[i]
		slot, match := CronMatchesMinute(row.CronExpr, row.Timezone, now)
		if !match {
			continue
		}
		// Claim minute slot first so overlapping ticks cannot double-fire.
		if !s.claimSlot(row.ID, slot) {
			continue
		}
		_ = s.executeRow(row, slot)
		ran++
	}
	return ran, nil
}

// claimSlot atomically sets last_slot when it differs; false means already claimed.
func (s *FlexibleScheduleService) claimSlot(id uint, slot string) bool {
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil || slot == "" {
		return false
	}
	res, err := q.Model(&models.FlexibleSchedule{}).
		Where("id", id).
		Where("last_slot != ?", slot).
		Update(map[string]any{"last_slot": slot})
	if err != nil || res == nil {
		return false
	}
	return res.RowsAffected > 0
}

func (s *FlexibleScheduleService) executeRow(row *models.FlexibleSchedule, slot string) error {
	lockKey := fmt.Sprintf("flexible_schedule:run:%d", row.ID)
	if !facades.Cache().Add(lockKey, "1", flexibleScheduleLockTTL) {
		_ = s.persistResult(row, "skipped", "busy", "", 0, slot)
		return apperrors.ErrScheduleBusy
	}
	defer func() { _ = facades.Cache().Forget(lockKey) }()

	start := time.Now()
	out, runErr := invokeFlexibleHandler(s.ctx, row)
	dur := time.Since(start).Milliseconds()
	status := "success"
	errMsg := ""
	if isSoftSkip(runErr) {
		status = "skipped"
		errMsg = runErr.Error()
		runErr = nil
	} else if runErr != nil {
		status = "failed"
		errMsg = runErr.Error()
	}
	out = truncateFlexibleOutput(out, flexibleScheduleOutputMax)
	_ = s.persistResult(row, status, errMsg, out, dur, slot)
	return runErr
}

func truncateFlexibleOutput(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	if max <= 0 {
		return ""
	}
	return s[:max]
}

func (s *FlexibleScheduleService) persistResult(row *models.FlexibleSchedule, status, errMsg, output string, dur int64, slot string) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"last_run_at":      now,
		"last_status":      status,
		"last_error":       errMsg,
		"last_output":      output,
		"last_duration_ms": dur,
	}
	if slot != "" {
		updates["last_slot"] = slot
	}
	_, err := appfacades.PlatformOrmQuery(s.ctx).Model(&models.FlexibleSchedule{}).Where("id", row.ID).Update(updates)
	return err
}

func invokeFlexibleHandler(ctx context.Context, row *models.FlexibleSchedule) (string, error) {
	switch row.Handler {
	case FlexibleHandlerTenantBackup:
		return runFlexibleTenantBackup(ctx, row)
	case FlexibleHandlerScheduleTestLog:
		return runFlexibleScheduleTestLog()
	default:
		return "", apperrors.ErrFlexibleScheduleHandlerInvalid
	}
}

func runFlexibleTenantBackup(ctx context.Context, row *models.FlexibleSchedule) (string, error) {
	if !tenancy.Enabled() {
		return "", softSkip("tenancy disabled")
	}
	if !facades.Config().GetBool("tenancy.backup_schedule_enabled", false) {
		return "", softSkip("TENANT_BACKUP_SCHEDULE_ENABLED=false")
	}
	keep := facades.Config().GetInt("tenancy.backup_keep", 10)
	var cmd string
	if row.TenantID > 0 {
		var t models.Tenant
		if err := appfacades.PlatformOrmQuery(ctx).Where("id", row.TenantID).First(&t); err != nil || t.ID == 0 {
			return "", apperrors.ErrFlexibleScheduleTenantInvalid
		}
		cmd = "tenant:backup " + t.Code
		if keep > 0 {
			cmd += " --keep=" + strconv.Itoa(keep)
		}
	} else {
		cmd = "tenant:backup-all"
		if keep > 0 {
			cmd += " --keep=" + strconv.Itoa(keep)
		}
	}
	if err := facades.Artisan().Call(cmd); err != nil {
		return cmd, err
	}
	return "ok: " + cmd, nil
}

func runFlexibleScheduleTestLog() (string, error) {
	if err := facades.Artisan().Call("app:schedule-test-log"); err != nil {
		return "", err
	}
	return "ok: app:schedule-test-log", nil
}

// FlexibleScheduleToJSON enriches a row for API responses.
func FlexibleScheduleToJSON(row models.FlexibleSchedule) map[string]any {
	tenantCode := ""
	if row.TenantID > 0 {
		var t models.Tenant
		if err := appfacades.PlatformOrmQuery(nil).Where("id", row.TenantID).First(&t); err == nil && t.ID > 0 {
			tenantCode = t.Code
		}
	}
	next, _ := PreviewNextRuns(row.CronExpr, row.Timezone, time.Now().UTC(), 3)
	metaName := row.Handler
	for _, h := range ListFlexibleHandlers() {
		if h.Key == row.Handler {
			metaName = h.Name
			break
		}
	}
	return map[string]any{
		"id":               row.ID,
		"name":             row.Name,
		"handler":          row.Handler,
		"handler_name":     metaName,
		"cron_expr":        row.CronExpr,
		"timezone":         row.Timezone,
		"tenant_id":        row.TenantID,
		"tenant_code":      tenantCode,
		"enabled":          row.Enabled,
		"last_run_at":      formatFlexibleTime(row.LastRunAt),
		"last_status":      row.LastStatus,
		"last_error":       row.LastError,
		"last_output":      row.LastOutput,
		"last_duration_ms": row.LastDurationMs,
		"last_slot":        row.LastSlot,
		"next_runs":        next,
	}
}

func formatFlexibleTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}
