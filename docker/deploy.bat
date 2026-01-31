@echo off
REM 高并发数据处理系统 - 一键部署脚本（Windows版本）

setlocal enabledelayedexpansion

echo.
echo ==========================================
echo 高并发数据处理系统 - 一键部署
echo ==========================================
echo.

REM 检查Docker是否安装
docker --version >nul 2>&1
if errorlevel 1 (
    echo ❌ Docker未安装，请先安装Docker Desktop
    pause
    exit /b 1
)

docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo ❌ Docker Compose未安装，请先安装Docker Desktop
    pause
    exit /b 1
)

echo ✓ Docker和Docker Compose已安装
echo.

REM 创建.env文件
if not exist .env (
    echo 📝 创建环境配置文件...
    copy .env.example .env
    echo ✓ .env文件已创建，请根据需要修改配置
)

echo.
echo 🔨 构建Docker镜像...
cd docker
docker-compose build
cd ..

echo.
echo 🚀 启动服务...
cd docker
docker-compose up -d
cd ..

echo.
echo ⏳ 等待服务启动...
timeout /t 5

echo.
echo ==========================================
echo ✅ 部署成功！
echo ==========================================
echo.
echo 📊 Web UI: http://localhost:8080
echo 🔗 API: http://localhost:8080/api
echo 💾 Redis: localhost:6379
echo.
echo 常用命令：
echo   查看日志: docker logs -f goserver-app
echo   停止服务: docker-compose -f docker/docker-compose.yml down
echo   重启服务: docker-compose -f docker/docker-compose.yml restart
echo.

pause
