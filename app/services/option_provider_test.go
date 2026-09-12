package services

import (
	"context"
	"testing"

	"github.com/goravel/framework/contracts/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOptionProvider struct {
	name string
}

func (p *stubOptionProvider) GetOptions(http.Context) (map[string]any, error) {
	return map[string]any{"options": []map[string]any{{"label": p.name, "value": p.name}}}, nil
}

func TestRegisterOptionProviderLookup(t *testing.T) {
	const typ = "demo_option_ext"
	RegisterOptionProvider(typ, func(ctx context.Context) OptionProvider {
		return &stubOptionProvider{name: typ}
	})
	t.Cleanup(func() {
		optionProviderMu.Lock()
		delete(optionProviderFactories, typ)
		optionProviderMu.Unlock()
	})

	p, ok := LookupOptionProvider(context.Background(), typ)
	require.True(t, ok)
	require.NotNil(t, p)

	data, err := p.GetOptions(nil)
	require.NoError(t, err)
	assert.Equal(t, typ, data["options"].([]map[string]any)[0]["value"])

	_, ok = LookupOptionProvider(context.Background(), "missing_option_type_xyz")
	assert.False(t, ok)
}
