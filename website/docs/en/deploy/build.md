# Build & deploy

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/deploy/build).

---

## 编译
```bash
# 检查编译是否通过
go build -o NUL .
```

### 本地编译（当前平台）

```bash
# 生产环境推荐使用静态编译
go build --ldflags "-extldflags -static -s -w" -o main .
```

### Linux 服务器交叉编译（Windows/Mac 编译 Linux 版本）

```bash
# Windows PowerShell:
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"; go build --ldflags "-extldflags -static -s -w" -o main .

# Windows CMD (需要分开执行):
set GOOS=linux
set GOARCH=amd64
go build --ldflags "-extldflags -static -s -w" -o main .
SET GOOS=windows
SET GOARCH=amd64

# Linux/Mac:
GOOS=linux GOARCH=amd64 go build --ldflags "-extldflags -static -s -w" -o main .
```

**重要提示：**
- 如果在 Windows 上编译，但要在 Linux 服务器上运行，必须使用交叉编译
- 使用 `GOOS=linux GOARCH=amd64` 指定目标平台
- 如果服务器是 ARM 架构，使用 `GOARCH=arm64`
- **推荐使用静态编译**：添加 `--ldflags "-extldflags -static -s -w"` 参数，生成独立可执行文件

---

## 方案一：单服务部署（端口 3000）

### 步骤 1：准备部署文件

在服务器上创建应用目录并上传必要文件：

```bash
# 在服务器上创建应用目录
sudo mkdir -p /www/goravel-admin
sudo chown -R www-data:www-data /www/goravel-admin

# 上传二进制文件
scp main user@server:/www/goravel-admin/

# 上传配置文件和其他必要文件
scp .env user@server:/www/goravel-admin/.env
scp -r storage/ user@server:/www/goravel-admin/
scp -r resources/ user@server:/www/goravel-admin/
scp -r public/ user@server:/www/goravel-admin/
```

### 步骤 2：配置 systemd 服务

```bash
# 上传服务文件
scp scripts/systemd/goravel-admin-3000.service user@server:/tmp/

# 在服务器上安装服务文件
sudo cp /tmp/goravel-admin-3000.service /etc/systemd/system/goravel-admin.service
sudo systemctl daemon-reload
```

编辑服务文件，根据实际情况修改路径和用户：

```bash
sudo nano /etc/systemd/system/goravel-admin.service
```

主要配置项：
- `User` 和 `Group`：运行服务的用户和组（如 `www-data`）
- `WorkingDirectory`：应用工作目录（如 `/www/goravel-admin`）
- `ExecStart`：二进制文件路径（如 `/www/goravel-admin/main`）
- `ReadWritePaths`：需要写入权限的目录（如 `/www/goravel-admin/storage`）


### 步骤 3：设置文件权限

```bash
# 设置二进制文件权限
sudo chmod +x /www/goravel-admin/main

# 设置存储目录权限
sudo chmod -R 775 /www/goravel-admin/storage
sudo chown -R www-data:www-data /www/goravel-admin/storage

# 设置配置文件权限（保护敏感信息）
sudo chmod 600 /www/goravel-admin/.env
sudo chown www-data:www-data /www/goravel-admin/.env
```

### 步骤 4：初始化应用

```bash
# 生成应用密钥
cd /www/goravel-admin
./main artisan key:generate

# 数据库迁移
./main artisan migrate

# 数据库填充（可选，仅首次初始化建议完整 seed）
./main artisan db:seed --force

# 增量部署：仅同步代码生成器模块的菜单与权限（幂等）
# ./main artisan db:seed --force --seeder=GeneratedModulesSeeder
```

详见 [代码生成器](/guide/code-generator)。

### 步骤 5：启动和管理服务

```bash
# 启动服务
sudo systemctl start goravel-admin

# 设置开机自启
sudo systemctl enable goravel-admin

# 查看服务状态
sudo systemctl status goravel-admin

# 停止服务
sudo systemctl stop goravel-admin

# 重启服务（⚠️ 会有短暂中断 1-3 秒）
sudo systemctl restart goravel-admin
```

