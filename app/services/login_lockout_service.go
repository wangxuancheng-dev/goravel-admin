package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/tenancy"
)

// LoginLockoutService 登录失败锁定服务。
// 两个维度：
//   - IP+用户名：连续失败达到阈值后锁定（同一 IP 不同账号互不影响）。
//   - 用户名（跨 IP）：防代理池轮换 IP 撞库；达到较低阈值后强制人机验证，
//     达到较高阈值后锁定账号。阈值比 IP 维度宽松，避免他人恶意锁死正常用户。
type LoginLockoutService interface {
	// IsLocked 检查该 IP+用户名 或该账号是否处于锁定状态，返回 (是否锁定, 锁定分钟数)。
	IsLocked(ip, username string) (bool, int)
	// GetFailureCount 返回当前失败计数（未锁定时的连续失败次数）。
	GetFailureCount(ip, username string) int
	// NeedsCaptcha 失败次数达到自适应阈值（IP 维度或账号维度）时需要人机验证。
	NeedsCaptcha(ip, username string) bool
	// RecordFailure 记录一次失败，返回 (当前 IP 维度失败次数, 是否触发锁定)。
	RecordFailure(ip, username string) (int, bool)
	// ClearFailures 登录成功后清除计数。
	ClearFailures(ip, username string)
	// UnlockUsername clears lock for a username across IPs (platform support).
	UnlockUsername(username string)
}

// adaptiveCaptchaIPFailures IP+用户名 维度失败达到该次数后强制人机验证。
const adaptiveCaptchaIPFailures = 2

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

// accountConfig 账号维度（跨 IP）阈值；<=0 表示关闭对应能力。
func (s *LoginLockoutServiceImpl) accountConfig() (captchaAfter, maxAttempts int, lockDuration, decay time.Duration) {
	cfg := facades.Config()
	captchaAfter = cfg.GetInt("login_security.account_captcha_after", 5)
	maxAttempts = cfg.GetInt("login_security.account_max_attempts", 20)
	lockMinutes := cfg.GetInt("login_security.account_lock_duration_minutes", 15)
	decayMinutes := cfg.GetInt("login_security.account_decay_minutes", 60)
	if decayMinutes < 1 {
		decayMinutes = 60
	}
	if lockMinutes < 1 {
		lockMinutes = 15
	}
	return captchaAfter, maxAttempts, time.Duration(lockMinutes) * time.Minute, time.Duration(decayMinutes) * time.Minute
}

// lockoutUsername 归一化用户名：MySQL 默认排序规则忽略大小写和尾部空格，
// "Admin" / "admin " 登录的是同一账号，不归一化就能换写法绕过计数。
func lockoutUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func (s *LoginLockoutServiceImpl) lockKey(ip, username string) string {
	return tenancy.CacheKey(s.ctx, fmt.Sprintf("login_lock:%s:%s", ip, lockoutUsername(username)))
}

func (s *LoginLockoutServiceImpl) attemptsKey(ip, username string) string {
	return tenancy.CacheKey(s.ctx, fmt.Sprintf("login_attempts:%s:%s", ip, lockoutUsername(username)))
}

func (s *LoginLockoutServiceImpl) accountLockKey(username string) string {
	return tenancy.CacheKey(s.ctx, fmt.Sprintf("login_account_lock:%s", lockoutUsername(username)))
}

func (s *LoginLockoutServiceImpl) accountAttemptsKey(username string) string {
	return tenancy.CacheKey(s.ctx, fmt.Sprintf("login_account_attempts:%s", lockoutUsername(username)))
}

func (s *LoginLockoutServiceImpl) unlockKey(username string) string {
	return tenancy.CacheKey(s.ctx, fmt.Sprintf("login_unlock:%s", lockoutUsername(username)))
}

// IsLocked 检查是否锁定。
// 锁定标记的值为加锁时间戳；人工解锁（UnlockUsername）只让解锁之前产生的锁失效，
// 解锁窗口内新产生的失败仍会正常锁定。
func (s *LoginLockoutServiceImpl) IsLocked(ip, username string) (bool, int) {
	unlockedAt := parseLockStamp(facades.Cache().GetString(s.unlockKey(username), ""))
	for _, key := range []string{s.lockKey(ip, username), s.accountLockKey(username)} {
		val := facades.Cache().GetString(key, "")
		if val == "" {
			continue
		}
		if unlockedAt > 0 && parseLockStamp(val) <= unlockedAt {
			continue
		}
		return true, facades.Cache().GetInt(key+"_ttl", 0)
	}
	return false, 0
}

