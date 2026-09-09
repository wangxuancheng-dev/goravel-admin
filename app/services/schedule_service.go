package services

import (
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/color"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
)

const (
	scheduleLastRunCacheTTL = 30 * 24 * time.Hour
	scheduleRunLockTTL      = 10 * time.Minute
	scheduleOutputMaxBytes  = 32 * 1024
)

var (
	artisanOutputMu sync.Mutex
	ansiEscapeRe    = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\x1b\][^\x07]*\x07|\x1b.`)
	ptermLevelRe    = regexp.MustCompile(`(?i)^(INFO|ERROR|WARNING|WARN|SUCCESS|DEBUG)\s+(.*)$`)
)

// ScheduleTask is a registered schedule event for admin listing.
type ScheduleTask struct {
	ID                  string `json:"id"`
	Command             string `json:"command"`
	Cron                string `json:"cron"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	OnOneServer         bool   `json:"on_one_server"`
	SkipIfStillRunning  bool   `json:"skip_if_still_running"`
	DelayIfStillRunning bool   `json:"delay_if_still_running"`
	LastRunAt           string `json:"last_run_at"`
	LastStatus          string `json:"last_status"`
	LastError           string `json:"last_error"`
	LastOutput          string `json:"last_output"`
	LastDurationMs      int64  `json:"last_duration_ms"`
}

// ScheduleRunResult is returned after a manual run.
type ScheduleRunResult struct {
	Command    string `json:"command"`
	Status     string `json:"status"`
	Error      string `json:"error"`
	Output     string `json:"output"`
	DurationMs int64  `json:"duration_ms"`
	RunAt      string `json:"run_at"`
}

type scheduleLastRunCache struct {
	Status     string `json:"status"`
	Error      string `json:"error"`
	Output     string `json:"output"`
	DurationMs int64  `json:"duration_ms"`
	RunAt      string `json:"run_at"`
}

type ScheduleService interface {
	List() ([]ScheduleTask, error)
	Run(command string) (*ScheduleRunResult, error)
}

type ScheduleServiceImpl struct {
	ctx context.Context
}

func NewScheduleService(ctx context.Context) ScheduleService {
	return &ScheduleServiceImpl{ctx: ctx}
}

var scheduleCommandDescriptions = map[string]string{
	"app:clear-logs":                 "清理过期系统日志",
	"app:clear-chunks":               "清理过期上传分片",
	"db:analyze-stats":               "分析数据库表统计信息",
	"order:create-sharding-tables":   "创建下月订单分表",
	"payment:create-sharding-tables": "创建下月支付记录分表",
	"es:retry-outbox":                "重试 Elasticsearch outbox 积压",
	"app:schedule-test-log":          "定时任务心跳测试",
}

