package services

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/tenancy"
)

// LoginLockoutService 登录失败锁定服务。
// 按 IP+用户名 维度计数，达到阈值后锁定一段时间。
// 同一 IP 不同账号互不影响，同一账号不同 IP 互不影响。
type LoginLockoutService interface {
	// IsLocked 检查该 IP+用户名 是否处于锁定状态，返回 (是否锁定, 剩余分钟数)。
	IsLocked(ip, username string) (bool, int)
	// RecordFailure 记录一次失败，返回 (当前失败次数, 是否触发锁定)。
	RecordFailure(ip, username string) (int, bool)
	// ClearFailures 登录成功后清除计数。
	ClearFailures(ip, username string)
}

type LoginLockoutServiceImpl struct {
	ctx context.Context
}

func NewLoginLockoutService(ctx context.Context) LoginLockoutService {
	return &LoginLockoutServiceImpl{ctx: ctx}
}

func (s *LoginLockoutServiceImpl) config() (maxAttempts int, lockDuration time.Duration, decayMinutes time.Duration) {
	cfg := facades.Config()
	maxAttempts = cfg.GetInt("login_security.max_attempts", 5)
	lockMinutes := cfg.GetInt("login_security.lock_duration_minutes", 15)
	decay := cfg.GetInt("login_security.decay_minutes", 5)
	return maxAttempts, time.Duration(lockMinutes) * time.Minute, time.Duration(decay) * time.Minute
}

func (s *LoginLockoutServiceImpl) lockKey(ip, username string) string {
	return tenancy.CacheKey(s.ctx, fmt.Sprintf("login_lock:%s:%s", ip, username))
}

func (s *LoginLockoutServiceImpl) attemptsKey(ip, username string) string {
	return tenancy.CacheKey(s.ctx, fmt.Sprintf("login_attempts:%s:%s", ip, username))
}

// IsLocked 检查是否锁定。
func (s *LoginLockoutServiceImpl) IsLocked(ip, username string) (bool, int) {
	key := s.lockKey(ip, username)
	val := facades.Cache().GetString(key, "")
	if val == "" {
		return false, 0
	}
	minutes := facades.Cache().GetInt(key+"_ttl", 0)
	return true, minutes
}

// RecordFailure 记录失败，到达阈值时写入锁定标记。
func (s *LoginLockoutServiceImpl) RecordFailure(ip, username string) (int, bool) {
	maxAttempts, lockDuration, decayMinutes := s.config()
	aKey := s.attemptsKey(ip, username)

	current := facades.Cache().GetInt(aKey, 0)
	current++

	_ = facades.Cache().Put(aKey, current, decayMinutes)

	if current >= maxAttempts {
		lKey := s.lockKey(ip, username)
		_ = facades.Cache().Put(lKey, "1", lockDuration)
		lockMinutes := int(lockDuration.Minutes())
		_ = facades.Cache().Put(lKey+"_ttl", lockMinutes, lockDuration)
		_ = facades.Cache().Forget(aKey)
		return current, true
	}
	return current, false
}

// ClearFailures 清除计数和锁定标记。
func (s *LoginLockoutServiceImpl) ClearFailures(ip, username string) {
	_ = facades.Cache().Forget(s.attemptsKey(ip, username))
	lKey := s.lockKey(ip, username)
	_ = facades.Cache().Forget(lKey)
	_ = facades.Cache().Forget(lKey + "_ttl")
}
