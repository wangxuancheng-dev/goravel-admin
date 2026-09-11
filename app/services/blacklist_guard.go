package services

import (
	"context"
	"sync"
	"time"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancyctx"
)

const (
	blacklistCacheTTL    = 30 * time.Second
	blacklistStaleMaxAge = 5 * time.Minute
)

type blacklistCacheEntry struct {
	patterns  []string
	fetchedAt time.Time
}

var (
	blacklistPatternCache      sync.Map // map[string]blacklistCacheEntry
	loadEnabledBlacklistPatternsFn = loadEnabledBlacklistPatterns
)

func blacklistCacheKey(ctx context.Context) string {
	if ctx == nil {
		ctx = context.Background()
	}
	if conn, ok := tenancyctx.ConnectionFrom(ctx); ok && conn != "" {
		return "blacklist:enabled:" + conn
	}
	return "blacklist:enabled:default"
}

// InvalidateBlacklistCache drops the in-process enabled-pattern cache for the current connection.
// Call after blacklist Create/Update/Delete so bans take effect immediately.
func InvalidateBlacklistCache(ctx context.Context) {
	blacklistPatternCache.Delete(blacklistCacheKey(ctx))
}

// ResetBlacklistCacheForTest clears all blacklist pattern cache entries (tests only).
func ResetBlacklistCacheForTest() {
	blacklistPatternCache.Range(func(key, _ any) bool {
		blacklistPatternCache.Delete(key)
		return true
	})
}

// EnabledBlacklistPatterns returns enabled blacklist IP patterns for the bound connection.
// Fresh hits skip DB (short TTL). On DB failure, returns last-known-good within stale window;
// otherwise returns the DB error so callers can fail-closed.
func EnabledBlacklistPatterns(ctx context.Context) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	key := blacklistCacheKey(ctx)
	now := time.Now()

	if v, ok := blacklistPatternCache.Load(key); ok {
		entry := v.(blacklistCacheEntry)
		if now.Sub(entry.fetchedAt) < blacklistCacheTTL {
			return clonePatterns(entry.patterns), nil
		}
	}

	patterns, err := loadEnabledBlacklistPatternsFn(ctx)
	if err != nil {
		if v, ok := blacklistPatternCache.Load(key); ok {
			entry := v.(blacklistCacheEntry)
			if now.Sub(entry.fetchedAt) < blacklistStaleMaxAge {
				return clonePatterns(entry.patterns), nil
			}
		}
		return nil, err
	}

	blacklistPatternCache.Store(key, blacklistCacheEntry{
		patterns:  patterns,
		fetchedAt: now,
	})
	return clonePatterns(patterns), nil
}

func loadEnabledBlacklistPatterns(ctx context.Context) ([]string, error) {
	var rows []models.Blacklist
	if err := appfacades.OrmQuery(ctx).Where("status", 1).Get(&rows); err != nil {
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

func clonePatterns(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
