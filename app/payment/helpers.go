package payment

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// ParseMethodConfig unmarshals payment_methods.config JSON.
func ParseMethodConfig(pm *models.PaymentMethod) (map[string]any, error) {
	if pm == nil || strings.TrimSpace(pm.Config) == "" {
		return map[string]any{}, nil
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(pm.Config), &config); err != nil {
		return nil, apperrors.ErrPaymentConfigRequired.WithError(err)
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
			return strconv.FormatInt(int64(t), 10)
		case int:
			return strconv.Itoa(t)
		case json.Number:
			return t.String()
		}
	}
	return ""
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

// DefaultNotifyURL builds APP_URL + /api/payment/notify/{type}[/{tenant}].
func DefaultNotifyURL(ctx context.Context, notifyType string) string {
	base := strings.TrimRight(facades.Config().GetString("app.url"), "/")
	code := ""
	if ctx != nil {
		code, _ = tenancyctx.CodeFrom(ctx)
	}
	return base + tenancy.PaymentNotifyPath(code, notifyType)
}
