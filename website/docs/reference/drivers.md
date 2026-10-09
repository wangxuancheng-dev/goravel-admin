# 可选驱动仓库

本仓库不再内嵌 `driver/` 源码备份。达梦等可选数据库驱动通过 Go module 依赖独立仓库。

生产队列请使用官方 Redis 队列：`QUEUE_CONNECTION=redis`（见 `config/queue.go`、[生产清单](/deploy/production)）。Kafka / NSQ / RabbitMQ / Redis Stream 等多队列驱动已从依赖与配置中移除。

| 驱动 | 模块路径 | 仓库 |
|------|----------|------|
| 达梦 DM | `github.com/wangxuancheng-dev/goravel-dm` | https://github.com/wangxuancheng-dev/goravel-dm |

## 使用

主工程 `go.mod` 已 `require` 上述模块；配置见 `config/database.go`。

更新示例：

```bash
go get github.com/wangxuancheng-dev/goravel-dm@latest
go mod tidy
```

## 测试

达梦集成测试（需 DM 环境与 build tag）：

```bash
go test -tags dm github.com/wangxuancheng-dev/goravel-dm -run TestDMCrudAndTransaction -v
```
