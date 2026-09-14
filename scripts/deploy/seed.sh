#!/bin/bash
# 数据填充脚本
# 在运行中的容器中执行数据填充

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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
    echo "none"
}

CONTAINER_NAME=$1

# 如果没有指定容器名，自动检测
if [ -z "$CONTAINER_NAME" ]; then
    CURRENT_COLOR=$(detect_current_version)
    if [ "$CURRENT_COLOR" = "none" ]; then
        echo -e "${RED}错误: 未找到运行中的容器${NC}"
        echo "用法: $0 <container-name>"
        echo "示例: $0 goravel-admin-blue"
        exit 1
    fi
    CONTAINER_NAME="goravel-admin-${CURRENT_COLOR}"
    echo -e "${YELLOW}自动检测到运行中的容器: $CONTAINER_NAME${NC}"
fi

# 检查容器是否存在
if ! docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${RED}错误: 容器 $CONTAINER_NAME 不存在或未运行${NC}"
    exit 1
fi

echo -e "${YELLOW}=== 执行数据填充 ===${NC}"
echo -e "容器: ${YELLOW}$CONTAINER_NAME${NC}"
echo -e "${YELLOW}警告: 数据填充可能会重复插入数据${NC}"
echo ""

# 确认操作
read -p "确认要执行数据填充吗？(y/N): " confirm
if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    echo -e "${YELLOW}取消数据填充${NC}"
    exit 0
fi

# 执行填充
echo "执行数据填充..."
# APP_ENV=production requires --force (Goravel db:seed ConfirmToProceed).
if docker exec $CONTAINER_NAME /www/main artisan db:seed --force; then
    echo -e "${GREEN}✓ 数据填充成功${NC}"
    exit 0
else
    echo -e "${RED}✗ 数据填充失败${NC}"
    exit 1
fi

