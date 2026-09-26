package services

import (
	"context"
	"strings"
	"sync"
	"time"

	appfacades "goravel/app/facades"
	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/tenancyctx"
	"goravel/app/utils"
)

const (
	allowlistCacheTTL    = 30 * time.Second
	allowlistStaleMaxAge = 5 * time.Minute
)

type allowlistCacheEntry struct {
	patterns  []string
	fetchedAt time.Time
}

var (
	allowlistPatternCache           sync.Map
	loadEnabledAllowlistPatternsFn  = loadEnabledAllowlistPatterns
)

func allowlistCacheKey(ctx context.Context) string {
	if ctx == nil {
		ctx = context.Background()
	}
	if conn, ok := tenancyctx.ConnectionFrom(ctx); ok && conn != "" {
		return "allowlist:enabled:" + conn
	}
	return "allowlist:enabled:default"
}

func InvalidateAllowlistCache(ctx context.Context) {
	allowlistPatternCache.Delete(allowlistCacheKey(ctx))
}

func EnabledAllowlistPatterns(ctx context.Context) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	key := allowlistCacheKey(ctx)
	now := time.Now()
	if v, ok := allowlistPatternCache.Load(key); ok {
		entry := v.(allowlistCacheEntry)
		if now.Sub(entry.fetchedAt) < allowlistCacheTTL {
			return clonePatterns(entry.patterns), nil
		}
	}
	patterns, err := loadEnabledAllowlistPatternsFn(ctx)
	if err != nil {
		if v, ok := allowlistPatternCache.Load(key); ok {
			entry := v.(allowlistCacheEntry)
			if now.Sub(entry.fetchedAt) < allowlistStaleMaxAge {
				return clonePatterns(entry.patterns), nil
			}
		}
		return nil, err
	}
	allowlistPatternCache.Store(key, allowlistCacheEntry{patterns: patterns, fetchedAt: now})
	return clonePatterns(patterns), nil
}

func loadEnabledAllowlistPatterns(ctx context.Context) ([]string, error) {
	// Tenant DBs may lag behind platform migrate; missing table = unrestricted.
	if !appfacades.SchemaHasTable(ctx, "allowlists") {
		return nil, nil
	}
	var rows []models.Allowlist
	if err := appfacades.OrmQuery(ctx).Where("status", 1).Get(&rows); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "doesn't exist") ||
			strings.Contains(msg, "does not exist") ||
			strings.Contains(msg, "no such table") ||
			strings.Contains(msg, "1146") {
			return nil, nil
		}
		return nil, err
	}
	patterns := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.IP == "" {
			continue
		}
		patterns = append(patterns, row.IP)
	}
	return patterns, nil
}

// EnsureClientIPAllowed enforces allowlist when at least one enabled pattern exists.
// Empty enabled set = allow all. Fail-closed on load errors.
func EnsureClientIPAllowed(ctx context.Context, clientIP string) error {
	patterns, err := EnabledAllowlistPatterns(ctx)
	if err != nil {
		return apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	if len(patterns) == 0 {
		return nil
	}
	for _, pattern := range patterns {
		if utils.IsIPInBlacklist(clientIP, pattern) {
			return nil
		}
	}
	return apperrors.ErrIPNotAllowed
}
