# Goralens debug dashboard

## Overview

[Goralens](https://github.com/beratkara/goralens) is a Laravel Telescope–style debug assistant for Goravel. It records requests, SQL queries, logs, exceptions, queued jobs, events, cache operations, mail, gate checks, outgoing HTTP calls, Artisan commands and scheduled tasks, and shows them in a browser dashboard grouped by request. Use it to track down N+1 queries, slow SQL, or everything a single request triggered.

**Development only.** It is disabled by default and records nothing unless your local `.env` turns it on.

## Enable

1. Turn it on in `.env` (already set in `.env.example`):

```env
GORALENS_ENABLED=true
```

2. Run migrations to create the `goralens_entries`, `goralens_entries_tags` and `goralens_monitoring` tables:

```bash
go run . artisan migrate
```

3. Start the server and open:

```
http://localhost:3000/goralens
```

> If it is enabled before the migration has run, startup logs a one-time "table goralens_monitoring doesn't exist" error. The app keeps running, and the error goes away after migrating.

## Configuration

Config file: `config/goralens.go`. Common environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `GORALENS_ENABLED` | `false` | Master switch; when off, nothing is recorded and the dashboard route is not registered |
| `GORALENS_PATH` | `goralens` | Dashboard path |
| `GORALENS_DRIVER` | `database` | Storage driver: `database` or `memory` (cleared on restart) |
| `GORALENS_DB_CONNECTION` | `DB_CONNECTION` | Storage connection, pinned to the landlord DB so entries never land in a tenant DB while tenant maintenance temporarily swaps the default connection |

**Turning off a recorder:** remove or comment out its entry in the `watchers` list in `config/goralens.go`. Redis recording also needs `github.com/beratkara/goralens/redishook` and is not enabled in this project.

**Sensitive data:** the `authorization` and `cookie` headers and fields such as `password` and `token` are replaced with `********` before storage. Add more under `hidden` in the config.

## Access control

Without a custom `Auth` callback, the dashboard is only reachable when `APP_ENV=local`; other environments get 403.

> Leave `GORALENS_ALLOWED_EMAILS` empty. Once set, every environment, local included, switches to checking the signed-in user. This project has not wired its admin login into Goralens (`goralens.ResolveUser`), so the dashboard would become unreachable.

## Pruning

Entries grow quickly; clean them up regularly:

```bash
# Delete entries older than 24 hours
go run . artisan goralens:prune --hours=24

# Delete all entries
go run . artisan goralens:clear
```

## Production

- `enabled` defaults to `false` in `config/goralens.go`, and `.env.production.example` sets it to `false` explicitly. Leaving it out of the server `.env` keeps it off. **Do not enable it in production.**
- The migration is registered for every environment, so the production landlord database gets three empty tables. This is harmless.
- The migration is landlord-only (see `database/migrations/tenant_skip.go`): `tenant:migrate` skips it on tenant DBs, so tenant databases never get these tables.
- While disabled, Goralens does not touch those tables or write anything.

## Notes

1. **Log noise:** this project logs SQL at info level, so the INSERTs Goralens uses to flush entries also show up in `storage/logs`. This is expected.
2. **Port:** Goralens has no port of its own; it is served by the app's HTTP server on `APP_PORT`.
3. **Version:** currently v0.1.0, built for Goravel v1.18. Check compatibility before upgrading Goravel.

## Uninstall

1. Remove the goralens code from `bootstrap/providers.go` and `config/logging.go`; remove its entries from `database/migrations/all.go`, `tenant_skip.go` and `tenant_skip_test.go`; delete `config/goralens.go` and `database/migrations/20260101000000_create_goralens_tables.go`
2. Remove the `GORALENS_*` variables from the `.env*` files and run `go mod tidy`
3. Drop the `goralens_entries`, `goralens_entries_tags` and `goralens_monitoring` tables manually

## References

- [Goralens repository](https://github.com/beratkara/goralens)
