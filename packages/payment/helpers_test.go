package payment

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withNotifyResolver(t *testing.T, fn func(context.Context, string) string) {
	t.Helper()
	prev := ResolveNotifyURL
	ResolveNotifyURL = fn
	t.Cleanup(func() { ResolveNotifyURL = prev })
}

func TestFirstStringKeepsNumbers(t *testing.T) {
	data := map[string]any{
		"dec":    12.5,
		"int":    float64(4200000001234),
		"i64":    int64(9007199254740993),
		"number": json.Number("9007199254740993"),
	}
	assert.Equal(t, "12.5", FirstString(data, "dec"))
	assert.Equal(t, "4200000001234", FirstString(data, "int"))
	assert.Equal(t, "9007199254740993", FirstString(data, "i64"))
	assert.Equal(t, "9007199254740993", FirstString(data, "number"))
}

func TestBoolFromConfig(t *testing.T) {
	for _, v := range []any{true, "true", "1", " Yes ", "on", float64(1)} {
		assert.True(t, BoolFromConfig(map[string]any{"k": v}, "k"), "%v", v)
	}
	for _, v := range []any{nil, false, "", "0", "false", "no", float64(0)} {
		assert.False(t, BoolFromConfig(map[string]any{"k": v}, "k"), "%v", v)
	}
	assert.False(t, BoolFromConfig(nil, "k"))
}

func TestRequireNotifyURLPrefersConfig(t *testing.T) {
	withNotifyResolver(t, nil)
	url, err := RequireNotifyURL(context.Background(), "wechat", map[string]any{"notify_url": " https://cfg/notify "})
	require.NoError(t, err)
	assert.Equal(t, "https://cfg/notify", url)
}

func TestRequireNotifyURLUsesResolver(t *testing.T) {
	withNotifyResolver(t, func(_ context.Context, typ string) string { return "https://host/notify/" + typ })
	url, err := RequireNotifyURL(context.Background(), "alipay", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "https://host/notify/alipay", url)
}

func TestRequireNotifyURLFailsWhenResolverMissing(t *testing.T) {
	withNotifyResolver(t, nil)
	_, err := RequireNotifyURL(context.Background(), "wechat", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrConfigRequired))
}

func TestRequireNotifyURLFailsWhenResolverReturnsEmpty(t *testing.T) {
	withNotifyResolver(t, func(context.Context, string) string { return "  " })
	_, err := RequireNotifyURL(context.Background(), "wechat", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrConfigRequired))
}
