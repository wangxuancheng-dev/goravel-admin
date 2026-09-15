# Docker 蓝绿部署脚本

## 文件说明

- `deploy-config.sh` - **端口配置文件**（统一管理所有端口配置）
- `docker-blue-green.sh` - 蓝绿部署主脚本，执行零停机部署
- `git-deploy.sh` - 从 Git 拉取代码并自动部署的脚本
- `rollback.sh` - 快速回滚到上一个版本
- `rollback-git.sh` - 从 Git 回滚到指定版本并部署
- `migrate.sh` - 单独执行数据库迁移脚本
- `seed.sh` - 单独执行数据填充脚本

## 端口配置

**重要：** 所有端口配置统一在 `deploy-config.sh` 文件中管理。

### 修改端口

只需修改 `scripts/deploy/deploy-config.sh` 文件：

```bash
# 编辑配置文件
vim scripts/deploy/deploy-config.sh

# 修改端口配置：
export BLUE_PORT="${BLUE_PORT:-3000}"      # 蓝环境端口
export GREEN_PORT="${GREEN_PORT:-3001}"    # 绿环境端口
export CONTAINER_PORT="${CONTAINER_PORT:-3000}"  # 容器内部端口
```

### 通过环境变量覆盖

```bash
# 临时使用其他端口
export BLUE_PORT=4000
export GREEN_PORT=4001
./scripts/deploy/docker-blue-green.sh
```

详细说明请参考：`scripts/deploy/README-PORT-CONFIG.md`

## 使用方法

### 方式一：手动部署（已拉取代码）

```bash
cd /www/goravel-admin
chmod +x scripts/deploy/docker-blue-green.sh
./scripts/deploy/docker-blue-green.sh
```

### 方式二：从 Git 拉取并部署

```bash
# 设置环境变量（可选）
export GIT_REPO_URL="https://github.com/your-username/goravel-admin.git"
export GIT_BRANCH="main"
export DEPLOY_DIR="/www/goravel-admin"

# 执行部署
chmod +x scripts/deploy/git-deploy.sh
./scripts/deploy/git-deploy.sh
```

或者直接修改脚本中的配置：

```bash
# 编辑脚本
vim scripts/deploy/git-deploy.sh

# 修改这些变量：
# GIT_REPO_URL="你的仓库地址"
# GIT_BRANCH="main"
# DEPLOY_DIR="/www/goravel-admin"
```

## 部署流程

1. **检测当前版本** - 自动检测运行的是 blue 还是 green
2. **构建新版本** - 在备用环境构建新镜像（包含启动脚本）
3. **启动新版本** - 启动新版本容器（不同端口）
4. **自动执行迁移** - 容器启动脚本会在应用启动前自动执行数据库迁移
5. **数据填充（可选）** - 提示是否执行数据填充
6. **健康检查** - 等待新版本通过健康检查
7. **切换流量** - 更新 Nginx 配置（如果存在）
8. **停止旧版本** - 停止旧版本容器

## Multi-tenant (TENANCY_DRIVER=database)

When new application code requires new columns/tables:

1. Build the **new** image first (do not switch traffic yet).
2. Run migrations with the **new** image CLI (not the old running container):

```bash
docker run --rm --env-file .env <new-image> /www/main artisan migrate
docker run --rm --env-file .env <new-image> /www/main artisan tenant:migrate-all
```

3. Confirm tenants are aligned (platform console Ops overview / schema status).
4. Then blue/green switch traffic to the new containers.

Notes:

- `migrate.sh` against the **currently running** container only has the **old** migrations if you have not switched yet.
- Platform UI migrate also uses the **running** binary only.
- Prefer migrate-before-cutover when code is not backward-compatible with old schema.
- Optional: put a tenant into **maintenance** (platform console) to return 503 on business traffic while ops run; platform `/api/platform` is unaffected.
- Ops failure alerts: set `TENANT_OPS_ALERT_WEBHOOK_URL` (falls back to `QUEUE_ALERT_WEBHOOK_URL` / `READY_ALERT_WEBHOOK_URL`). Queue backlog: `queue:alert-backlog`.

