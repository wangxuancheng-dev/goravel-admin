# 多租户：规模与推荐配置

> 本页从 [多租户](/advanced/tenancy) 拆出：生产调参与容量规划。业务开发只需读 [多租户：业务开发须知](/advanced/tenancy-dev)。

以下为**经验起点**，需按监控回调，**不是硬性配额**。户多时首要瓶颈通常是**租户库连接池**（不是 Redis）。Redis 全站共用（缓存键经 `tenancy.CacheKey` 加 `t{id}:` 前缀），一般升规格即可；队列与缓存吵邻居时再考虑拆实例或分 DB。

## 出厂默认能支撑多少

「活跃」指**近期有流量、进程内已注册连接池的租户**，不是 `tenants` 表总行数。

| 场景 | 建议 |
|------|------|
| **默认池 + 单机部署** | 稳妥大约 **&lt; 50 活跃**；储备 **50–200**（建议已收紧 `OPEN`） |
| **调参 + Redis 队列 + API/Worker 分开** | **数百～约 1k 活跃** |
| **再加 `tenants.host` 分库主机 + `queue-schedule` Worker** | **几千活跃**（本产品目标量级） |
| **再加按区域多套完整部署（cell）** | **超出单舰队**时复制栈（见本节下文） |

默认 `TENANCY_POOL_MAX_OPEN_CONNS=20`、`TENANCY_REGISTERED_MAX=0`（不限）。**几千商户不要原样上生产**，按下表调参。

**容量公式（租户库）：**

`进程内已注册的租户池数 × TENANCY_POOL_MAX_OPEN_CONNS × API实例数` ≪ 数据库 `max_connections`（预留平台库、备份、运维余量）。

同机多库时所有 `tenant_*` 仍计入**同一** MySQL/PG 实例的连接上限。

## 运行时双模式（少量 / 大量）

| 机制 | 配置 | 少量租户 | 大量租户 |
|------|------|----------|----------|
| 连接注册表空闲淘汰 | `TENANCY_REGISTERED_IDLE_TTL`（默认 900s） | 长时间无访问会释放池 | 避免注册池涨到总户数 |
| 注册池硬上限 | `TENANCY_REGISTERED_MAX`（默认 0=不限） | 可不设 | 建议 300–500 |
| 全舰队命令分页 | `TENANCY_SCOPE_AUTO_BATCH_AT`（200） / `TENANCY_SCOPE_AUTO_BATCH`（100） | 户数 ≤200 时一次跑完 | 超过阈值自动分页；分钟级任务带 rotate 游标 |
| 强制页大小 | `TENANCY_SCOPE_BATCH` | 0=跟自动策略 | 可强制每页 N 户 |
| 灵活定时 | `next_run_at` + tick 扇出 + `schedule` 队列（`QUEUE_SCHEDULE_CONCURRENT`） | 行为与按分钟匹配一致 | tick **只入队**；Worker 并行执行 handler |
| 全舰队备份 | `tenant:backup-all` / `backup-scheduled` | 小舰队可一次入队 | 入队 `long-running`；定时按天轮转 `TENANT_BACKUP_SCHEDULE_BATCH` |
| 全舰队 migrate/seed | `TENANCY_MIGRATE_CONCURRENCY` / `--concurrency` | 默认 2；单机同 host 建议 2–5 | 多 `tenants.host` 时可 20–50（真实并行 DDL/seed；勿盲目开满打爆 DB） |

## 全舰队 schema 迁移（`tenant:migrate-all`）

一户一库时每户独立 DDL/seed，墙钟大致为 `户数 × 单户耗时 / 并发`。`MigrateTenant` / `SeedTenant` 使用每 goroutine 绑定的 Schema+Orm（不翻转全局 `database.default`），故 `TENANCY_MIGRATE_CONCURRENCY` 对 migrate-all 与 seed-all 均为 **真正并行**。压力落在各租户库所在的 MySQL/PG 上。

| 部署 | 建议并发 | 说明 |
|------|----------|------|
| 单机 / 同 host 多库 | 2–5 | 默认 2；再高易顶 `max_connections` / 磁盘 IO |
| 约 100 户 / 一台租户库机，多 host | 20–50 | 开户填不同 `tenants.host` 分散；仍是**全局**并发上限，非按 host 分组 |
| 平台 UI 批量 migrate/seed | `TENANCY_MIGRATE_CONCURRENCY`（`tenant_ops_fleet`） | 与 CLI 同路径；`QUEUE_LONG_RUNNING_CONCURRENT=1` 即可 |

