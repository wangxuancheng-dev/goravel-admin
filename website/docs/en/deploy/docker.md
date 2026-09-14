# Docker production deploy

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/deploy/docker).

---

## 📋 概述

本项目支持 Docker Compose 蓝绿部署，实现零停机更新。适合本地没有 Docker 环境，但服务器有 Docker 的场景。

## 🚀 快速开始

### 1. 本地开发（无需 Docker）

```bash
# 正常开发，提交代码
git add .
git commit -m "更新功能"
git push origin main
```

### 2. 服务器部署

#### 首次部署

```bash
# SSH 登录服务器
ssh user@your-server.com

# 创建部署目录
mkdir -p /www/goravel-admin
cd /www/goravel-admin

# 克隆仓库
git clone https://github.com/your-username/goravel-admin.git .

# 配置环境变量
cp .env.example .env  # 如果存在
vim .env  # 编辑配置

# 执行部署
chmod +x scripts/deploy/git-deploy.sh
./scripts/deploy/git-deploy.sh
```

#### 后续部署

```bash
# SSH 登录服务器
ssh user@your-server.com

# 进入部署目录
cd /www/goravel-admin

# 方式一：使用 Git 部署脚本（推荐）
./scripts/deploy/git-deploy.sh

# 方式二：手动拉取代码后部署
git pull origin main
./scripts/deploy/docker-blue-green.sh
```

## 📁 文件结构

```
项目根目录/
├── docker-compose.blue.yml      # 蓝环境配置（端口 3000）
├── docker-compose.green.yml     # 绿环境配置（端口 3001）
├── Dockerfile                    # Docker 镜像构建文件
└── scripts/
    └── deploy/
        ├── docker-blue-green.sh  # 蓝绿部署主脚本
        ├── git-deploy.sh         # Git 拉取并部署脚本
        ├── rollback.sh           # 快速回滚脚本
        ├── rollback-git.sh       # Git 版本回滚脚本
        └── README.md             # 详细说明文档
```

## 🔧 配置说明

### 端口配置

**重要：** 所有端口配置统一在 `scripts/deploy/deploy-config.sh` 文件中管理。

只需修改一个文件即可更改所有端口：

```bash
# 编辑配置文件
vim scripts/deploy/deploy-config.sh

# 修改端口配置：
export BLUE_PORT="${BLUE_PORT:-3000}"      # 蓝环境端口（默认 3000）
export GREEN_PORT="${GREEN_PORT:-3001}"    # 绿环境端口（默认 3001）
export CONTAINER_PORT="${CONTAINER_PORT:-3000}"  # 容器内部端口（默认 3000）
```

或通过环境变量临时覆盖：

```bash
export BLUE_PORT=4000
export GREEN_PORT=4001
./scripts/deploy/docker-blue-green.sh
```

详细说明请参考：`scripts/deploy/README-PORT-CONFIG.md`

### 环境变量

在服务器上创建 `.env` 文件，配置数据库、Redis 等：

```env
APP_ENV=production
APP_HOST=0.0.0.0
APP_PORT=3000
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=your_database
DB_USERNAME=your_username
DB_PASSWORD=your_password
# ... 其他配置
```

### Git 仓库配置

如果使用 `git-deploy.sh`，可以设置环境变量：

```bash
export GIT_REPO_URL="https://github.com/your-username/goravel-admin.git"
export GIT_BRANCH="main"
export DEPLOY_DIR="/www/goravel-admin"
```

或直接编辑 `scripts/deploy/git-deploy.sh` 文件中的配置。

## 📊 部署流程

1. **检测当前版本** - 自动检测运行的是 `blue` 还是 `green`
2. **构建新版本** - 在备用环境构建新 Docker 镜像（包含启动脚本）
3. **启动新版本** - 启动新版本容器（使用不同端口）
4. **自动执行迁移** - 容器启动脚本会在应用启动前自动执行数据库迁移
5. **数据填充（可选）** - 提示是否执行数据填充
6. **健康检查** - 等待新版本通过健康检查
7. **切换流量** - 更新 Nginx 配置（如果存在）
8. **停止旧版本** - 停止旧版本容器

## 🗄️ 数据库迁移和填充

### 自动迁移（容器启动时执行）

