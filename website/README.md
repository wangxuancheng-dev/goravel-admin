# Goravel Admin Docs (VitePress)

**唯一项目文档正文。** 只编辑：

- `docs/` — 简体中文（默认）
- `docs/en/` — English

仓库根目录 `docs/` 仅存放 Swagger 生成物，与本文档站无关。

## Develop

```bash
cd website
npm install
npm run dev
```

本地默认：`http://localhost:5175`

## Build

```bash
cd website
npm install
npm run build
npm run serve
```

产物目录：`docs/.vitepress/dist`

## Cloudflare Workers 部署

与后台前端相同技术栈（Workers + Assets），**单独一个 Worker**，不要和 `html/` / `html-react/` 的 `admin` Worker 混用。

```bash
cd website
npm install
npm run build
npx wrangler deploy
```

**配置说明：**

| 项 | 值 |
|----|-----|
| 根目录 | `website/` |
| Worker 名 | `goravel-admin-docs`（见 `wrangler.toml`） |
| 静态资源 | `docs/.vitepress/dist` |
| 路由兜底 | `worker.js`（clean URL → `.html` / `index.html`） |
| 建议自定义域名 | `docs.你的域名`（例如 `docs.xuancheng888.top`） |

文档站是静态站，**不需要**配置 `VITE_API_*`。

挂子域时 VitePress `base` 保持 `/` 即可。若必须挂在路径下（如 `https://admin.example.com/docs/`），需在 `docs/.vitepress/config.mts` 设置 `base: '/docs/'` 后再构建。

### Cloudflare Pages（可选）

Dashboard → Workers & Pages → 连接本仓库：

| 项 | 值 |
|----|-----|
| Root directory | `website` |
| Build command | `npm run build` |
| Build output directory | `docs/.vitepress/dist` |
| Node.js version | `20` |

Pages 同样建议绑独立 `docs.` 子域。

## Layout

| Path | Content |
|------|---------|
| `docs/` | 简体中文 |
| `docs/en/` | English |
| `docs/.vitepress/config.mts` | Nav / sidebar / i18n |
| `wrangler.toml` / `worker.js` | Cloudflare Workers 部署 |
