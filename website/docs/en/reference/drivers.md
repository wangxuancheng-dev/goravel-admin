# Optional driver repositories

Dameng (DM) support is currently removed from this project (no `goravel-dm` dependency or `dm` connection). Supported databases: MySQL and PostgreSQL via `config/database.go`.

Production queue: `QUEUE_CONNECTION=redis`.

If DM is needed later, re-add `github.com/wangxuancheng-dev/goravel-dm` and register its service provider.

Chinese notes: [可选驱动仓库](/reference/drivers).
