#!/bin/bash
# Docker Compose 蓝绿部署脚本
# 在服务器上执行此脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../../"  # 回到项目根目录

# 加载部署配置
source "$SCRIPT_DIR/deploy-config.sh"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检测当前运行的版本
detect_current_version() {
    if docker ps --format "{{.Names}}" | grep -q "goravel-admin-blue"; then
        if [ "$(docker inspect -f '{{.State.Running}}' goravel-admin-blue 2>/dev/null)" = "true" ]; then
            echo "blue"
            return
        fi
    fi
    if docker ps --format "{{.Names}}" | grep -q "goravel-admin-green"; then
        if [ "$(docker inspect -f '{{.State.Running}}' goravel-admin-green 2>/dev/null)" = "true" ]; then
            echo "green"
            return
        fi
    fi
    echo "blue"  # 默认从 blue 开始
}

CURRENT_COLOR=$(detect_current_version)
NEXT_COLOR=$([ "$CURRENT_COLOR" = "blue" ] && echo "green" || echo "blue")
CURRENT_PORT=$([ "$CURRENT_COLOR" = "blue" ] && echo "$BLUE_PORT" || echo "$GREEN_PORT")
NEXT_PORT=$([ "$NEXT_COLOR" = "blue" ] && echo "$BLUE_PORT" || echo "$GREEN_PORT")

echo -e "${GREEN}=== Docker Compose 蓝绿部署 ===${NC}"
echo -e "当前运行: ${YELLOW}$CURRENT_COLOR${NC} (端口: $CURRENT_PORT)"
echo -e "部署到: ${YELLOW}$NEXT_COLOR${NC} (端口: $NEXT_PORT)"
echo ""

# 1. 检查 Docker 和 Docker Compose
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误: Docker 未安装${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}错误: Docker Compose 未安装${NC}"
    exit 1
fi

# 使用 docker compose 或 docker-compose
COMPOSE_CMD="docker-compose"
if ! command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker compose"
fi

# 导出端口配置到环境变量（供 docker-compose 使用）
export BLUE_PORT GREEN_PORT CONTAINER_PORT

# 2. 构建新版本镜像
echo -e "${GREEN}[1/7] 构建新版本镜像...${NC}"
echo "注意: 镜像包含启动脚本，会在应用启动前自动执行数据库迁移"
$COMPOSE_CMD -f docker-compose.${NEXT_COLOR}.yml build --no-cache

if [ $? -ne 0 ]; then
    echo -e "${RED}构建失败！${NC}"
    exit 1
fi

# 3. 启动新版本容器
echo -e "${GREEN}[2/7] 启动新版本容器 (端口 $NEXT_PORT)...${NC}"
$COMPOSE_CMD -f docker-compose.${NEXT_COLOR}.yml up -d

if [ $? -ne 0 ]; then
    echo -e "${RED}启动失败！${NC}"
    exit 1
fi

# 4. 等待容器启动（启动脚本会自动执行迁移）
echo -e "${GREEN}[3/7] 等待容器启动并执行迁移...${NC}"
echo "注意: 容器启动脚本会在应用启动前自动执行数据库迁移"

# 等待容器启动并完成迁移
# 启动脚本会在应用启动前执行迁移，所以我们需要等待应用完全启动
for i in {1..30}; do
    # 检查容器是否运行
    if ! docker ps --format "{{.Names}}" | grep -q "goravel-admin-${NEXT_COLOR}"; then
        echo "容器未运行，等待中... ($i/30)"
        sleep 2
        continue
    fi
    
    # 检查应用是否已启动（迁移完成后应用才会启动）
    if docker exec goravel-admin-${NEXT_COLOR} /www/main artisan --version > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 容器已启动，迁移已完成，应用已就绪${NC}"
        break
    fi
    
    if [ $i -eq 30 ]; then
        echo -e "${RED}✗ 容器启动超时，可能迁移失败${NC}"
        echo "查看容器日志:"
        docker logs --tail 50 goravel-admin-${NEXT_COLOR}
        $COMPOSE_CMD -f docker-compose.${NEXT_COLOR}.yml down
        exit 1
    fi
    
    echo "等待容器启动和迁移完成... ($i/30)"
    sleep 2
done

# 验证迁移是否成功（可选，作为双重检查）
echo "验证数据库迁移状态..."
if docker exec goravel-admin-${NEXT_COLOR} /www/main artisan migrate:status > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 数据库迁移验证通过${NC}"
else
    echo -e "${YELLOW}警告: 无法验证迁移状态，但应用已启动${NC}"
fi

