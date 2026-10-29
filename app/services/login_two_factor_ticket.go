package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/tenancy"
)

// 两步登录票据：第一步（人机验证 + 密码）通过且账号绑定了谷歌验证码时签发，
// 第二步凭票据 + 谷歌验证码登录，无需再次人机验证。票据与作用域、账号、IP 绑定，短时有效。
// 没有有效票据时，携带谷歌验证码的请求仍需先通过人机验证，避免以“填任意谷歌码”绕过验证码撞密码。

const (
	loginTwoFactorTicketTTL = 5 * time.Minute
	// TOTP 默认允许 ±1 个 30s 周期，已使用的验证码在此时间内不可重放。
	totpReplayWindow = 90 * time.Second
)

func loginTwoFactorTicketKey(ctx context.Context, ticket string) string {
	return tenancy.CacheKey(ctx, "login_2fa_ticket:"+ticket)
}

// 用户名放最后：SplitN(4) 后用户名里含 "|" 也不会错位。
func loginTwoFactorTicketValue(scope string, subjectID uint, username, ip string) string {
	return fmt.Sprintf("%s|%d|%s|%s", scope, subjectID, ip, lockoutUsername(username))
}

// IssueLoginTwoFactorTicket 签发第二步登录票据。
func IssueLoginTwoFactorTicket(ctx context.Context, scope string, subjectID uint, username, ip string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	ticket := hex.EncodeToString(buf)
	if err := facades.Cache().Put(loginTwoFactorTicketKey(ctx, ticket), loginTwoFactorTicketValue(scope, subjectID, username, ip), loginTwoFactorTicketTTL); err != nil {
		return "", err
	}
	return ticket, nil
}

// LoginTwoFactorTicketSubject 校验票据属于该作用域 + 用户名 + IP，返回签发时的账号 ID（0 表示无效）。
func LoginTwoFactorTicketSubject(ctx context.Context, ticket, scope, username, ip string) uint {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" || len(ticket) > 128 {
		return 0
	}
	val := facades.Cache().GetString(loginTwoFactorTicketKey(ctx, ticket), "")
	parts := strings.SplitN(val, "|", 4)
	if len(parts) != 4 {
		return 0
	}
	if parts[0] != scope || parts[2] != ip || parts[3] != lockoutUsername(username) {
		return 0
	}
	var id uint
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return 0
	}
	return id
}

// ConsumeLoginTwoFactorTicket 登录成功后作废票据。
func ConsumeLoginTwoFactorTicket(ctx context.Context, ticket string) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return
	}
	_ = facades.Cache().Forget(loginTwoFactorTicketKey(ctx, ticket))
}

var (
	dummyPasswordHashOnce sync.Once
	dummyPasswordHash     string
)

// BurnPasswordCheck 用户名不存在时也做一次哈希比对，使响应耗时与“密码错误”一致，防止按耗时枚举账号。
func BurnPasswordCheck(password string) {
	dummyPasswordHashOnce.Do(func() {
		dummyPasswordHash, _ = facades.Hash().Make("login-timing-equalizer")
	})
	if dummyPasswordHash != "" {
		_ = facades.Hash().Check(password, dummyPasswordHash)
	}
}

// VerifyTOTPOnce 校验谷歌验证码，并拒绝在有效窗口内重放同一验证码（防截获后复用）。
func VerifyTOTPOnce(ctx context.Context, scope string, subjectID uint, secret, code string) bool {
	code = strings.TrimSpace(code)
	if !NewGoogleAuthenticatorServiceImpl(ctx).Verify(secret, code) {
		return false
	}
	key := tenancy.CacheKey(ctx, fmt.Sprintf("totp_used:%s:%d:%s", scope, subjectID, code))
	// Add 仅在 key 不存在时写入（Redis 下为原子 SETNX），多实例并发也只有一个请求能用上。
	return facades.Cache().Add(key, "1", totpReplayWindow)
}
