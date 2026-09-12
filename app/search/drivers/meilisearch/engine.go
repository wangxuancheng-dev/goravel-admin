package meilisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/goravel/framework/contracts/config"
	meili "github.com/meilisearch/meilisearch-go"

	"goravel/app/search"
)

// Engine is a production Meilisearch driver (switch via SEARCH_DRIVER=meilisearch).
type Engine struct {
	client meili.ServiceManager
	cfg    config.Config
	host   string
	apiKey string
	prefix string

	ensureMu   sync.Mutex
	ensureDone sync.Map // full index uid -> struct{}
}

// NewEngine creates a Meilisearch engine from config.
func NewEngine(cfg config.Config) (*Engine, error) {
	host := strings.TrimSpace(cfg.GetString("meilisearch.host", "http://127.0.0.1:7700"))
	if host == "" {
		return nil, fmt.Errorf("meilisearch.host is empty")
	}
	apiKey := cfg.GetString("meilisearch.api_key", "")
	opts := []meili.Option{}
	if strings.TrimSpace(apiKey) != "" {
		opts = append(opts, meili.WithAPIKey(apiKey))
	}
	client := meili.New(host, opts...)
	return &Engine{
		client: client,
		cfg:    cfg,
		host:   host,
		apiKey: apiKey,
		prefix: cfg.GetString("meilisearch.index_prefix", ""),
	}, nil
}

// NewEngineWithClient injects a client (tests).
func NewEngineWithClient(cfg config.Config, client meili.ServiceManager) *Engine {
	return &Engine{
		client: client,
		cfg:    cfg,
		host:   cfg.GetString("meilisearch.host", ""),
		apiKey: cfg.GetString("meilisearch.api_key", ""),
		prefix: cfg.GetString("meilisearch.index_prefix", ""),
	}
}

func (e *Engine) Name() string { return search.DriverMeilisearch }

func (e *Engine) QueryReady() bool { return e != nil && e.client != nil }

func (e *Engine) fullIndex(index string) string {
	return e.prefix + index
}

func (e *Engine) Ping(ctx context.Context) error {
	if e.client == nil {
		return fmt.Errorf("meilisearch client nil")
	}
	_, err := e.client.HealthWithContext(ctx)
	return err
}

func (e *Engine) Index(ctx context.Context, index, documentID string, document map[string]any) error {
	if err := e.EnsureIndex(ctx, index); err != nil {
		return err
	}
	uid := e.fullIndex(index)
	def, _ := search.MatchResource(index)
	primary := ""
	if d, ok := search.Definition(def); ok {
		primary = d.PrimaryKey
		if documentID == "" {
			id, err := search.DocumentID(d, document)
			if err != nil {
				return err
			}
			documentID = id
		}
		// Ensure primary key field is present for Meili.
		if primary != "" {
			if _, ok := document[primary]; !ok {
				document[primary] = documentID
			}
		}
	}
	idx := e.client.Index(uid)
	var err error
	if primary != "" {
		_, err = idx.AddDocumentsWithContext(ctx, []map[string]any{document}, primary)
	} else {
		_, err = idx.AddDocumentsWithContext(ctx, []map[string]any{document})
	}
	return err
}

func (e *Engine) Delete(ctx context.Context, index, documentID string) error {
	uid := e.fullIndex(index)
	_, err := e.client.Index(uid).DeleteDocumentWithContext(ctx, documentID)
	return err
}