### 步骤 6：查看日志

```bash
# 实时查看日志
sudo journalctl -u goravel-admin -f

# 查看最近 100 行日志
sudo journalctl -u goravel-admin -n 100

# 查看今天的日志
sudo journalctl -u goravel-admin --since today
```

### 步骤 7：验证部署

```bash
# 检查服务是否运行
sudo systemctl is-active goravel-admin

# 检查端口是否监听
sudo netstat -tlnp | grep 3000
# 或使用 ss 命令
sudo ss -tlnp | grep 3000

```

### 配置外部访问

如果无法从外部 IP 访问，需要修改 `.env` 文件：

```env
# 修改为 0.0.0.0 允许所有网络接口访问
APP_HOST=0.0.0.0
APP_PORT=3000
```

然后重启服务：
```bash
sudo systemctl restart goravel-admin
```

**检查防火墙设置：**

```bash
# CentOS/RHEL 系统
sudo firewall-cmd --list-ports
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --reload

# Ubuntu/Debian 系统
sudo ufw status
sudo ufw allow 3000/tcp
sudo ufw reload
```

---

## 方案二：双目录双端口零停机部署（蓝绿部署 + Nginx 切换）⭐ 推荐

这是生产环境最可靠的零停机部署方案，通过双目录双端口 + Nginx 切换实现真正的零停机。

### 工作原理

- 使用两个版本目录：`v1`（端口 3000）和 `v2`（端口 3001）
- 两个实例可以同时运行
- Nginx 负载均衡在两个端口之间切换
- 部署新版本时：
  1. 部署新版本到备用目录（如 `v2`，端口 3001）
  2. 启动新版本服务
  3. 健康检查通过后，Nginx 切换流量到新端口
  4. 停止旧版本服务
  5. 实现真正的零停机，且可以快速回滚

### 目录结构

```
/www/goravel-admin/
├── v1/                    # 版本 1 目录（端口 3000）
│   ├── main              # 可执行文件
│   ├── .env              # 配置文件（APP_PORT=3000）
│   ├── storage/          # 存储目录
│   └── ...
└── v2/                    # 版本 2 目录（端口 3001）
    ├── main              # 可执行文件
    ├── .env              # 配置文件（APP_PORT=3001）
    ├── storage/          # 存储目录
    └── ...
```


### 步骤 1：准备部署文件

在服务器上创建应用目录结构：

```bash
# 在服务器上创建应用目录
sudo mkdir -p /www/goravel-admin/{v1,v2}
sudo chown -R www-data:www-data /www/goravel-admin

# 上传第一个版本到 v1 目录
scp main user@server:/www/goravel-admin/v1/
scp .env user@server:/www/goravel-admin/v1/.env
scp -r storage/ user@server:/www/goravel-admin/v1/
scp -r resources/ user@server:/www/goravel-admin/v1/
scp -r public/ user@server:/www/goravel-admin/v1/
```

**重要提示：.env 文件配置**
- 每个版本目录（v1、v2）都有独立的 `.env` 文件
- **v1 目录的 `.env` 文件中配置 `APP_PORT=3000`**
- **v2 目录的 `.env` 文件中配置 `APP_PORT=3001`**
- 其他配置（如数据库连接）可以不同版本不同配置

### 步骤 2：配置双实例 systemd 服务

项目已提供两个服务文件：
- `scripts/systemd/goravel-admin-v1.service` - v1 目录，端口 3000
- `scripts/systemd/goravel-admin-v2.service` - v2 目录，端口 3001

**服务文件特点：**
- `WorkingDirectory=/www/goravel-admin/v1` 或 `/www/goravel-admin/v2`
- `ExecStart=/www/goravel-admin/v1/main` 或 `/www/goravel-admin/v2/main`
- 端口由各自目录的 `.env` 文件中的 `APP_PORT` 配置管理

在服务器上安装：

