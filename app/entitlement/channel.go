package entitlement

import "strings"

// Client labels used when building ClientView (informational only).
const (
	ChannelAdmin = "admin"
	ChannelPC    = "pc"
	ChannelH5    = "h5"
)

// NormalizeChannel returns a known client label or empty string.
func NormalizeChannel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ChannelAdmin:
		return ChannelAdmin
	case ChannelPC, "desktop":
		return ChannelPC
	case ChannelH5, "mobile", "wap":
		return ChannelH5
	default:
		return ""
	}
}
