package search

import "context"

// nullEngine 空实现：不索引、检索返回空结果。
type nullEngine struct{}

func (nullEngine) Name() string { return DriverNull }

func (nullEngine) Ping(ctx context.Context) error { return nil }

func (nullEngine) Index(ctx context.Context, index, documentID string, document map[string]any) error {
	return nil
}

func (nullEngine) Delete(ctx context.Context, index, documentID string) error { return nil }

func (nullEngine) Search(ctx context.Context, index string, req SearchRequest) (*SearchResult, error) {
	return &SearchResult{Total: 0, Hits: nil}, nil
}

func (nullEngine) EnsureIndex(ctx context.Context, index string) error { return nil }

// NewNullEngine 供 provider / 测试使用。
func NewNullEngine() Engine { return nullEngine{} }
