# 数据处理系统 - 测试指南

## 🚀 快速开始

### 1. 启动服务

```bash
# 方式1：本地运行（需要先启动Redis）
make redis-start  # 启动Redis
go run cmd/main.go  # 启动应用

# 方式2：Docker运行
cd docker
./deploy.sh  # Linux/Mac
# 或
.\deploy.bat  # Windows
```

服务启动后访问：http://localhost:8080

---

## 📋 手动测试命令

### 1. 健康检查
```bash
curl http://localhost:8080/health
```

### 2. 获取配置
```bash
curl http://localhost:8080/api/config
```

### 3. 处理单条数据
```bash
curl -X POST http://localhost:8080/api/process \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "id": "001",
      "name": "用户登录事件",
      "timestamp": "2026-03-01T10:30:00Z",
      "user_id": "user_12345",
      "ip": "192.168.1.100",
      "action": "login"
    }
  }'
```

### 4. 批量处理数据
```bash
curl -X POST http://localhost:8080/api/process/batch \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"id": "001", "name": "订单创建", "amount": 299.99},
      {"id": "002", "name": "订单支付", "amount": 299.99, "status": "paid"},
      {"id": "003", "name": "订单发货", "carrier": "顺丰"},
      {"id": "004", "name": "订单完成"},
      {"id": "005", "name": "用户退款", "amount": 299.99, "reason": "质量问题"}
    ]
  }'
```

### 5. 队列处理（先往Redis队列添加数据）

添加数据到队列：
```bash
# 使用redis-cli添加数据到队列
redis-cli LPUSH data:queue '{"event":"page_view","user_id":"u001"}'
redis-cli LPUSH data:queue '{"event":"click","user_id":"u001"}'
redis-cli LPUSH data:queue '{"event":"purchase","user_id":"u001","amount":199.99}'
```

然后处理队列：
```bash
curl -X POST http://localhost:8080/api/process/queue \
  -H "Content-Type: application/json" \
  -d '{
    "queue_key": "data:queue",
    "batch_size": 100
  }'
```

### 6. 查看处理指标
```bash
curl http://localhost:8080/api/metrics
```

### 7. 重置指标
```bash
curl -X POST http://localhost:8080/api/metrics/reset
```

### 8. 查看Redis键
```bash
curl "http://localhost:8080/api/redis/keys?pattern=*"
```

---

## 🤖 自动化测试

### 使用测试脚本（推荐）

**Linux/Mac:**
```bash
chmod +x test_api.sh
./test_api.sh
```

**Windows:**
```powershell
.\test_api.ps1
```

测试脚本会自动执行：
1. 健康检查
2. 获取配置
3. 获取初始指标
4. 单条数据处理
5. 批量数据处理（5条）
6. 队列数据添加
7. 查看更新后的指标
8. 查看Redis键

---

## 📊 测试数据示例

### 电商订单数据
```json
{
  "id": "ORDER_20260301_001",
  "type": "order",
  "status": "created",
  "amount": 599.99,
  "currency": "CNY",
  "items": [
    {"sku": "SKU001", "name": "iPhone 15", "price": 5999.00, "qty": 1},
    {"sku": "SKU002", "name": "AirPods Pro", "price": 1999.00, "qty": 1}
  ],
  "customer": {
    "id": "CUST_001",
    "name": "张三",
    "phone": "13800138000",
    "address": "北京市朝阳区xxx"
  },
  "timestamp": "2026-03-01T10:00:00Z"
}
```

### 用户行为日志
```json
{
  "event": "user_action",
  "type": "click",
  "page": "/product/12345",
  "element": "add_to_cart",
  "user_id": "user_abc123",
  "session_id": "sess_xyz789",
  "ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "timestamp": "2026-03-01T10:30:00Z"
}
```

### 传感器数据
```json
{
  "device_id": "SENSOR_001",
  "type": "temperature",
  "value": 25.6,
  "unit": "celsius",
  "location": {"lat": 39.9042, "lng": 116.4074},
  "battery": 87,
  "timestamp": "2026-03-01T10:30:00Z"
}
```

---

## 🖥️ Web UI 操作

访问 http://localhost:8080 可以进行可视化操作：

1. **查看系统状态** - 实时显示应用和Redis连接状态
2. **监控指标** - 查看已处理数量、失败数量、平均延迟
3. **配置管理** - 通过标签页切换修改服务器/Redis/应用配置
4. **数据处理** - 
   - 单条数据：在文本框输入JSON数据，点击"开始处理"
   - 队列处理：指定队列键名和批处理大小
5. **快速操作** - 刷新指标、重置指标、清空Redis

---

## 📈 性能测试

### 批量压力测试
```bash
# 生成1000条测试数据并批量处理
for i in {1..1000}; do
  curl -s -X POST http://localhost:8080/api/process \
    -H "Content-Type: application/json" \
    -d "{\"data\":{\"id\":\"$i\",\"value\":$i}}" > /dev/null &
done
wait

echo "批量测试完成"
curl http://localhost:8080/api/metrics
```

### 使用 Apache Bench (ab)
```bash
# 安装 ab: sudo apt-get install apache2-utils

# 准备测试数据文件
echo '{"data":{"id":"1","test":"value"}}' > post_data.json

# 1000请求，100并发
ab -n 1000 -c 100 -p post_data.json -T application/json http://localhost:8080/api/process
```

---

## 🔍 调试技巧

### 查看应用日志
```bash
# 本地运行直接查看控制台输出

# Docker运行
docker logs -f goserver-app
```

### 查看Redis数据
```bash
# 连接到Redis
redis-cli -p 6379

# 常用命令
keys *                    # 查看所有键
lrange data:queue 0 -1    # 查看队列内容
info                      # 查看Redis信息
monitor                   # 实时监控命令（调试用）
```

### 使用浏览器开发者工具
1. 打开 http://localhost:8080
2. 按 F12 打开开发者工具
3. 切换到 Network 标签
4. 操作页面，观察API请求和响应

---

## ❓ 常见问题

**Q: 连接Redis失败？**
```
Failed to connect to Redis: dial tcp 127.0.0.1:6379
```
A: 确保Redis已启动：`docker run -d -p 6379:6379 redis:7-alpine`

**Q: 端口被占用？**
```
listen tcp :8080: bind: address already in use
```
A: 修改 .env 中的 `SERVER_PORT=8081` 或使用其他端口

**Q: JSON格式错误？**
A: 确保请求头包含 `-H "Content-Type: application/json"`，且数据是有效的JSON格式。

---

## 📞 更多帮助

- 查看 README.md 了解项目详情
- 查看 Makefile 了解可用命令
- 访问 Web UI: http://localhost:8080
