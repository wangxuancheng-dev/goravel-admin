# Search

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/advanced/search).

---

默认关闭。内置**订单**同步；其他业务通过 `RegisterDefinition` 扩展。

## 开关

```ini
SEARCH_ENABLED=true
SEARCH_DRIVER=elasticsearch   # 或 meilisearch | null
SEARCH_QUEUE=search
SEARCH_SYNC_WORKER=auto       # auto=任一已注册资源 sync 开启则起 worker
SEARCH_OUTBOX_ENABLED=true

SEARCH_SYNC_ORDERS=true
SEARCH_ORDERS_INDEX=orders
```

### Elasticsearch

```ini
ELASTICSEARCH_URLS=http://127.0.0.1:9200
# ELASTICSEARCH_USERNAME=
# ELASTICSEARCH_PASSWORD=
# ELASTICSEARCH_INDEX_PREFIX=
# ELASTICSEARCH_ORDERS_ANALYZER=auto
```

### Meilisearch

```ini
SEARCH_DRIVER=meilisearch
MEILISEARCH_HOST=http://127.0.0.1:7700
MEILISEARCH_API_KEY=
MEILISEARCH_INDEX_PREFIX=
```

驱动已接官方 `meilisearch-go`：Ping / Index / Delete / Search / EnsureIndex（按 `IndexDefinition` 写 settings）。

## 命令

```bash
go run . artisan search:init-orders-index [--tenant=]
go run . artisan search:sync-orders [--tenant=]
go run . artisan search:retry-outbox [--tenant=]
```

需要常驻消费 `SEARCH_QUEUE`（默认 `search`）的 Queue Worker。

## 多租户

索引短名带租户前缀：`{code}_orders`。未绑定租户时 **fail-closed**。队列任务必须带 `tenant_id`。

## 扩展其他模块（推荐方式）

1. 在业务模块 `init` / ServiceProvider 中注册定义：

```go
func init() {
    search.RegisterDefinition(search.IndexDefinition{
        Key:        "products", // 与 config 键一致
        PrimaryKey: "id",
        Searchable: []string{"name", "sku"},
        Filterable: []string{"id", "status"},
        Sortable:   []string{"id", "created_at"},
    })
}
```

2. `.env` / `config/search.go` 增加：

```ini
SEARCH_SYNC_PRODUCTS=true
SEARCH_PRODUCTS_INDEX=products
```

（配置路径：`search.indexes.products.sync_enabled` / `name`）

3. 业务包内实现 `Document` + `Push` +（可选）队列/outbox，参考 `app/search/orders/`。
4. **Elasticsearch**：订单有专用 mapping；其他资源需自行建索引/mapping，或在模块里封装 Ensure。  
   **Meilisearch**：`EnsureIndex` 会按已注册 `IndexDefinition` 自动更新 settings。

## 就绪探针

`SEARCH_ENABLED=true` 时，`GET /ready` 会对当前驱动 `Ping`（失败 → 503）。

## 生产建议

1. ES/Meili 与 App 同 VPC  
2. 专用 `search` 队列与 Worker  
3. 监控 outbox 积压（`search:retry-outbox`）  
4. 切换驱动后重新 init + 全量 sync  
5. 不用搜索时保持 `SEARCH_ENABLED=false`
