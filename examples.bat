@echo off
REM 示例脚本：演示如何使用API处理数据（Windows版本）

setlocal enabledelayedexpansion

set API_URL=http://localhost:8080/api

echo.
echo === 高并发数据处理系统 - API使用示例 ===
echo.

REM 1. 获取系统健康状态
echo 1. 检查系统健康状态...
powershell -Command "Invoke-RestMethod -Uri 'http://localhost:8080/health' | ConvertTo-Json"
echo.

REM 2. 获取当前配置
echo 2. 获取系统配置...
powershell -Command "Invoke-RestMethod -Uri '%API_URL%/config' | ConvertTo-Json"
echo.

REM 3. 处理单条数据
echo 3. 处理单条数据...
powershell -Command ^
  "$body = @{ data = @{ id = 1; name = 'example'; timestamp = [int][double]::Parse((Get-Date -UFormat %%s)) } } | ConvertTo-Json" ^
  "Invoke-RestMethod -Uri '%API_URL%/process' -Method POST -Body $body -ContentType 'application/json' | ConvertTo-Json"
echo.

REM 4. 获取处理指标
echo 4. 获取处理指标...
powershell -Command "Invoke-RestMethod -Uri '%API_URL%/metrics' | ConvertTo-Json"
echo.

REM 5. 获取Redis键列表
echo 5. 获取Redis键列表...
powershell -Command "Invoke-RestMethod -Uri '%API_URL%/redis/keys?pattern=*' | ConvertTo-Json"
echo.

echo === 示例完成 ===
echo.
echo 提示: 您也可以使用Postman进行API测试
echo.
pause
