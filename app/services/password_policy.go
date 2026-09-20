package services

import (
	"context"
	"strconv"
	"strings"
	"unicode"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/utils"
)

// PasswordPolicy 密码策略（便于单测注入，不依赖 facades）。
type PasswordPolicy struct {
	MinLength      int
	RequireLetter  bool
	RequireNumber  bool
	RequireSpecial bool
}

// ResolvePasswordPolicy 优先读 configs 表 group=login_security，再回退到 facades.Config / .env。
// DB 中已存在的 key（含值为 "0"）一律以库为准，不再回退到环境默认的严格策略。
func ResolvePasswordPolicy(ctx context.Context) PasswordPolicy {
	cfg := facades.Config()
	db := utils.GetConfigGroupMap(ctx, "login_security")

	minLen := cfg.GetInt("login_security.password_min_length", 8)
	if raw, ok := db["password_min_length"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > 0 {
			minLen = n
		}
	}
	if minLen < 1 {
		minLen = 8
	}

	requireLetter := cfg.GetBool("login_security.password_require_letter", true)
	if raw, ok := db["password_require_letter"]; ok {
		requireLetter = utils.ParseConfigBool(raw)
	}
	requireNumber := cfg.GetBool("login_security.password_require_number", true)
	if raw, ok := db["password_require_number"]; ok {
		requireNumber = utils.ParseConfigBool(raw)
	}
	requireSpecial := cfg.GetBool("login_security.password_require_special", false)
	if raw, ok := db["password_require_special"]; ok {
		requireSpecial = utils.ParseConfigBool(raw)
	}

	return PasswordPolicy{
		MinLength:      minLen,
		RequireLetter:  requireLetter,
		RequireNumber:  requireNumber,
		RequireSpecial: requireSpecial,
	}
}

// DefaultPasswordPolicy 从配置读取密码策略（无请求上下文时使用）。
func DefaultPasswordPolicy() PasswordPolicy {
	return ResolvePasswordPolicy(context.Background())
}

// ValidatePasswordPolicy 按 login_security 配置校验密码强度。
func ValidatePasswordPolicy(password string) error {
	return ValidatePasswordAgainstPolicy(password, DefaultPasswordPolicy())
}

// ValidatePasswordPolicyCtx 带上下文的密码策略校验（多租户 / DB 覆盖）。
func ValidatePasswordPolicyCtx(ctx context.Context, password string) error {
	return ValidatePasswordAgainstPolicy(password, ResolvePasswordPolicy(ctx))
}

// ValidatePasswordAgainstPolicy 按给定策略校验密码（纯逻辑，便于单测）。
func ValidatePasswordAgainstPolicy(password string, policy PasswordPolicy) error {
	if policy.MinLength < 1 {
		policy.MinLength = 8
	}
	runes := []rune(password)
	if len(runes) < policy.MinLength {
		return apperrors.ErrPasswordTooWeak.WithParams(map[string]any{
			"min_length": policy.MinLength,
		})
	}

	var hasLetter, hasNumber, hasSpecial bool
	for _, r := range runes {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if policy.RequireLetter && !hasLetter {
		return apperrors.ErrPasswordTooWeak
	}
	if policy.RequireNumber && !hasNumber {
		return apperrors.ErrPasswordTooWeak
	}
	if policy.RequireSpecial && !hasSpecial {
		return apperrors.ErrPasswordTooWeak
	}
	return nil
}
