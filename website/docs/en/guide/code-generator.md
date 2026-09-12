# Code generator

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/guide/code-generator).

---

代码生成器在生成 CRUD 代码后，可通过 **Install 服务** 自动注册菜单、权限，并写入可复现的 manifest 文件。

## 流程概览

```
开发：生成代码 →（可选）自动安装菜单/权限 → 写入 database/seeders/modules/{module}.json
部署：migrate → 仅执行 GeneratedModulesSeeder → 同步菜单/权限
```

| 步骤 | 开发环境 | 生产环境 |
|------|----------|----------|
| 生成代码 | 代码生成器 →「生成代码」 | 随版本发布（Git） |
| 安装菜单/权限 | 保存时自动安装，或点「安装菜单与权限」 | 见下方部署命令 |
| 持久化配置 | `database/seeders/modules/*.json` 提交 Git | 同左，随镜像/代码部署 |

## 开发环境

### 1. 配置（代码生成器页面）

Vue（`html/`）与 React（`html-react/`）页面能力一致。勾选 **树形列表** 时：

- Vue 生成 `TreeListPage` 树表 + 表单页
- React 生成树形 `Table`（展开/收起、无分页、`page_size: 1000`）+ `FormModal`（`parent_id` 使用 `TreeSelect`）

树形模块建议在字段中包含 `parent_id`（及用于展示的名称字段，如 `name` / `title`）。勾选树形时后端 `Index` 无搜索条件返回树形 `list`，有搜索时返回扁平列表；前端会自动将扁平数据组装为树。

- **菜单标题**：侧边栏显示名（如「文章管理」）
- **父级菜单**：留空 = 顶级；也可挂到「系统管理」等目录下
- **菜单排序**：`sort` 值，越小越靠前
- **保存时自动安装菜单与权限**（默认开启）

### 2. 生成并安装

1. 填写模块名、表名、字段
2. 点击 **生成代码**（保存文件 + 若勾选则安装）
3. 或代码已存在时，单独点击 **安装菜单与权限**

安装完成后：

- `menus` / `permissions` 表写入或更新（按 `slug` 幂等）
- 生成 `database/seeders/modules/{module_name}.json`

### 3. 本地仅同步 manifest（不跑完整 seed）

```bash
go run . artisan db:seed --seeder=GeneratedModulesSeeder
```

## 生产环境

### 命令写法

生产使用编译后的二进制，**不要**依赖 `go run .`：

```bash
# 容器内示例
/www/main artisan db:seed --seeder=GeneratedModulesSeeder

# 裸机示例
./main artisan db:seed --seeder=GeneratedModulesSeeder
```

Docker 部署可参考 `scripts/deploy/seed.sh`（默认跑**完整** `db:seed`，见下文注意）。

### 推荐部署顺序

1. 发布新代码（含 `database/seeders/modules/*.json`）
2. `./main artisan migrate`
3. `./main artisan db:seed --seeder=GeneratedModulesSeeder`

### 注意

| 操作 | 是否建议在生产执行 |
|------|-------------------|
| `db:seed --seeder=GeneratedModulesSeeder` | ✅ 可以（幂等，只处理生成器模块） |
| `db:seed`（完整） | ⚠️ 仅适合**首次初始化**；会跑 Menu/Permission/Admin 等全部 seeder，可能重复初始化基础数据 |
| 代码生成器「安装菜单与权限」API | ✅ 可用，但 manifest 需另存 Git 才便于复现 |

`GeneratedModulesSeeder` 已注册在 `bootstrap/seeders.go`，完整 `db:seed` 时也会执行；生产增量部署建议**只跑单个 seeder**，避免误触其他 seeder。

## 自动生成的权限

根据生成器选项（增/删/改/导出）推导，与前端 `PermissionButton` 使用的 slug 一致：

| 能力 | slug 示例（模块 `article`） |
|------|---------------------------|
| 列表 | `article.index` |
| 详情 | `article.show` |
| 创建 | `article.store` |
| 更新 | `article.update` |
| 删除 | `article.destroy` |
| 导出 | `article.export` |

API 路径前缀：`/api/admin/{table_name}`（如 `/api/admin/articles`）。

## 实现位置（维护参考）

| 文件 | 说明 |
|------|------|
| `app/services/module_manifest.go` | 从模块名 + 选项构建 manifest |
| `app/services/module_installer.go` | 幂等安装菜单/权限、写 JSON |
| `database/seeders/generated_modules_seeder.go` | 读取 `modules/*.json` 并安装 |
| `POST /api/admin/code-generator/install-module` | 单独安装接口 |
| `POST /api/admin/code-generator/save` | 保存代码，`install.enabled=true` 时附带安装 |

## 仍需手动的项

代码生成器**不会**自动完成以下工作，发布前请对照 [HARDCODED_CHECKLIST.md](/guide/hardcoded-checklist)：

- `permission.*` / `menu.*` 中英文 i18n（角色树有 `description` 兜底，但建议补全）
- `app/utils/operation_title.go` 非标准接口的操作日志标题
- 非超管角色的权限分配
- 观测、日志等特殊页面的硬编码映射

## 已有模块补装示例（article）

1. 打开 **开发工具 → 代码生成器**
2. 模块名 `article`，表名 `articles`，菜单标题「文章管理」
3. 点击 **安装菜单与权限**（无需重新生成代码）
4. 提交 `database/seeders/modules/article.json` 到 Git
