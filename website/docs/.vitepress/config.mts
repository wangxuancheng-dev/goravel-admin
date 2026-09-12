import { defineConfig } from 'vitepress'

const zhNav = [
  { text: '指南', link: '/guide/getting-started' },
  { text: '部署', link: '/deploy/production' },
  { text: '进阶', link: '/advanced/tenancy' },
  { text: '参考', link: '/reference/api' },
]

const enNav = [
  { text: 'Guide', link: '/en/guide/getting-started' },
  { text: 'Deploy', link: '/en/deploy/production' },
  { text: 'Advanced', link: '/en/advanced/tenancy' },
  { text: 'Reference', link: '/en/reference/api' },
]

const zhSidebar = {
  '/guide/': [
    {
      text: '开始使用',
      collapsed: false,
      items: [
        { text: '项目简介', link: '/' },
        { text: '1. Docker 快速开始', link: '/guide/getting-started' },
        { text: '2. 开源定位与模块', link: '/guide/opensource' },
        { text: '3. 系统架构', link: '/guide/architecture' },
      ],
    },
    {
      text: '二次开发',
      collapsed: false,
      items: [
        { text: '4. 开发指南', link: '/guide/development' },
        { text: '5. 代码生成器', link: '/guide/code-generator' },
        { text: '6. 双前端对齐', link: '/guide/frontend-parity' },
        { text: '7. 权限按钮配置', link: '/guide/permission-button' },
        { text: '硬编码检查清单', link: '/guide/hardcoded-checklist' },
      ],
    },
    {
      text: '质量与协作',
      collapsed: false,
      items: [
        { text: '8. 测试指南', link: '/guide/testing' },
        { text: '9. 贡献指南', link: '/guide/contributing' },
        { text: '变更日志', link: '/guide/changelog' },
      ],
    },
  ],
  '/deploy/': [
    {
      text: '部署上线',
      collapsed: false,
      items: [
        { text: '1. 生产清单', link: '/deploy/production' },
        { text: '2. 编译与部署', link: '/deploy/build' },
        { text: '3. Docker 生产部署', link: '/deploy/docker' },
      ],
    },
  ],
  '/advanced/': [
    {
      text: '进阶能力（按需）',
      collapsed: false,
      items: [
        { text: '1. 多租户', link: '/advanced/tenancy' },
        { text: '2. 支付参考', link: '/advanced/payments' },
        { text: '3. 搜索（ES / Meili）', link: '/advanced/search' },
        { text: '4. 分表指南', link: '/advanced/sharding' },
        { text: '5. 分表迁移', link: '/advanced/sharding-migration' },
        { text: '6. 分表查询服务', link: '/advanced/sharding-query' },
        { text: '7. SaaS 核对清单', link: '/advanced/saas' },
        { text: '8. 迁移策略', link: '/advanced/migration-strategy' },
        { text: '9. AI 模块提示词', link: '/advanced/ai-module' },
      ],
    },
  ],
  '/reference/': [
    {
      text: '参考手册',
      collapsed: false,
      items: [
        { text: 'API 概览', link: '/reference/api' },
        { text: '错误码', link: '/reference/error-codes' },
        { text: 'MySQL', link: '/reference/mysql' },
        { text: 'PostgreSQL', link: '/reference/postgresql' },
        { text: '系统日志', link: '/reference/system-log' },
        { text: '生产 pprof', link: '/reference/pprof' },
        { text: 'Facades 管理', link: '/reference/facades' },
        { text: '可选驱动仓库', link: '/reference/drivers' },
      ],
    },
  ],
}

