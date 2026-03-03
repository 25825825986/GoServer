@echo off
chcp 65001 >nul
REM 高并发数据处理系统 - 一键部署脚本(Windows版本)

setlocal enabledelayedexpansion

echo.
echo ==========================================
echo 高并发数据处理系统 - 一键部署
echo ==========================================
echo.

REM 检查Docker是否安装
docker --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker未安装，请先安装Docker Desktop
    echo 下载地址: https://www.docker.com/products/docker-desktop
    pause
    exit /b 1
)

docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker Compose未安装，请先安装Docker Desktop
    pause
    exit /b 1
)

echo [OK] Docker和Docker Compose已安装
echo.

REM 检查Docker是否运行
docker info >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker Desktop未运行，请先启动Docker Desktop
    echo.
    echo 按任意键退出...
    pause >nul
    exit /b 1
)

echo [OK] Docker守护进程运行正常
echo.

REM 获取脚本所在目录
cd /d "%~dp0"
set "SCRIPT_DIR=%CD%"

REM 切换到项目根目录（docker的父目录）
cd ..
set "PROJECT_ROOT=%CD%"

echo 项目根目录: %PROJECT_ROOT%
echo.

REM 创建.env文件
if not exist .env (
    echo [INFO] 创建环境配置文件...
    if exist .env.example (
        copy .env.example .env
        echo [OK] .env文件已创建，请根据需要修改配置
    ) else (
        echo [WARN] .env.example不存在，跳过创建.env
    )
) else (
    echo [OK] .env文件已存在
)
echo.

echo [BUILD] 构建Docker镜像...
cd /d "%SCRIPT_DIR%"
docker-compose build
if errorlevel 1 (
    echo [ERROR] 构建失败
    pause
    exit /b 1
)

echo.
echo [START] 启动服务...
docker-compose up -d
if errorlevel 1 (
    echo [ERROR] 启动失败
    pause
    exit /b 1
)

echo.
echo [WAIT] 等待服务启动...
timeout /t 5 /nobreak >nul

echo.
echo ==========================================
echo [SUCCESS] 部署成功！
echo ==========================================
echo.
echo  访问地址:
echo   - Web UI: http://localhost:8081
echo   - TCP Server: localhost:8080
echo   - Health: http://localhost:8081/health
echo   - Redis: localhost:6379
echo.
echo 常用命令:
echo   查看日志: docker logs -f goserver-app
echo   停止服务: docker-compose -f docker/docker-compose.yml down
echo   重启服务: docker-compose -f docker/docker-compose.yml restart
echo.

pause
