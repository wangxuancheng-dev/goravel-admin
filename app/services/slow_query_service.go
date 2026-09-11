package services

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	appfacades "goravel/app/facades"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/models"
	"goravel/app/tenancy"
)

var (
	sqlLogLinePattern = regexp.MustCompile(`^\[(.*?)\]\s+\w+\.\w+:\s+\[(\d+(?:\.\d+)?)ms\]\s+\[rows:(-?\d+)\]\s+(.*)$`)
	singleQuoted      = regexp.MustCompile(`'[^']*'`)
	doubleQuoted      = regexp.MustCompile(`"[^"]*"`)
	numberLiteral     = regexp.MustCompile(`\b\d+\b`)
	spacePattern      = regexp.MustCompile(`\s+`)
)

const (
	slowQueryInsertBatchSize   = 20
	maxSlowSQLTextLength       = 4000
	maxNormalizedSQLTextLength = 4000
)

type SlowQueryTopItem struct {
	SQLHash       string  `json:"sql_hash"`
	NormalizedSQL string  `json:"normalized_sql"`
	Count         int     `json:"count"`
	AvgDurationMS float64 `json:"avg_duration_ms"`
	MaxDurationMS float64 `json:"max_duration_ms"`
	LastSeenAt    string  `json:"last_seen_at"`
	LatestSQL     string  `json:"latest_sql"`
	LatestTraceID string  `json:"latest_trace_id"`
}

type SlowQueryService interface {
	CollectFromLatestLog(minDurationMS float64) error
	GetTopN(hours, limit int, minDurationMS float64) ([]SlowQueryTopItem, error)
	GetByTraceID(traceID string, limit int) ([]models.SlowQueryLog, error)
}

type SlowQueryServiceImpl struct {
	ctx context.Context
}

func NewSlowQueryService(ctx context.Context) SlowQueryService {
	return &SlowQueryServiceImpl{ctx: ctx}
}

