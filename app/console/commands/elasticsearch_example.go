package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/binding"
	"goravel/app/clients"
	"goravel/app/search"
)

// ElasticsearchExample 演示 Ping / 写入 / 搜索 / 删除（需 SEARCH_DRIVER=elasticsearch）。
type ElasticsearchExample struct{}

func (r *ElasticsearchExample) Signature() string {
	return "search:es-example"
}

func (r *ElasticsearchExample) Description() string {
	return "Elasticsearch 驱动示例：ping | index | search | delete"
}

func (r *ElasticsearchExample) Extend() command.Extend {
	return command.Extend{
		Category: "search",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "op",
				Aliases: []string{"o"},
				Usage:   "操作: ping（默认）| index | search | delete",
				Value:   "ping",
			},
			&command.StringFlag{
				Name:  "id",
				Usage: "index/delete 时的文档 ID，默认 demo-1",
				Value: "demo-1",
			},
			&command.StringFlag{
				Name:  "q",
				Usage: "search 时的关键词，默认 示例",
				Value: "示例",
			},
		},
	}
}

func (r *ElasticsearchExample) Handle(ctx console.Context) error {
	if !search.Enabled() || search.Driver() != search.DriverElasticsearch {
		ctx.Warning("请设置 SEARCH_ENABLED=true 且 SEARCH_DRIVER=elasticsearch。")
		return nil
	}

	es, err := resolveElasticsearchExampleClient()
	if err != nil {
		ctx.Error(fmt.Sprintf("无法解析 ES 客户端: %v", err))
		return err
	}

	op := ctx.Option("op")
	id := ctx.Option("id")
	q := ctx.Option("q")
	runCtx := context.Background()
	index := elasticsearchExampleIndex()

	switch op {
	case "ping", "":
		res, err := es.Ping(es.Ping.WithContext(runCtx))
		if err != nil {
			ctx.Error(err.Error())
			return err
		}
		if res != nil {
			defer res.Body.Close()
			if res.IsError() {
				err := fmt.Errorf("ping: %s", res.Status())
				ctx.Error(err.Error())
				return err
			}
		}
		ctx.Success("ping ok")
	case "index":
		doc := map[string]any{
			"title":   "Goravel 搜索示例",
			"content": "这是一条写入 Elasticsearch 的演示文档",
		}
		if err := clients.ElasticsearchIndexValue(runCtx, es, index, id, doc); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("indexed id=" + id + " index=" + index)
	case "search":
		raw, err := elasticsearchExampleSearchMatch(runCtx, es, index, "content", q, 10)
		if err != nil {
			ctx.Error(err.Error())
			return err
		}
		var parsed struct {
			Hits struct {
				Total struct {
					Value int64 `json:"value"`
				} `json:"total"`
			} `json:"hits"`
		}
		_ = json.Unmarshal(raw, &parsed)
		ctx.Success(fmt.Sprintf("search q=%q hits=%d", q, parsed.Hits.Total.Value))
	case "delete":
		if err := clients.ElasticsearchDeleteDocument(runCtx, es, index, id); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("deleted id=" + id)
	default:
		ctx.Error("unknown op: " + op)
	}
	return nil
}

func resolveElasticsearchExampleClient() (*elasticsearch.Client, error) {
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

func elasticsearchExampleIndex() string {
	short := facades.Config().GetString("elasticsearch.demo_index", "goravel_demo")
	return clients.ElasticsearchIndexName(facades.Config(), short)
}

func elasticsearchExampleSearchMatch(ctx context.Context, es *elasticsearch.Client, index, field, query string, size int) (json.RawMessage, error) {
	if size <= 0 {
		size = 10
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