失败户会写 `provision_status=failed` + `last_migrate_error`，可筛失败后批量重试，不必每次全舰队重跑。发版：新镜像先平台 `migrate`，再 `tenant:migrate-all`（可 `--concurrency`），再切流量。

## 并发旋钮对照（与队列一起调）

这些参数**不是**同一个东西，调高其中一个不会自动提高另一个：

| 参数 | 控制什么 | 典型场景 |
|------|----------|----------|
| `TENANCY_MIGRATE_CONCURRENCY` | **本进程内** `tenant:migrate-all` / `tenant:seed-all` 同时处理几户（goroutine） | CLI 发版窗口全舰队 DDL/seed |
| `QUEUE_LONG_RUNNING_CONCURRENT` | **每个** long-running Worker 同时跑几个任务 | 单户 `tenant_ops`；**批量 migrate/seed/backup 只需 1**（fleet 占一槽） |
| `QUEUE_SCHEDULE_CONCURRENT` | **每个** schedule Worker 同时跑几个灵活定时/采集 job | 日常分钟级业务，与 migrate **抢租户库 IO** |
| `QUEUE_CONCURRENT` | default 队列并行度 | 导出、通知等普通任务 |

**一起调的原则（同机租户库）：**

1. **发版跑 `migrate-all` / `seed-all` 时**：把 `QUEUE_SCHEDULE_CONCURRENT` 临时降到 `1–2`（或先停 schedule tick），`QUEUE_LONG_RUNNING_CONCURRENT` 保持 `1`，**不要**同时在平台 UI 再点批量迁移——否则 CLI 并发 + Worker 并发叠在同一批 MySQL/PG 上。
2. **墙钟**：`≈ ceil(户数 / TENANCY_MIGRATE_CONCURRENCY) × 单户 migrate 耗时`；seed 通常远快于 migrate，可共用同一并发值。
3. **日常运行**：`TENANCY_MIGRATE_CONCURRENCY` 只影响 CLI，可长期保持较大；真正常驻压力看 `QUEUE_SCHEDULE_CONCURRENT × Worker 台数` 与 `TENANCY_POOL_MAX_OPEN_CONNS`。
4. **经验上界**：单机同 host 建议 CLI 并发 `2–5`；多 `tenants.host` 才开到 `20–50`。队列侧 long-running 默认 `1` 即可，schedule 按采集 SLA 单独加。
5. **批量 migrate/seed**：CLI 与平台 UI 批量都读 `TENANCY_MIGRATE_CONCURRENCY`。`QUEUE_LONG_RUNNING_CONCURRENT` 不必跟着开大。发版时仍勿同时 CLI + UI 两路叠压同一批库。

**平台 UI 批量 migrate / seed / backup：**

1. **批量** migrate/seed 入队 **一个** `tenant_ops_fleet` 任务，内部用 `TENANCY_MIGRATE_CONCURRENCY`（与 CLI `migrate-all` 同路径）。单户操作仍走普通 `tenant_ops`。`QUEUE_LONG_RUNNING_CONCURRENT` 对批量迁保持 **1** 即可（fleet 只占一个队列槽）。
2. Worker 机须跑 `queue-long-running`（勿设 `APP_DISABLED_RUNNERS=queue-*`）；API 机通常只 Dispatch。
3. 改 `.env` 后须 **重启消费 long-running 的 Worker 进程**（并发在启动时读入，见 `bootstrap/runners.go`）。只改 env 不重启 Worker → 仍是旧并发。只重启 API、Worker 不重启 → UI 入队照常但并行度不变。
4. 批量迁速度靠调大 `TENANCY_MIGRATE_CONCURRENCY`（改完 **下次** 入队的 fleet 即生效，或重启 Worker 前已入队的不受影响）。backup 批量同样走 `tenant_ops_fleet`，并行看 `TENANCY_BACKUP_CONCURRENCY`（`QUEUE_LONG_RUNNING_CONCURRENT=1` 即可）。

