---
layout: home

hero:
  name: Goravel Admin
  text: 中小 SaaS 运维 + 后台脚手架
  tagline: Goravel · 一户一库 · 平台控制台 · React（主）/ Vue · RBAC · Docker
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: 开源定位与模块
      link: /guide/opensource
    - theme: alt
      text: GitHub
      link: https://github.com/wangxuancheng-dev/goravel-admin

features:
  - title: 建议阅读顺序
    details: 快速开始 → 开源定位（能力分档）→ 架构 → 开发 → 生产最小路径 / SaaS 清单。规模旋钮、搜索、分表按需再看。
  - title: 平台运维
    details: 一户一库、平台控制台、开户迁移门禁、health/ready 与队列积压告警。
  - title: 双前端
    details: React（html-react）为默认前端；Vue 对齐同一套 Admin API。见双前端对齐清单。
  - title: 核心与进阶
    details: RBAC、日志、导出、代码生成开箱可用；分表 / ES / 多租户 / 支付网关按模块开启。
---

## 文档导航

| 章节 | 说明 |
|------|------|
| [快速开始](/guide/getting-started) | 本机或 Docker 第一次跑起来 |
| [开源定位与模块](/guide/opensource) | 模块边界、**能力分档**、阅读路径 |
| [系统架构](/guide/architecture) | 二次开发前摸清结构 |
| [开发指南](/guide/development) / [代码生成器](/guide/code-generator) | 加业务 CRUD（生成器优先） |
| [生产清单](/deploy/production) / [SaaS 核对清单](/advanced/saas) | 上线前核对 |
| [多租户业务须知](/advanced/tenancy-dev) | 写业务前必读（短） |
| [进阶](/advanced/tenancy) / [规模](/advanced/tenancy-scaling) / [参考](/reference/api) | 租户运维全文、户多再调、支付/分表等 |

租户后台：https://acme.xuancheng888.top/login?tenant_code=acme
租户运维：https://admin.xuancheng888.top/platform/login
账号/密码（两处相同）：demo / demo123（勿用于生产）