func (s *ScheduleServiceImpl) List() ([]ScheduleTask, error) {
	events := appfacades.Schedule().Events()
	tasks := make([]ScheduleTask, 0, len(events))
	seen := make(map[string]struct{}, len(events))

	for _, event := range events {
		if event == nil {
			continue
		}
		command := strings.TrimSpace(event.GetCommand())
		if command == "" {
			continue
		}
		cron := strings.TrimSpace(event.GetCron())
		id := scheduleTaskID(command, cron)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		task := ScheduleTask{
			ID:                  id,
			Command:             command,
			Cron:                cron,
			Name:                strings.TrimSpace(event.GetName()),
			Description:         scheduleCommandDescriptions[command],
			OnOneServer:         event.IsOnOneServer(),
			SkipIfStillRunning:  event.GetSkipIfStillRunning(),
			DelayIfStillRunning: event.GetDelayIfStillRunning(),
			LastStatus:          "never",
		}
		if cached := s.getLastRun(command); cached != nil {
			task.LastRunAt = cached.RunAt
			task.LastStatus = cached.Status
			task.LastError = cached.Error
			task.LastOutput = cached.Output
			task.LastDurationMs = cached.DurationMs
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *ScheduleServiceImpl) Run(command string) (*ScheduleRunResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, apperrors.ErrScheduleCommandRequired
	}
	if strings.ContainsAny(command, "\n\r;&|") || strings.Contains(command, "  ") {
		return nil, apperrors.ErrScheduleCommandNotAllowed
	}
	if !s.isScheduledCommand(command) {
		return nil, apperrors.ErrScheduleCommandNotAllowed
	}

	lockKey := scheduleRunLockKey(command)
	lock := facades.Cache().Lock(lockKey, scheduleRunLockTTL)
	if !lock.Get() {
		return nil, apperrors.ErrScheduleBusy
	}
	defer lock.Release()

	started := time.Now()
	runAt := started.Format("2006-01-02 15:04:05")
	output, callErr := callArtisanWithCapturedOutput(command)
	durationMs := time.Since(started).Milliseconds()

	result := &ScheduleRunResult{
		Command:    command,
		DurationMs: durationMs,
		RunAt:      runAt,
		Status:     "success",
		Output:     output,
	}
	if callErr != nil {
		result.Status = "failed"
		result.Error = callErr.Error()
	}

	s.putLastRun(command, &scheduleLastRunCache{
		Status:     result.Status,
		Error:      result.Error,
		Output:     result.Output,
		DurationMs: result.DurationMs,
		RunAt:      result.RunAt,
	})

	if callErr != nil {
		return result, apperrors.WrapError(callErr, apperrors.ErrScheduleRunFailed.Code, apperrors.ErrScheduleRunFailed.Message)
	}
	return result, nil
}

// callArtisanWithCapturedOutput captures ctx.Info/Error output.
// Goravel commands write via support/color (pterm), not os.Stdout, so stdout redirect is useless.
func callArtisanWithCapturedOutput(command string) (string, error) {
	artisanOutputMu.Lock()
	defer artisanOutputMu.Unlock()

	var callErr error
	raw := color.CaptureOutput(func(_ io.Writer) {
		callErr = facades.Artisan().Call(command)
	})
	return truncateScheduleOutput(normalizeScheduleOutput(raw)), callErr
}

func normalizeScheduleOutput(s string) string {
	s = ansiEscapeRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\u00a0", " ")

	lines := strings.Split(s, "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := ptermLevelRe.FindStringSubmatch(line); len(m) == 3 {
			level := strings.ToUpper(m[1])
			if level == "WARN" {
				level = "WARNING"
			}
			msg := strings.TrimSpace(m[2])
			if msg == "" {
				normalized = append(normalized, "["+level+"]")
			} else {
				normalized = append(normalized, "["+level+"] "+msg)
			}
			continue
		}
		normalized = append(normalized, line)
	}
	return strings.Join(normalized, "\n")
}

func truncateScheduleOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	if len(output) <= scheduleOutputMaxBytes {
		return output
	}
	truncated := output[:scheduleOutputMaxBytes]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated + "\n...[truncated]"
}

func (s *ScheduleServiceImpl) isScheduledCommand(command string) bool {
	for _, event := range appfacades.Schedule().Events() {
		if event == nil {
			continue
		}
		if strings.TrimSpace(event.GetCommand()) == command {
			return true
		}
	}
	return false
}

func (s *ScheduleServiceImpl) getLastRun(command string) *scheduleLastRunCache {
	raw := facades.Cache().GetString(scheduleLastRunCacheKey(command), "")
	if raw == "" {
		return nil
	}
	var cached scheduleLastRunCache
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil
	}
	return &cached
}

func (s *ScheduleServiceImpl) putLastRun(command string, cached *scheduleLastRunCache) {
	payload, err := json.Marshal(cached)
	if err != nil {
		return
	}
	_ = facades.Cache().Put(scheduleLastRunCacheKey(command), string(payload), scheduleLastRunCacheTTL)
}

func scheduleTaskID(command, cron string) string {
	if cron == "" {
		return command
	}
	return command + "|" + cron
}

func scheduleLastRunCacheKey(command string) string {
	return "schedule:last_run:" + command
}

func scheduleRunLockKey(command string) string {
	return "schedule:run:lock:" + command
}