| 改哪个 | 要不要重启 | 说明 |
|--------|------------|------|
| `QUEUE_LONG_RUNNING_CONCURRENT` / `QUEUE_SCHEDULE_CONCURRENT` / `QUEUE_CONCURRENT` | **要**：重启对应队列 Worker | 启动时固定 Concurrent |
| `TENANCY_MIGRATE_CONCURRENCY` | CLI **下次** `artisan tenant:migrate-all` 即生效；长期跑的 API/Worker 不必为它重启 | 也可用一次性 `--concurrency=N`，无需改 env |
| 平台 UI 点迁移 | 不重启也能入队 | 并行度仍取决于已启动的 Worker 并发 |

**建议取值（可直接抄，同机一库主机）：**

| 场景 | `TENANCY_MIGRATE_CONCURRENCY`（CLI / UI 批量 migrate·seed） | `QUEUE_LONG_RUNNING_CONCURRENT`（单户 / backup 批量） | `QUEUE_SCHEDULE_CONCURRENT` |
|------|--------------------------------------|----------------------------------------|------------------------------|
| 本地 / 小舰队 &lt;50 户 | **2–3**（默认 2） | **1** | **5–10**（默认常 10） |
| 单机同 host，发版 migrate-all | **3–5** | **1**（发版窗口勿开大） | 临时 **1–2** 或停 tick |
| ~100 户/台库机，已分 `tenants.host` | **20–40** | **1–2** | **5–10** |
| 几千户、多 host | **40–50**（上限 100，慎用） | **1–2**；更快就加 Worker 台 | 按 SLA，常从 **10** 起 |

经验：UI 批量迁调 `TENANCY_MIGRATE_CONCURRENCY`（与 CLI 相同）；`QUEUE_LONG_RUNNING_CONCURRENT` 默认 **1** 即可。墙钟粗算 `ceil(户数/并发) × 单户 migrate 秒数`（本仓库实测单户 DDL 约十余秒量级）。

## 活跃量级 × 调参对照

| 活跃商户 | 租户池 | 队列 / 进程 | Redis |
|----------------|------|-------------|-------|
| &lt; 50 | 默认 `IDLE=2` / `OPEN=20` 可留；可先降到 `OPEN=10` | 可同机 Worker；`QUEUE_CONNECTION=redis` | 单机或小规格云 Redis |
| 50–200 | `IDLE=1–2`，`OPEN=3–5`；缩短 idle/lifetime（如 120 / 600） | API 与 Worker **分角色**（见 [生产清单](/deploy/production) §4.1）；`QUEUE_LONG_RUNNING_CONCURRENT` 保持较小 | 托管 Redis，盯 `used_memory` 与 backlog |
| 200–1000 | `OPEN=2–3`；`REGISTERED_MAX=300` 可选 | 独立 Worker：`default` + `long-running` + **`schedule`** | 更大规格；导出导入注意吵邻居 |
| 几千 | `OPEN=2`；`REGISTERED_MAX=300–500`；`tenants.host` **分库主机**；migrate 可 `TENANCY_MIGRATE_CONCURRENCY=20–50` | Worker 可水平扩；`QUEUE_SCHEDULE_CONCURRENT` 按采集耗时调 | 托管高可用；吵邻居再拆缓存/队列 |

## 机器规模与角色分布（经验）

同一二进制；用 `APP_DISABLED_RUNNERS` 区分角色（详见 [生产清单 §4.1](/deploy/production)）。以下 **vCPU / 内存** 为云主机粗算，按实际 QPS、采集耗时、导出频次加减。

| 活跃户量级 | API | Queue Worker | Scheduler | 平台 DB | 租户 DB | Redis |
|--------------|-----|--------------|-----------|---------|---------|-------|
| &lt; 50 | 1× 2vCPU 4GB（可同机跑 Worker） | 可合并 | 合并或单独 1 台 | 1×中小规格 | 可与平台同机多库 | 1×小规格 |
| 50–200 | 2× 2vCPU 4GB（只 HTTP） | 1× 2vCPU 4GB | **只 1 台** 开 schedule | 1×（提高`max_connections`） | 同机或开始拆 1 台租户库机 | 1×托管 |
| 200–1000 | 2–4× API | 2+× Worker（含 `schedule`） | 1 台 | 独立平台库 | **多台** `tenants.host` 分片 | 1×中大规格 HA |
| 几千 | 4+× API（LB） | 3+× Worker；`QUEUE_SCHEDULE_CONCURRENT`按需调 | 1 台 | 独立 + 备份 | 多台租户库（按 host 分配开户） | HA；吵邻居再拆 |