func (e *Engine) EnsureIndex(ctx context.Context, index string) error {
	index = strings.TrimSpace(index)
	if index == "" {
		return fmt.Errorf("meilisearch index name empty")
	}
	uid := e.fullIndex(index)
	if _, ok := e.ensureDone.Load(uid); ok {
		return nil
	}
	e.ensureMu.Lock()
	defer e.ensureMu.Unlock()
	if _, ok := e.ensureDone.Load(uid); ok {
		return nil
	}

	resource, ok := search.MatchResource(index)
	primary := "id"
	var searchable, filterable, sortable []string
	if ok {
		if def, ok := search.Definition(resource); ok {
			primary = def.PrimaryKey
			searchable = def.Searchable
			filterable = def.Filterable
			sortable = def.Sortable
		}
	}

	if _, err := e.client.GetIndexWithContext(ctx, uid); err != nil {
		if _, err := e.client.CreateIndexWithContext(ctx, &meili.IndexConfig{
			Uid:        uid,
			PrimaryKey: primary,
		}); err != nil {
			// Race / already exists — continue to settings.
			if !strings.Contains(strings.ToLower(err.Error()), "already") {
				// Still try settings; GetIndex may succeed now.
				if _, getErr := e.client.GetIndexWithContext(ctx, uid); getErr != nil {
					return fmt.Errorf("meilisearch create index %s: %w", uid, err)
				}
			}
		}
	}

	settings := &meili.Settings{}
	if len(searchable) > 0 {
		settings.SearchableAttributes = searchable
	}
	if len(filterable) > 0 {
		settings.FilterableAttributes = filterable
	}
	if len(sortable) > 0 {
		settings.SortableAttributes = sortable
	}
	if len(searchable) > 0 || len(filterable) > 0 || len(sortable) > 0 {
		if _, err := e.client.Index(uid).UpdateSettingsWithContext(ctx, settings); err != nil {
			return fmt.Errorf("meilisearch update settings %s: %w", uid, err)
		}
	}
	e.ensureDone.Store(uid, struct{}{})
	return nil
}

func (e *Engine) Search(ctx context.Context, index string, req search.SearchRequest) (*search.SearchResult, error) {
	uid := e.fullIndex(index)
	page := req.Page
	if page < 1 {
		page = 1
	}
	size := req.PageSize
	if size < 1 {
		size = 15
	}

	mreq := &meili.SearchRequest{
		Offset: int64((page - 1) * size),
		Limit:  int64(size),
	}
	if len(req.SearchFields) > 0 {
		mreq.AttributesToSearchOn = req.SearchFields
	}
	if filter := buildMeiliFilter(req.Filters); filter != "" {
		mreq.Filter = filter
	}
	if req.SortField != "" {
		order := "asc"
		if req.SortDesc {
			order = "desc"
		}
		mreq.Sort = []string{req.SortField + ":" + order}
	}

	res, err := e.client.Index(uid).SearchWithContext(ctx, strings.TrimSpace(req.Keyword), mreq)
	if err != nil {
		return nil, err
	}

	hits := make([]map[string]any, 0, len(res.Hits))
	for _, raw := range res.Hits {
		switch h := raw.(type) {
		case map[string]any:
			hits = append(hits, h)
		default:
			b, err := json.Marshal(h)
			if err != nil {
				continue
			}
			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				continue
			}
			hits = append(hits, m)
		}
	}
	total := res.TotalHits
	if total == 0 {
		total = res.EstimatedTotalHits
	}
	return &search.SearchResult{Total: total, Hits: hits}, nil
}

func buildMeiliFilter(filters []search.Filter) string {
	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		field := strings.TrimSpace(f.Field)
		if field == "" {
			continue
		}
		switch strings.ToLower(f.Op) {
		case "term":
			parts = append(parts, field+" = "+meiliLiteral(f.Value))
		case "gte":
			parts = append(parts, field+" >= "+meiliLiteral(f.Value))
		case "lte":
			parts = append(parts, field+" <= "+meiliLiteral(f.Value))
		}
	}
	return strings.Join(parts, " AND ")
}

func meiliLiteral(v any) string {
	switch x := v.(type) {
	case string:
		escaped := strings.ReplaceAll(x, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		return fmt.Sprint(x)
	}
}

func (e *Engine) Host() string                 { return e.host }
func (e *Engine) APIKey() string               { return e.apiKey }
func (e *Engine) FullIndexName(short string) string { return e.fullIndex(short) }
func (e *Engine) Client() meili.ServiceManager { return e.client }
