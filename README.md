# 日志实时处理系统

一个基于 Go 开发的高并发日志实时收集、处理和存储系统。

## 功能特性

- **实时日志收集**：通过 TCP 端口接收日志数据，支持高并发连接
- **日志级别分类**：支持 Debug、Info、Warn、Error、Fatal 级别
- **多维度索引**：按时间、级别、来源自动建立索引
- **Web 管理界面**：实时查看日志流、统计分析、日志筛选
- **Redis 持久化**：日志数据存储在 Redis，支持过期清理

## 快速开始

### 1. 启动 Redis

```bash
docker run -d -p 6379:6379 --name redis-logs redis:7-alpine
```

### 2. 启动系统

```bash
go run cmd/main.go
# 或使用编译好的程序
./goserver.exe
```

### 3. 访问 Web 界面

打开浏览器访问：http://localhost:8081

## TCP 日志上报协议

### 单条日志上报

```json
{
  "id": "log-001",
  "cmd": "log",
  "data": {
    "timestamp": 1709452800000,
    "level": "error",
    "source": "api-server",
    "message": "数据库连接失败",
    "tags": ["db", "critical"],
    "metadata": {"error_code": 5001}
  }
}
```

### 批量日志上报

```json
{
  "id": "batch-001",
  "cmd": "batch",
  "data": [
    {"level": "info", "source": "app", "message": "服务启动"},
    {"level": "debug", "source": "app", "message": "配置加载完成"}
  ]
}
```

### 查询日志

```json
{
  "id": "query-001",
  "cmd": "query",
  "filters": {
    "level": "error",
    "source": "api-server",
    "limit": 50
  }
}
```

## HTTP API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/metrics` | GET | 获取系统指标统计 |
| `/api/logs` | GET | 查询日志（支持筛选） |
| `/api/logs/recent` | GET | 获取最近日志 |
| `/api/logs/stats` | GET | 获取今日日志统计 |
| `/api/logs/levels` | GET | 获取级别分布 |
| `/api/logs` | DELETE | 清空所有日志 |

## 项目结构

```
├── cmd/
│   └── main.go              # 程序入口
├── internal/
│   ├── api/                 # HTTP API 服务
│   ├── cache/               # Redis 客户端
│   ├── config/              # 配置管理
│   ├── network/             # TCP 服务器
│   ├── pool/                # Worker 协程池
│   ├── processor/           # 日志处理器
│   └── protocol/            # 通信协议
├── web/
│   └── public/
│       └── index.html       # Web 管理界面
└── docker/
    └── docker-compose.yml   # Docker 部署配置
```

## 配置说明

通过环境变量或 `.env` 文件配置：

```env
# 服务器配置
SERVER_PORT=8080          # TCP 日志接收端口
SERVER_READ_TIMEOUT=10
SERVER_WRITE_TIMEOUT=10
SERVER_MAX_WORKERS=100

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Worker配置
WORKERS=10                # 并发处理Worker数
QUEUE_SIZE=1000           # 任务队列大小
```

## 技术栈

- **Go**: 高性能后端服务
- **Redis**: 日志数据存储和索引
- **Gin**: HTTP Web 框架
- **WebSocket**: 实时日志推送（待实现）

## License

MIT
