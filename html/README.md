# 后台管理系统前端（Vue 对等端）

默认前端是 React（`html-react/`）。本目录为 Vue 3 对等实现，API 契约对齐；改功能时 React 优先，见 [双前端对齐](../website/docs/guide/frontend-parity.md)。


基于 Vue 3 + Element Plus + vxe-table 的后台管理系统。

## 技术栈

- Vue 3
- Element Plus
- vxe-table
- Vue Router
- Pinia
- Axios
- ECharts
- vue-i18n

## 环境配置

项目使用 `.env` 文件管理环境变量。

1. 复制环境变量示例文件：
```bash
cp .env.example .env
```

2. 编辑 `.env` 文件，配置 API 地址：
```env
# API 基础地址
VITE_API_BASE_URL=http://127.0.0.1:3000

# API 前缀
VITE_API_PREFIX=/api/admin

# WebSocket 基础地址（可选，如果配置了单独的 WebSocket 域名）
# 如果不配置，将使用 VITE_API_BASE_URL
# VITE_WS_BASE_URL=wss://wss.xuancheng888.top
# 或者
# VITE_WS_BASE_URL=https://wss.xuancheng888.top
```

## 安装依赖

```bash
npm install
```

## 开发

```bash
npm run dev
```

开发服务器将在 `http://localhost:3007` 启动。

## 构建

```bash
npm run build
```

构建后的文件会输出到 `dist` 目录。

## 功能模块

- 登录认证
- 仪表盘
- 管理员管理
- 角色管理
- 权限管理
- 菜单管理
- 部门管理
- 字典管理
- 操作日志
- 登录日志
- 系统日志
- **AI 实验室**（文本 / 视觉 / 图片 / 语音 SDK 演示，需配置服务端 `AI_API_KEY`）

## 多语言支持

项目支持中文和英文两种语言，可以通过页面右上角的语言切换按钮进行切换。

