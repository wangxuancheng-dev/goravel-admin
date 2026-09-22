package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/goravel/framework/facades"
	"github.com/robfig/cron/v3"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
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

func isSoftSkip(err error) bool {
	_, ok := err.(*flexibleSoftSkip)
	return ok
}

// FlexibleHandlerMeta describes a whitelist handler for the admin UI.
type FlexibleHandlerMeta struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	TenantAware    bool   `json:"tenant_aware"`
	DefaultCron    string `json:"default_cron,omitempty"`
	DefaultPayload string `json:"default_payload,omitempty"`
	SeedDefault    bool   `json:"-"`
}

var flexibleCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// ListFlexibleHandlers returns the code whitelist (not user-extensible at runtime).
// Tenant backup is NOT listed: it stays on kernel DailyAt + TENANT_BACKUP_* env.
func ListFlexibleHandlers() []FlexibleHandlerMeta {
	return []FlexibleHandlerMeta{
		{
			Key:            FlexibleHandlerScheduleTestLog,
			Name:           "Schedule test log",
			Description:    "Safe demo: writes storage/logs/schedule-test-*.log via app:schedule-test-log.",
			TenantAware:    true,
			DefaultCron:    "*/5 * * * *",
			DefaultPayload: "{}",
			SeedDefault:    true,
		},
	}
}

type flexibleHandlerFn func(ctx context.Context, row *models.FlexibleSchedule) (string, error)

func flexibleHandlerRegistry() map[string]flexibleHandlerFn {
	return map[string]flexibleHandlerFn{
		FlexibleHandlerScheduleTestLog: func(_ context.Context, _ *models.FlexibleSchedule) (string, error) {
			return runFlexibleScheduleTestLog()
		},
	}
}

// FlexibleScheduleService manages landlord flexible_schedules rows.
type FlexibleScheduleService struct {
	ctx context.Context
}

func NewFlexibleScheduleService(ctx context.Context) *FlexibleScheduleService {
	return &FlexibleScheduleService{ctx: ctx}
}

// EnsureDefaults seeds whitelist SeedDefault rows for the current tenant.
func (s *FlexibleScheduleService) EnsureDefaults() {
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return
	}
	s.purgeLegacyTenantBackupRows()
	s.purgeSharedWhitelistRows()
	tid, err := s.scopeTenantID()
	if err != nil {
		return
	}
	s.ensureWhitelistRowsForTenant(tid)
}

// purgeLegacyTenantBackupRows removes older UI seeds; backup cadence is kernel-owned.
func (s *FlexibleScheduleService) purgeLegacyTenantBackupRows() {
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return
	}
	_, _ = q.Where("handler", FlexibleHandlerTenantBackup).Delete(&models.FlexibleSchedule{})
}

// purgeSharedWhitelistRows drops landlord-wide tenant_id=0 rows when tenancy is on.
func (s *FlexibleScheduleService) purgeSharedWhitelistRows() {
	if !tenancy.Enabled() {
		return
	}
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return
	}
	for _, h := range ListFlexibleHandlers() {
		_, _ = q.Where("handler", h.Key).Where("tenant_id", 0).Delete(&models.FlexibleSchedule{})
	}
}

func (s *FlexibleScheduleService) scopeTenantID() (uint, error) {
	if !tenancy.Enabled() {
		return 0, nil
	}
	tid, ok := helpers.GetTenantIDFromAnyContext(s.ctx)
	if !ok || tid == 0 {
		return 0, apperrors.ErrParamsError
	}
	return tid, nil
}

