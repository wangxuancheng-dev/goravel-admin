package meilisearch

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/config"

	"goravel/app/search"
)

// Engine Meilisearch 驱动骨架。切换 SEARCH_DRIVER=meilisearch 后在此接入官方 SDK。
type Engine struct {
	cfg    config.Config
	host   string
	apiKey string
	prefix string
}

// NewEngine 创建 Meilisearch 引擎（当前为骨架：写入/检索返回 ErrUnsupported）。
func NewEngine(cfg config.Config) (*Engine, error) {
	return &Engine{
		cfg:    cfg,
		host:   cfg.GetString("meilisearch.host", "http://127.0.0.1:7700"),
		apiKey: cfg.GetString("meilisearch.api_key", ""),
		prefix: cfg.GetString("meilisearch.index_prefix", ""),
	}, nil
}

func (e *Engine) Name() string { return search.DriverMeilisearch }

func (e *Engine) fullIndex(index string) string {
	return e.prefix + index
}

func (e *Engine) Ping(ctx context.Context) error {
	return fmt.Errorf("%w: meilisearch ping（请安装 meilisearch-go 并实现本驱动）", search.ErrUnsupported)
}

func (e *Engine) Index(ctx context.Context, index, documentID string, document map[string]any) error {
	return fmt.Errorf("%w: meilisearch Index", search.ErrUnsupported)
}

func (e *Engine) Delete(ctx context.Context, index, documentID string) error {
	return fmt.Errorf("%w: meilisearch Delete", search.ErrUnsupported)
}

func (e *Engine) Search(ctx context.Context, index string, req search.SearchRequest) (*search.SearchResult, error) {
	return nil, fmt.Errorf("%w: meilisearch Search", search.ErrUnsupported)
}

func (e *Engine) EnsureIndex(ctx context.Context, index string) error {
	return fmt.Errorf("%w: meilisearch EnsureIndex", search.ErrUnsupported)
}

// Host 配置的 Meili 地址（实现 SDK 时使用）。
func (e *Engine) Host() string { return e.host }

// APIKey 配置的 API Key。
func (e *Engine) APIKey() string { return e.apiKey }

// FullIndexName 带前缀索引名。
func (e *Engine) FullIndexName(short string) string { return e.fullIndex(short) }
