package services

import (
	"unicode"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
)

// PasswordPolicy 密码策略（便于单测注入，不依赖 facades）。
type PasswordPolicy struct {
	MinLength      int
	RequireLetter  bool
	RequireNumber  bool
	RequireSpecial bool
}

// DefaultPasswordPolicy 从配置读取密码策略。
func DefaultPasswordPolicy() PasswordPolicy {
	cfg := facades.Config()
	minLen := cfg.GetInt("login_security.password_min_length", 8)
	if minLen < 1 {
		minLen = 8
	}
	return PasswordPolicy{
		MinLength:      minLen,
		RequireLetter:  cfg.GetBool("login_security.password_require_letter", true),
		RequireNumber:  cfg.GetBool("login_security.password_require_number", true),
		RequireSpecial: cfg.GetBool("login_security.password_require_special", false),
	}
}

// ValidatePasswordPolicy 按 login_security 配置校验密码强度。
func ValidatePasswordPolicy(password string) error {
	return ValidatePasswordAgainstPolicy(password, DefaultPasswordPolicy())
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
