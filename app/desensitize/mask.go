package desensitize

import (
	"strings"
	"unicode/utf8"
)

// Strategy names used in config/desensitize.modules.
const (
	StrategyPhone    = "phone"
	StrategyEmail    = "email"
	StrategyHide     = "hide"
	StrategyKeepEnds = "keep_ends"
)

// Mask applies a named strategy to a plaintext value.
func Mask(value, strategy string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case StrategyPhone:
		return MaskPhone(value)
	case StrategyEmail:
		return MaskEmail(value)
	case StrategyHide:
		return "***"
	case StrategyKeepEnds:
		return MaskKeepEnds(value, 1, 1)
	default:
		return MaskKeepEnds(value, 1, 1)
	}
}

// MaskPhone keeps a readable prefix/suffix for common mobile lengths.
func MaskPhone(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	runes := []rune(raw)
	n := len(runes)
	switch {
	case n >= 11:
		return string(runes[:3]) + "****" + string(runes[n-4:])
	case n >= 7:
		return string(runes[:2]) + "****" + string(runes[n-2:])
	case n >= 3:
		return string(runes[:1]) + "***"
	default:
		return "***"
	}
}

// MaskEmail keeps first local char and full domain: a***@example.com
func MaskEmail(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	at := strings.Index(raw, "@")
	if at < 0 {
		return MaskKeepEnds(raw, 1, 0)
	}
	local := raw[:at]
	domain := raw[at:]
	if local == "" {
		return "***" + domain
	}
	first, _ := utf8.DecodeRuneInString(local)
	if first == utf8.RuneError {
		return "***" + domain
	}
	return string(first) + "***" + domain
}

// MaskKeepEnds keeps prefixLen leading and suffixLen trailing runes.
func MaskKeepEnds(raw string, prefixLen, suffixLen int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	runes := []rune(raw)
	n := len(runes)
	if prefixLen < 0 {
		prefixLen = 0
	}
	if suffixLen < 0 {
		suffixLen = 0
	}
	if prefixLen+suffixLen >= n {
		return "***"
	}
	return string(runes[:prefixLen]) + "***" + string(runes[n-suffixLen:])
}

// LooksMasked reports values that already contain a mask marker (edit-form guard).
func LooksMasked(value string) bool {
	return strings.Contains(value, "*")
}
