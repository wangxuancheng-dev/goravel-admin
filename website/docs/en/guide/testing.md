# Testing

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/guide/testing).

---

本项目后端（Go / Goravel）提供单元测试与集成测试。

## 后端测试 (Go - Goravel)

后端测试分为两类：

- **快速测试**：默认质量门禁，只运行当前根模块内的包，不能依赖外部服务。
- **集成测试 / 冒烟测试**：需要数据库等依赖；CI 的 `backend-integration` job 会跑 `./tests/feature/...`。

### 当前实际结构

```
tests/
├── test_case.go                         # Goravel 测试基类
└── feature/
    ├── admin_auth_test.go               # 登录校验、未授权访问
    ├── admin_auth_extra_test.go         # 错密登录、无权限访问
    ├── permission_guard_test.go         # 订单/导出无权限；导出下载非所有者 forbidden
    ├── quality_gates_test.go            # 有权限放行、模块关闭、支付 notify 501
    ├── admin_smoke_test.go              # 冒烟：登录成功、info、menus/tree
    ├── admin_module_test.go             # info.config 模块开关字段
    ├── api_auth_test.go                 # health、用户注册/登录校验
    ├── platform_tenancy_test.go         # 平台登录 / 租户头
    ├── tenant_isolation_test.go         # 双租户库隔离
    ├── tenancy_failclosed_test.go       # 导出/搜索缺 tenant fail-closed
    └── blacklist_guard_test.go          # IP 黑名单拦截、平台未授权

app/services/
└── blacklist_guard_test.go              # 黑名单短缓存 / 过期回退 / fail-closed

app/http/middleware/
└── permission_match_test.go             # 权限路径通配匹配

app/http/controllers/admin/
└── resource_ownership_test.go           # 导出/附件归属纯逻辑

app/http/helpers/
└── time_converter_test.go               # 快速 Go 测试

app/utils/
├── sharding_helper_test.go
└── traceid/traceid_test.go

html/ / html-react/
└── src/utils/*.test.*                   # vitest：tenant、buildSearchParams、apiFactory、normalize、timeRange、storage（Vue 另含 xss）
```

Optional drivers live in separate repos — see [Optional drivers](/en/reference/drivers).

### 运行测试

```bash
# 快速后端门禁（CI 默认使用，排除 tests/feature、node_modules）
mapfile -t PKGS < <(go list ./... | grep -vE '/tests/feature$|node_modules')
go test -count=1 -timeout=3m "${PKGS[@]}"

# 冒烟 / feature 集成测试（需 migrate 后的数据库）
go test -v -timeout=2m ./tests/feature/...

# 仅 helper / utils
go test -v -timeout=30s ./app/http/helpers ./app/utils/...

# 前端最小单测（CI frontend-* jobs）
(cd html && npm test)
(cd html-react && npm test)
```

### 集成测试（按需运行）

Kafka / NSQ / RabbitMQ / Redis Stream / Dameng drivers are separate repositories. Default `go test ./...` in this repo does not test their sources. When needed:

```bash
go test -v github.com/wangxuancheng-dev/goravel-redis-stream/...
go test -v -timeout=2m github.com/wangxuancheng-dev/goravel-kafka/...     # needs 127.0.0.1:9092
go test -v -timeout=2m github.com/wangxuancheng-dev/goravel-nsq/...       # needs 127.0.0.1:4150
go test -v -timeout=2m github.com/wangxuancheng-dev/goravel-rabbitmq/...  # needs 127.0.0.1:5672
```

Dameng integration needs the official driver, `dm` build tag, and a reachable instance:

```bash
set DM_TEST_DSN=dm://SYSDBA:SYSDBA@127.0.0.1:5236
go test -tags dm github.com/wangxuancheng-dev/goravel-dm -run TestDMCrudAndTransaction -v
```

### 卡住问题排查顺序

如果 `go test ./...` 变慢或无输出，按下面顺序缩小范围：

