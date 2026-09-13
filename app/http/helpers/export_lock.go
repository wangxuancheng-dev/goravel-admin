package helpers

import (
	"fmt"
	"time"

	"goravel/app/utils"

	"github.com/goravel/framework/contracts/cache"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
)

const defaultExportLockTTL = 10 * time.Second

// ExportLockResult 导出锁结果。
type ExportLockResult struct {
	AdminID      uint
	Unauthorized bool
	Blocked      bool
	lock         cache.Lock
	acquired     bool
}

// Release 在任务入队失败等场景下立即释放导出锁。
func (r *ExportLockResult) Release() {
	if r == nil || !r.acquired || r.lock == nil {
		return
	}
	r.lock.Release()
	r.acquired = false
}

// AcquireExportLock 尝试获取导出防重复锁。
func AcquireExportLock(ctx http.Context, resource string) ExportLockResult {
	return AcquireExportLockWithTTL(ctx, resource, defaultExportLockTTL)
}

// AcquireExportLockWithTTL 带自定义 TTL 的导出锁。
func AcquireExportLockWithTTL(ctx http.Context, resource string, ttl time.Duration) ExportLockResult {
	adminID, err := GetAdminIDFromContext(ctx)
	if err != nil {
		return ExportLockResult{Unauthorized: true}
	}

	lockKey := TenantCacheKey(ctx, fmt.Sprintf("export:%s:lock:%d", resource, adminID))
	lock := facades.Cache().Lock(lockKey, ttl)
	if !lock.Get() {
		return ExportLockResult{AdminID: adminID, Blocked: true}
	}
	return ExportLockResult{AdminID: adminID, lock: lock, acquired: true}
}

// ResolveExportDisk 读取导出存储盘配置。
func ResolveExportDisk(ctx http.Context) string {
	return utils.ResolveFileDisk(ctx)
}
