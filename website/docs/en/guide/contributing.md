# Contributing

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/guide/contributing).

---

感谢您对 Goravel Admin 项目的关注！我们欢迎任何形式的贡献。

## 目录

- [行为准则](#行为准则)
- [如何贡献](#如何贡献)
- [开发环境](#开发环境)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [Pull Request 流程](#pull-request-流程)

---

## 行为准则

请在参与项目时保持尊重和友善。我们致力于维护一个开放、包容的社区环境。

---

## 如何贡献

### 报告 Bug

1. 检查 [Issues](https://github.com/wangxuancheng-dev/goravel-admin/issues) 确认问题未被报告
2. 使用 Bug 模板创建新 Issue
3. 提供详细的复现步骤和环境信息

### 提出新功能

1. 先在 Issues 中讨论您的想法
2. 获得维护者认可后再开始开发
3. 遵循项目的设计理念

### 提交代码

1. Fork 项目
2. 创建功能分支
3. 编写代码和测试
4. 提交 Pull Request

---

## 开发环境

### 前置要求

| 工具 | 版本 |
|------|------|
| Go | 1.21+ |
| Node.js | 20+ |
| MySQL | 8.0+ |
| Redis | 7.0+ |

### 环境搭建

```bash
# 1. 克隆项目
git clone https://github.com/wangxuancheng-dev/goravel-admin.git
cd goravel-admin

# 2. 安装后端依赖
go mod tidy

# 3. 复制环境配置
cp .env.example .env

# 4. 配置数据库连接
# 编辑 .env 文件

# 5. 运行数据库迁移
go run . artisan migrate
go run . artisan db:seed

# 6. 启动后端服务
go run . --no-ansi
# 或使用 air 热重载
air

# 7. 安装前端依赖
cd html
npm install

# 8. 启动前端开发服务器
npm run dev
```

### 开发工具推荐

- **IDE**: VSCode / GoLand
- **Go 扩展**: gopls
- **Vue 扩展**: Volar
- **调试工具**: Delve (Go)

---

## 代码规范

### Go 代码规范

```go
// 1. 使用 gofmt 格式化代码
gofmt -w .

// 2. 遵循 Effective Go 指南
// https://go.dev/doc/effective_go

// 3. 函数命名使用驼峰式
func GetAdminList() {}

// 4. 常量使用大写下划线
const MAX_PAGE_SIZE = 100

// 5. 错误处理
if err != nil {
    return nil, err
}

// 6. 注释规范
// GetAdminList 获取管理员列表
// 参数 page 为页码
func GetAdminList(page int) {}
```

### Vue/TypeScript 代码规范

```typescript
// 1. 使用 Composition API
import { ref, computed, onMounted } from 'vue'

// 2. 组件命名使用 PascalCase
// components/UserList.vue

// 3. Composables 命名使用 use 前缀
// composables/useCrud.ts

// 4. 类型定义
interface User {
  id: number
  name: string
}

// 5. 使用 const 声明不变的引用
const tableRef = ref<VxeTableInstance>()

// 6. 导出函数使用具名导出
export function useCrud() {}
```

### CSS/SCSS 规范

```scss
// 1. 使用 BEM 命名规范
.admin-list {
  &__header {}
  &__content {}
  &--active {}
}

// 2. 使用变量管理颜色
$primary-color: #409eff;

// 3. 避免过深的嵌套（最多 3 层）
```

---

## 提交规范

### Commit Message 格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档更新 |
| `style` | 代码格式（不影响功能） |
| `refactor` | 重构 |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 构建/工具相关 |

### 示例

```bash
# 新功能
feat(admin): add batch delete feature

# Bug 修复
fix(auth): fix token refresh issue

# 文档
docs: update API documentation

# 重构
refactor(composables): convert useCrud to TypeScript
```

---

## Pull Request 流程

### 1. 创建分支

```bash
# 从 main 分支创建功能分支
git checkout main
git pull origin main
git checkout -b feature/your-feature-name
```

### 2. 开发和测试

```bash
# 后端构建
go build ./...

# 后端测试
go test -v -timeout=2m ./...

# 类型检查
(cd html && npm run type-check)

# 前端构建
(cd html && npm run build)
```

### 3. 提交代码

```bash
git add .
git commit -m "feat(module): add new feature"
git push origin feature/your-feature-name
```

### 4. 创建 Pull Request

1. 在 GitHub 上创建 PR
2. 填写 PR 模板
3. 关联相关 Issue
4. 请求 Code Review

### 5. Code Review

- 回复评论并修改代码
- 确保所有 CI 检查通过
- 等待维护者合并

---

## PR 检查清单

在提交 PR 前，请确认：

- [ ] 代码符合项目规范
- [ ] 添加了必要的测试
- [ ] 所有测试通过
- [ ] 更新了相关文档
- [ ] Commit message 符合规范
- [ ] 无冲突可合并

---

## 目录结构

提交代码时，请遵循项目的目录结构：

```
.
├── app/                    # 后端应用代码
│   ├── http/controllers/  # 控制器
│   ├── models/            # 数据模型
│   ├── services/          # 业务服务
│   └── utils/             # 工具函数
├── html/                   # 前端应用代码
│   └── src/
│       ├── api/           # API 请求
│       ├── components/    # 通用组件
│       ├── composables/   # 可复用逻辑
│       ├── views/         # 页面组件
│       └── types/         # TypeScript 类型
├── tests/                  # 测试代码
│   ├── unit/              # 单元测试
│   └── feature/           # 集成测试
└── docs/                   # 文档
```

---

## 联系我们

- **Issues**: [GitHub Issues](https://github.com/wangxuancheng-dev/goravel-admin/issues)
- **Discussions**: [GitHub Discussions](https://github.com/wangxuancheng-dev/goravel-admin/discussions)
- **Discord**: [Goravel Discord](https://discord.gg/cFc5csczzS)

---

感谢您的贡献！🎉

