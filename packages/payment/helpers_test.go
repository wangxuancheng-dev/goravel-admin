package payment

import (
	"context"
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