func (s *SlowQueryServiceImpl) CollectFromLatestLog(minDurationMS float64) error {
	// Shared process log mixes all tenants; ingesting into the caller's tenant DB would leak SQL.
	// Cron/platform tooling can collect later; tenant HTTP observability only reads its own table.
	if tenancy.Enabled() {
		return nil
	}
	if !appfacades.SchemaHasTable(s.ctx, "slow_query_logs") {
		return nil
	}

	logPath, err := latestAppLogPath()
	if err != nil || logPath == "" {
		return nil
	}

	fileInfo, err := os.Stat(logPath)
	if err != nil {
		return nil
	}

	offsetKey := "observability:slow_sql:offset:" + filepath.Base(logPath)
	offsetStr := facades.Cache().GetString(offsetKey, "0")
	offset, _ := strconv.ParseInt(offsetStr, 10, 64)
	if offset < 0 || offset > fileInfo.Size() {
		offset = 0
	}

	f, err := os.Open(logPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	if _, err := f.Seek(offset, 0); err != nil {
		return nil
	}

	scanner := bufio.NewScanner(f)
	const maxScanTokenSize = 1024 * 1024
	scanner.Buffer(make([]byte, 0, 128*1024), maxScanTokenSize)

	batch := make([]models.SlowQueryLog, 0, 64)
	for scanner.Scan() {
		line := scanner.Text()
		entry, ok := parseSlowQueryLine(line, minDurationMS)
		if !ok {
			continue
		}
		batch = append(batch, entry)
		if len(batch) >= slowQueryInsertBatchSize {
			_ = appfacades.OrmQuery(s.ctx).Create(&batch)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		_ = appfacades.OrmQuery(s.ctx).Create(&batch)
	}

	newOffset, _ := f.Seek(0, 1)
	_ = facades.Cache().Put(offsetKey, strconv.FormatInt(newOffset, 10), 60*60*24*30)
	return nil
}

func (s *SlowQueryServiceImpl) GetTopN(hours, limit int, minDurationMS float64) ([]SlowQueryTopItem, error) {
	if hours <= 0 {
		hours = 24
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	var logs []models.SlowQueryLog
	if err := appfacades.OrmQuery(s.ctx).
		Model(&models.SlowQueryLog{}).
		Where("created_at >= ?", cutoff).
		Where("duration_ms >= ?", minDurationMS).
		Order("created_at desc").
		Limit(5000).
		Get(&logs); err != nil {
		return nil, err
	}

	type agg struct {
		count int
		sum   float64
		max   float64
		last  string
		sql   string
		trace string
		norm  string
	}
	grouped := make(map[string]*agg)
	for _, item := range logs {
		key := item.SQLHash
		if key == "" {
			key = fmt.Sprintf("plain:%x", sha1.Sum([]byte(item.NormalizedSQL)))
		}
		if _, ok := grouped[key]; !ok {
			grouped[key] = &agg{norm: item.NormalizedSQL}
		}
		g := grouped[key]
		g.count++
		g.sum += item.DurationMS
		if item.DurationMS > g.max {
			g.max = item.DurationMS
		}
		lastSeen := ""
		if item.CreatedAt != nil {
			lastSeen = item.CreatedAt.ToDateTimeString()
		}
		// ToDateTimeString 为固定格式，可直接按字典序比较先后
		if lastSeen != "" && lastSeen > g.last {
			g.last = lastSeen
			g.sql = item.SQLText
			g.trace = item.TraceID
		}
	}

	results := make([]SlowQueryTopItem, 0, len(grouped))
	for hash, g := range grouped {
		avg := 0.0
		if g.count > 0 {
			avg = g.sum / float64(g.count)
		}
		results = append(results, SlowQueryTopItem{
			SQLHash:       hash,
			NormalizedSQL: g.norm,
			Count:         g.count,
			AvgDurationMS: avg,
			MaxDurationMS: g.max,
			LastSeenAt:    g.last,
			LatestSQL:     g.sql,
			LatestTraceID: g.trace,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].MaxDurationMS == results[j].MaxDurationMS {
			return results[i].Count > results[j].Count
		}
		return results[i].MaxDurationMS > results[j].MaxDurationMS
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (s *SlowQueryServiceImpl) GetByTraceID(traceID string, limit int) ([]models.SlowQueryLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var rows []models.SlowQueryLog
	err := appfacades.OrmQuery(s.ctx).
		Model(&models.SlowQueryLog{}).
		Where("trace_id", traceID).
		Order("id desc").
		Limit(limit).
		Get(&rows)
	return rows, err
}

func latestAppLogPath() (string, error) {
	paths, err := filepath.Glob(filepath.Join("storage", "logs", "goravel-*.log"))
	if err != nil || len(paths) == 0 {
		return "", err
	}
	sort.Slice(paths, func(i, j int) bool {
		fi, errI := os.Stat(paths[i])
		fj, errJ := os.Stat(paths[j])
		if errI != nil || errJ != nil {
			return paths[i] > paths[j]
		}
		return fi.ModTime().After(fj.ModTime())
	})
	return paths[0], nil
}

func parseSlowQueryLine(line string, minDurationMS float64) (models.SlowQueryLog, bool) {
	matches := sqlLogLinePattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) != 5 {
		return models.SlowQueryLog{}, false
	}

	durationMS, err := strconv.ParseFloat(matches[2], 64)
	if err != nil || durationMS < minDurationMS {
		return models.SlowQueryLog{}, false
	}
	sqlText := strings.TrimSpace(matches[4])
	if sqlText == "" {
		return models.SlowQueryLog{}, false
	}
	lower := strings.ToLower(sqlText)
	if strings.HasPrefix(lower, "show status") || strings.HasPrefix(lower, "show variables") {
		return models.SlowQueryLog{}, false
	}
	if strings.Contains(lower, "slow_query_logs") {
		// Exclude self-observability SQL to avoid noisy recursive records.
		return models.SlowQueryLog{}, false
	}

	rowsAffected, _ := strconv.ParseInt(matches[3], 10, 64)
	normalized := normalizeSQL(sqlText)
	hash := sha1.Sum([]byte(normalized))
	return models.SlowQueryLog{
		SQLText:       truncateText(sqlText, maxSlowSQLTextLength),
		NormalizedSQL: truncateText(normalized, maxNormalizedSQLTextLength),
		SQLHash:       hex.EncodeToString(hash[:]),
		DurationMS:    durationMS,
		RowsAffected:  rowsAffected,
		Source:        "gorm-log",
		OccurredAt:    matches[1],
	}, true
}

func normalizeSQL(sqlText string) string {
	normalized := singleQuoted.ReplaceAllString(sqlText, "?")
	normalized = doubleQuoted.ReplaceAllString(normalized, "?")
	normalized = numberLiteral.ReplaceAllString(normalized, "?")
	normalized = strings.ToLower(strings.TrimSpace(normalized))
	normalized = spacePattern.ReplaceAllString(normalized, " ")
	return normalized
}

func truncateText(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}
