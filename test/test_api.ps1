# 数据处理系统 - API测试脚本 (PowerShell)
# 使用方法: .\test_api.ps1

$API_URL = "http://localhost:8080"

function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) { Write-Output $args }
    $host.UI.RawUI.ForegroundColor = $fc
}

Write-Output "======================================"
Write-Output "  数据处理系统 - API测试"
Write-Output "======================================"
Write-Output ""

# 1. 健康检查
Write-ColorOutput Cyan "[1/8] 健康检查..."
try {
    $response = Invoke-RestMethod -Uri "$API_URL/health" -Method GET
    $response | ConvertTo-Json
} catch {
    Write-ColorOutput Red "错误: $_"
}
Write-Output ""

# 2. 获取配置
Write-ColorOutput Cyan "[2/8] 获取系统配置..."
try {
    $response = Invoke-RestMethod -Uri "$API_URL/api/config" -Method GET
    $response | ConvertTo-Json -Depth 10
} catch {
    Write-ColorOutput Red "错误: $_"
}
Write-Output ""

# 3. 获取当前指标
Write-ColorOutput Cyan "[3/8] 获取当前指标..."
try {
    $response = Invoke-RestMethod -Uri "$API_URL/api/metrics" -Method GET
    $response | ConvertTo-Json
} catch {
    Write-ColorOutput Red "错误: $_"
}
Write-Output ""

# 4. 处理单条数据
Write-ColorOutput Cyan "[4/8] 处理单条数据..."
$singleData = @{
    data = @{
        id = "001"
        name = "用户登录事件"
        timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
        user_id = "user_12345"
        ip = "192.168.1.100"
        action = "login"
    }
} | ConvertTo-Json -Depth 10

try {
    $response = Invoke-RestMethod -Uri "$API_URL/api/process" -Method POST -ContentType "application/json" -Body $singleData
    $response | ConvertTo-Json
} catch {
    Write-ColorOutput Red "错误: $_"
}
Write-Output ""

# 5. 批量处理数据
Write-ColorOutput Cyan "[5/8] 批量处理5条数据..."
$batchData = @{
    data = @(
        @{ id = "001"; name = "订单创建"; amount = 299.99; currency = "CNY"; timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") },
        @{ id = "002"; name = "订单支付"; amount = 299.99; currency = "CNY"; status = "paid"; timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") },
        @{ id = "003"; name = "订单发货"; tracking_no = "SF1234567890"; carrier = "顺丰"; timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") },
        @{ id = "004"; name = "订单完成"; timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") },
        @{ id = "005"; name = "用户退款"; amount = 299.99; reason = "商品质量问题"; timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") }
    )
} | ConvertTo-Json -Depth 10

try {
    $response = Invoke-RestMethod -Uri "$API_URL/api/process/batch" -Method POST -ContentType "application/json" -Body $batchData
    $response | ConvertTo-Json
} catch {
    Write-ColorOutput Red "错误: $_"
}
Write-Output ""

# 6. 向队列添加数据
Write-ColorOutput Cyan "[6/8] 向Redis队列添加数据..."
for ($i = 1; $i -le 5; $i++) {
    $queueItem = @{
        data = @{
            event = "queue_item_$i"
            timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
        }
    } | ConvertTo-Json
    
    try {
        Invoke-RestMethod -Uri "$API_URL/api/process" -Method POST -ContentType "application/json" -Body $queueItem > $null
    } catch {}
}
Write-Output "已添加5条数据到处理队列"
Write-Output ""
Start-Sleep -Seconds 1

# 7. 查看更新后的指标
Write-ColorOutput Cyan "[7/8] 查看更新后的指标..."
try {
    $response = Invoke-RestMethod -Uri "$API_URL/api/metrics" -Method GET
    $response | ConvertTo-Json
} catch {
    Write-ColorOutput Red "错误: $_"
}
Write-Output ""

# 8. 查看Redis中的键
Write-ColorOutput Cyan "[8/8] 查看Redis中的键..."
try {
    $response = Invoke-RestMethod -Uri "$API_URL/api/redis/keys?pattern=*" -Method GET
    $response | ConvertTo-Json
} catch {
    Write-ColorOutput Red "错误: $_"
}
Write-Output ""

Write-ColorOutput Green "======================================"
Write-ColorOutput Green "  测试完成!"
Write-ColorOutput Green "======================================"
Write-Output ""
Write-Output "Web UI: http://localhost:8080"
Write-Output ""

# 可选：重置指标
$answer = Read-Host "是否重置指标? (y/n)"
if ($answer -eq "y") {
    Write-ColorOutput Yellow "重置指标..."
    try {
        $response = Invoke-RestMethod -Uri "$API_URL/api/metrics/reset" -Method POST
        $response | ConvertTo-Json
    } catch {
        Write-ColorOutput Red "错误: $_"
    }
    Write-Output ""
}

Write-Output "测试脚本执行完毕!"