# 6. 询问是否执行数据填充（可选）
echo -e "${GREEN}[5/7] 数据填充（可选）...${NC}"
if [ -z "$RUN_SEED" ]; then
    read -p "是否执行数据填充？这可能会重复插入数据 (y/N): " seed_confirm
    if [ "$seed_confirm" = "y" ] || [ "$seed_confirm" = "Y" ]; then
        RUN_SEED="true"
    else
        RUN_SEED="false"
    fi
fi

if [ "$RUN_SEED" = "true" ]; then
    echo "执行数据填充..."
    # APP_ENV=production requires --force (Goravel db:seed ConfirmToProceed).
    if docker exec goravel-admin-${NEXT_COLOR} /www/main artisan db:seed --force; then
        echo -e "${GREEN}✓ 数据填充成功${NC}"
    else
        echo -e "${YELLOW}警告: 数据填充失败，但继续部署${NC}"
    fi
else
    echo -e "${YELLOW}跳过数据填充${NC}"
fi

# 7. 健康检查
echo -e "${GREEN}[6/7] 执行健康检查...${NC}"
HEALTH_CHECK_FAILED=true
for i in {1..30}; do
    # 检查容器是否运行
    if ! docker ps --format "{{.Names}}" | grep -q "goravel-admin-${NEXT_COLOR}"; then
        echo "容器未运行，等待中... ($i/30)"
        sleep 2
        continue
    fi
    
    # 检查健康状态
    HEALTH=$(docker inspect --format='{{.State.Health.Status}}' goravel-admin-${NEXT_COLOR} 2>/dev/null || echo "none")
    
    # 尝试 HTTP 健康检查
    if curl -f -s http://localhost:$NEXT_PORT/health > /dev/null 2>&1 || \
       curl -f -s http://localhost:$NEXT_PORT/ > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 新版本健康检查通过 (尝试 $i/30)${NC}"
        HEALTH_CHECK_FAILED=false
        break
    fi
    
    if [ "$HEALTH" = "healthy" ]; then
        echo -e "${GREEN}✓ 新版本健康检查通过 (Docker 健康检查)${NC}"
        HEALTH_CHECK_FAILED=false
        break
    fi
    
    echo "等待健康检查... ($i/30)"
    sleep 2
done

if [ "$HEALTH_CHECK_FAILED" = "true" ]; then
    echo -e "${RED}✗ 健康检查失败，停止新版本并回滚${NC}"
    $COMPOSE_CMD -f docker-compose.${NEXT_COLOR}.yml down
    exit 1
fi

# 8. 切换 Nginx 流量（如果有 Nginx）
echo -e "${GREEN}[7/7] 切换 Nginx 流量...${NC}"
NGINX_CONF="/etc/nginx/sites-available/goravel-admin"
if [ -f "$NGINX_CONF" ]; then
    # 备份配置
    BACKUP_FILE="${NGINX_CONF}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$NGINX_CONF" "$BACKUP_FILE"
    echo "已备份 Nginx 配置到: $BACKUP_FILE"
    
    # 更新 upstream 配置
    sed -i "s/server 127.0.0.1:$BLUE_PORT/server 127.0.0.1:$NEXT_PORT/" "$NGINX_CONF"
    sed -i "s/server 127.0.0.1:$GREEN_PORT/server 127.0.0.1:$NEXT_PORT/" "$NGINX_CONF"
    
    # 测试并重载 Nginx
    if nginx -t 2>/dev/null; then
        nginx -s reload
        echo -e "${GREEN}✓ Nginx 流量已切换到端口 $NEXT_PORT${NC}"
    else
        echo -e "${YELLOW}警告: Nginx 配置测试失败，恢复备份${NC}"
        mv "$BACKUP_FILE" "$NGINX_CONF"
    fi
else
    echo -e "${YELLOW}提示: Nginx 配置文件不存在，跳过流量切换${NC}"
    echo -e "${YELLOW}请手动配置负载均衡指向端口 $NEXT_PORT${NC}"
fi

# 9. 等待流量切换完成
echo -e "${GREEN}等待流量切换完成...${NC}"
sleep 5

# 10. 停止旧版本服务
echo -e "${GREEN}停止旧版本服务 ($CURRENT_COLOR)...${NC}"
$COMPOSE_CMD -f docker-compose.${CURRENT_COLOR}.yml down

echo ""
echo -e "${GREEN}=== 部署完成 ===${NC}"
echo -e "当前运行版本: ${YELLOW}$NEXT_COLOR${NC} (端口: $NEXT_PORT)"
echo -e "旧版本: ${YELLOW}$CURRENT_COLOR${NC} (已停止)"
echo ""
echo "查看日志: docker logs -f goravel-admin-${NEXT_COLOR}"
echo "查看状态: docker ps | grep goravel-admin"

