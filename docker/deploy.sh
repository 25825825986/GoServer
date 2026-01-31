#!/bin/bash

# 高并发数据处理系统 - 一键部署脚本

set -e

echo "=========================================="
echo "高并发数据处理系统 - 一键部署"
echo "=========================================="
echo ""

# 检查Docker是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ Docker未安装，请先安装Docker"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose未安装，请先安装Docker Compose"
    exit 1
fi

echo "✓ Docker和Docker Compose已安装"
echo ""

# 创建.env文件
if [ ! -f .env ]; then
    echo "📝 创建环境配置文件..."
    cp .env.example .env
    echo "✓ .env文件已创建，请根据需要修改配置"
fi

echo ""
echo "🔨 构建Docker镜像..."
docker-compose -f docker/docker-compose.yml build

echo ""
echo "🚀 启动服务..."
docker-compose -f docker/docker-compose.yml up -d

echo ""
echo "⏳ 等待服务启动..."
sleep 5

# 检查服务是否启动成功
if docker-compose -f docker/docker-compose.yml ps | grep -q "goserver-app.*Up"; then
    echo ""
    echo "=========================================="
    echo "✅ 部署成功！"
    echo "=========================================="
    echo ""
    echo "📊 Web UI: http://localhost:8080"
    echo "🔗 API: http://localhost:8080/api"
    echo "💾 Redis: localhost:6379"
    echo ""
    echo "常用命令："
    echo "  查看日志：docker logs -f goserver-app"
    echo "  停止服务：docker-compose -f docker/docker-compose.yml down"
    echo "  重启服务：docker-compose -f docker/docker-compose.yml restart"
    echo ""
else
    echo ""
    echo "❌ 服务启动失败！"
    echo ""
    echo "查看错误日志："
    docker-compose -f docker/docker-compose.yml logs app
    exit 1
fi
