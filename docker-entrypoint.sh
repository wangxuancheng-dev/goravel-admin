#!/bin/sh
# Docker 容器启动脚本：迁移 → 可选 seed → 启动应用

set -e

echo "=== 容器启动脚本 ==="

SKIP_MIGRATE="${SKIP_MIGRATE:-false}"
RUN_SEED="${RUN_SEED:-false}"

if [ "$SKIP_MIGRATE" != "true" ]; then
    echo "执行数据库迁移..."
    if /www/main artisan migrate; then
        echo "✓ 数据库迁移完成"
    else
        echo "✗ 数据库迁移失败，容器退出"
        exit 1
    fi
else
    echo "跳过数据库迁移 (SKIP_MIGRATE=true)"
fi

if [ "$RUN_SEED" = "true" ]; then
    echo "执行数据库填充 (RUN_SEED=true)..."
    if /www/main artisan db:seed; then
        echo "✓ 数据库填充完成"
    else
        echo "⚠ 数据库填充失败（可能已填充过），继续启动"
    fi
else
    echo "跳过数据库填充 (RUN_SEED!=true)"
fi

echo "启动应用..."
exec /www/main "$@"
