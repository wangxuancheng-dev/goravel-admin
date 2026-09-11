package utils

import (
	"context"
	"strings"
	"sync"
	"time"

	appfacades "goravel/app/facades"
)

const shardingTableCacheTTL = 5 * time.Minute

type shardingTableCacheEntry struct {
	exists    bool
	checkedAt time.Time
}

var (
	shardingTableCache   sync.Map // map[string]shardingTableCacheEntry
	shardingTableCreateM sync.Map // map[string]*sync.Mutex
)

func shardingCacheKeyFrom(ctx context.Context, tableName string) string {
	return appfacades.SchemaConnectionKeyFrom(ctx) + ":" + tableName
}

func shardingTableCreateMutex(ctx context.Context, tableName string) *sync.Mutex {
	key := shardingCacheKeyFrom(ctx, tableName)
	actual, _ := shardingTableCreateM.LoadOrStore(key, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// ShardingTableExists 检查分表是否存在（无租户 ctx 时用当前 Schema 连接，适合 CLI WithTenantConnection）。
func ShardingTableExists(tableName string) bool {
	return ShardingTableExistsCtx(context.Background(), tableName)
}

// ShardingTableExistsCtx 按 ctx 租户连接检查分表是否存在（带短缓存）。
func ShardingTableExistsCtx(ctx context.Context, tableName string) bool {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	key := shardingCacheKeyFrom(ctx, tableName)
	if v, ok := shardingTableCache.Load(key); ok {
		entry := v.(shardingTableCacheEntry)
		if time.Since(entry.checkedAt) < shardingTableCacheTTL {
			return entry.exists
		}
	}
	exists := appfacades.SchemaHasTable(ctx, tableName)
	shardingTableCache.Store(key, shardingTableCacheEntry{
		exists:    exists,
		checkedAt: time.Now(),
	})
	return exists
}

// MarkShardingTableExists 标记分表已存在（创建成功后调用）。
func MarkShardingTableExists(tableName string) {
	MarkShardingTableExistsCtx(context.Background(), tableName)
}

// MarkShardingTableExistsCtx 按连接标记分表已存在。
func MarkShardingTableExistsCtx(ctx context.Context, tableName string) {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	shardingTableCache.Store(shardingCacheKeyFrom(ctx, tableName), shardingTableCacheEntry{
		exists:    true,
		checkedAt: time.Now(),
	})
}

// InvalidateShardingTableCache 清除单个分表缓存（一般无需调用）。
func InvalidateShardingTableCache(tableName string) {
	InvalidateShardingTableCacheCtx(context.Background(), tableName)
}

// InvalidateShardingTableCacheCtx 按连接清除分表缓存。
func InvalidateShardingTableCacheCtx(ctx context.Context, tableName string) {
	if ctx == nil {
		ctx = context.Background()
	}
	shardingTableCache.Delete(shardingCacheKeyFrom(ctx, strings.TrimSpace(tableName)))
}

// WithShardingTableCreateLock 对同一分表名串行化建表，避免并发 DDL 竞态。
func WithShardingTableCreateLock(tableName string, fn func() error) error {
	return WithShardingTableCreateLockCtx(context.Background(), tableName, fn)
}

// WithShardingTableCreateLockCtx 按连接串行化建表。
func WithShardingTableCreateLockCtx(ctx context.Context, tableName string, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	mu := shardingTableCreateMutex(ctx, tableName)
	mu.Lock()
	defer mu.Unlock()
	return fn()
}