const enSidebar = {
  '/en/guide/': [
    {
      text: 'Getting started',
      collapsed: false,
      items: [
        { text: 'Introduction', link: '/en/' },
        { text: '1. Docker quick start', link: '/en/guide/getting-started' },
        { text: '2. Open-source scope', link: '/en/guide/opensource' },
        { text: '3. Architecture', link: '/en/guide/architecture' },
      ],
    },
    {
      text: 'Secondary development',
      collapsed: false,
      items: [
        { text: '4. Development guide', link: '/en/guide/development' },
        { text: '5. Code generator', link: '/en/guide/code-generator' },
        { text: '6. Frontend parity', link: '/en/guide/frontend-parity' },
        { text: '7. Permission buttons', link: '/en/guide/permission-button' },
        { text: 'Hardcoded checklist', link: '/en/guide/hardcoded-checklist' },
      ],
    },
    {
      text: 'Quality & collaboration',
      collapsed: false,
      items: [
        { text: '8. Testing', link: '/en/guide/testing' },
        { text: '9. Contributing', link: '/en/guide/contributing' },
        { text: 'Changelog', link: '/en/guide/changelog' },
      ],
    },
  ],
  '/en/deploy/': [
    {
      text: 'Deploy',
      collapsed: false,
      items: [
        { text: '1. Production checklist', link: '/en/deploy/production' },
        { text: '2. Build & deploy', link: '/en/deploy/build' },
        { text: '3. Docker production', link: '/en/deploy/docker' },
      ],
    },
  ],
  '/en/advanced/': [
    {
      text: 'Advanced (optional)',
      collapsed: false,
      items: [
        { text: '1. Multi-tenancy', link: '/en/advanced/tenancy' },
        { text: '2. Payments', link: '/en/advanced/payments' },
        { text: '3. Search', link: '/en/advanced/search' },
        { text: '4. Sharding guide', link: '/en/advanced/sharding' },
        { text: '5. Sharding migration', link: '/en/advanced/sharding-migration' },
        { text: '6. Sharding query service', link: '/en/advanced/sharding-query' },
        { text: '7. SaaS checklist', link: '/en/advanced/saas' },
        { text: '8. Migration strategy', link: '/en/advanced/migration-strategy' },
        { text: '9. AI module prompts', link: '/en/advanced/ai-module' },
      ],
    },
  ],
  '/en/reference/': [
    {
      text: 'Reference',
      collapsed: false,
      items: [
        { text: 'API overview', link: '/en/reference/api' },
        { text: 'Error codes', link: '/en/reference/error-codes' },
        { text: 'MySQL', link: '/en/reference/mysql' },
        { text: 'PostgreSQL', link: '/en/reference/postgresql' },
        { text: 'System logs', link: '/en/reference/system-log' },
        { text: 'Production pprof', link: '/en/reference/pprof' },
        { text: 'Facades', link: '/en/reference/facades' },
        { text: 'Optional drivers', link: '/en/reference/drivers' },
      ],
    },
  ],
}

export default defineConfig({
  title: 'Goravel Admin',
  description: 'Goravel admin starter — Vue / React dual frontend',
  lastUpdated: true,
  cleanUrls: true,
  ignoreDeadLinks: true,
  srcExclude: ['zh/**'],
  markdown: {
    languageAlias: {
      env: 'ini',
    },
  },

  locales: {
    root: {
      label: '简体中文',
      lang: 'zh-CN',
      title: 'Goravel Admin',
      description: '基于 Goravel 的开箱可跑后台：Vue3 + React 双前端',
      themeConfig: {
        nav: zhNav,
        sidebar: zhSidebar,
        outline: { label: '本页目录' },
        docFooter: { prev: '上一页', next: '下一页' },
        lastUpdated: { text: '最后更新' },
        returnToTopLabel: '回到顶部',
        sidebarMenuLabel: '菜单',
        darkModeSwitchLabel: '主题',
      },
    },
    en: {
      label: 'English',
      lang: 'en-US',
      link: '/en/',
      title: 'Goravel Admin',
      description: 'Goravel admin starter with Vue 3 and React frontends',
      themeConfig: {
        nav: enNav,
        sidebar: enSidebar,
        outline: { label: 'On this page' },
      },
    },
  },

  themeConfig: {
    logo: 'https://www.goravel.dev/logo.png?v=1.14.x',
    socialLinks: [
      { icon: 'github', link: 'https://github.com/wangxuancheng-dev/goravel-admin' },
    ],
    search: { provider: 'local' },
  },
})
