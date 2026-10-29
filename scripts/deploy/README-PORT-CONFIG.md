# 端口配置说明

## 📍 统一配置端口

所有端口配置现在统一在 **`scripts/deploy/deploy-config.sh`** 文件中管理。

## 🔧 修改端口

只需修改 `scripts/deploy/deploy-config.sh` 文件中的端口配置：

```bash
# 编辑配置文件
vim scripts/deploy/deploy-config.sh

# 修改这些变量：
export BLUE_PORT="${BLUE_PORT:-3000}"      # 蓝环境端口（宿主机端口）
export GREEN_PORT="${GREEN_PORT:-3001}"    # 绿环境端口（宿主机端口）
export CONTAINER_PORT="${CONTAINER_PORT:-3000}"  # 容器内部端口（应用监听端口）
```

## 📝 配置说明

- **BLUE_PORT**: 蓝环境在宿主机上映射的端口（默认 3000）
- **GREEN_PORT**: 绿环境在宿主机上映射的端口（默认 3001）
- **CONTAINER_PORT**: 容器内部应用监听的端口（默认 3000，通常不需要修改）
- **APP_BIND**: 宿主机端口绑定地址（默认 `127.0.0.1`，只有本机 nginx 能访问，外部无法绕过 nginx 直连）。nginx 在另一台机器时设为本机内网 IP；不要设为 `0.0.0.0`

## 🔄 使用环境变量覆盖

也可以通过环境变量临时覆盖配置（无需修改文件）：

```bash
# 临时使用其他端口
export BLUE_PORT=4000
export GREEN_PORT=4001
./scripts/deploy/docker-blue-green.sh
```

## 📋 自动应用范围

修改配置后，以下文件会自动使用新端口：

- ✅ `docker-compose.blue.yml` - 蓝环境配置
- ✅ `docker-compose.green.yml` - 绿环境配置
- ✅ `scripts/deploy/docker-blue-green.sh` - 部署脚本
- ✅ `scripts/deploy/rollback.sh` - 回滚脚本
- ✅ Nginx 配置更新（如果配置了）

## ⚠️ 注意事项

1. **容器内部端口** (`CONTAINER_PORT`) 通常不需要修改，除非应用配置了其他端口
2. **宿主机端口** (`BLUE_PORT`, `GREEN_PORT`) 需要确保端口未被占用
3. 修改端口后，如果使用 Nginx，需要更新 Nginx 配置中的 upstream
4. 修改端口后需要重新构建镜像和启动容器

## 💡 示例

### 修改为 4000 和 4001

```bash
# 编辑配置文件
vim scripts/deploy/deploy-config.sh

# 修改为：
export BLUE_PORT="${BLUE_PORT:-4000}"
export GREEN_PORT="${GREEN_PORT:-4001}"
```

### 检查端口是否被占用

```bash
# 检查端口是否被占用
netstat -tlnp | grep 4000
netstat -tlnp | grep 4001

# 或使用 ss 命令
ss -tlnp | grep 4000
ss -tlnp | grep 4001
```