## 数据库迁移和填充

### 自动迁移（容器启动时执行）

**重要：** 迁移在容器启动时自动执行，确保在应用启动前完成：
- ✅ **启动前执行** - 容器启动脚本会在应用启动前执行迁移，避免字段不存在错误
- ✅ **自动执行** - 无需手动操作，每次容器启动都会执行
- ✅ **幂等操作** - Goravel 的 `migrate` 命令是安全的，可以重复执行
- ✅ **失败退出** - 如果迁移失败，容器会退出，部署脚本会自动检测并回滚
- ⚙️ **可配置** - 通过环境变量 `SKIP_MIGRATE=true` 可以跳过迁移

### 数据填充

部署脚本会提示是否执行数据填充：
- ⚠️ **需要确认** - 会提示用户是否执行
- ⚠️ **可能重复** - 填充可能会重复插入数据，请谨慎使用
- 💡 **建议** - 通常只在首次部署或需要更新基础数据时执行

### 通过环境变量控制

可以通过环境变量跳过填充提示：

```bash
# 自动执行填充（不提示）
export RUN_SEED=true
./scripts/deploy/docker-blue-green.sh

# 跳过填充（不提示）
export RUN_SEED=false
./scripts/deploy/docker-blue-green.sh
```

### 单独执行迁移或填充

如果需要单独执行迁移或填充：

```bash
# 执行数据库迁移
chmod +x scripts/deploy/migrate.sh
./scripts/deploy/migrate.sh                    # 自动检测容器
./scripts/deploy/migrate.sh goravel-admin-blue # 指定容器

# 执行数据填充
chmod +x scripts/deploy/seed.sh
./scripts/deploy/seed.sh                      # 自动检测容器
./scripts/deploy/seed.sh goravel-admin-blue    # 指定容器
```

## 前置要求

- Docker 和 Docker Compose 已安装
- 服务器上已配置 `.env` 文件
- 如果需要 Nginx 切换，需要配置 Nginx

## 健康检查

应用需要提供 `/health` 端点，已在 `routes/web.go` 中配置。

## 故障处理

如果部署失败，脚本会自动回滚：
- 停止新版本容器
- 保持旧版本运行

## 回滚操作

### 快速回滚

如果刚部署的版本有问题，可以快速回滚到上一个版本：

```bash
cd /www/goravel-admin
chmod +x scripts/deploy/rollback.sh
./scripts/deploy/rollback.sh
```

**特点：**
- 自动检测当前运行的版本（blue/green）
- 切换到另一个环境
- 如果旧容器已删除，可以选择从 Git 回滚或重新构建
- 自动切换 Nginx 流量（如果配置了）

### Git 版本回滚

回滚到特定的 Git 提交、标签或分支：

```bash
cd /www/goravel-admin
chmod +x scripts/deploy/rollback-git.sh

# 查看最近提交（不带参数）
./scripts/deploy/rollback-git.sh

# 回滚到上一个提交
./scripts/deploy/rollback-git.sh HEAD~1

# 回滚到指定提交
./scripts/deploy/rollback-git.sh abc1234

# 回滚到指定标签
./scripts/deploy/rollback-git.sh v1.0.0

# 回滚到远程分支
./scripts/deploy/rollback-git.sh origin/main
```

**特点：**
- 支持提交哈希、标签、分支
- 自动执行部署流程
- 需要确认操作

## 查看日志

```bash
# 查看运行中的容器日志
docker logs -f goravel-admin-blue
docker logs -f goravel-admin-green

# 查看容器状态
docker ps | grep goravel-admin

# 查看容器健康状态
docker inspect --format='{{.State.Health.Status}}' goravel-admin-blue
docker inspect --format='{{.State.Health.Status}}' goravel-admin-green
```