```bash
# 1. 只编译，不运行测试，判断是否是包初始化/编译问题
go test -run TestNonExistent -count=0 -v ./...

# 2. 分区检查
go test -run TestNonExistent -count=0 -v ./app/...
go test -run TestNonExistent -count=0 -v ./routes ./config ./bootstrap ./database/...

# 3. 再运行真实快速测试
go test -v -timeout=2m ./...
```

注意：不要把需要外部服务的集成测试加入默认快速门禁；它们应独立运行并设置明确超时。

### 创建测试

使用 Artisan 命令创建测试：

```bash
go run . artisan make:test unit/MyServiceTest
go run . artisan make:test feature/MyFeatureTest
```

### 测试示例

```go
// tests/unit/token_service_test.go
package unit

import (
    "testing"
    "github.com/stretchr/testify/suite"
    "goravel/tests"
)

type TokenServiceTestSuite struct {
    suite.Suite
    tests.TestCase
}

func TestTokenServiceTestSuite(t *testing.T) {
    suite.Run(t, new(TokenServiceTestSuite))
}

func (s *TokenServiceTestSuite) SetupTest() {
    // 每个测试前执行
}

func (s *TokenServiceTestSuite) TearDownTest() {
    // 每个测试后执行
}

func (s *TokenServiceTestSuite) TestHashToken() {
    // 测试 token 哈希
    s.Equal(64, len(hashToken("test")))
}

func (s *TokenServiceTestSuite) TestGenerateRandomToken() {
    token1 := generateRandomToken()
    token2 := generateRandomToken()
    
    s.Len(token1, 40)
    s.NotEqual(token1, token2)
}
```

---

## 测试最佳实践

### 1. 命名约定

```go
// Go: TestSuiteName + TestMethodName
func (s *TokenServiceTestSuite) TestHashToken_ValidInput()
```

### 2. 表格驱动测试 (Go)

```go
func (s *TokenServiceTestSuite) TestHashToken() {
    tests := []struct {
        name     string
        input    string
        expected int
    }{
        {"正常 token", "test-token", 64},
        {"空 token", "", 64},
    }
    
    for _, tt := range tests {
        s.Run(tt.name, func() {
            got := hashToken(tt.input)
            s.Len(got, tt.expected)
        })
    }
}
```

---

## CI/CD 集成

真实工作流见 [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)：

| Job | 内容 |
|-----|------|
| `backend` | Fast `go test` excluding `tests/feature` |
| `backend-integration` | MySQL 8 + Redis 7 → `artisan migrate` → `go test ./tests/feature/...` |
| `frontend-vue` / `frontend-react` | `type-check` + `build:ci` |

本地复现 integration：准备好 MySQL/Redis，配置 `.env` 后执行 migrate，再跑 feature 包。

---

## Feature 冒烟测试

位于 `tests/feature/`：

| 文件 | 覆盖内容 |
|------|----------|
| `admin_auth_test.go` | 登录校验、未授权 |
| `admin_smoke_test.go` | 登录成功、info、menus/tree |
| `admin_module_test.go` | `info.config` 模块开关字段 |
| `api_auth_test.go` | health、用户注册/登录校验 |
| `platform_tenancy_test.go` | 平台登录/info/租户列表/health/改密；tenancy 开启时 admin 缺租户头与未知租户拒绝；缓存/存储前缀/搜索索引短名隔离 |
| `tenant_isolation_test.go` | 双租户真实建库 + migrate；configs 写入 A 不可见 B；无 CREATE 权限时 Skip |
| `app/search/config_test.go` | 订单索引短名租户隔离段 / `IsOrdersIndexShortName` |

```bash
go test -v -timeout=5m ./tests/feature/...
```

前置：可连数据库、已 `migrate`；不依赖 Docker-in-Docker。勿假设存在已删除的 `admin_api_test.go` / `blacklist_api_test.go` 套件名。

---

## 常见问题

### Q: 后端测试为什么放在 tests 目录？

**A:** 这是 Goravel 框架的推荐做法，参考 [官方文档](https://www.goravel.dev/zh_CN/testing/getting-started.html)。

### Q: 如何添加新测试？

**A:** 后端：`go run . artisan make:test unit/MyTest`
