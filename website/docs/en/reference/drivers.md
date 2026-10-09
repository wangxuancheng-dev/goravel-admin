# Optional driver repositories

This project no longer vendors driver source under `driver/`. Optional database drivers (e.g. Dameng) are separate Go modules.

For queues in production, use the official Redis queue: `QUEUE_CONNECTION=redis` (see `config/queue.go` and [Production](/en/deploy/production)). Kafka / NSQ / RabbitMQ / Redis Stream drivers have been removed from dependencies and configuration.

| Driver | Module path | Repository |
|--------|-------------|------------|
| Dameng (DM) | `github.com/wangxuancheng-dev/goravel-dm` | https://github.com/wangxuancheng-dev/goravel-dm |

## Usage

The root `go.mod` already `require`s this module. Configure via `config/database.go`.

```bash
go get github.com/wangxuancheng-dev/goravel-dm@latest
go mod tidy
```

## Testing

DM integration (needs a DM environment + build tag):

```bash
go test -tags dm github.com/wangxuancheng-dev/goravel-dm -run TestDMCrudAndTransaction -v
```

Full Chinese notes: [可选驱动仓库](/reference/drivers).
