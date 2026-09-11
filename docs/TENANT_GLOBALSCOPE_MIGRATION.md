# 多租户说明（GlobalScopes 文档已过时）

> **重要：** 当前生产路径是 `TENANCY_DRIVER=database`（一户一库 / Schema），通过 `Tenant` 中间件 + `OrmQuery(ctx)` **切换数据库连接**隔离，**不是**行级 `tenant_id` GlobalScope。
>
> 权威约定见 [TENANT_RESERVED.md](./TENANT_RESERVED.md)。

本文档描述的是早期「单库 + `tenant_id` 列 + GlobalScopes」设想，**请勿再按本文改造业务模型**。若你仍维护旧的行级多租户实验分支，可仅作历史参考；新功能一律走连接级租户。
