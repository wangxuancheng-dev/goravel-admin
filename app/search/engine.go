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

// ErrUnsupported means the current driver does not implement an operation (caller may fall back to DB).
var ErrUnsupported = errors.New("search: operation not supported by current driver")

// Engine is a swappable search backend (Scout-style).
// Package boundary (extractable later as a Goravel package):
//   - Engine + Resolve + indexes/config/outbox live here; SDK clients in drivers/*.
//   - Domain packages must NOT import app/services; order DB access via SetOrderLoaders.
//   - App wires engine factory + loaders via providers.SearchServiceProvider.
//   - search/orders is an optional adapter (document/Push/Query) for this admin app.
type Engine interface {
	Name() string
	Ping(ctx context.Context) error
	Index(ctx context.Context, index, documentID string, document map[string]any) error
	Delete(ctx context.Context, index, documentID string) error
	Search(ctx context.Context, index string, req SearchRequest) (*SearchResult, error)
	// EnsureIndex 按驱动语义确保索引存在（ES 建 mapping；Meili 建 index + settings）。
	EnsureIndex(ctx context.Context, index string) error
}

// Filter 通用过滤条件（驱动负责翻译；业务层勿写引擎专属 DSL）。
type Filter struct {
	Field string
	Op    string // 可移植操作：term | gte | lte
	Value any
}

// SearchRequest 引擎无关的检索请求。
// SearchFields 仅为字段名（如 order_no），不要写 ES 的 order_no^2；加权用 FieldBoosts。
type SearchRequest struct {
	Keyword      string
	SearchFields []string           // 纯字段名，跨驱动可移植
	FieldBoosts  map[string]float64 // 可选；仅支持加权的驱动（如 ES）使用
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

// QueryReadyEngine 可选能力：驱动声明全文/筛选检索是否已真正可用。
type QueryReadyEngine interface {
	QueryReady() bool
}


// EngineQueryReady 判断引擎是否可用于订单等业务检索。
func EngineQueryReady(e Engine) bool {
	if e == nil || e.Name() == DriverNull {
		return false
	}
	if q, ok := e.(QueryReadyEngine); ok {
		return q.QueryReady()
	}
	return true
}
