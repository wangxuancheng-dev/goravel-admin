package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/facades"

	"goravel/app/binding"
	"goravel/app/clients"
	"goravel/app/search"
)

// Engine Elasticsearch 驱动。
type Engine struct {
	client *elasticsearch.Client
	cfg    config.Config
}

// NewEngine 从容器取 ES 客户端；失败则尝试新建。
func NewEngine(cfg config.Config) (*Engine, error) {
	raw, err := facades.App().Make(binding.ElasticsearchClient)
	if err == nil {
		if c, ok := raw.(*elasticsearch.Client); ok && c != nil {
			return &Engine{client: c, cfg: cfg}, nil
		}
	}
	c, err := clients.NewElasticsearchClient(cfg, "")
	if err != nil {
		return nil, err
	}
	return &Engine{client: c, cfg: cfg}, nil
}

// NewEngineWithClient 注入已有客户端（测试用）。
func NewEngineWithClient(cfg config.Config, c *elasticsearch.Client) *Engine {
	return &Engine{client: c, cfg: cfg}
}

func (e *Engine) Name() string { return search.DriverElasticsearch }

func (e *Engine) Ping(ctx context.Context) error {
	res, err := e.client.Ping(e.client.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch ping: %s", res.Status())
	}
	return nil
}

func (e *Engine) fullIndex(index string) string {
	return clients.ElasticsearchIndexName(e.cfg, index)
}

func (e *Engine) Index(ctx context.Context, index, documentID string, document map[string]any) error {
	return clients.ElasticsearchIndexValue(ctx, e.client, e.fullIndex(index), documentID, document)
}

func (e *Engine) Delete(ctx context.Context, index, documentID string) error {
	return clients.ElasticsearchDeleteDocument(ctx, e.client, e.fullIndex(index), documentID)
}

func (e *Engine) EnsureIndex(ctx context.Context, index string) error {
	if search.IsOrdersIndexShortName(index) {
		return EnsureOrdersIndexNamed(ctx, e, index)
	}
	// Other registered resources: module code should create mapping (or use Meili auto-settings).
	// Unknown / unregistered: only verify the index already exists.
	full := e.fullIndex(index)
	res, err := e.client.Indices.Exists([]string{full}, e.client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	if res != nil {
		defer res.Body.Close()
		if res.StatusCode == 200 {
			return nil
		}
	}
	return fmt.Errorf("elasticsearch index %s does not exist", full)
}

func (e *Engine) Search(ctx context.Context, index string, req search.SearchRequest) (*search.SearchResult, error) {
	filters := make([]any, 0, len(req.Filters))
	rangeAcc := map[string]map[string]any{}

	for _, f := range req.Filters {
		switch strings.ToLower(f.Op) {
		case "term":
			filters = append(filters, map[string]any{"term": map[string]any{f.Field: f.Value}})
		case "gte", "lte":
			if rangeAcc[f.Field] == nil {
				rangeAcc[f.Field] = map[string]any{}
			}
			rangeAcc[f.Field][f.Op] = f.Value
		}
	}
	for field, rng := range rangeAcc {
		filters = append(filters, map[string]any{"range": map[string]any{field: rng}})
	}

	boolQ := map[string]any{"filter": filters}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		fields := buildElasticsearchSearchFields(req.SearchFields, req.FieldBoosts)
		if len(fields) == 0 {
			fields = []string{"*"}
		}
		boolQ["must"] = []any{
			map[string]any{
				"multi_match": map[string]any{
					"query":  kw,
					"type":   "best_fields",
					"fields": fields,
				},
			},
		}
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	size := req.PageSize
	if size < 1 {
		size = 15
	}

	body := map[string]any{
		"query": map[string]any{"bool": boolQ},
		"from":  (page - 1) * size,
		"size":  size,
	}
	if req.SortField != "" {
		order := "asc"
		if req.SortDesc {
			order = "desc"
		}
		body["sort"] = []any{map[string]any{req.SortField: map[string]any{"order": order}}}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(e.fullIndex(index)),
		e.client.Search.WithBody(bytes.NewReader(payload)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch search: %s: %s", res.Status(), string(b))
	}

	var parsed struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	hits := make([]map[string]any, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		hits = append(hits, h.Source)
	}
	return &search.SearchResult{Total: parsed.Hits.Total.Value, Hits: hits}, nil
}

// buildElasticsearchSearchFields 将可移植字段名 + 可选加权转为 ES multi_match fields（field^boost）。
func buildElasticsearchSearchFields(fields []string, boosts map[string]float64) []string {
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for _, raw := range fields {
		field := strings.TrimSpace(raw)
		if field == "" {
			continue
		}
		// 兼容误传入的 ES 语法，剥掉已有 ^boost
		if i := strings.IndexByte(field, '^'); i >= 0 {
			field = field[:i]
		}
		if boosts != nil {
			if b, ok := boosts[field]; ok && b > 0 {
				out = append(out, fmt.Sprintf("%s^%g", field, b))
				continue
			}
		}
		out = append(out, field)
	}
	return out
}

// Client 暴露底层客户端（命令/高级用途）。
func (e *Engine) Client() *elasticsearch.Client { return e.client }

// FullIndexName 带前缀的索引名。
func (e *Engine) FullIndexName(short string) string { return e.fullIndex(short) }