```bash
# 上传服务文件
scp scripts/systemd/goravel-admin-v1.service user@server:/tmp/
scp scripts/systemd/goravel-admin-v2.service user@server:/tmp/

# 在服务器上安装
sudo cp /tmp/goravel-admin-v1.service /etc/systemd/system/
sudo cp /tmp/goravel-admin-v2.service /etc/systemd/system/
sudo systemctl daemon-reload
```

编辑服务文件，根据实际情况修改路径和用户：

```bash
sudo nano /etc/systemd/system/goravel-admin-v1.service
sudo nano /etc/systemd/system/goravel-admin-v2.service
```

主要配置项：
- `User` 和 `Group`：运行服务的用户和组（如 `www-data`）
- `WorkingDirectory`：应用工作目录（如 `/www/goravel-admin/v1` 或 `/www/goravel-admin/v2`）
- `ExecStart`：二进制文件路径（如 `/www/goravel-admin/v1/main` 或 `/www/goravel-admin/v2/main`）
- `ReadWritePaths`：需要写入权限的目录（如 `/www/goravel-admin/v1/storage` 或 `/www/goravel-admin/v2/storage`）

### 步骤 3：设置文件权限

```bash
# 设置 v1 目录权限
sudo chmod +x /www/goravel-admin/v1/main
sudo chmod -R 775 /www/goravel-admin/v1/storage
sudo chown -R www-data:www-data /www/goravel-admin/v1/storage
sudo chmod 600 /www/goravel-admin/v1/.env
sudo chown www-data:www-data /www/goravel-admin/v1/.env

# 设置 v2 目录权限（如果已上传）
sudo chmod +x /www/goravel-admin/v2/main 2>/dev/null || true
sudo chmod -R 775 /www/goravel-admin/v2/storage 2>/dev/null || true
sudo chown -R www-data:www-data /www/goravel-admin/v2/storage 2>/dev/null || true
sudo chmod 600 /www/goravel-admin/v2/.env 2>/dev/null || true
sudo chown www-data:www-data /www/goravel-admin/v2/.env 2>/dev/null || true
```

### 步骤 4：初始化应用

```bash
# 生成应用密钥（在 v1 目录）
cd /www/goravel-admin/v1
./main artisan key:generate

# 数据库迁移
./main artisan migrate

# 数据库填充（可选，仅首次初始化建议完整 seed）
./main artisan db:seed --force

# 增量部署：仅同步代码生成器模块的菜单与权限（幂等）
# ./main artisan db:seed --force --seeder=GeneratedModulesSeeder
```

详见 [代码生成器](/guide/code-generator)。

```bash
# 试运行
./main

```
### 步骤 5：配置 Nginx 负载均衡

项目已提供 Nginx 配置文件：`scripts/nginx/goravel-admin.conf`

在服务器上安装：

```bash
# 上传 Nginx 配置
scp scripts/nginx/goravel-admin.conf user@server:/tmp/

# 在服务器上安装
sudo cp /tmp/goravel-admin.conf /etc/nginx/sites-available/goravel-admin
sudo ln -s /etc/nginx/sites-available/goravel-admin /etc/nginx/sites-enabled/
sudo nginx -t
sudo nginx -s reload
```

**Nginx 配置说明：**
nginx.conf 参考
编辑 `/etc/nginx/sites-available/goravel-admin`，修改 `server_name` 为你的域名：



### 步骤 6：启动第一个实例

```bash
# 启动 v1 实例（端口 3000）
sudo systemctl start goravel-admin-v1
# 开机自启
sudo systemctl enable goravel-admin-v1

# 检查状态
sudo systemctl status goravel-admin-v1

```


### 查看日志

```bash
# 查看 v1 服务日志
sudo journalctl -u goravel-admin-v1 -f

# 查看 v2 服务日志
sudo journalctl -u goravel-admin-v2 -f

# 查看最近 100 行日志
sudo journalctl -u goravel-admin-v1 -n 100
sudo journalctl -u goravel-admin-v2 -n 100
```

### 验证部署

