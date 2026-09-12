package services

import (
	"context"
	"strings"
	"sync"

	"github.com/goravel/framework/contracts/http"
)

// OptionProvider 选项提供者接口
// 所有选项提供者都需要实现此接口
type OptionProvider interface {
	// GetOptions 获取选项列表
	// 返回的 map 应该包含 "options" 键，值为选项数组
	// 可以包含其他额外的数据，如 "list" 等
	GetOptions(ctx http.Context) (map[string]any, error)
}

// OptionProviderFactory builds a request-scoped OptionProvider.
type OptionProviderFactory func(ctx context.Context) OptionProvider

var (
	optionProviderMu        sync.RWMutex
	optionProviderFactories = map[string]OptionProviderFactory{}
)

// RegisterOptionProvider registers a factory for /api/admin/options?type=...
// Call from init() in a provider package. Built-in types may also be registered this way.
func RegisterOptionProvider(optionType string, factory OptionProviderFactory) {
	optionType = strings.TrimSpace(optionType)
	if optionType == "" || factory == nil {
		return
	}
	optionProviderMu.Lock()
	defer optionProviderMu.Unlock()
	optionProviderFactories[optionType] = factory
}

// LookupOptionProvider returns a registered provider for type, if any.
func LookupOptionProvider(ctx context.Context, optionType string) (OptionProvider, bool) {
	optionType = strings.TrimSpace(optionType)
	optionProviderMu.RLock()
	factory, ok := optionProviderFactories[optionType]
	optionProviderMu.RUnlock()
	if !ok || factory == nil {
		return nil, false
	}
	p := factory(ctx)
	if p == nil {
		return nil, false
	}
	return p, true
}
