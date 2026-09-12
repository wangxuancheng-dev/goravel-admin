---
layout: home

hero:
  name: Goravel Admin
  text: 开箱可跑的后台管理系统
  tagline: Golang Goravel · Vue3 + Element Plus / React + Ant Design · RBAC · 代码生成 · Docker
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
    details: 快速开始 → 开源定位 → 架构 → 开发指南 → 生产清单。进阶能力（租户 / 支付 / 搜索 / 分表）按需查阅。
  - title: 双前端
    details: Vue 与 React 共用同一套 Admin API；新功能按 FRONTEND_PARITY 约定同发。
  - title: 核心开箱、进阶可选
    details: RBAC、日志、导出、代码生成器默认可跑；分表 / ES / 多租户 / 支付网关按模块开启。
---

## 文档怎么读

| 优先级 | 章节 | 适合谁 |
|--------|------|--------|
| 1 | [快速开始](/guide/getting-started) | 本机或 Docker 第一次跑起来 |
| 2 | [开源定位与模块](/guide/opensource) | 判断能不能用、开哪些模块 |
| 3 | [系统架构](/guide/architecture) | 二次开发前摸清结构 |
| 4 | [开发指南](/guide/development) / [代码生成器](/guide/code-generator) | 加业务 CRUD |
| 5 | [生产清单](/deploy/production) | 上线前核对 |
| 按需 | [进阶](/advanced/tenancy) / [参考](/reference/api) | 租户、支付、分表、数据库等 |

演示站：https://admin.xuancheng888.top （demo / demo123，勿用于生产）
