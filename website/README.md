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
npm run build
npm run serve
```

## Layout

| Path | Content |
|------|---------|
| `docs/` | 简体中文 |
| `docs/en/` | English |
| `docs/.vitepress/config.mts` | Nav / sidebar / i18n |
