package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/redis/go-redis/v9"

	"goravel/app/clients"
)

// LockResult 锁操作结果
type LockResult struct {
	Acquired bool          // 是否成功获取锁
	Error    error         // 错误信息
	Client   *redis.Client // Redis 客户端（如果使用 Redis）
}

// TryAcquireLock 尝试获取分布式锁（原子操作）
// lockKey: 锁的键名
// lockValue: 锁的值（用于标识锁的拥有者）
// ttl: 锁的过期时间
// 返回 LockResult，包含是否成功获取锁和错误信息
func TryAcquireLock(lockKey, lockValue string, ttl time.Duration) *LockResult {
	result := &LockResult{
		Acquired: false,
	}

	// 优先使用 Redis SETNX 原子操作
	redisClient, err := clients.GetRedisClient("default")
	if err != nil {
		facades.Log().Warningf("获取 Redis 客户端失败，降级到缓存锁: %v", err)
		// 降级到普通缓存检查（不保证原子性，但至少提供基本保护）
		return tryAcquireLockWithCache(lockKey, lockValue, ttl)
	}

	// 使用 Redis SETNX 原子操作（SET key value NX EX seconds）
	// SetNX 会自动设置过期时间，如果键已存在且未过期，返回 false
	ctx := context.Background()
	acquired, err := redisClient.SetNX(ctx, lockKey, lockValue, ttl).Result()
	if err != nil {
		facades.Log().Errorf("Redis 获取锁失败: key=%s, error=%v", lockKey, err)
		result.Error = fmt.Errorf("获取锁失败: %v", err)
		// 注意：使用公共 Redis 客户端，不需要手动关闭
		return result
	}

	if !acquired {
		// 锁已被占用，检查锁的剩余过期时间
		ttlResult, _ := redisClient.TTL(ctx, lockKey).Result()
		if ttlResult <= 0 {
			// 锁已过期，删除它并重试
			redisClient.Del(ctx, lockKey)
			// 重试一次
			acquired, err = redisClient.SetNX(ctx, lockKey, lockValue, ttl).Result()
			if err != nil {
				facades.Log().Errorf("Redis 重试获取锁失败: key=%s, error=%v", lockKey, err)
				result.Error = fmt.Errorf("获取锁失败: %v", err)
				// 注意：使用公共 Redis 客户端，不需要手动关闭
				return result
			}
		}
	}

	result.Acquired = acquired
	if acquired {
		result.Client = redisClient // 只有获取成功才保留客户端
	} else {
		result.Client = nil
	}
	return result
}

// ReleaseLock 释放锁
// 注意：只有锁的拥有者才能释放锁（通过 lockValue 验证）
func ReleaseLock(lockKey, lockValue string, client *redis.Client) error {
	if client == nil {
		// 如果没有 Redis 客户端，使用缓存删除
		_ = facades.Cache().Forget(lockKey)
		return nil
	}

	ctx := context.Background()
	// 使用 Lua 脚本确保只有锁的拥有者才能释放锁
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := client.Eval(ctx, script, []string{lockKey}, lockValue).Result()
	if err != nil {
		// 如果 Lua 脚本执行失败，尝试直接删除（降级处理）
		return client.Del(ctx, lockKey).Err()
	}
	return nil
}

// tryAcquireLockWithCache 使用缓存实现锁（降级方案，不保证原子性）
// 注意：由于缓存操作的检查-设置不是原子的，在高并发下可能仍有竞态条件
// 但至少可以提供基本的保护
func tryAcquireLockWithCache(lockKey, lockValue string, ttl time.Duration) *LockResult {
	result := &LockResult{
		Acquired: false,
	}

	// 检查是否已有锁
	var cachedValue string
	cacheErr := facades.Cache().Get(lockKey, &cachedValue)
	// 如果 Get 返回 nil（成功）且值不为空，说明锁已存在
	if cacheErr == nil && cachedValue != "" {
		// 锁已存在
		result.Error = fmt.Errorf("锁已被占用")
		return result
	}

	// 设置锁（使用 Put，如果键已存在会覆盖，但至少可以防止大部分重复请求）
	if err := facades.Cache().Put(lockKey, lockValue, ttl); err != nil {
		facades.Log().Errorf("设置缓存锁失败: key=%s, error=%v", lockKey, err)
		result.Error = fmt.Errorf("设置锁失败: %v", err)
		return result
	}

	// 再次检查，确保锁设置成功（双重检查，减少竞态条件的影响）
	// 短暂延迟，让其他并发请求有机会检测到锁
	time.Sleep(50 * time.Millisecond)
	var verifyValue string
	if facades.Cache().Get(lockKey, &verifyValue) == nil {
		if verifyValue == lockValue {
			// 锁设置成功且值匹配
			result.Acquired = true
			return result
		}
		// 值不匹配，说明被其他请求覆盖了（竞态条件）
		result.Error = fmt.Errorf("锁已被占用")
		return result
	}

	// 锁设置后无法验证，可能是缓存问题
	result.Error = fmt.Errorf("设置锁失败")
	return result
}

// LockGuard 锁保护器，自动管理锁的生命周期
type LockGuard struct {
	lockKey   string
	lockValue string
	client    *redis.Client
	acquired  bool
}

// AcquireLock 获取锁
// lockKey: 锁的键名（会自动添加用户ID前缀，格式：lockKey:userID）
// userID: 用户ID（用于生成唯一的锁键）
// ttl: 锁的过期时间
// 返回 LockGuard 和错误，如果锁已被占用，返回 ErrLockAcquired 错误
func AcquireLock(lockKey string, userID uint, ttl time.Duration) (*LockGuard, error) {
	// 生成完整的锁键和值
	fullLockKey := fmt.Sprintf("%s:%d", lockKey, userID)
	lockValue := fmt.Sprintf("%d_%d", userID, time.Now().Unix())

	// 尝试获取锁
	lockResult := TryAcquireLock(fullLockKey, lockValue, ttl)
	if lockResult.Error != nil {
		return nil, lockResult.Error
	}
	if !lockResult.Acquired {
		return nil, fmt.Errorf("锁已被占用")
	}

	return &LockGuard{
		lockKey:   fullLockKey,
		lockValue: lockValue,
		client:    lockResult.Client,
		acquired:  true,
	}, nil
}

// Release 释放锁
func (g *LockGuard) Release() {
	if !g.acquired {
		return
	}

	if g.client != nil {
		if err := ReleaseLock(g.lockKey, g.lockValue, g.client); err != nil {
			facades.Log().Errorf("释放 Redis 锁失败: key=%s, error=%v", g.lockKey, err)
		}
		g.client = nil
	} else {
		// 使用缓存时，直接删除
		_ = facades.Cache().Forget(g.lockKey)
	}
	g.acquired = false
}
