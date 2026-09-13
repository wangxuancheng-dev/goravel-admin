<p align="center"><img src="https://www.goravel.dev/logo.png?v=1.14.x" width="300"></p>

[English](./README.md) | 中文

## 关于 Goravel 

Goravel 是一个功能完整、可扩展性良好的 Web 应用框架。作为起始脚手架，帮助 Gopher 快速构建自己的应用程序。

框架风格与 [Laravel](https://github.com/laravel/laravel) 保持一致，让 Phper 无需学习新框架，也能愉快地使用 Golang！致敬 Laravel！

欢迎 Star、PR 和 Issues！

## 后台管理系统

本项目包含一个基于 Goravel 框架构建的完整后台管理系统。

```bash
git clone https://github.com/wangxuancheng-dev/goravel-admin.git
```

> 文档：https://docs.xuancheng888.top/  
> 演示站：https://admin.xuancheng888.top

账号: demo  
密码: demo123

### 适用场景

**适合：** 企业内部后台、运营管理端、Goravel + Vue / React 二次开发底座。

**不适合：** 直接当金融交易核心、超大规模商业 SaaS 中台；分表 / ES / 多队列需额外运维，请按需开启。

模块分层、最小生产配置与进阶配置说明见：[开源定位与模块](https://docs.xuancheng888.top/guide/opensource)。本地预览文档：`cd website && npm run dev`。

### 截图展示

<p align="center">
  <img src="./images/login.png" alt="登录页面" width="800">
  <p align="center">登录页面</p>
</p>

<p align="center">
  <img src="./images/admin.png" alt="后台管理界面" width="800">
  <p align="center">后台管理界面</p>
</p>

<p align="center">
  <img src="./images/react.jpg" alt="React 后台管理界面" width="800">
  <p align="center">React 后台管理界面</p>
</p>

<p align="center">
  <img src="./images/generator.png" alt="代码生成器管理界面" width="800">
  <p align="center">代码生成器界面</p>
</p>

<p align="center">
  <img src="./images/monitor.png" alt="系统监控" width="800">
  <p align="center">系统监控</p>
</p>

<p align="center">
  <img src="./images/ai.png" alt="AI 代码生成器" width="800">
  <p align="center">AI 代码生成器</p>
</p>

<p align="center">
  <img src="./images/ai-lab.png" alt="AI 实验室" width="800">
  <p align="center">AI 实验室（文本 / 视觉 / 图片 / 语音，演示站可用，按账号限流）</p>
</p>

<p align="center">
  <img src="./images/pages.png" alt="cloudflare" width="800">
  <p align="center">cloudflare</p>
</p>

### 功能特性

#### 核心模块
- **认证与授权**
  - 基于角色的访问控制（RBAC）
  - 权限管理
  - 多令牌管理
  - 在线用户监控与踢出

- **管理员管理**
  - 管理员用户管理
  - 部门管理
  - 角色管理
  - 权限分配
  - 密码重置

- **系统配置**
  - 菜单管理（动态菜单）
  - 字典管理
  - 系统配置
  - 黑名单管理

- **日志与监控**
  - 操作日志（自动记录）
  - 登录日志
  - 系统日志（带追踪 ID）
  - 服务监控

- **附加功能**
  - 数据统计仪表盘
  - 通知中心（WebSocket 实时通知）
  - 数据导出管理
  - **AI 实验室**（顶级菜单，演示 Goravel AI SDK：文本对话、图片理解、图片生成、语音合成、语音转写；配置 `AI_API_KEY` 后可用，按管理员账号限流）
  - 多语言支持（中文/英文）
  - 响应式 UI 设计

#### 进阶模块（可选）

以下能力**默认不必开启**，按业务需要再接入（详见 [开源定位](./website/docs/guide/opensource.md)）：

- 订单 / 支付按月分表、用户余额哈希分表
- Elasticsearch 订单同步与检索
- Redis 异步队列、导出长任务、多队列驱动（Kafka / RabbitMQ / NSQ 等）
- OpenTelemetry 导出到 Jaeger / Grafana
- 支付管理示例（**非完整收单**：无可用回调 / 退款，见 [开源定位](./website/docs/guide/opensource.md)）

### 技术栈

**后端：**
- Goravel 框架（Go）
- JWT 认证
- RBAC 权限系统
- WebSocket 支持
- 数据库迁移与填充

**前端（二选一）：**

| | Vue（`html/`） | React（`html-react/`） |
|---|---|---|
| UI | Element Plus + vxe-table | Ant Design 6 |
| 状态 | Pinia | Zustand |
| 路由 | Vue Router | React Router 7 |
| 国际化 | vue-i18n | react-i18next |
| 共用 | Vite、Axios、ECharts，同一套 Admin API | Vite、Axios、ECharts，同一套 Admin API |

两套前端都对接 `/api/admin`，**新功能需 Vue + React 同发**。React 已覆盖主要管理模块（含代码生成器、AI 实验室等）。详情见 [html-react/README.md](./html-react/README.md)。

### 三分钟 Docker 跑通（推荐）

```bash
cp .env.docker.example .env
docker compose up -d --build
# 等待健康检查通过后访问 http://localhost:3000
# 默认账号 admin / admin123（请尽快修改）
```

前端本地开发仍用 Vite（Vue `html/` → `:3007`，React `html-react/` → `:3008`），API 指向 `http://127.0.0.1:3000`。

### 快速开始（本机 Go + 自备数据库）

1. **后端配置：**
   ```bash
   # 安装依赖
   go mod download
   
   # 在 .env 中配置数据库
   # 运行数据库迁移和填充
   go run . artisan migrate
   go run . artisan db:seed
   
   # 启动服务
   go run . --no-ansi
   # 或使用 air 进行热重载
   air
   ```

2. **前端配置（Vue）：**
   ```bash
   cd html
   
   # 安装依赖
   npm install
   
   # 在 .env 中配置 API 地址
   # VITE_API_BASE_URL=http://127.0.0.1:3000
   # VITE_API_PREFIX=/api/admin
   
   # 启动开发服务器（默认 http://localhost:3007）
   npm run dev
   ```

3. **前端配置（React，可选）：**
   ```bash
   cd html-react
   cp .env.example .env
   npm install
   npm run dev
   ```
   开发地址默认 `http://localhost:3008`。请在根目录 `.env` 的 CORS 中放行该端口（及 `Accept-Language`），然后**重启 Go 后端**。更多说明见 [html-react/README.md](./html-react/README.md)。

4. **默认登录：**
   - 用户名：`admin`
   - 密码：`admin123`
   - （首次登录后请修改默认密码）

### 构建与部署

详细的编译打包和部署说明，包括跨平台编译、Docker 部署、systemd 服务配置等，请参考 [编译部署](./website/docs/deploy/build.md)。

### API 文档

后台管理 API 接口前缀为 `/api/admin`。除登录和验证码接口外，所有接口都需要 JWT 认证。

详细的 API 文档请查看 [routes/admin.go](./routes/admin.go)

#### Swagger API 文档

项目包含 Swagger API 文档，支持交互式 API 探索。

错误码总览（前端联调建议必读）：[错误码](./website/docs/reference/error-codes.md)

**访问 Swagger 文档：**

Swagger JSON 文档访问地址：
- 本地开发：`http://localhost:3000/swagger/index.html`
- 生产环境：`https://your-domain.com/swagger/index.html`

**重新生成 Swagger 文档：**

修改 API 路由或添加新接口后，需要重新生成 Swagger 文档：

```bash
# 生成 Swagger 文档
swag init
```

这将根据代码中的 Swagger 注解（示例见 `main.go`）重新生成 `docs/docs.go`、`docs/swagger.json` 和 `docs/swagger.yaml` 文件。

### 项目结构

```
.
├── app/
│   ├── http/
│   │   ├── controllers/admin/    # 后台控制器
│   │   ├── middleware/           # 自定义中间件（JWT、权限、操作日志）
│   │   └── helpers/              # 辅助函数
│   ├── models/                   # 数据库模型
│   └── services/                 # 业务逻辑服务
├── routes/
│   └── admin.go                  # 后台路由
├── database/
│   ├── migrations/               # 数据库迁移
│   └── seeders/                  # 数据库填充
├── html/                         # 前端 Vue 应用
│   └── src/
│       ├── views/                # 页面组件
│       ├── components/           # 可复用组件
│       ├── api/                 # API 客户端
│       └── store/               # Pinia 状态管理
├── html-react/                   # 前端 React 应用（与 Vue 对照）
│   └── src/
│       ├── pages/                # 页面组件
│       ├── components/           # 可复用组件
│       ├── api/                  # API 客户端
│       └── stores/               # Zustand 状态管理
├── config/                       # 配置文件
├── docs/                         # 文档目录
│   ├── API.md                    # API 接口文档
│   ├── ARCHITECTURE.md           # 架构设计文档
│   ├── BUILD.md                  # 编译打包与部署
│   ├── SHARDING_MIGRATION.md     # 数据库分表指南
│   └── ...                       # 其他文档
├── CONTRIBUTING.md               # 贡献指南
├── CHANGELOG.md                  # 版本变更记录
└── images/                       # 截图文件
```

### 数据库分表（进阶）

项目支持按月分表策略（订单等）。**新用户可先忽略**，仅在数据量大时再启用。  
详细说明：[分表迁移](./website/docs/advanced/sharding-migration.md)、[开源定位](./website/docs/guide/opensource.md)。

### 生产配置

- **最小生产配置**（后台管理为主，不分表 / 不用 ES）：见 [开源定位](./website/docs/guide/opensource.md)
- **完整进阶配置**（队列、分表、ES、OTEL）：见 [开源定位](./website/docs/guide/opensource.md)

### 安全特性

- 基于 JWT 令牌的认证
- 权限中间件保护路由
- 自动操作日志记录
- 日志中敏感数据过滤
- 登录接口限流
- IP/管理员黑名单管理
- 令牌撤销支持

## 快速入门

### 启动服务

`go run . --no-ansi` 或 `air`

[关于 air]：https://www.goravel.dev/getting-started/installation.html#live-reload

### Cloudflare Workers 部署

将前端应用部署到 Cloudflare Workers（Vue：`html/`，React：`html-react/`，配置相同）：

```bash
# 构建前端应用（React 示例；Vue 则 cd html）
cd html-react
# 注意：Cloudflare Workers 构建环境会自动运行 npm ci
# 如果遇到 Rollup 可选依赖问题，请使用以下构建命令：
npm install --include=optional @rollup/rollup-linux-x64-gnu && npm run build

# 或者使用项目提供的 CI 构建脚本：
npm run build:ci

# 部署到 Cloudflare Workers（需使用本目录的 wrangler.toml + worker.js，否则刷新子路由会 404）
npx wrangler deploy
```

**配置说明：**

- **根目录：** `html-react`（Vue 为 `html`）
- **环境变量（变量和机密）：**
  - `VITE_API_BASE_URL`: `https://api.xuancheng888.top`
  - `VITE_API_PREFIX`: `/api/admin`
- **自定义域名：** `admin.xuancheng888.top`

**注意：** `worker.js` 会处理 SPA 路由，当路径不是静态资源时回退到 `index.html`。仅上传 `dist`、不带 Worker 时，刷新 `/admins` 等路径会 404。

### Cloudflare 部署文档站（VitePress）

文档在 `website/`，与后台前端分开部署（独立 Worker / 域名）：

```bash
cd website
npm install
npm run build
npx wrangler deploy
# 或：npm run deploy
```

- **根目录：** `website/`（`wrangler.toml` + `worker.js`）
- **产物：** `docs/.vitepress/dist`
- **建议自定义域名：** `docs.xuancheng888.top`（勿与 `admin.` 共用同一 Worker）
- 详细说明见 [website/README.md](./website/README.md)

### 性能分析

pprof 性能分析工具地址：http://localhost:3000/debug/pprof/

### 二进制压缩

为了减小二进制文件大小，可以使用 UPX（Ultimate Packer for eXecutables）压缩编译后的可执行文件：

**Windows：**

1. 下载 UPX（Windows 64 位版本）：
   - 官网下载：https://github.com/upx/upx/releases/latest
   - 选择 `upx-5.0.2-win64.zip`（或最新版本）
   - 解压到无中文或空格的路径（例如：`F:\tools\upx`）
   - 确保 `upx.exe` 可访问

2. 压缩二进制文件（PowerShell）：
   ```powershell
   # 方式1：临时添加 UPX 到环境变量（推荐）
   $env:PATH += ";F:\tools\upx"
   
   # 进入项目目录
   cd F:\www\go\admin\goravel-admin
   
   # 最高级别压缩（-9）
   upx -9 main
   ```

**Linux/macOS：**

```bash
# 安装 UPX（如果尚未安装）
# Ubuntu/Debian: sudo apt-get install upx
# macOS: brew install upx

# 压缩二进制文件
upx -9 main
```

## 文档

### 项目文档

**推荐：** [VitePress 文档站](./website/README.md) — **只维护** `website/docs/`（及 `website/docs/en/`）

```bash
cd website && npm install && npm run dev
```

| 文档 | 说明 |
|------|------|
| [文档站说明](./website/README.md) | 本地预览 / 构建 |
| [开源定位](./website/docs/guide/opensource.md) | 开源定位、核心/进阶模块 |
| [快速开始](./website/docs/guide/getting-started.md) | Docker 三分钟跑通 |
| [系统架构](./website/docs/guide/architecture.md) | 系统架构 |
| [开发指南](./website/docs/guide/development.md) | 二次开发指南 |
| [生产清单](./website/docs/deploy/production.md) | 生产上线清单 |
| [前端开发指南（Vue）](./html/DEVELOPMENT.md) | Vue 前端 |
| [前端说明（React）](./html-react/README.md) | React 前端 |

> 根目录 `docs/` 仅保留 Swagger 产物（`docs.go` / `swagger.json` / `swagger.yaml`），项目说明文档不在此维护。

### Goravel 框架文档

在线文档 [https://www.goravel.dev](https://www.goravel.dev)

> 要优化文档，请向文档仓库提交 PR
> [https://github.com/goravel/docs](https://github.com/goravel/docs)

## 社区

欢迎在 Discord 中讨论。

[https://discord.gg/cFc5csczzS](https://discord.gg/cFc5csczzS)

## 许可证

Goravel 框架是在 [MIT 许可证](https://opensource.org/licenses/MIT) 下发布的开源软件。


