package esdemo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/goravel/framework/facades"

	"goravel/app/binding"
	"goravel/app/clients"
)

// Client 解析容器中的 ES 客户端（仅供 CLI 演示与本地排查）。
func Client() (*elasticsearch.Client, error) {
	raw, err := facades.App().Make(binding.ElasticsearchClient)
	if err != nil {
		return nil, err
	}
	c, ok := raw.(*elasticsearch.Client)
	if !ok || c == nil {
		return nil, fmt.Errorf("elasticsearch 客户端未注册：请设置 SEARCH_ENABLED=true 且 SEARCH_DRIVER=elasticsearch")
	}
	return c, nil
}

// DemoIndex 演示索引全名（含 index_prefix）。
func DemoIndex() string {
	short := facades.Config().GetString("elasticsearch.demo_index", "goravel_demo")
	return clients.ElasticsearchIndexName(facades.Config(), short)
}

// Ping 集群连通性。
func Ping(ctx context.Context) error {
	es, err := Client()
	if err != nil {
		return err
	}
	res, err := es.Ping(es.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	if res != nil {
		defer res.Body.Close()
	}
	if res != nil && res.IsError() {
		return fmt.Errorf("ping: %s", res.Status())
	}
	return nil
}

// IndexDocument 写入演示文档。
func IndexDocument(ctx context.Context, index, documentID string, doc any) error {
	es, err := Client()
	if err != nil {
		return err
	}
	return clients.ElasticsearchIndexValue(ctx, es, index, documentID, doc)
}

// DeleteDocument 删除演示文档。
func DeleteDocument(ctx context.Context, index, documentID string) error {
	es, err := Client()
	if err != nil {
		return err
	}
	return clients.ElasticsearchDeleteDocument(ctx, es, index, documentID)
}

// SearchMatch 简单 match 查询。
func SearchMatch(ctx context.Context, index, field, query string, size int) (json.RawMessage, error) {
	if size <= 0 {
		size = 10
	}
	es, err := Client()
	if err != nil {
		return nil, err
	}
	q := map[string]any{
		"query": map[string]any{
			"match": map[string]any{field: query},
		},
		"size": size,
	}
	body, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}
	res, err := es.Search(
		es.Search.WithContext(ctx),
		es.Search.WithIndex(index),
		es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.IsError() {
		return nil, fmt.Errorf("search: %s: %s", res.Status(), strings.TrimSpace(string(raw)))
	}
	return raw, nil
}
