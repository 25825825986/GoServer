.PHONY: help build run test clean docker-build docker-up docker-down logs

# 变量
APP_NAME = goserver
MAIN_PATH = ./cmd/main.go
DOCKER_DIR = ./docker

help: ## 显示帮助信息
	@echo "高并发数据处理系统 - 开发工具"
	@echo ""
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 编译项目
	@echo "[BUILD] 编译项目..."
	@go build -o $(APP_NAME) $(MAIN_PATH)
	@echo "[OK] 编译完成: $(APP_NAME)"

run: ## 运行项目（需要Redis运行）
	@echo "[RUN] 启动应用..."
	@go run $(MAIN_PATH)

dev: ## 开发模式运行（自动重启）
	@echo "[DEV] 开发模式启动..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	@air

test: ## 运行测试
	@echo "[TEST] 运行测试..."
	@go test -v -cover ./...

bench: ## 运行基准测试
	@echo "[BENCH] 运行基准测试..."
	@go test -bench=. -benchmem ./...

clean: ## 清理构建产物
	@echo "[CLEAN] 清理构建产物..."
	@rm -f $(APP_NAME)
	@rm -rf dist/
	@echo "[OK] 清理完成"

fmt: ## 代码格式化
	@echo "[FMT] 格式化代码..."
	@go fmt ./...
	@echo "[OK] 格式化完成"

lint: ## 代码检查
	@echo "[LINT] 代码检查..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run ./...

mod-tidy: ## 整理依赖
	@echo "[MOD] 整理依赖..."
	@go mod tidy
	@echo "[OK] 依赖整理完成"

redis-start: ## 启动本地Redis容器
	@echo "[REDIS] 启动Redis..."
	@docker run -d -p 6379:6379 --name redis-goserver redis:7-alpine
	@echo "[OK] Redis已启动"

redis-stop: ## 停止Redis容器
	@echo "[STOP] 停止Redis..."
	@docker stop redis-goserver
	@docker rm redis-goserver
	@echo "[OK] Redis已停止"

docker-build: ## 构建Docker镜像
	@echo "[DOCKER] 构建Docker镜像..."
	@cd $(DOCKER_DIR) && docker-compose build
	@echo "[OK] 镜像构建完成"

docker-up: ## 启动Docker容器
	@echo "[DOCKER] 启动Docker容器..."
	@cd $(DOCKER_DIR) && docker-compose up -d
	@echo "[OK] 容器已启动"
	@echo "[INFO] 访问: http://localhost:8080"

docker-down: ## 停止Docker容器
	@echo "[STOP] 停止Docker容器..."
	@cd $(DOCKER_DIR) && docker-compose down
	@echo "[OK] 容器已停止"

docker-logs: ## 查看Docker日志
	@cd $(DOCKER_DIR) && docker-compose logs -f app

docker-restart: ## 重启Docker容器
	@echo "[RESTART] 重启Docker容器..."
	@cd $(DOCKER_DIR) && docker-compose restart
	@echo "[OK] 容器已重启"

logs: ## 查看应用日志
	@docker logs -f goserver-app

install-deps: ## 安装开发依赖
	@echo "[INSTALL] 安装开发工具..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "[OK] 开发工具安装完成"

check: fmt lint test ## 完整检查（格式化+检查+测试）
	@echo "[DONE] 所有检查完成"

all: clean build ## 完整构建

.DEFAULT_GOAL := help
