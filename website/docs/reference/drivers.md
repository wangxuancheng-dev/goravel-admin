# 可选驱动仓库

达梦（DM）支持已从本项目移除（无 `goravel-dm` 依赖与 `dm` 连接）。当前支持的数据库：MySQL、PostgreSQL（见 `config/database.go`）。

生产队列：`QUEUE_CONNECTION=redis`。

若后续需要 DM，再次引入 `github.com/wangxuancheng-dev/goravel-dm` 并注册其 ServiceProvider。
