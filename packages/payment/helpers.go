package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseConfigJSON unmarshals payment method config JSON into a map.
func ParseConfigJSON(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigRequired, err)
	}
	if config == nil {
		config = map[string]any{}
	}
	return config, nil
}

// StringFromConfig returns a string config value.
func StringFromConfig(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	v, _ := config[key].(string)
	return v
}

// FirstString returns the first non-empty string among keys in data.
func FirstString(data map[string]any, keys ...string) string {
	if data == nil {
		return ""
	}
	for _, k := range keys {
		v, ok := data[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			if s := strings.TrimSpace(t); s != "" {
				return s
			}
		case float64:
			// 'f', -1 keeps decimals ("12.5", not "12") and never uses exponent form.
			// Integers above 2^53 are already rounded when JSON is decoded into float64:
			// decode with json.Decoder.UseNumber, or have the provider send them as strings.
			return strconv.FormatFloat(t, 'f', -1, 64)
		case float32:
			return strconv.FormatFloat(float64(t), 'f', -1, 32)
		case int:
			return strconv.Itoa(t)
		case int64:
			return strconv.FormatInt(t, 10)
		case uint64:
			return strconv.FormatUint(t, 10)
		case json.Number:
			return t.String()
		}
	}
	return ""
}

// BoolFromConfig reads a flag that may be stored as a bool, number or string
// ("true", "1", "yes", "on"). Missing or anything else is false.
func BoolFromConfig(config map[string]any, key string) bool {
	if config == nil {
		return false
	}
	switch t := config[key].(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1", "yes", "on":
			return true
		}
	}
	return false
}

// OptionalFloat returns the first parseable float among keys in data.
func OptionalFloat(data map[string]any, keys ...string) *float64 {
	if data == nil {
		return nil
	}
	for _, k := range keys {
		v, ok := data[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case float64:
			return &t
		case float32:
			f := float64(t)
			return &f
		case int:
			f := float64(t)
			return &f
		case int64:
			f := float64(t)
			return &f
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
			if err == nil {
				return &f
			}
		case json.Number:
			f, err := t.Float64()
			if err == nil {
				return &f
			}
		}
	}
	return nil
}

// DefaultNotifyURL returns ResolveNotifyURL(ctx, gatewayType) when wired by the host,
// or "" when it is not. Use RequireNotifyURL for drivers that send the URL to a provider.
func DefaultNotifyURL(ctx context.Context, gatewayType string) string {
	if ResolveNotifyURL != nil {
		return strings.TrimSpace(ResolveNotifyURL(ctx, gatewayType))
	}
	return ""
}

// RequireNotifyURL returns config["notify_url"] when set, else the host's ResolveNotifyURL.
// It fails with ErrConfigRequired instead of returning "", so a host that forgot to wire
// ResolveNotifyURL never places provider orders with an empty callback address.
func RequireNotifyURL(ctx context.Context, gatewayType string, config map[string]any) (string, error) {
	if v := strings.TrimSpace(StringFromConfig(config, "notify_url")); v != "" {
		return v, nil
	}
	if ResolveNotifyURL == nil {
		return "", fmt.Errorf("%w: notify_url is empty and payment.ResolveNotifyURL is not wired", ErrConfigRequired)
	}
	if v := DefaultNotifyURL(ctx, gatewayType); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("%w: payment.ResolveNotifyURL returned an empty notify_url for %q", ErrConfigRequired, gatewayType)
}