```bash
# 检查服务状态
sudo systemctl status goravel-admin-v1
sudo systemctl status goravel-admin-v2

# 检查端口监听
sudo ss -tlnp | grep 3000
sudo ss -tlnp | grep 3001

```


### （蓝绿部署 + Nginx 切换）

```bash
# 启动服务
sudo systemctl start goravel-admin-v1
sudo systemctl start goravel-admin-v2

# 停止服务
sudo systemctl stop goravel-admin-v1
sudo systemctl stop goravel-admin-v2

# 重启服务
sudo systemctl restart goravel-admin-v1
sudo systemctl restart goravel-admin-v2


# 查看状态
sudo systemctl status goravel-admin-v1
sudo systemctl status goravel-admin-v2

# 查看日志
sudo journalctl -u goravel-admin-v1 -f
sudo journalctl -u goravel-admin-v2 -f

# 设置开机自启
sudo systemctl enable goravel-admin-v1
sudo systemctl enable goravel-admin-v2

# 检查端口监听
sudo ss -tlnp | grep 3000
sudo ss -tlnp | grep 3001
```

### 清理所有服务

```bash

# 停止所有服务
sudo systemctl stop goravel-admin-v1 goravel-admin-v2 2>/dev/null

# 禁用所有服务
sudo systemctl disable goravel-admin-v1 goravel-admin-v2 2>/dev/null

# 删除服务文件
sudo rm -f /etc/systemd/system/goravel-admin*.service

# 重新加载 systemd
sudo systemctl daemon-reload

# 重置失败状态
sudo systemctl reset-failed goravel-admin* 2>/dev/null

# 如果还有进程在运行，强制停止
sudo pkill -f 'goravel-admin' || true
sudo pkill -f '/www/goravel-admin' || true
```

---

## 方案三：Docker Compose 蓝绿部署（零停机）⭐ 推荐容器化方案

这是使用 Docker 容器化的零停机部署方案，适合本地没有 Docker 环境，但服务器有 Docker 的场景。

### 工作原理

- 使用两个 Docker Compose 配置：`blue`（端口 3000）和 `green`（端口 3001）
- 两个容器可以同时运行
- 部署新版本时：
  1. 在备用环境（如 `green`）构建并启动新版本
  2. 健康检查通过后，切换流量到新版本
  3. 停止旧版本容器
  4. 实现真正的零停机，且可以快速回滚

### 前置要求

- 服务器已安装 Docker 和 Docker Compose
- 服务器可以访问 Git 仓库（或手动上传代码）
- 已配置 `.env` 文件

### 步骤 1：在服务器上初始化

```bash
# SSH 登录服务器
ssh user@your-server.com

# 创建部署目录
sudo mkdir -p /www/goravel-admin
sudo chown -R $USER:$USER /www/goravel-admin
cd /www/goravel-admin

# 克隆仓库（首次）
git clone https://github.com/your-username/goravel-admin.git .

# 或者如果已经克隆过
git pull origin main
```

### 步骤 2：配置环境变量

```bash
# 确保 .env 文件存在
cd /www/goravel-admin
cp .env.example .env  # 如果存在
# 编辑 .env 文件，设置数据库等配置
vim .env
```

### 步骤 3：执行部署

#### 方式一：从 Git 拉取并部署（推荐）

```bash
# 设置环境变量（可选）
export GIT_REPO_URL="https://github.com/your-username/goravel-admin.git"
export GIT_BRANCH="main"
export DEPLOY_DIR="/www/goravel-admin"

# 执行部署脚本
chmod +x scripts/deploy/git-deploy.sh
./scripts/deploy/git-deploy.sh
```

#### 方式二：手动部署（已拉取代码）

```bash
cd /www/goravel-admin
git pull origin main  # 拉取最新代码

# 执行部署脚本
chmod +x scripts/deploy/docker-blue-green.sh
./scripts/deploy/docker-blue-green.sh
```

### 部署流程说明

脚本会自动执行以下步骤：

