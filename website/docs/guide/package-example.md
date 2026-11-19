# Goravel 扩展包示例（feedback）

本仓库用官方 [扩展包开发](https://www.goravel.dev/zh_CN/digging-deeper/package-development.html) 方式放了一个可复用示例：`packages/feedback`。

**目标：** 多 Goravel 项目复用业务功能；**不**自动安装菜单/权限（由宿主自行配置）。

## 包里有什么

| 内容 | 说明 |
|------|------|
| 表 `feedbacks` | 迁移在包内，并已登记到本仓 `database/migrations/all.go`（租户 migrate 会跑到） |
| Service | `NewService()`，经 `ResolveQuery` 查库 |
| HTTP | `GET/POST /api/admin/package-demos/feedbacks` 等 |
| 安装脚本 | `packages/feedback/setup/setup.go`（`package:install` 用） |

## 多租户

本仓在 `FeedbackServiceProvider` 注入：

```go
feedback.ResolveQuery = func(ctx context.Context) orm.Query {
    return appfacades.OrmQuery(ctx)
}
```

一户一库时请求进哪个租户就写哪个库。其他 Goravel 项目若也有租户，同样注入自家的租户 Query；单库项目可不注入（回落 `facades.Orm().Query()`）。

## 本仓已接线

1. `bootstrap/providers.go`：`feedback.ServiceProvider` + `FeedbackServiceProvider`
2. `routes/admin.go`：登录后、**无** Permission 中间件的分组里 `RegisterAdminRoutes`（演示用；上线可再挂权限）
3. 迁移：`all.go` 引用包内 migration

试调（需管理员 JWT）：

```bash
curl -H "Authorization: Bearer <token>" http://127.0.0.1:3000/api/admin/package-demos/feedbacks
curl -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d "{\"contact\":\"alice\",\"content\":\"hello package\"}" \
  http://127.0.0.1:3000/api/admin/package-demos/feedbacks
```

多租户本地可加 Header `X-Tenant-ID`（与现有后台一致）。

## 其它 Goravel 项目怎么用

1. 依赖本 module 的 `packages/feedback`，或把该目录发成独立 module  
2. 注册 `ServiceProvider`，注入 `ResolveQuery`（若需要）  
3. 发布/拷贝 migration 并 migrate  
4. 在自家路由组调用 `feedback/http.RegisterAdminRoutes`  
5. 菜单与权限按各项目自己的 RBAC 配置

详见包内 `packages/feedback/README.md`。