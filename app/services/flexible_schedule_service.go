package services

import (
	"context"
	"encoding/json"
	"fmt"
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
	FlexibleHandlerScheduleTestLog = "schedule_test_log"
	FlexibleScheduleTimezoneUTC    = "UTC"

	flexibleScheduleOutputMax = 16 * 1024
	flexibleScheduleLockTTL   = 30 * time.Minute
)

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

// FlexibleScheduleService manages landlord flexible_schedules rows (UTC cron, per-tenant).
type FlexibleScheduleService struct {
	ctx context.Context
}

func NewFlexibleScheduleService(ctx context.Context) *FlexibleScheduleService {
	return &FlexibleScheduleService{ctx: ctx}
}

// EnsureDefaults seeds whitelist SeedDefault rows for the current tenant.
func (s *FlexibleScheduleService) EnsureDefaults() {
	if appfacades.PlatformOrmQuery(s.ctx) == nil {
		return
	}
	tid, err := s.scopeTenantID()
	if err != nil {
		return
	}
	s.ensureWhitelistRowsForTenant(tid)
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
			Timezone: FlexibleScheduleTimezoneUTC,
			TenantID: tenantID,
			Payload:  payload,
			Enabled:  true,
		}
		if next := ComputeNextRunAt(cronExpr, time.Now()); next != nil {
			row.NextRunAt = next
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

// PreviewNextRuns returns upcoming fire times in UTC (YYYY-MM-DD HH:mm:ss).
func PreviewNextRuns(expr string, from time.Time, count int) ([]string, error) {
	if err := ValidateCronExpr(expr); err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 5
	}
	if count > 20 {
		count = 20
	}
	sched, err := flexibleCronParser.Parse(strings.TrimSpace(expr))
	if err != nil {
		return nil, apperrors.ErrFlexibleScheduleCronInvalid.WithError(err)
	}
	t := from.UTC()
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		t = sched.Next(t)
		out = append(out, t.UTC().Format("2006-01-02 15:04:05"))
	}
	return out, nil
}

// CronMatchesMinute reports whether expr should fire in the UTC minute of now.
func CronMatchesMinute(expr string, now time.Time) (slot string, ok bool) {
	expr = strings.TrimSpace(expr)
	nowUTC := now.UTC()
	sched, err := flexibleCronParser.Parse(expr)
	if err != nil {
		return "", false
	}
	minuteStart := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), nowUTC.Hour(), nowUTC.Minute(), 0, 0, time.UTC)
	next := sched.Next(minuteStart.Add(-time.Second))
	if next.Before(minuteStart) || !next.Before(minuteStart.Add(time.Minute)) {
		return "", false
	}
	return minuteStart.Format("2006-01-02T15:04"), true
}

// ComputeNextRunAt returns the next fire time after `after` (UTC), or nil if expr is invalid.
func ComputeNextRunAt(expr string, after time.Time) *time.Time {
	expr = strings.TrimSpace(expr)
	sched, err := flexibleCronParser.Parse(expr)
	if err != nil {
		return nil
	}
	n := sched.Next(after.UTC())
	return &n
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

// FlexibleScheduleInput is update payload (create/delete disabled).
type FlexibleScheduleInput struct {
	CronExpr string
	Payload  *string
	Enabled  *bool
}

// Create is disabled: rows are seeded per tenant from the whitelist.
func (s *FlexibleScheduleService) Create(_ FlexibleScheduleInput) (*models.FlexibleSchedule, error) {
	return nil, apperrors.ErrFlexibleScheduleMutationForbidden
}

// Update allows cron_expr / enabled / payload for the current tenant's row only.
func (s *FlexibleScheduleService) Update(id uint, in FlexibleScheduleInput) (*models.FlexibleSchedule, error) {
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
		if next := ComputeNextRunAt(expr, time.Now()); next != nil {
			updates["next_run_at"] = *next
		}
	}
	if in.Enabled != nil {
		updates["enabled"] = *in.Enabled
		if *in.Enabled {
			expr := row.CronExpr
			if v, ok := updates["cron_expr"].(string); ok && strings.TrimSpace(v) != "" {
				expr = v
			}
			if next := ComputeNextRunAt(expr, time.Now()); next != nil {
				updates["next_run_at"] = *next
			}
		}
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

// TickDue runs enabled rows that are due (next_run_at <= now, or legacy null next_run_at).
// Caps executions per tick so large fleets do not stampede tenant connections.
func (s *FlexibleScheduleService) TickDue(now time.Time) (ran int, err error) {
	q := appfacades.PlatformOrmQuery(s.ctx)
	if q == nil {
		return 0, nil
	}
	nowUTC := now.UTC()
	limit := facades.Config().GetInt("tenancy.flex_schedule_tick_limit", 200)
	if limit < 1 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}

	var rows []models.FlexibleSchedule
	if err := q.Model(&models.FlexibleSchedule{}).
		Where("enabled", true).
		Where("(next_run_at IS NULL OR next_run_at <= ?)", nowUTC).
		Order("next_run_at asc").
		Order("id asc").
		Limit(limit).
		Find(&rows); err != nil {
		return 0, nil
	}
	for i := range rows {
		row := &rows[i]
		slot, match := CronMatchesMinute(row.CronExpr, nowUTC)
		if !match {
			// Stale next_run_at (or null): advance without executing.
			_ = s.advanceNextRunAt(row, nowUTC)
			continue
		}
		if !s.claimSlot(row.ID, slot) {
			_ = s.advanceNextRunAt(row, nowUTC)
			continue
		}
		_ = s.executeRow(row, slot)
		ran++
	}
	return ran, nil
}

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
	if runErr != nil {
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
	if next := ComputeNextRunAt(row.CronExpr, now); next != nil {
		updates["next_run_at"] = *next
	}
	_, err := appfacades.PlatformOrmQuery(s.ctx).Model(&models.FlexibleSchedule{}).Where("id", row.ID).Update(updates)
	return err
}

func (s *FlexibleScheduleService) advanceNextRunAt(row *models.FlexibleSchedule, after time.Time) error {
	next := ComputeNextRunAt(row.CronExpr, after)
	if next == nil {
		return nil
	}
	_, err := appfacades.PlatformOrmQuery(s.ctx).Model(&models.FlexibleSchedule{}).Where("id", row.ID).Update(map[string]any{
		"next_run_at": *next,
	})
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
	next, _ := PreviewNextRuns(row.CronExpr, time.Now().UTC(), 3)
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
		"timezone":         FlexibleScheduleTimezoneUTC,
		"tenant_id":        row.TenantID,
		"payload":          FlexiblePayloadMap(&row),
		"enabled":          row.Enabled,
		"last_run_at":      formatFlexibleTimeUTC(row.LastRunAt),
		"last_status":      row.LastStatus,
		"last_error":       row.LastError,
		"last_output":      row.LastOutput,
		"last_duration_ms": row.LastDurationMs,
		"last_slot":        row.LastSlot,
		"next_runs":        next,
	}
}

func formatFlexibleTimeUTC(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}
