package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnabledBlacklistPatternsFreshCacheSkipsLoader(t *testing.T) {
	ResetBlacklistCacheForTest()
	t.Cleanup(ResetBlacklistCacheForTest)

	ctx := context.Background()
	calls := 0
	loadEnabledBlacklistPatternsFn = func(context.Context) ([]string, error) {
		calls++
		return []string{"10.0.0.1"}, nil
	}
	t.Cleanup(func() { loadEnabledBlacklistPatternsFn = loadEnabledBlacklistPatterns })

	first, err := EnabledBlacklistPatterns(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.1"}, first)
	assert.Equal(t, 1, calls)

	second, err := EnabledBlacklistPatterns(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.1"}, second)
	assert.Equal(t, 1, calls, "fresh TTL should skip DB loader")
}

func TestEnabledBlacklistPatternsStaleFallbackOnLoaderError(t *testing.T) {
	ResetBlacklistCacheForTest()
	t.Cleanup(ResetBlacklistCacheForTest)

	ctx := context.Background()
	loadEnabledBlacklistPatternsFn = func(context.Context) ([]string, error) {
		return []string{"10.0.0.8"}, nil
	}
	t.Cleanup(func() { loadEnabledBlacklistPatternsFn = loadEnabledBlacklistPatterns })

	_, err := EnabledBlacklistPatterns(ctx)
	require.NoError(t, err)

	key := blacklistCacheKey(ctx)
	v, ok := blacklistPatternCache.Load(key)
	require.True(t, ok)
	entry := v.(blacklistCacheEntry)
	entry.fetchedAt = time.Now().Add(-2 * time.Minute) // expired TTL, still within stale window
	blacklistPatternCache.Store(key, entry)

	loadEnabledBlacklistPatternsFn = func(context.Context) ([]string, error) {
		return nil, errors.New("db down")
	}

	got, err := EnabledBlacklistPatterns(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.8"}, got)
}

func TestEnabledBlacklistPatternsFailClosedWhenStaleTooOld(t *testing.T) {
	ResetBlacklistCacheForTest()
	t.Cleanup(ResetBlacklistCacheForTest)

	ctx := context.Background()
	key := blacklistCacheKey(ctx)
	blacklistPatternCache.Store(key, blacklistCacheEntry{
		patterns:  []string{"10.0.0.9"},
		fetchedAt: time.Now().Add(-10 * time.Minute),
	})

	loadEnabledBlacklistPatternsFn = func(context.Context) ([]string, error) {
		return nil, errors.New("db down")
	}
	t.Cleanup(func() { loadEnabledBlacklistPatternsFn = loadEnabledBlacklistPatterns })

	_, err := EnabledBlacklistPatterns(ctx)
	require.Error(t, err)
}

func TestInvalidateBlacklistCacheForcesReload(t *testing.T) {
	ResetBlacklistCacheForTest()
	t.Cleanup(ResetBlacklistCacheForTest)

	ctx := context.Background()
	calls := 0
	loadEnabledBlacklistPatternsFn = func(context.Context) ([]string, error) {
		calls++
		return []string{"10.0.0.2"}, nil
	}
	t.Cleanup(func() { loadEnabledBlacklistPatternsFn = loadEnabledBlacklistPatterns })

	_, err := EnabledBlacklistPatterns(ctx)
	require.NoError(t, err)
	InvalidateBlacklistCache(ctx)
	_, err = EnabledBlacklistPatterns(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}
