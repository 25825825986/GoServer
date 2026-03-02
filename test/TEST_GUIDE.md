# 高并发数据处理系统 - 测试指南

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        客户端层                              │
│              (TCP长连接 / HTTP API / Web UI)                │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      TCP Server 层                           │
│  • 支持1万并发连接                                           │
│  • 连接管理（心跳检测、超时处理）                              │
│  • 协议解析（JSON格式）                                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Worker Pool 层                          │
│  • Goroutine池（可配置工作协程数）                            │
│  • 任务队列（异步处理）                                       │
│  • 性能指标统计                                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Processor 层                            │
│  • 单条/批量数据处理                                         │
│  • 消息队列（发布订阅模式）                                   │
│  • Redis数据存储                                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Redis 层                               │
│  • 数据缓存（24小时TTL）                                     │
│  • 消息队列（List结构）                                       │
│  • 连接池管理                                                │
└─────────────────────────────────────────────────────────────┘
```

## 通信协议

### 请求消息格式
```json
{
  "id": "请求唯一ID",
  "cmd": "命令类型",
  "channel": "队列/频道名称（可选）",
  "data": "数据负载"
}
```

### 命令类型
| 命令 | 说明 |
|------|------|
| `process` | 处理单条数据 |
| `batch` | 批量处理数据 |
| `publish` | 发布消息到队列 |
| `subscribe` | 订阅频道 |
| `ping` | 心跳检测 |
| `metrics` | 获取性能指标 |
| `config` | 获取系统配置 |

### 响应消息格式
```json
{
  "id": "对应请求ID",
  "status": "ok/error",
  "data": "响应数据",
  "error": "错误信息",
  "latency": "处理延迟（毫秒）"
}
```

## 快速开始

### 1. 启动服务

```bash
# 启动Redis（如果没有运行）
docker run -d -p 6379:6379 --name redis-goserver redis:7-alpine

# 启动TCP服务器
go run cmd/main.go
```

### 2. 使用TCP客户端测试

```bash
# 进入测试目录
cd test

# 单条数据测试
go run tcp_client.go -type=single

# 批量数据测试
go run tcp_client.go -type=batch

# 发布订阅测试
go run tcp_client.go -type=pubsub

# 压力测试（100连接，每连接100请求）
go run tcp_client.go -type=stress -c 100 -n 100
```

### 3. 使用telnet手动测试

```bash
# 连接服务器
telnet localhost 8080

# 发送单条数据（输入后按回车）
{"id":"1","cmd":"process","data":{"name":"test","value":123}}

# 心跳检测
{"id":"2","cmd":"ping"}

# 获取指标
{"id":"3","cmd":"metrics"}

# 订阅频道
{"id":"4","cmd":"subscribe","channel":"news"}

# 发布消息（在另一个终端）
{"id":"5","cmd":"publish","channel":"news","data":{"title":"Hello"}}
```

## 性能测试

### 目标指标
- **并发连接**：10,000+
- **QPS**：5,000+
- **平均延迟**：< 50ms

### 压力测试命令

```bash
# 小规模测试（验证功能）
go run tcp_client.go -type=stress -c 10 -n 100

# 中等规模测试
go run tcp_client.go -type=stress -c 100 -n 1000

# 大规模测试（接近目标）
go run tcp_client.go -type=stress -c 1000 -n 1000

# 极限测试（需要调整系统ulimit）
go run tcp_client.go -type=stress -c 10000 -n 100
```

### 使用wrk进行HTTP压力测试（如果启用HTTP接口）

```bash
# 安装wrk
# Mac: brew install wrk
# Linux: 源码编译

# 运行压力测试
wrk -t12 -c400 -d30s http://localhost:8080/health
```

### 调整系统限制（Linux/Mac）

```bash
# 查看当前限制
ulimit -n

# 临时增加文件描述符限制（支持1万连接）
ulimit -n 20000

# 永久修改（需要sudo，写入/etc/security/limits.conf）
* soft nofile 20000
* hard nofile 20000
```

## 监控指标

### 实时指标查询

```bash
# 通过TCP客户端查询
echo '{"id":"1","cmd":"metrics"}' | nc localhost 8080
```

### 指标说明

| 指标 | 说明 |
|------|------|
| `processed_count` | 成功处理的数据条数 |
| `failed_count` | 处理失败的数据条数 |
| `published_count` | 发布的消息数 |
| `subscribed_count` | 订阅的频道数 |
| `avg_latency` | 平均处理延迟（毫秒） |
| `last_process` | 最后处理时间 |

### 使用Redis命令行查看数据

```bash
# 连接到Redis
redis-cli

# 查看所有键
keys *

# 查看数据
lrange data:queue 0 -1

# 查看队列长度
llen data:queue

# 监控Redis命令
monitor
```

## Docker部署

```bash
# 构建并启动
cd docker
docker-compose up -d

# 查看日志
docker-compose logs -f app

# 停止
docker-compose down
```

## 配置文件

`.env` 文件示例：

```bash
# 服务器配置
SERVER_PORT=8080
SERVER_READ_TIMEOUT=10
SERVER_WRITE_TIMEOUT=10
SERVER_MAX_WORKERS=100

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# 应用配置
APP_NAME=DataProcessor
APP_VERSION=1.0.0
BATCH_SIZE=100
WORKERS=10          # Goroutine池大小
QUEUE_SIZE=1000     # 任务队列大小
```

## 模块说明

| 模块 | 路径 | 职责 |
|------|------|------|
| protocol | `internal/protocol/` | 通信协议定义 |
| network | `internal/network/` | TCP服务器、连接管理 |
| pool | `internal/pool/` | Goroutine工作池 |
| processor | `internal/processor/` | 业务处理器 |
| cache | `internal/cache/` | Redis客户端 |
| config | `internal/config/` | 配置管理 |

## 常见问题

### Q: 连接数达到上限？
A: 增加系统文件描述符限制：`ulimit -n 20000`

### Q: Redis连接失败？
A: 检查Redis是否运行：`docker ps | grep redis`

### Q: QPS达不到5000？
A: 调整WORKERS数量（建议CPU核心数的2-4倍）

### Q: 如何处理更多并发？
A: 可以水平扩展，部署多个实例，使用负载均衡
