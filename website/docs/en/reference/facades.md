# Facades

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/reference/facades).

---

## 概述

Goravel v1.17 引入了 Facades 按需安装和卸载功能。由于项目已经升级到 v1.17 并完成了代码结构简化，现在可以使用 `package:install` 和 `package:uninstall` 命令来管理 facades。

## 当前状态

项目当前使用的是完整的 `goravel/goravel`（不是 `goravel/goravel-lite`），所有 facades 已经默认安装。facades 文件位于 `app/facades` 目录中。

## 可用的 Facades

当前项目中已安装的 facades：

- `App` - 应用实例
- `Artisan` - 命令行工具
- `Auth` - 身份验证
- `Cache` - 缓存
- `Config` - 配置
- `Crypt` - 加密
- `DB` - 数据库查询构建器
- `Event` - 事件系统
- `Gate` - 授权
- `Grpc` - gRPC 客户端
- `Hash` - 哈希
- `Http` - HTTP 客户端
- `Lang` - 多语言
- `Log` - 日志
- `Mail` - 邮件
- `Orm` - ORM
- `Process` - 进程管理
- `Queue` - 队列
- `RateLimiter` - 速率限制
- `Route` - 路由
- `Schedule` - 任务调度
- `Schema` - 数据库模式
- `Seeder` - 数据填充
- `Session` - 会话
- `Storage` - 文件存储
- `Testing` - 测试
- `Validation` - 验证
- `View` - 视图

## 安装 Facades

### 安装指定的 Facade

```bash
# 安装 Route facade
./artisan package:install Route

# 安装多个 facades
./artisan package:install Route Cache Queue
```

### 安装所有 Facades

```bash
# 安装所有 facades（交互式选择）
./artisan package:install --all

# 使用默认驱动安装所有 facades
./artisan package:install --all --default
```

**注意**：使用交互式安装时：
- 按下 `x` 选择要安装的 facades
- 按下 `Enter` 确认安装
- 如果直接按下 `Enter` 而不选择，默认没有 facades 被选中

## 卸载 Facades

### 卸载指定的 Facade

```bash
# 卸载 Route facade
./artisan package:uninstall Route

# 卸载多个 facades
./artisan package:uninstall Route Cache Queue
```

## 使用场景

### 场景 1：优化项目体积

如果你的项目不需要某些 facades，可以卸载它们以减少编译后的二进制文件大小：

```bash
# 卸载不需要的 facades
./artisan package:uninstall Grpc Gate Testing
```

### 场景 2：按需添加功能

如果项目开始时只使用基本功能，后续需要添加新功能时再安装对应的 facades：

```bash
# 初始只安装基本 facades
./artisan package:install Config Log Route

# 后续需要邮件功能时
./artisan package:install Mail
```

### 场景 3：迁移到轻量版

如果将来想迁移到 `goravel/goravel-lite`，可以先卸载不需要的 facades，然后切换到 lite 版本：

```bash
# 卸载不需要的 facades
./artisan package:uninstall Grpc Gate Testing View

# 然后修改 go.mod，将 goravel/goravel 改为 goravel/goravel-lite
```

## 注意事项

1. **服务提供者依赖**：卸载 facade 不会自动卸载对应的服务提供者。如果服务提供者在 `bootstrap/providers.go` 中注册，需要手动移除。

2. **代码引用**：卸载 facade 前，确保代码中没有使用该 facade，否则会导致编译错误。

3. **配置文件**：某些 facades 可能需要在配置文件中进行配置（如 `config/http.go`、`config/cache.go` 等），卸载后可能需要清理相关配置。

4. **向后兼容**：当前项目已安装所有 facades，如果不需要优化体积，可以保持现状。

## 检查 Facade 使用情况

在卸载 facade 前，可以使用以下命令检查代码中的使用情况：

```bash
# 检查 Route facade 的使用
grep -r "facades.Route()" app/ routes/ --include="*.go"

# 检查 Cache facade 的使用
grep -r "facades.Cache()" app/ --include="*.go"
```

### 项目中 Facades 使用情况分析

根据代码扫描，项目中实际使用的 facades：

