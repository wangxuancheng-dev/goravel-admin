# 多租户：业务开发须知

写业务模块只需要读这一页。架构、部署、运维见 [多租户](/advanced/tenancy)，调参见 [规模与推荐配置](/advanced/tenancy-scaling)。

单库模式（`TENANCY_DRIVER=off`）下，下面的写法同样正确。所以**永远按多租户的方式写**，不用区分两种模式。

## 三条规则

### 1. 查库：拿到 ctx 就往下传

```go
// 租户业务数据：用请求 / 任务的 ctx（http.Context 本身就是 context.Context）
appfacades.OrmQuery(ctx).Where("status", 1).Find(&orders)
appfacades.OrmTransaction(ctx, func(tx orm.Query) error { ... })

// 平台数据（tenants、platform_*、套餐订阅等）
appfacades.PlatformOrmQuery(ctx)   // 手上有 ctx 时
appfacades.PlatformQuery()         // 没有 ctx：命令行、定时任务、运维
```

- 业务服务的构造函数接收 `context.Context`，并保存为 `s.ctx`。
- 不要写 `facades.Orm().Query()`，它不知道当前是哪个租户。
- 迁移里的数据操作用 `migrationQuery()`。

### 2. 异步：用 `tenancyctx.Go`

```go
adminID := ...  // 需要从 http.Context 读的值，在启动之前读出来
tenancyctx.Go(ctx, "audit_export", func(ctx context.Context) {
    appfacades.OrmQuery(ctx).Create(&row) // 仍是同一个租户
})
```

`tenancyctx.Go` 做了三件事：

- 保留租户。
- 去掉请求的取消信号：请求一结束，原来的 ctx 就会被取消。
- 兜底 panic，不会因为一个后台任务把进程拖垮。

框架会在请求结束后回收 `http.Context`，所以协程里不能再读它。

### 3. 缓存与存储：加租户前缀

```go
facades.Cache().Put(tenancy.CacheKey(ctx, "report:"+id), v, ttl)
path := tenancy.StoragePrefix(ctx) + "exports/" + name
```

key 不经 `tenancy.CacheKey`，所有租户会读到同一份。把 key 拼装封装在辅助函数里也可以（例如 `func (s *X) lockKey(...)` 内部调用 `tenancy.CacheKey`），检查工具认得这种写法。

## 写错了会怎样

这几条都不靠人记，写错了会在开发阶段就报出来：

| 写错的方式 | 结果 |
|---|---|
| 忘传 ctx：`OrmQuery(nil)` | 多租户开启时直接报 `ErrNilQueryContext`，不会悄悄落到平台库 |
| 用租户 ctx 读写平台表：`OrmQuery(ctx).Model(&models.Tenant{})` | 租户连接拒绝执行，报 `tenantdb.ErrPlatformTable`，提示改用 `PlatformQuery()` |
| `facades.Orm().Query()`、`PlatformOrmQuery(nil)`、缓存不加前缀、`go func` 里用请求 ctx | CI 的 `tenantlint` 不通过 |
| 某个接口把 A 租户的数据返回给 B 租户 | `TestEveryAdminRouteIsTenantIsolated` 会逐个扫描所有 GET 接口，新增的接口也自动纳入 |
| 原生 SQL 跨库读取 | 开启 `TENANCY_DB_DEDICATED_USER` 后，租户用的数据库账号只能访问自己的库，由数据库服务器直接拒绝 |

本地自查：

```bash
go run ./scripts/tenantlint
go test ./tests/feature -run 'TestEveryAdminRouteIsTenantIsolated|TestTenantResolutionSecurity|TestTenantConnectionRefusesPlatformTables'
```

## 确实需要例外时

有些 key 本来就是全局的，例如平台告警防抖、OIDC state。这种情况在调用处或函数注释里写明原因：

```go
//tenantlint:allow cache: 平台级告警防抖，不是租户数据
func AlertQueueBacklogIfNeeded(...) { ... }
```

规则名有 `orm`、`nilctx`、`cache`、`goctx` 四个，冒号后面的原因必填。

## 新增一张平台表

1. 模型实现 `models.PlatformModel`，并加入 `app/models/platform.go` 的 `platformModels`。
2. 迁移的 `Up` / `Down` 以 `if SkipOnTenantConnection() { return nil }` 开头，并在 `IsLandlordOnlyMigration` 中登记。

两边不一致时，`TestPlatformTablesMatchLandlordOnlyMigrations` 会失败。

## 需要有经验的人把关的代码

下面这几类代码会同时涉及平台和租户，或者本来就要跨租户，上面的防护覆盖不全，合并前请找熟悉多租户的人 review：

- 租户解析：`middleware/tenant.go`、`BindHTTP`、子域 / 独立域名解析。
- 平台运维：`tenant_ops_*`、备份恢复、fleet 批量操作。它们本来就是跨租户的。
- 数据库以外的共享资源：搜索索引名、队列 payload、对象存储 bucket。
- `WithTenantConnection` / `WithSchemaContext`：只用于迁移、seeder 这类无法接收 ctx 的代码，业务代码不要用。
