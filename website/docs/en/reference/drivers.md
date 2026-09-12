# Optional driver repositories

This project no longer vendors driver source under `driver/`. Queue and DM drivers are separate Go modules.

| Driver | Module path | Repository |
|--------|-------------|------------|
| Dameng (DM) | `github.com/wangxuancheng-dev/goravel-dm` | https://github.com/wangxuancheng-dev/goravel-dm |
| Kafka | `github.com/wangxuancheng-dev/goravel-kafka` | https://github.com/wangxuancheng-dev/goravel-kafka |
| NSQ | `github.com/wangxuancheng-dev/goravel-nsq` | https://github.com/wangxuancheng-dev/goravel-nsq |
| RabbitMQ | `github.com/wangxuancheng-dev/goravel-rabbitmq` | https://github.com/wangxuancheng-dev/goravel-rabbitmq |
| Redis Stream | `github.com/wangxuancheng-dev/goravel-redis-stream` | https://github.com/wangxuancheng-dev/goravel-redis-stream |

## Usage

The root `go.mod` already `require`s these modules. Configure via `config/database.go` (DM) and `config/queue.go` (queues).

```bash
go get github.com/wangxuancheng-dev/goravel-dm@latest
go get github.com/wangxuancheng-dev/goravel-kafka@latest
go get github.com/wangxuancheng-dev/goravel-nsq@latest
go get github.com/wangxuancheng-dev/goravel-rabbitmq@latest
go get github.com/wangxuancheng-dev/goravel-redis-stream@latest
go mod tidy
```

## Testing

Queue drivers are separate modules; default `go test ./...` in this repo does not enter their trees. Run tests in each driver repo when needed.

DM integration (needs a DM environment + build tag):

```bash
go test -tags dm github.com/wangxuancheng-dev/goravel-dm -run TestDMCrudAndTransaction -v
```

Full Chinese notes: [可选驱动仓库](/reference/drivers).