**重要：** 迁移在容器启动时自动执行，确保在应用启动前完成：
- ✅ **启动前执行** - 容器启动脚本会在应用启动前执行迁移，避免字段不存在错误
- ✅ **自动执行** - 无需手动操作，每次容器启动都会执行
- ✅ **幂等操作** - Goravel 的 `migrate` 命令是安全的，可以重复执行
- ✅ **失败退出** - 如果迁移失败，容器会退出，部署脚本会自动检测并回滚
- ⚙️ **可配置** - 通过环境变量 `SKIP_MIGRATE=true` 可以跳过迁移

### 数据填充

- ⚠️ **需要确认** - 部署时会提示是否执行数据填充
- ⚠️ **可能重复** - 填充可能会重复插入数据，请谨慎使用
- 💡 **建议** - 通常只在首次部署或需要更新基础数据时执行

### 通过环境变量控制

```bash
# 自动执行填充（不提示）
export RUN_SEED=true
./scripts/deploy/docker-blue-green.sh

# 跳过填充（不提示）
export RUN_SEED=false
./scripts/deploy/docker-blue-green.sh
```

### 单独执行

```bash
# 执行数据库迁移
./scripts/deploy/migrate.sh

# 执行数据填充（默认完整 db:seed，仅适合首次初始化）
./scripts/deploy/seed.sh

# 仅同步代码生成器模块菜单/权限（生产增量部署推荐）
# docker exec <container> /www/main artisan db:seed --force --seeder=GeneratedModulesSeeder
```

详见 [代码生成器](/guide/code-generator)。

## 🔍 查看状态

```bash
# 查看运行中的容器
docker ps | grep goravel-admin

# 查看容器日志
docker logs -f goravel-admin-blue
docker logs -f goravel-admin-green

# 查看健康状态
docker inspect --format='{{.State.Health.Status}}' goravel-admin-blue
```

## 🔄 回滚

> **Static SPA only:** use versioned dirs + `current` symlink (see [Production](/en/deploy/production) Admin SPA). This section is **API blue/green**. Percentage canary belongs to Nginx/LB/Cloudflare, not `rollback.sh`.

### 快速回滚（推荐）

如果刚部署的版本有问题，快速回滚到上一个版本：

```bash
cd /www/goravel-admin
chmod +x scripts/deploy/rollback.sh
./scripts/deploy/rollback.sh
```

**特点：**
- ✅ 自动检测当前版本
- ✅ 零停机切换
- ✅ 自动健康检查
- ✅ 自动切换 Nginx 流量

### Git 版本回滚

回滚到特定的 Git 提交、标签或分支：

```bash
cd /www/goravel-admin
chmod +x scripts/deploy/rollback-git.sh

# 查看最近提交
./scripts/deploy/rollback-git.sh

# 回滚到上一个提交
./scripts/deploy/rollback-git.sh HEAD~1

# 回滚到指定提交
./scripts/deploy/rollback-git.sh abc1234

# 回滚到指定标签
./scripts/deploy/rollback-git.sh v1.0.0
```

### 手动回滚

如果需要手动控制：

```bash
cd /www/goravel-admin

# 方式一：手动切换容器
docker-compose -f docker-compose.blue.yml up -d  # 启动 blue
docker-compose -f docker-compose.green.yml down  # 停止 green

# 方式二：回滚代码后重新部署
git checkout <previous-commit>
./scripts/deploy/docker-blue-green.sh
```

## ⚠️ 故障处理

如果部署失败：
- 脚本会自动停止新版本容器
- 旧版本继续运行
- 检查日志：`docker logs goravel-admin-<color>`

## 📝 注意事项

1. **首次部署**：确保服务器已安装 Docker 和 Docker Compose
2. **环境变量**：确保 `.env` 文件配置正确
3. **健康检查**：应用需要提供 `/health` 端点（已配置）
4. **端口冲突**：确保 3000 和 3001 端口未被占用
5. **存储持久化**：`storage` 目录会持久化到宿主机

## 🔗 相关文档

- 详细部署文档：[编译与部署](/en/deploy/build)
- 脚本说明：`scripts/deploy/README.md`

## 💡 提示

- 本地开发无需 Docker，只需 Go 环境
- 部署在服务器上自动完成，无需手动操作
- 支持零停机部署，用户体验无影响
- 支持快速回滚，降低部署风险

