# Hardcoded checklist

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/guide/hardcoded-checklist).

---

本文档用于避免“新增功能后遗漏改动”。当你新增后台模块、接口或页面时，请按下面清单逐项确认。

## 一、权限与菜单（必查）

**推荐（代码生成器模块）：** 使用开发工具 → 代码生成器 →「安装菜单与权限」，或保存代码时勾选自动安装。详见 [代码生成器](/guide/code-generator)。manifest 文件位于 `database/seeders/modules/*.json`，生产部署执行：

```bash
./main artisan db:seed --seeder=GeneratedModulesSeeder
```

**手动（非生成器模块或需改种子默认值）：**

- [ ] 在 `database/seeders/permission_seeder.go` 增加权限种子：
  - `slug`（如 `article.update`）
  - `method`（GET/POST/PUT/DELETE）
  - `path`（接口路径，支持 `*`）
- [ ] 菜单种子是否已补齐（如有新增菜单）。
- [ ] 角色是否已分配新权限（本地测试账号至少验证 1 个非超管角色）。

## 二、操作日志标题（必查）

- [ ] 新接口能否命中权限 slug（优先）。
- [ ] 非标准接口（如 `.../verify`、`.../export`、`.../batch-*`）是否在
      `app/utils/operation_title.go` 的 `generateDefaultTitle()` 增加兜底映射。
- [ ] `html/src/views/log/OperationLogList.vue`：
  - [ ] `defaultTitleSlugs` 是否需要补新 slug（下拉预置项）
  - [ ] `legacyTitleMap` 是否需要补历史 key 兼容映射（如连字符旧格式）

## 三、多语言（必查）

- [ ] `html/src/i18n/locales/zh-CN.json` 增加：
  - `permission.<slug>`
  - `menu.<slug>`（如果有新菜单）
- [ ] `html/src/i18n/locales/en-US.json` 增加同名 key。
- [ ] 页面上不出现原始 key（如 `permission.xxx`、`menu.xxx`）。

## 四、路由与前端页面（常查）

- [ ] 新页面组件路径符合 `html/src/router/index.js` 动态加载约定。
- [ ] 页面权限按钮 `permission` 配置与后端 slug 一致。
- [ ] 新页面是否需要加入菜单搜索/面包屑的特殊映射（如有特殊命名）。

## 五、日志与观测相关（按需）

- [ ] 如果是“日志与观测”子能力：
  - [ ] `html/src/views/log/ObservabilityHub.vue` 的 `tabAccess` / `visibleTabs` 是否补齐
  - [ ] 无权限时是否要求“隐藏且静默”（`skipErrorMessage` + 页面 403 处理）
- [ ] 如果是系统日志模块筛选：
  - [ ] `html/src/views/log/SystemLogList.vue` 的模块下拉与映射是否补齐

## 六、操作日志采集策略（按需）

- [ ] 新接口是否应记录操作日志：
  - [ ] 方法是否在 `config/operation_log.go` 的 `allowed_methods`
  - [ ] 路径是否被 `excluded_paths` / `excluded_path_prefixes` 排除

## 七、返回字段与前端映射（必查）

- [ ] 后端模型对外字段必须显式声明 `json` 标签（推荐 snake_case）。
  - 例如：`json:"created_at"`、`json:"trace_id"`、`json:"user_agent"`。
  - 避免依赖默认序列化导致的 `CreatedAt/TraceID` 与前端字段不一致。
- [ ] 明确接口实际返回字段风格（snake_case / PascalCase）。
- [ ] 前端转换逻辑（`transformXxxData`）与实际返回一致，避免“页面空数据”。
- [ ] 若需兼容历史字段，写清注释和计划移除时间。

## 八、验证用例（发布前）

- [ ] 新增/编辑/删除各跑一遍，确认操作日志有标题且翻译正确。
- [ ] 在“操作日志”标题下拉可搜到新增 slug，并可筛选命中。
- [ ] 用无权限角色验证：页面隐藏/按钮禁用/接口提示符合预期。
- [ ] 中英文切换下，新增模块标题与按钮文案正常。

---

## 推荐原则

1. 优先让标题来自权限 slug，路径推导只做兜底。  
2. 新增功能时优先补“种子 + i18n + 日志标题”，避免后续补洞。  
3. 页面若依赖接口权限，尽量做“无权限不展示 + 403 静默处理”。
