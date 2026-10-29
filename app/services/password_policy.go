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

// ToJSON 下发给前端做即时提示（后端仍是最终校验，弱口令字典等只在后端）。
func (p PasswordPolicy) ToJSON() map[string]any {
	return map[string]any{
		"min_length":      p.MinLength,
		"require_letter":  p.RequireLetter,
		"require_number":  p.RequireNumber,
		"require_special": p.RequireSpecial,
	}
}

// ResolvePasswordPolicy 优先读 configs 表 group=login_security，再回退到 facades.Config / .env。
// DB 中已存在的 key（含值为 "0"）一律以库为准，不再回退到环境默认的严格策略。
func ResolvePasswordPolicy(ctx context.Context) PasswordPolicy {
	return resolvePasswordPolicy(utils.GetConfigGroupMap(ctx, "login_security"))
}

// ConfigPasswordPolicy 只读 .env / config 的策略，不受租户库配置影响（平台管理员使用）。
func ConfigPasswordPolicy() PasswordPolicy {
	return resolvePasswordPolicy(nil)
}

func resolvePasswordPolicy(db map[string]string) PasswordPolicy {
	cfg := facades.Config()

	minLen := cfg.GetInt("login_security.password_min_length", 8)
	if raw, ok := db["password_min_length"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > 0 {
			minLen = n
		}
	}
	if minLen < 1 {
		minLen = 8
	}

	flag := func(key string, def bool) bool {
		v := cfg.GetBool("login_security."+key, def)
		if raw, ok := db[key]; ok {
			v = utils.ParseConfigBool(raw)
		}
		return v
	}

	return PasswordPolicy{
		MinLength:      minLen,
		RequireLetter:  flag("password_require_letter", true),
		RequireNumber:  flag("password_require_number", true),
		RequireSpecial: flag("password_require_special", true),
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

// ValidatePasswordPolicyCtx 带上下文的密码策略校验（多租户 / DB 覆盖），并拒绝包含用户名的密码。
func ValidatePasswordPolicyCtx(ctx context.Context, password, username string) error {
	return ValidateAccountPassword(password, username, ResolvePasswordPolicy(ctx))
}

// ValidateAccountPassword 策略校验 + 不得包含用户名。
func ValidateAccountPassword(password, username string, policy PasswordPolicy) error {
	if err := ValidatePasswordAgainstPolicy(password, policy); err != nil {
		return err
	}
	if passwordContainsUsername(password, username) {
		return apperrors.ErrPasswordContainsUsername
	}
	return nil
}

// ValidatePasswordAgainstPolicy 按给定策略校验密码（纯逻辑，便于单测）。
// 常见弱口令（如 Admin@123、P@ssw0rd）满足复杂度也会被拒绝，这类密码是撞库首选。
func ValidatePasswordAgainstPolicy(password string, policy PasswordPolicy) error {
	if policy.MinLength < 1 {
		policy.MinLength = 8
	}
	runes := []rune(password)

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

	// 一次列出所有缺少的字符类型，避免用户逐条试错
	var missing apperrors.TransList
	if policy.RequireLetter && !hasLetter {
		missing = append(missing, apperrors.PasswordCharLetter)
	}
	if policy.RequireNumber && !hasNumber {
		missing = append(missing, apperrors.PasswordCharNumber)
	}
	if policy.RequireSpecial && !hasSpecial {
		missing = append(missing, apperrors.PasswordCharSpecial)
	}
	tooShort := len(runes) < policy.MinLength
	switch {
	case tooShort && len(missing) > 0:
		return apperrors.ErrPasswordRequirementsUnmet.WithParams(map[string]any{
			"min_length": policy.MinLength,
			"missing":    missing,
		})
	case tooShort:
		return apperrors.ErrPasswordTooShort.WithParams(map[string]any{
			"min_length": policy.MinLength,
		})
	case len(missing) > 0:
		return apperrors.ErrPasswordMissingChars.WithParams(map[string]any{
			"missing": missing,
		})
	}
	if isCommonPassword(password) {
		return apperrors.ErrPasswordTooCommon
	}
	return nil
}

// commonPasswordWords 弱口令字典词根：去掉数字/符号（及还原 l33t 替换）后等于这些词的密码视为弱口令。
var commonPasswordWords = map[string]struct{}{}

func init() {
	for _, w := range []string{
		"password", "passwd", "pass", "pwd", "admin", "administrator", "root", "super", "superadmin",
		"system", "sys", "manager", "user", "guest", "test", "demo", "default", "login", "welcome",
		"changeme", "letmein", "iloveyou", "qwerty", "qwer", "qwert", "asdf", "asdfgh", "zxcv", "zxcvbn",
		"qaz", "qazwsx", "abc", "abcd", "abcde", "abcdef", "aaa", "aaaa", "woaini", "goravel", "developer",
	} {
		commonPasswordWords[w] = struct{}{}
	}
}

var leetReplacer = strings.NewReplacer("@", "a", "4", "a", "0", "o", "1", "i", "!", "i", "3", "e", "$", "s", "5", "s", "7", "t")

func isCommonPassword(password string) bool {
	lower := strings.ToLower(password)
	// 首尾的数字/符号是常见的“凑复杂度”后缀，只对中间部分做 l33t 还原（P@ssw0rd1 → password）。
	inner := strings.TrimFunc(lower, func(r rune) bool { return !unicode.IsLetter(r) })
	for _, candidate := range []string{lettersOnly(lower), lettersOnly(leetReplacer.Replace(inner))} {
		if _, ok := commonPasswordWords[candidate]; ok && candidate != "" {
			return true
		}
	}
	// 0~2 个字母 + 键盘数字序列，如 12345678、Aa123456!、Q1234567!
	return len([]rune(lettersOnly(lower))) <= 2 && hasSequentialDigits(lower)
}

func lettersOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// hasSequentialDigits 是否含 4 位以上连续递增/递减或相同的数字串（1234、4321、8888）。
func hasSequentialDigits(s string) bool {
	const unset = 2 // step 只会是 -1/0/1
	run, step := 0, unset
	prev := rune(-1)
	for _, r := range s {
		if r < '0' || r > '9' {
			run, prev = 0, -1
			continue
		}
		if prev < 0 {
			run, step = 1, unset
		} else {
			switch d := int(r - prev); {
			case d < -1 || d > 1:
				run, step = 1, unset
			case step == unset || d == step:
				run, step = run+1, d
			default:
				run, step = 2, d
			}
		}
		prev = r
		if run >= 4 {
			return true
		}
	}
	return false
}

// passwordContainsUsername 密码包含用户名（忽略大小写，用户名至少 3 个字符时检查）。
func passwordContainsUsername(password, username string) bool {
	u := strings.ToLower(strings.TrimSpace(username))
	if len([]rune(u)) < 3 {
		return false
	}
	p := strings.ToLower(password)
	return strings.Contains(p, u) || strings.Contains(leetReplacer.Replace(p), u)
}
