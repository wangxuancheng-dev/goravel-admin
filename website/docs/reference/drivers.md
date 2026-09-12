# 可选驱动仓库

本仓库不再内嵌 `driver/` 源码备份。队列 / 达梦驱动均通过 Go module 依赖独立仓库。

| 驱动 | 模块路径 | 仓库 |
|------|----------|------|
| 达梦 DM | `github.com/wangxuancheng-dev/goravel-dm` | https://github.com/wangxuancheng-dev/goravel-dm |
| Kafka | `github.com/wangxuancheng-dev/goravel-kafka` | https://github.com/wangxuancheng-dev/goravel-kafka |
| NSQ | `github.com/wangxuancheng-dev/goravel-nsq` | https://github.com/wangxuancheng-dev/goravel-nsq |
| RabbitMQ | `github.com/wangxuancheng-dev/goravel-rabbitmq` | https://github.com/wangxuancheng-dev/goravel-rabbitmq |
| Redis Stream | `github.com/wangxuancheng-dev/goravel-redis-stream` | https://github.com/wangxuancheng-dev/goravel-redis-stream |

## 使用

主工程 `go.mod` 已 `require` 上述模块；配置见 `config/database.go`（DM）与 `config/queue.go`（队列）。

更新示例：

```bash
go get github.com/wangxuancheng-dev/goravel-dm@latest
go get github.com/wangxuancheng-dev/goravel-kafka@latest
go get github.com/wangxuancheng-dev/goravel-nsq@latest
go get github.com/wangxuancheng-dev/goravel-rabbitmq@latest
go get github.com/wangxuancheng-dev/goravel-redis-stream@latest
go mod tidy
```

## 测试

队列驱动为独立 module，默认 `go test ./...` 不会进入其源码树。需要时在对应仓库内运行测试，或临时 `go test` 该模块路径。

达梦集成测试（需 DM 环境与 build tag）：

```bash
go test -tags dm github.com/wangxuancheng-dev/goravel-dm -run TestDMCrudAndTransaction -v
```
