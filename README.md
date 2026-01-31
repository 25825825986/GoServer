# 高并发数据处理系统

**轻量级、可扩展的 Go + Redis 并发数据处理平台**

这是一个基于 Go 与 Redis 的高并发数据处理示例项目，包含 Web UI、监控指标、任务队列与 Docker 一键部署方案，适合用于原型验证与教学示例。

---

## 📌 关键要点

- 语言：Go 1.21+
- 运行方式：Docker / 本地运行
- 存储：Redis（可选外部服务或容器）
- 主要功能：并发任务处理、队列批量处理、实时监控、配置管理

---

## 🚀 快速开始 (Docker 推荐)

1. 进入部署目录并运行部署脚本：

```bash
cd docker
# Linux / macOS
chmod +x deploy.sh && ./deploy.sh
# Windows
# deploy.bat
```

2. 访问 Web UI： http://localhost:8080
3. 验证健康： `curl http://localhost:8080/health`

---

## 🛠 本地开发

- 安装依赖： `go mod download`
- 启动本地 Redis（可选）： `docker run -d -p 6379:6379 redis:7-alpine`
- 运行： `go run cmd/main.go`
- 编译： `go build -o goserver ./cmd/main.go`
- 测试： `go test ./...`

---

## ⚙️ 配置（.env）

复制 `.env.example` 为 `.env`，常用变量：

- `SERVER_PORT`（默认 `8080`）
- `SERVER_MAX_WORKERS`（默认 `100`）
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- `BATCH_SIZE`, `WORKERS`, `QUEUE_SIZE`

---

## 🔌 主要 API 速查

- `GET /health` — 健康检查
- `GET /api/config` — 获取配置
- `POST /api/config` — 更新配置
- `POST /api/process` — 处理单条数据
- `POST /api/process/batch` — 批量处理
- `POST /api/process/queue` — 处理队列
- `GET /api/metrics` — 获取指标
- `POST /api/metrics/reset` — 重置指标
- `GET /api/redis/keys` — 列出 Redis 键

---

## 📁 项目结构（简要）

```
.
├── cmd/              # 应用入口
├── internal/         # 核心模块 (api, cache, config, processor)
├── web/              # Web UI 静态文件
├── docker/           # Docker 构建与部署脚本
├── .env.example
└── go.mod / go.sum
```

---

## ✅ 部署与运维要点

- Docker Compose 提供一键部署与日志查看
- 使用 `.env` 管理运行时参数
- 生产环境建议使用 HTTPS、设置 Redis 密码、并配置监控和备份

---

## 🧭 架构概览

- HTTP 层（Gin）处理请求并调度到内部处理器
- 数据处理层使用工作线程池与批处理策略
- Redis 用作队列与缓存，支持快速读写
- Web UI 提供实时监控、配置与操作界面

---

## 📋 检查清单（部署前）

- [ ] Docker & Docker Compose
- [ ] `.env` 已配置
- [ ] 端口 (`SERVER_PORT`) 未被占用
- [ ] Redis 可用
- [ ] 日志与监控配置就绪

---

## 📖 关于文档合并

已将仓库内的 `QUICKSTART.md`、`DEPLOYMENT.md`、`ARCHITECTURE.md`、`CHECKLIST.md`、`PROJECT_SUMMARY.md` 和 `GETTING_STARTED.md` 的内容合并到本 `README.md`。其它 `.md` 文件已替换为指向本文件的占位说明。

---

## 📫 贡献与反馈

欢迎提 Issue 或 Pull Request。开发流程建议：

- Fork -> 新建分支 -> 提交 -> 发起 PR

---

MIT License


## 🐳 Docker命令参考

```bash
# 查看容器状态
docker-compose -f docker/docker-compose.yml ps

# 查看应用日志
docker-compose -f docker/docker-compose.yml logs -f app

# 查看Redis日志
docker-compose -f docker/docker-compose.yml logs -f redis

# 进入容器
docker-compose -f docker/docker-compose.yml exec app sh

# 停止服务
docker-compose -f docker/docker-compose.yml down

# 清理所有容器和镜像
docker-compose -f docker/docker-compose.yml down -v --rmi all
```

## 📝 日志

应用日志输出到标准输出，可以通过以下方式查看：

```bash
# 查看最近100行日志
docker logs --tail 100 goserver-app

# 实时跟踪日志
docker logs -f goserver-app
```

## 🔐 安全建议

- 生产环境中修改Redis密码
- 使用HTTPS而不是HTTP
- 限制API访问IP范围
- 定期更新依赖包
- 启用请求认证

## 🐛 故障排查

### Redis连接失败
```bash
# 检查Redis是否运行
docker ps | grep redis

# 检查Redis日志
docker logs goserver-redis

# 测试Redis连接
redis-cli -h localhost -p 6379 ping
```

### 应用启动失败
```bash
# 查看应用日志
docker logs goserver-app

# 检查端口占用
netstat -an | grep 8080
```

### 内存使用过高
- 减少 `POOL_SIZE` 或 `QUEUE_SIZE`
- 清理长期未使用的Redis键
- 检查数据处理逻辑是否有内存泄漏

## 📈 下一步改进

- [ ] 添加数据库持久化支持
- [ ] 实现分布式追踪
- [ ] 添加认证和授权
- [ ] 支持Prometheus指标导出
- [ ] 实现自动扩缩容
- [ ] 添加数据加密传输

## 📄 许可证

MIT License

## 👥 贡献

欢迎提交Issue和Pull Request！

## 📞 联系方式

如有问题，请提交Issue或联系开发者。

---

**祝您使用愉快！** 🎉