**必须保留的 Facades：**
- `Schema` - 多处使用（分表、迁移等）
- `Lang` - 多处使用（多语言翻译）
- `Route` - 路由系统
- `Log` - 日志记录
- `Orm` - ORM 查询
- `Config` - 配置读取
- `Queue` - 队列系统
- `Auth` - 身份验证
- `Cache` - 缓存
- `Http` - HTTP 客户端
- `Session` - 会话管理
- `Validation` - 数据验证
- `Hash` - 哈希加密
- `Crypt` - 加密解密
- `Mail` - 邮件发送
- `Storage` - 文件存储
- `Artisan` - 命令行工具
- `Schedule` - 任务调度
- `Event` - 事件系统
- `DB` - 数据库查询构建器
- `RateLimiter` - 速率限制

**可能未使用的 Facades（需要确认）：**
- `Grpc` - gRPC 客户端（项目中未发现使用）✅ **已成功卸载**
- `Gate` - 授权门面（项目中未发现使用）✅ **已成功卸载**
- `Testing` - 测试工具（仅在测试代码中使用，生产环境不需要）
- `View` - 视图渲染（项目是纯 API，不使用视图模板）❌ **无法卸载**（依赖于 Route）
- `Process` - 进程管理（项目中未发现使用）❌ **无法卸载**（基础 facade）
- `Seeder` - 数据填充（仅在数据库填充时使用）

**卸载限制说明：**

1. **`Process` - 基础 Facade**
   - 原因：`Process` 是框架核心功能，属于基础 facade
   - 状态：无法卸载
   - 建议：保留，即使当前未使用

2. **`View` - 依赖 Route**
   - 原因：`View` facade 依赖于 `Route` facade
   - 原因分析：HTTP 驱动配置（`config/http.go`）中的 `template` 配置可能使用 View 功能
   - 状态：如果 `Route` 已安装，`View` 无法卸载
   - 解决方案：
     - 如果确实不需要视图功能，需要先检查 `config/http.go` 中的 `template` 配置
     - 如果项目是纯 API，可以考虑移除 `template` 配置后再尝试卸载
     - 或者保留 `View` facade（即使不使用，影响也不大）

**注意**：
1. `Testing` facade 在测试代码中使用，如果只优化生产环境，可以保留
2. 即使某些 facade 在当前代码中未直接使用，它们可能被框架内部或其他包使用，卸载前请仔细测试
3. 建议先在不重要的环境中测试卸载，确认无误后再应用到生产环境
4. 基础 facades（如 `Process`、`App`、`Config`、`Artisan`）无法卸载

## 示例：卸载不需要的 Facades

### 实际卸载示例

根据实际测试，以下 facades 可以成功卸载：

```bash
# 成功卸载 Grpc 和 Gate
./artisan package:uninstall Grpc Gate

# 输出：
# SUCCESS  Facade Grpc uninstalled successfully
# SUCCESS  Facade Gate uninstalled successfully
```

### 无法卸载的 Facades

```bash
# 尝试卸载 View（会失败）
./artisan package:uninstall View
# 错误：Facade View is depended on Route facades, cannot be uninstalled

# 尝试卸载 Process（会失败）
./artisan package:uninstall Process
# 警告：Facade Process is a base facade, cannot be uninstalled
```

### 完整的卸载流程

```bash
# 1. 检查使用情况
grep -r "facades.Grpc()" app/ --include="*.go"
grep -r "facades.Gate()" app/ --include="*.go"
grep -r "facades.Testing()" app/ --include="*.go"

# 2. 尝试卸载（会提示哪些无法卸载）
./artisan package:uninstall Grpc Gate View Process Testing

# 3. 只卸载可以卸载的
./artisan package:uninstall Grpc Gate Testing

# 4. 检查编译是否正常
go build .

# 5. 运行测试确保功能正常
go test ./...

# 6. 如果一切正常，可以提交更改
```

### 如果确实需要卸载 View

如果项目是纯 API，不使用视图功能，可以尝试以下步骤：

```bash
# 1. 检查 config/http.go 中的 template 配置
# 如果配置了 template，可能需要移除或注释掉

# 2. 尝试先卸载 Route（如果项目不使用视图，Route 可能也不需要）
# 注意：这会影响整个路由系统，请谨慎操作

# 3. 或者保留 View facade，即使不使用影响也不大
```

## 重新安装 Facade

如果误卸载了需要的 facade，可以重新安装：

```bash
# 重新安装 Route facade
./artisan package:install Route
```

## 相关文档

- [Goravel Facades 文档](https://www.goravel.dev/zh_CN/architecture-concepts/facades.html)
- [代码结构简化指南](/advanced/migration-strategy)
