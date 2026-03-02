# 高并发数据处理系统

基于Go语言的轻量级高并发数据处理系统，支持TCP长连接、消息队列（发布订阅）、批量处理和实时监控。

## 系统特性

- **高并发处理**：单机支持1万+并发TCP连接
- **高性能**：目标QPS 5000+，平均延迟<50ms
- **消息队列**：内置发布订阅模式，支持Redis消息队列
- **连接管理**：心跳检测、连接超时、自动重连
- **Goroutine池**：可配置的工作协程池，控制并发度
- **监控统计**：实时性能指标，支持Prometheus格式
- **协议设计**：JSON通信协议，易于扩展和调试

## 系统架构

```
客户端(TCP长连接)
    │
    ▼
TCP Server (Go net/tcp)
    │
    ▼
Worker Pool (Goroutine池)
    │
    ▼
Processor (业务处理器)
    │
    ├──► Redis (数据存储/消息队列)
    └──► 订阅推送
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- Redis 7.x
- Docker (可选)

### 2. 端口配置

⚠️ **重要**：系统使用两个不同端口，**不能混用**！

| 端口 | 协议 | 用途 | 访问方式 |
|------|------|------|----------|
| **8080** | TCP | 数据服务 | TCP客户端连接 |
| **8081** | HTTP | Web管理界面 | 浏览器访问 |

浏览器如果访问8080会导致TCP服务器报错（因为收到了HTTP请求而不是JSON数据）。

### 3. 本地运行

```bash
# 1. 启动Redis
docker run -d -p 6379:6379 --name redis-goserver redis:7-alpine

# 2. 运行服务
go run cmd/main.go

# 3. 浏览器访问Web界面（必须是8081端口！）
http://localhost:8081

# 4. TCP客户端测试（连接8080端口）
cd test
go run tcp_client.go -type=single

### 3. Docker部署

```bash
cd docker
./deploy.sh  # Linux/Mac
# 或
.\deploy.bat  # Windows
```

## 通信协议

### 请求格式
```json
{
  "id": "请求ID",
  "cmd": "process|batch|publish|subscribe|metrics",
  "channel": "队列名",
  "data": {}
}
```

### 命令说明

| 命令 | 功能 | 示例 |
|------|------|------|
| `process` | 处理单条数据 | `{"cmd":"process","data":{"name":"order"}}` |
| `batch` | 批量处理 | `{"cmd":"batch","data":[...]}` |
| `publish` | 发布消息 | `{"cmd":"publish","channel":"news","data":{}}` |
| `subscribe` | 订阅频道 | `{"cmd":"subscribe","channel":"news"}` |
| `metrics` | 获取指标 | `{"cmd":"metrics"}` |
| `ping` | 心跳检测 | `{"cmd":"ping"}` |

## 性能测试

```bash
# 压力测试：100连接，每连接1000请求
cd test
go run tcp_client.go -type=stress -c 100 -n 1000

# 预期结果：
# QPS: >5000
# Avg Latency: <50ms
```

## 配置说明

`.env` 文件：

```bash
# 服务器
SERVER_PORT=8080
SERVER_MAX_WORKERS=100

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# 应用
WORKERS=10          # Goroutine池大小
QUEUE_SIZE=1000     # 任务队列长度
BATCH_SIZE=100      # 批处理大小
```

## 项目结构

```
goserver/
├── cmd/
│   └── main.go              # 应用入口
├── internal/
│   ├── protocol/            # 通信协议
│   │   └── message.go
│   ├── network/             # TCP服务器
│   │   ├── server.go
│   │   └── connection.go
│   ├── pool/                # Goroutine池
│   │   └── worker.go
│   ├── processor/           # 业务处理器
│   │   └── processor.go
│   ├── cache/               # Redis客户端
│   │   └── redis.go
│   └── config/              # 配置管理
│       └── config.go
├── test/
│   ├── tcp_client.go        # TCP测试客户端
│   └── TEST_GUIDE.md        # 测试指南
├── docker/
│   ├── docker-compose.yml
│   └── Dockerfile
└── README.md
```

## 研究任务完成情况

- [x] **任务一**：学习Go并发编程，研究net/tcp标准库
- [x] **任务二**：设计系统架构（网络层、协议层、业务逻辑层）
- [x] **任务三**：实现TCP服务器核心功能（连接管理、协议解析、业务处理）
- [x] **任务四**：Goroutine池优化、心跳检测、连接超时
- [x] **任务五**：消息队列（发布订阅）、监控统计
- [x] **任务六**：压力测试工具、性能优化

## 技术亮点

1. **连接管理**：每个连接独立goroutine，支持10K并发
2. **Worker Pool**：限制并发处理数，防止资源耗尽
3. **心跳机制**：30秒间隔检测，90秒超时断开
4. **协议设计**：JSON格式，易于调试和扩展
5. **发布订阅**：支持实时消息推送

## License

MIT License