1. **检测当前版本** - 自动检测运行的是 `blue` 还是 `green`
2. **构建新版本** - 在备用环境构建新 Docker 镜像
3. **启动新版本** - 启动新版本容器（使用不同端口）
4. **健康检查** - 等待新版本通过健康检查（最多 30 次，每次 2 秒）
5. **切换 Nginx 流量** - 如果存在 Nginx 配置，自动更新并重载
6. **停止旧版本** - 停止旧版本容器

### 查看部署状态

```bash
# 查看运行中的容器
docker ps | grep goravel-admin

# 查看容器日志
docker logs -f goravel-admin-blue
docker logs -f goravel-admin-green

# 查看容器健康状态
docker inspect --format='{{.State.Health.Status}}' goravel-admin-blue
docker inspect --format='{{.State.Health.Status}}' goravel-admin-green
```

### 回滚方案

如果需要回滚到上一个版本：

```bash
cd /www/goravel-admin

# 方式一：手动切换
# 如果当前运行的是 green，切换到 blue
docker-compose -f docker-compose.blue.yml up -d
# 然后停止 green
docker-compose -f docker-compose.green.yml down

# 方式二：使用 Git 回滚代码后重新部署
git checkout <previous-commit>
./scripts/deploy/docker-blue-green.sh
```

### 配置文件说明

- `docker-compose.blue.yml` - 蓝环境配置（端口 3000）
- `docker-compose.green.yml` - 绿环境配置（端口 3001）
- `scripts/deploy/docker-blue-green.sh` - 蓝绿部署主脚本
- `scripts/deploy/git-deploy.sh` - 从 Git 拉取并部署的脚本

### 健康检查

# 应用已配置 `/health`（存活）与 `/ready`（就绪，含 DB/Redis 探测），见 [生产清单](/en/deploy/production)。

### 故障处理

如果部署过程中健康检查失败，脚本会自动：
- 停止新版本容器
- 保持旧版本继续运行
- 退出并报告错误

### 本地开发流程

1. **本地开发** - 在本地修改代码（无需 Docker）
2. **提交代码** - `git add . && git commit -m "更新" && git push`
3. **服务器部署** - SSH 到服务器执行 `./scripts/deploy/git-deploy.sh`

### 优势

- ✅ **零停机部署** - 新版本就绪后再切换流量
- ✅ **快速回滚** - 可以快速切换回旧版本
- ✅ **本地无需 Docker** - 本地开发环境简单
- ✅ **自动化** - 一键部署脚本
- ✅ **健康检查** - 自动验证新版本是否正常

---

## 部署方案对比

| 方案 | 复杂度 | 零停机 | 适用场景 | 推荐度 |
|------|--------|--------|----------|--------|
| 单服务部署 | ⭐ | ❌ | 开发/测试环境 | ⭐⭐ |
| 蓝绿部署 + Nginx | ⭐⭐⭐ | ✅ | 生产环境（非容器） | ⭐⭐⭐⭐ |
| Docker Compose 蓝绿 | ⭐⭐ | ✅ | 生产环境（容器化） | ⭐⭐⭐⭐⭐ |

---

## Docs site on Cloudflare

VitePress lives under `website/`. Use a **separate subdomain** (e.g. `docs.example.com`), not the admin SPA Worker.

### Workers + Assets

```bash
cd website
npm install
npm run build
npx wrangler deploy
# or npm run deploy
```

| Item | Notes |
|------|--------|
| Config | `website/wrangler.toml`, `website/worker.js` |
| Output | `docs/.vitepress/dist` |
| Worker name | `goravel-admin-docs` (do not reuse the `admin` Worker) |
| Domain | Bind a `docs.` hostname; no `VITE_API_*` needed |

`worker.js` maps clean paths like `/guide/getting-started` to `.html` so refresh works.

### Pages (optional)

Root=`website`, Build=`npm run build`, Output=`docs/.vitepress/dist`, Node 20.

See [website/README.md](https://github.com/wangxuancheng-dev/goravel-admin/blob/main/website/README.md).