func (s *FlexibleScheduleService) ensureWhitelistRowsForTenant(tenantID uint) {
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return
	}
	for _, h := range ListFlexibleHandlers() {
		if !h.SeedDefault {
			continue
		}
		var existing models.FlexibleSchedule
		if err := q.Where("handler", h.Key).Where("tenant_id", tenantID).First(&existing); err == nil && existing.ID > 0 {
			continue
		}
		cronExpr := strings.TrimSpace(h.DefaultCron)
		if cronExpr == "" {
			cronExpr = "0 * * * *"
		}
		payload, err := normalizeFlexiblePayload(h.DefaultPayload)
		if err != nil {
			payload = "{}"
		}
		row := models.FlexibleSchedule{
			Name:     h.Name,
			Handler:  h.Key,
			CronExpr: cronExpr,
			Timezone: "UTC",
			TenantID: tenantID,
			Payload:  payload,
			Enabled:  true,
		}
		if err := q.Create(&row); err != nil {
			errorlog.Record(s.ctx, "flexible_schedule", "ensure whitelist row failed", map[string]any{
				"error":     err.Error(),
				"handler":   h.Key,
				"tenant_id": tenantID,
			}, "flexible schedule ensure row failed: %v", err)
		}
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

// List returns flexible schedules for the current tenant only.
func (s *FlexibleScheduleService) List() ([]models.FlexibleSchedule, error) {
	s.EnsureDefaults()
	tid, err := s.scopeTenantID()
	if err != nil {
		return nil, err
	}
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return nil, apperrors.ErrParamsError
	}
	var rows []models.FlexibleSchedule
	if err := q.Model(&models.FlexibleSchedule{}).Where("tenant_id", tid).Order("id asc").Find(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// GetByID loads one row owned by the current tenant.
func (s *FlexibleScheduleService) GetByID(id uint) (*models.FlexibleSchedule, error) {
	tid, err := s.scopeTenantID()
	if err != nil {
		return nil, err
	}
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return nil, apperrors.ErrParamsError
	}
	var row models.FlexibleSchedule
	if err := q.Where("id", id).Where("tenant_id", tid).First(&row); err != nil || row.ID == 0 {
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
	Payload  *string
	Enabled  *bool
}

// Create is disabled: rows are seeded per tenant from the whitelist.
func (s *FlexibleScheduleService) Create(_ FlexibleScheduleInput) (*models.FlexibleSchedule, error) {
	return nil, apperrors.ErrFlexibleScheduleMutationForbidden
}

// Update allows cron_expr / timezone / enabled / payload for the current tenant's row only.
func (s *FlexibleScheduleService) Update(id uint, in FlexibleScheduleInput, _ bool) (*models.FlexibleSchedule, error) {
	row, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
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
	if in.Payload != nil {
		normalized, err := normalizeFlexiblePayload(*in.Payload)
		if err != nil {
			return nil, apperrors.ErrParamsError.WithError(err)
		}
		updates["payload"] = normalized
	}
	if len(updates) == 0 {
		return row, nil
	}
	if _, err := appfacades.PlatformOrmQuery(s.ctx).Model(row).Where("id", id).Update(updates); err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

// Delete is disabled: seeded whitelist rows are not removed from tenant admin.
func (s *FlexibleScheduleService) Delete(_ uint) error {
	return apperrors.ErrFlexibleScheduleMutationForbidden
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
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return 0, nil
	}
	s.purgeLegacyTenantBackupRows()
	s.purgeSharedWhitelistRows()
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
	fn, ok := flexibleHandlerRegistry()[row.Handler]
	if !ok || fn == nil {
		return "", apperrors.ErrFlexibleScheduleHandlerInvalid
	}
	return fn(ctx, row)
}

func runFlexibleScheduleTestLog() (string, error) {
	if err := facades.Artisan().Call("app:schedule-test-log"); err != nil {
		return "", err
	}
	return "ok: app:schedule-test-log", nil
}

// normalizeFlexiblePayload accepts empty / object JSON and stores compact object text.
func normalizeFlexiblePayload(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return "", err
	}
	if obj == nil {
		obj = map[string]any{}
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FlexiblePayloadMap returns handler options for a row (empty object when unset).
func FlexiblePayloadMap(row *models.FlexibleSchedule) map[string]any {
	out := map[string]any{}
	if row == nil {
		return out
	}
	raw := strings.TrimSpace(row.Payload)
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		return map[string]any{}
	}
	return out
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
		"payload":          FlexiblePayloadMap(&row),
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