**角色要点：**

| 角色 | `APP_DISABLED_RUNNERS` | 必开队列 Runner |
|------|-------------------------|----------------|
| API | `queue-*` | 无（只 Dispatch） |
| Worker | 留空 | `queue-default`、`queue-long-running`、**`queue-schedule`**（灵活定时）、可选 `queue-search` |
| Scheduler | 可 `queue-*`（若不消费） | 只跑 `goravel:schedule`；**全站只 1 台** |

分钟级采集吞吐粗算：`并发槽 ≈ QUEUE_SCHEDULE_CONCURRENT × Worker 台数`。若 3000 户/分钟且每户耗时 5s，约需 **250** 并发槽（或降频）。

## 示例配置

约 100 活跃户、2 个 API：

```ini
TENANCY_POOL_MAX_IDLE_CONNS=1
TENANCY_POOL_MAX_OPEN_CONNS=5
TENANCY_POOL_CONN_MAX_IDLETIME=120
TENANCY_POOL_CONN_MAX_LIFETIME=600
TENANCY_REGISTERED_IDLE_TTL=900

CACHE_STORE=redis
QUEUE_CONNECTION=redis
QUEUE_CONCURRENT=2
QUEUE_LONG_RUNNING_CONCURRENT=1
QUEUE_SCHEDULE_CONCURRENT=5
# API 机：APP_DISABLED_RUNNERS=queue-*
```

几千活跃起点：

```ini
TENANCY_POOL_MAX_IDLE_CONNS=1
TENANCY_POOL_MAX_OPEN_CONNS=2
TENANCY_REGISTERED_IDLE_TTL=900
TENANCY_REGISTERED_MAX=400
TENANCY_SCOPE_AUTO_BATCH_AT=200
TENANCY_SCOPE_AUTO_BATCH=100
TENANCY_FLEX_SCHEDULE_TICK_LIMIT=2000
TENANCY_FLEX_SCHEDULE_QUEUE=schedule
TENANCY_MIGRATE_CONCURRENCY=40
QUEUE_SCHEDULE_CONCURRENT=10
TENANT_BACKUP_SCHEDULE_ENABLED=true
TENANT_BACKUP_SCHEDULE_BATCH=100
QUEUE_LONG_RUNNING_CONCURRENT=1
# 开户时填 tenants.host 分库主机；API 关 queue-* ；Worker 开 queue-schedule
```

**建议监控：** MySQL/PG `Threads_connected`、队列 pending / `queue:alert-backlog`、`schedule` / `long-running` 积压、Redis 内存与连接数。接近上限时先下调 `TENANCY_POOL_*` 或扩容 DB/Worker，**不要盲目加 API 副本**（副本会放大连接占用）。

## 超出单舰队：按区域多套部署

单舰队目标大约是**几千活跃**（见上表）。户数继续增长时，优先**新开一套完整部署（区域 / cell）**，而不是在同一平台库上无限加 API 或堆更多 `tenant_*`。

推荐模式：

```text
区域 A（存量）              区域 B（新区）
平台库 A + API/Worker…  →  平台库 B + API/Worker…
老租户继续留在 A            新开户只写入 B
```

| 要点 | 说明 |
|------|------|
| 每区域一套栈 | 独立平台库、`TENANCY_DRIVER=database`、API / Worker / Scheduler / Redis、租户库（可再按 `tenants.host` 分机） |
| 开户指向 | 注册/销售入口默认打到**当前写入区域**的 platform/API；从某日起新户只进新区即可 |
| 租户如何回访 | 子域/根域按区域区分，或在网关/DNS 上维护极薄的 `tenant_code → 区域 API` 目录；勿把老户域名指到错区域 |
| 平台控制台 | 每区各自 `/platform`；本产品**无**跨区域统一控制面 |
| 迁户 | 尽量不搬；跨区等于备份恢复 + 改域名/目录，成本高 |
| 冷户 | `disabled` / 归档或 DROP 后保留平台行；勿让大量无流量但仍 `ready` 的库拖垮单区 backup / `migrate-all` |

**没有内置的** `CELL_ID`、全球租户路由表或一键跨区迁移。区域扩展是**运维级复制部署**；组内仍按本文「规模与推荐配置」调参即可。单组常见容量约一千户量级时最省心，总规模用「组数 × 组容量」放大。
