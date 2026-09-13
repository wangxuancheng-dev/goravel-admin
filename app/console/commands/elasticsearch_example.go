package commands

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/search"
)

// ElasticsearchExample demos Ping / Index / Search / Delete via search.Engine
// (requires SEARCH_ENABLED=true and SEARCH_DRIVER=elasticsearch).
type ElasticsearchExample struct{}

func (r *ElasticsearchExample) Signature() string {
	return "search:es-example"
}

func (r *ElasticsearchExample) Description() string {
	return "Elasticsearch driver example via search.Engine: ping | index | search | delete"
}

func (r *ElasticsearchExample) Extend() command.Extend {
	return command.Extend{
		Category: "search",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "op",
				Aliases: []string{"o"},
				Usage:   "op: ping (default) | index | search | delete",
				Value:   "ping",
			},
			&command.StringFlag{
				Name:  "id",
				Usage: "document id for index/delete (default demo-1)",
				Value: "demo-1",
			},
			&command.StringFlag{
				Name:  "q",
				Usage: "search keyword (default demo)",
				Value: "demo",
			},
		},
	}
}

func (r *ElasticsearchExample) Handle(ctx console.Context) error {
	if !search.Enabled() || search.Driver() != search.DriverElasticsearch {
		ctx.Warning("Set SEARCH_ENABLED=true and SEARCH_DRIVER=elasticsearch.")
		return nil
	}

	engine, err := search.Resolve()
	if err != nil {
		ctx.Error(fmt.Sprintf("resolve engine: %v", err))
		return err
	}

	op := ctx.Option("op")
	id := ctx.Option("id")
	q := ctx.Option("q")
	runCtx := context.Background()
	index := facades.Config().GetString("elasticsearch.demo_index", "goravel_demo")

	switch op {
	case "ping", "":
		if err := engine.Ping(runCtx); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("ping ok")
	case "index":
		doc := map[string]any{
			"title":   "Goravel search demo",
			"content": "demo document for Elasticsearch driver",
		}
		if err := engine.Index(runCtx, index, id, doc); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("indexed id=" + id + " index=" + index)
	case "search":
		res, err := engine.Search(runCtx, index, search.SearchRequest{
			Keyword:      q,
			SearchFields: []string{"content", "title"},
			Page:         1,
			PageSize:     10,
		})
		if err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success(fmt.Sprintf("search q=%q hits=%d", q, res.Total))
	case "delete":
		if err := engine.Delete(runCtx, index, id); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success("deleted id=" + id)
	default:
		ctx.Error("unknown op: " + op)
	}
	return nil
}
