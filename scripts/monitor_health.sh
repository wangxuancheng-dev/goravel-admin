#!/bin/bash
# 健康检查监控脚本 - 用于检测服务在更新时是否中断
#  chmod +x monitor_health.sh
# ./monitor_health.sh http://localhost:3000 0.2

# 配置
SERVICE_URL="${1:-http://localhost:3000}"
CHECK_INTERVAL="${2:-0.5}"  # 检查间隔（秒），默认0.5秒
TIMEOUT="${3:-2}"           # 请求超时时间（秒）
LOG_FILE="${4:-/tmp/health_check.log}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 统计信息
TOTAL_REQUESTS=0
SUCCESS_REQUESTS=0
FAILED_REQUESTS=0
START_TIME=$(date +%s)
LAST_FAILURE_TIME=0
MAX_RESPONSE_TIME=0
MIN_RESPONSE_TIME=999999

# 创建日志文件
touch "$LOG_FILE"

echo "=========================================="
echo "服务健康检查监控"
echo "=========================================="
echo "服务地址: $SERVICE_URL"
echo "检查间隔: ${CHECK_INTERVAL}秒"
echo "超时时间: ${TIMEOUT}秒"
echo "日志文件: $LOG_FILE"
echo "=========================================="
echo ""

# 信号处理
cleanup() {
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    echo ""
    echo "=========================================="
    echo "监控结束"
    echo "=========================================="
    echo "总请求数: $TOTAL_REQUESTS"
    echo "成功请求: $SUCCESS_REQUESTS"
    echo "失败请求: $FAILED_REQUESTS"
    if [ $TOTAL_REQUESTS -gt 0 ]; then
        SUCCESS_RATE=$(echo "scale=2; $SUCCESS_REQUESTS * 100 / $TOTAL_REQUESTS" | bc)
        echo "成功率: ${SUCCESS_RATE}%"
    fi
    echo "监控时长: ${DURATION}秒"
    if [ $MAX_RESPONSE_TIME -lt 999999 ]; then
        echo "最大响应时间: ${MAX_RESPONSE_TIME}ms"
        echo "最小响应时间: ${MIN_RESPONSE_TIME}ms"
    fi
    echo "=========================================="
    exit 0
}

trap cleanup SIGINT SIGTERM

# 检查服务健康状态
check_health() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local start_ms=$(date +%s%N)
    
    # 存活探针（进程）
    response=$(curl -s -w "\n%{http_code}\n%{time_total}" --max-time "$TIMEOUT" "$SERVICE_URL/health" 2>&1)
    
    # 若只要就绪探测，可改为: "$SERVICE_URL/ready"
    if [ $? -ne 0 ] || echo "$response" | grep -q "404\|Not Found"; then
        response=$(curl -s -w "\n%{http_code}\n%{time_total}" --max-time "$TIMEOUT" "$SERVICE_URL/ready" 2>&1)
    fi
    
    local end_ms=$(date +%s%N)
    local response_time=$(( (end_ms - start_ms) / 1000000 ))  # 转换为毫秒
    
    # 解析响应
    local http_code=$(echo "$response" | tail -n 2 | head -n 1)
    local curl_time=$(echo "$response" | tail -n 1)
    
    TOTAL_REQUESTS=$((TOTAL_REQUESTS + 1))
    
    # 检查是否成功（HTTP 200-299 或 其他成功状态）
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        SUCCESS_REQUESTS=$((SUCCESS_REQUESTS + 1))
        
        # 更新响应时间统计
        if [ $response_time -gt $MAX_RESPONSE_TIME ]; then
            MAX_RESPONSE_TIME=$response_time
        fi
        if [ $response_time -lt $MIN_RESPONSE_TIME ]; then
            MIN_RESPONSE_TIME=$response_time
        fi
        
        # 显示成功（绿色）
        echo -e "${GREEN}✓${NC} [$timestamp] HTTP $http_code | ${response_time}ms | 总计: $TOTAL_REQUESTS | 成功: $SUCCESS_REQUESTS | 失败: $FAILED_REQUESTS"
        echo "[$timestamp] SUCCESS | HTTP $http_code | ${response_time}ms" >> "$LOG_FILE"
    else
        FAILED_REQUESTS=$((FAILED_REQUESTS + 1))
        LAST_FAILURE_TIME=$(date +%s)
        
        # 显示失败（红色）
        echo -e "${RED}✗${NC} [$timestamp] HTTP $http_code | ${response_time}ms | 错误: 服务不可用"
        echo "[$timestamp] FAILED | HTTP $http_code | ${response_time}ms | Response: $(echo "$response" | head -n 1)" >> "$LOG_FILE"
        
        # 如果连续失败，发出警告
        if [ $FAILED_REQUESTS -gt 0 ] && [ $((FAILED_REQUESTS % 5)) -eq 0 ]; then
            echo -e "${YELLOW}⚠ 警告: 已连续失败 $FAILED_REQUESTS 次${NC}"
        fi
    fi
}

# 主循环
echo "开始监控... (按 Ctrl+C 停止)"
echo ""

while true; do
    check_health
    sleep "$CHECK_INTERVAL"
done