func lockStamp() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// parseLockStamp 兼容旧值 "1"（视为最早的锁）。
func parseLockStamp(v string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// GetFailureCount 返回当前失败计数。
func (s *LoginLockoutServiceImpl) GetFailureCount(ip, username string) int {
	return facades.Cache().GetInt(s.attemptsKey(ip, username), 0)
}

// NeedsCaptcha 自适应人机验证：IP 维度或账号维度失败次数达到阈值。
func (s *LoginLockoutServiceImpl) NeedsCaptcha(ip, username string) bool {
	if s.GetFailureCount(ip, username) >= adaptiveCaptchaIPFailures {
		return true
	}
	captchaAfter, _, _, _ := s.accountConfig()
	return captchaAfter > 0 && facades.Cache().GetInt(s.accountAttemptsKey(username), 0) >= captchaAfter
}

// RecordFailure 记录失败，到达阈值时写入锁定标记。
func (s *LoginLockoutServiceImpl) RecordFailure(ip, username string) (int, bool) {
	accountLocked := s.recordAccountFailure(username)

	maxAttempts, lockDuration, decayMinutes := s.config()
	aKey := s.attemptsKey(ip, username)

	current := facades.Cache().GetInt(aKey, 0)
	current++

	_ = facades.Cache().Put(aKey, current, decayMinutes)

	if current >= maxAttempts {
		lKey := s.lockKey(ip, username)
		_ = facades.Cache().Put(lKey, lockStamp(), lockDuration)
		lockMinutes := int(lockDuration.Minutes())
		_ = facades.Cache().Put(lKey+"_ttl", lockMinutes, lockDuration)
		_ = facades.Cache().Forget(aKey)
		return current, true
	}
	return current, accountLocked
}

// recordAccountFailure 账号维度计数，达到上限时锁定账号（跨 IP）。
func (s *LoginLockoutServiceImpl) recordAccountFailure(username string) bool {
	_, maxAttempts, lockDuration, decay := s.accountConfig()
	key := s.accountAttemptsKey(username)
	current := facades.Cache().GetInt(key, 0) + 1
	_ = facades.Cache().Put(key, current, decay)
	if maxAttempts <= 0 || current < maxAttempts {
		return false
	}
	lKey := s.accountLockKey(username)
	_ = facades.Cache().Put(lKey, lockStamp(), lockDuration)
	_ = facades.Cache().Put(lKey+"_ttl", int(lockDuration.Minutes()), lockDuration)
	_ = facades.Cache().Forget(key)
	return true
}

// ClearFailures 清除计数和锁定标记。
func (s *LoginLockoutServiceImpl) ClearFailures(ip, username string) {
	_ = facades.Cache().Forget(s.attemptsKey(ip, username))
	lKey := s.lockKey(ip, username)
	_ = facades.Cache().Forget(lKey)
	_ = facades.Cache().Forget(lKey + "_ttl")
	_ = facades.Cache().Forget(s.accountAttemptsKey(username))
}

// UnlockUsername stamps an unlock time so IsLocked ignores IP locks created before it
// (IP-scoped keys cannot be enumerated). Locks created afterwards still apply.
func (s *LoginLockoutServiceImpl) UnlockUsername(username string) {
	username = strings.TrimSpace(username)
	if username == "" {
		return
	}
	_, lockDuration, _ := s.config()
	_, _, accountLockDuration, _ := s.accountConfig()
	if accountLockDuration > lockDuration {
		lockDuration = accountLockDuration
	}
	_ = facades.Cache().Put(s.unlockKey(username), lockStamp(), lockDuration)
	s.ClearFailures("", username)
	s.ClearFailures("0.0.0.0", username)
	s.ClearFailures("127.0.0.1", username)
	aKey := s.accountLockKey(username)
	_ = facades.Cache().Forget(aKey)
	_ = facades.Cache().Forget(aKey + "_ttl")
}
