package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/goravel/framework/facades"

	"goravel/app/search"
)

var ensureIndexOnce sync.Once
var ensureIndexErr error

// EnsureOrdersIndex 创建订单索引（若不存在）；进程内只执行一次。
func EnsureOrdersIndex(ctx context.Context, e *Engine) error {
	ensureIndexOnce.Do(func() {
		ensureIndexErr = initOrdersIndex(ctx, e)
	})
	return ensureIndexErr
}

// InitOrdersIndex 供 artisan 显式初始化（可重复调用）。
func InitOrdersIndex(ctx context.Context, e *Engine) error {
	return initOrdersIndex(ctx, e)
}

func initOrdersIndex(ctx context.Context, e *Engine) error {
	if e == nil || e.client == nil {
		return fmt.Errorf("elasticsearch client not available")
	}
	index := e.fullIndex(search.OrdersIndexShortName())
	res, err := e.client.Indices.Exists([]string{index}, e.client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	if res != nil {
		defer res.Body.Close()
		if res.StatusCode == 200 {
			return nil
		}
	}

	indexAnalyzer, searchAnalyzer := ResolveOrdersTextAnalyzers(ctx, e.client)
	body, err := json.Marshal(ordersIndexBody(indexAnalyzer, searchAnalyzer))
	if err != nil {
		return err
	}
	createRes, err := e.client.Indices.Create(
		index,
		e.client.Indices.Create.WithContext(ctx),
		e.client.Indices.Create.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return err
	}
	defer createRes.Body.Close()
	if createRes.IsError() {
		b, _ := io.ReadAll(createRes.Body)
		if createRes.StatusCode == 400 && bytes.Contains(b, []byte("resource_already_exists")) {
			return nil
		}
		return fmt.Errorf("es create index %s: %s: %s", index, createRes.Status(), string(b))
	}
	return nil
}

func ordersIndexBody(indexAnalyzer, searchAnalyzer string) map[string]any {
	textMapping := textFieldMapping(indexAnalyzer, searchAnalyzer)
	return map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"id":            map[string]any{"type": "long"},
				"order_no":      map[string]any{"type": "keyword"},
				"user_id":       map[string]any{"type": "long"},
				"amount":        map[string]any{"type": "double"},
				"status":        map[string]any{"type": "keyword"},
				"remark":        textMapping,
				"product_names": textMapping,
				"created_at": map[string]any{
					"type":   "date",
					"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||strict_date_optional_time||epoch_millis",
				},
				"updated_at": map[string]any{
					"type":   "date",
					"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||strict_date_optional_time||epoch_millis",
				},
			},
		},
	}
}

// ResolveOrdersTextAnalyzers 解析订单文本分词器。
func ResolveOrdersTextAnalyzers(ctx context.Context, es *elasticsearch.Client) (indexAnalyzer, searchAnalyzer string) {
	mode := strings.ToLower(strings.TrimSpace(facades.Config().GetString("elasticsearch.orders_analyzer", "auto")))
	searchOverride := strings.TrimSpace(facades.Config().GetString("elasticsearch.orders_search_analyzer", ""))

	switch mode {
	case "", "standard", "default":
		return "", ""
	case "auto":
		if es != nil && analyzerAvailable(ctx, es, "ik_max_word") {
			search := "ik_smart"
			if searchOverride != "" {
				search = searchOverride
			}
			facades.Log().Info("elasticsearch orders index: using IK analyzers (ik_max_word / " + search + ")")
			return "ik_max_word", search
		}
		facades.Log().Info("elasticsearch orders index: IK not available, using default standard analyzer")
		return "", ""
	default:
		index := mode
		searchA := searchOverride
		if searchA == "" {
			if index == "ik_max_word" {
				searchA = "ik_smart"
			} else {
				searchA = index
			}
		}
		if es != nil && !analyzerAvailable(ctx, es, index) {
			facades.Log().Warningf("elasticsearch orders analyzer %q unavailable, fallback to standard", index)
			return "", ""
		}
		facades.Log().Infof("elasticsearch orders index: using analyzers (%s / %s)", index, searchA)
		return index, searchA
	}
}

func analyzerAvailable(ctx context.Context, es *elasticsearch.Client, analyzer string) bool {
	if es == nil || strings.TrimSpace(analyzer) == "" {
		return false
	}
	payload, err := json.Marshal(map[string]any{
		"analyzer": analyzer,
		"text":     "中文分词检测",
	})
	if err != nil {
		return false
	}
	res, err := es.Indices.Analyze(
		es.Indices.Analyze.WithContext(ctx),
		es.Indices.Analyze.WithBody(bytes.NewReader(payload)),
	)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	if res.IsError() {
		return false
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return false
	}
	var parsed struct {
		Tokens []struct {
			Token string `json:"token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return false
	}
	return len(parsed.Tokens) > 0
}

func textFieldMapping(indexAnalyzer, searchAnalyzer string) map[string]any {
	m := map[string]any{"type": "text"}
	if indexAnalyzer != "" {
		m["analyzer"] = indexAnalyzer
	}
	if searchAnalyzer != "" {
		m["search_analyzer"] = searchAnalyzer
	}
	return m
}
