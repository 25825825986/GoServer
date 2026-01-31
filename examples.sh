#!/bin/bash

# 示例脚本：演示如何使用API处理数据

API_URL="http://localhost:8080/api"

echo "=== 高并发数据处理系统 - API使用示例 ==="
echo ""

# 1. 获取系统健康状态
echo "1. 检查系统健康状态..."
curl -s http://localhost:8080/health | jq .
echo ""

# 2. 获取当前配置
echo "2. 获取系统配置..."
curl -s $API_URL/config | jq .
echo ""

# 3. 处理单条数据
echo "3. 处理单条数据..."
curl -s -X POST $API_URL/process \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "id": 1,
      "name": "example",
      "timestamp": "'$(date +%s)'"
    }
  }' | jq .
echo ""

# 4. 批量处理数据
echo "4. 批量处理数据..."
curl -s -X POST $API_URL/process/batch \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"id": 1, "value": "data1"},
      {"id": 2, "value": "data2"},
      {"id": 3, "value": "data3"}
    ]
  }' | jq .
echo ""

# 5. 获取处理指标
echo "5. 获取处理指标..."
curl -s $API_URL/metrics | jq .
echo ""

# 6. 获取Redis键列表
echo "6. 获取Redis键列表..."
curl -s "$API_URL/redis/keys?pattern=*" | jq .
echo ""

# 7. 重置指标
echo "7. 重置处理指标..."
curl -s -X POST $API_URL/metrics/reset | jq .
echo ""

echo "=== 示例完成 ==="
echo ""
echo "💡 提示：使用 jq 工具可以更好地查看JSON输出"
echo "   安装: brew install jq (Mac) 或 apt-get install jq (Linux)"
