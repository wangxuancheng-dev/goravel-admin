# 开发调试面板 Goralens

## 概述

[Goralens](https://github.com/beratkara/goralens) 是 Goravel 版的 Laravel Telescope，会记录请求、SQL、日志、异常、队列任务、事件、缓存、邮件、权限判断、对外 HTTP 调用、Artisan 命令和定时任务，并在浏览器面板中按请求串联展示。适合排查 N+1 查询、慢 SQL，或查看某个请求到底触发了什么。

**仅用于开发环境。** 项目已默认关闭，只有在本地 `.env` 中显式开启才会记录数据。

## 启用

1. 在 `.env` 中开启（`.env.example` 已默认写好）：

```env
GORALENS_ENABLED=true
```

2. 执行迁移，创建 `goralens_entries`、`goralens_entries_tags`、`goralens_monitoring` 三张表：

```bash
go run . artisan migrate
```

3. 启动服务后访问：

```
http://localhost:3000/goralens
```

> 若 `.env` 已开启但尚未执行迁移，启动时会报一次 `goralens_monitoring` 表不存在，不影响运行，迁移后即消失。

## 配置

配置文件：`config/goralens.go`，常用环境变量：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GORALENS_ENABLED` | `false` | 总开关；关闭时不记录、不注册面板路由 |
| `GORALENS_PATH` | `goralens` | 面板访问路径 |
| `GORALENS_DRIVER` | `database` | 存储驱动：`database` 或 `memory`（重启即清空） |
| `GORALENS_DB_CONNECTION` | `DB_CONNECTION` | 存储使用的数据库连接，固定为中心库；租户维护操作临时切换默认连接时，记录也不会写入租户库 |

**关闭某类记录：** 在 `config/goralens.go` 的 `watchers` 列表中删除或注释对应项即可。Redis 记录需额外安装 `github.com/beratkara/goralens/redishook`，项目默认未启用。

**敏感数据：** `authorization`、`cookie` 请求头及 `password`、`token` 等字段会在入库前替换为 `********`，可在配置的 `hidden` 中追加。

## 访问控制

未配置自定义 `Auth` 回调时，面板只在 `APP_ENV=local` 下可访问；其它环境返回 403。

> 请保持 `GORALENS_ALLOWED_EMAILS` 为空。一旦设置，包括 local 在内的所有环境都会改为校验当前登录用户，而项目尚未把后台登录用户接入 Goralens（`goralens.ResolveUser`），面板会因此无法访问。

## 清理数据

记录增长很快，建议定期清理：

```bash
# 删除 24 小时前的记录
go run . artisan goralens:prune --hours=24

# 清空全部记录
go run . artisan goralens:clear
```

## 生产环境

- `config/goralens.go` 中 `enabled` 默认值为 `false`，`.env.production.example` 也显式设为 `false`。服务器 `.env` 不配置即为关闭，**请勿在生产开启**。
- 迁移对所有环境注册，生产中心库会有三张空表，不影响运行。
- 该迁移为 landlord-only（见 `database/migrations/tenant_skip.go`），`tenant:migrate` 时在租户库上跳过，租户库不会建这三张表。
- 关闭状态下不访问上述表，也不产生额外写入。

## 注意事项

1. **日志噪音**：项目会以 info 级别记录 SQL，Goralens 批量写入记录的 INSERT 也会出现在 `storage/logs` 中，属正常现象。
2. **端口**：Goralens 没有独立端口，挂在应用自身的 HTTP 服务上，跟随 `APP_PORT`。
3. **版本**：当前为 v0.1.0，适配 Goravel v1.18；升级 Goravel 前请确认其兼容性。

## 卸载

1. 从 `bootstrap/providers.go`、`config/logging.go` 中移除 goralens 相关代码；从 `database/migrations/all.go`、`tenant_skip.go`、`tenant_skip_test.go` 中移除对应条目；删除 `config/goralens.go` 和 `database/migrations/20260101000000_create_goralens_tables.go`
2. 删除 `.env*` 中的 `GORALENS_*` 变量，执行 `go mod tidy`
3. 手动删除 `goralens_entries`、`goralens_entries_tags`、`goralens_monitoring` 三张表

## 参考资源

- [Goralens 仓库](https://github.com/beratkara/goralens)
