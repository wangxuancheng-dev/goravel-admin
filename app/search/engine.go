package search

import (
	"context"
	"errors"
)

// 驱动名常量。
const (
	DriverElasticsearch = "elasticsearch"
	DriverMeilisearch   = "meilisearch"
	DriverNull          = "null"
)

// ErrUnsupported 表示当前驱动尚未实现某能力（如 Meilisearch 骨架）。
var ErrUnsupported = errors.New("search: operation not supported by current driver")

// Engine 可切换的搜索引擎抽象（类似 Laravel Scout Engine）。
type Engine interface {
	Name() string
	Ping(ctx context.Context) error
	Index(ctx context.Context, index, documentID string, document map[string]any) error
	Delete(ctx context.Context, index, documentID string) error
	Search(ctx context.Context, index string, req SearchRequest) (*SearchResult, error)
	// EnsureIndex 按驱动语义确保索引存在（ES 建 mapping；Meili 建 index + settings）。
	EnsureIndex(ctx context.Context, index string) error
}

// Filter 通用过滤条件。
type Filter struct {
	Field string
	Op    string // term | gte | lte
	Value any
}

// SearchRequest 引擎无关的检索请求。
type SearchRequest struct {
	Keyword      string
	SearchFields []string // 如 order_no^2, product_names
	Filters      []Filter
	Page         int
	PageSize     int
	SortField    string
	SortDesc     bool
}

// SearchResult 引擎无关的检索结果；Hits 为原始文档 map。
type SearchResult struct {
	Total int64
	Hits  []map[string]any
}
