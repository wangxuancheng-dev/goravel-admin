package entitlement

// ClientView is the client bootstrap payload for entitlements.
type ClientView struct {
	Version  int64                `json:"version"`
	PlanCode string               `json:"plan_code,omitempty"`
	Channel  string               `json:"channel"`
	Features map[string]bool      `json:"features"`
	Limits   map[string]LimitView `json:"limits"`
}

// Present builds a ClientView from an EffectiveSet.
// channel is informational (which client asked); feature on/off is global.
func Present(set *EffectiveSet, channel string, usage map[string]int64) ClientView {
	ch := NormalizeChannel(channel)
	if ch == "" {
		ch = ChannelAdmin
	}
	view := ClientView{
		Channel:  ch,
		Features: map[string]bool{},
		Limits:   map[string]LimitView{},
	}
	if set == nil {
		return view
	}
	view.Version = set.Version
	view.PlanCode = set.PlanCode

	for key, enabled := range set.Features {
		view.Features[key] = enabled
	}
	for key, limit := range set.Limits {
		used := int64(0)
		if usage != nil {
			used = usage[key]
		}
		lv := LimitView{Used: used}
		if limit < 0 {
			lv.Unlimited = true
			lv.Remaining = -1
		} else {
			lv.Limit = limit
			rem := limit - used
			if rem < 0 {
				rem = 0
			}
			lv.Remaining = rem
		}
		view.Limits[key] = lv
	}
	return view
}
