package utils

import (
	"strings"
	"sync"
	"time"

	"github.com/goravel/framework/facades"
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

func shardingTableCreateMutex(tableName string) *sync.Mutex {
	actual, _ := shardingTableCreateM.LoadOrStore(tableName, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// ShardingTableExists 检查分表是否存在（带短缓存，减轻热路径 HasTable 压力）。
func ShardingTableExists(tableName string) bool {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return false
	}
	if v, ok := shardingTableCache.Load(tableName); ok {
		entry := v.(shardingTableCacheEntry)
		if time.Since(entry.checkedAt) < shardingTableCacheTTL {
			return entry.exists
		}
	}
	exists := facades.Schema().HasTable(tableName)
	shardingTableCache.Store(tableName, shardingTableCacheEntry{
		exists:    exists,
		checkedAt: time.Now(),
	})
	return exists
}

// MarkShardingTableExists 标记分表已存在（创建成功后调用）。
func MarkShardingTableExists(tableName string) {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return
	}
	shardingTableCache.Store(tableName, shardingTableCacheEntry{
		exists:    true,
		checkedAt: time.Now(),
	})
}

// InvalidateShardingTableCache 清除单个分表缓存（一般无需调用）。
func InvalidateShardingTableCache(tableName string) {
	shardingTableCache.Delete(strings.TrimSpace(tableName))
}

// WithShardingTableCreateLock 对同一分表名串行化建表，避免并发 DDL 竞态。
func WithShardingTableCreateLock(tableName string, fn func() error) error {
	mu := shardingTableCreateMutex(tableName)
	mu.Lock()
	defer mu.Unlock()
	return fn()
}
