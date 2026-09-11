package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/binding"
	esdemo "goravel/app/elasticsearch/demo"
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

	if _, err := facades.App().Make(binding.ElasticsearchClient); err != nil {
		ctx.Error(fmt.Sprintf("无法解析 ES 客户端: %v", err))
		return err
	}

	op := ctx.Option("op")
	id := ctx.Option("id")
	q := ctx.Option("q")
	runCtx := context.Background()
	index := esdemo.DemoIndex()

	switch op {
	case "ping", "":
		if err := esdemo.Ping(runCtx); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("ping ok")
	case "index":
		doc := map[string]any{
			"title":   "Goravel 搜索示例",
			"content": "这是一条写入 Elasticsearch 的演示文档",
		}
		if err := esdemo.IndexDocument(runCtx, index, id, doc); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("indexed id=" + id + " index=" + index)
	case "search":
		raw, err := esdemo.SearchMatch(runCtx, index, "content", q, 10)
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
		if err := esdemo.DeleteDocument(runCtx, index, id); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("deleted id=" + id)
	default:
		ctx.Error("unknown op: " + op)
	}
	return nil
}
