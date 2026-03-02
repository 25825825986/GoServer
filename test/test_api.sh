#!/bin/bash

# 数据处理系统 - API测试脚本
# 使用方法: ./test_api.sh

API_URL="http://localhost:8080"

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "  数据处理系统 - API测试"
echo "======================================"
echo ""

# 1. 健康检查
echo -e "${BLUE}[1/8] 健康检查...${NC}"
curl -s $API_URL/health | jq .
echo ""

# 2. 获取配置
echo -e "${BLUE}[2/8] 获取系统配置...${NC}"
curl -s $API_URL/api/config | jq .
echo ""

# 3. 获取当前指标
echo -e "${BLUE}[3/8] 获取当前指标...${NC}"
curl -s $API_URL/api/metrics | jq .
echo ""

# 4. 处理单条数据
echo -e "${BLUE}[4/8] 处理单条数据...${NC}"
curl -s -X POST $API_URL/api/process \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "id": "001",
      "name": "用户登录事件",
      "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
      "user_id": "user_12345",
      "ip": "192.168.1.100",
      "action": "login"
    }
  }' | jq .
echo ""

# 5. 批量处理数据
echo -e "${BLUE}[5/8] 批量处理5条数据...${NC}"
curl -s -X POST $API_URL/api/process/batch \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"id": "001", "name": "订单创建", "amount": 299.99, "currency": "CNY", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"},
      {"id": "002", "name": "订单支付", "amount": 299.99, "currency": "CNY", "status": "paid", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"},
      {"id": "003", "name": "订单发货", "tracking_no": "SF1234567890", "carrier": "顺丰", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"},
      {"id": "004", "name": "订单完成", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"},
      {"id": "005", "name": "用户退款", "amount": 299.99, "reason": "商品质量问题", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}
    ]
  }' | jq .
echo ""

# 6. 向队列添加数据并处理
echo -e "${BLUE}[6/8] 向Redis队列添加数据...${NC}"
# 先往队列添加一些数据
for i in {1..5}; do
  curl -s -X POST $API_URL/api/process \
    -H "Content-Type: application/json" \
    -d "{\"data\": {\"event\": \"queue_item_$i\", \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}}" > /dev/null
done
echo "已添加5条数据到处理队列"
echo ""

# 7. 查看更新后的指标
echo -e "${BLUE}[7/8] 查看更新后的指标...${NC}"
curl -s $API_URL/api/metrics | jq .
echo ""

# 8. 查看Redis中的键
echo -e "${BLUE}[8/8] 查看Redis中的键...${NC}"
curl -s "$API_URL/api/redis/keys?pattern=*" | jq .
echo ""

echo -e "${GREEN}======================================"
echo "  测试完成!"
echo "======================================${NC}"
echo ""
echo "Web UI: http://localhost:8080"
echo ""

# 可选：重置指标
read -p "是否重置指标? (y/n): " answer
if [ "$answer" = "y" ]; then
    echo -e "${YELLOW}重置指标...${NC}"
    curl -s -X POST $API_URL/api/metrics/reset | jq .
    echo ""
fi

echo "测试脚本执行完毕!"
