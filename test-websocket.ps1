# WebSocket 压测测试脚本 (PowerShell)
# 用途: 启动测试服务器并运行 WebSocket 压测

Write-Host "🚀 WebSocket 压测测试脚本" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan

# 配置参数
$TEST_SERVER_PORT = 3000
$REALTIME_PORT = 8088

# 清理旧进程
Write-Host "`n🧹 清理旧进程..." -ForegroundColor Yellow
Get-Process | Where-Object { $_.ProcessName -eq "test_server" } | Stop-Process -Force -ErrorAction SilentlyContinue
Get-Process | Where-Object { $_.ProcessName -eq "go-stress" } | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

# 构建项目
Write-Host "`n🔨 构建项目..." -ForegroundColor Yellow
go build -o go-stress.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ 构建失败" -ForegroundColor Red
    exit 1
}
Write-Host "✅ 构建完成" -ForegroundColor Green

# 启动测试服务器
Write-Host "`n🎯 启动 WebSocket 测试服务器 (端口:$TEST_SERVER_PORT)..." -ForegroundColor Cyan
$serverJob = Start-Job -ScriptBlock {
    Set-Location $using:PWD
    cd testserver
    go run test_server.go
}
Start-Sleep -Seconds 3

# 检查服务器是否启动成功
Write-Host "🔍 检查服务器状态..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:$TEST_SERVER_PORT/api/health" -UseBasicParsing -TimeoutSec 2
    Write-Host "✅ 测试服务器启动成功" -ForegroundColor Green
} catch {
    Write-Host "❌ 测试服务器启动失败" -ForegroundColor Red
    Stop-Job $serverJob
    Remove-Job $serverJob
    exit 1
}

# 显示可用的测试场景
Write-Host "`n📋 可用的测试场景:" -ForegroundColor Cyan
Write-Host "   1. 快速测试 (websocket-quick.yaml) - 5并发, 20请求" -ForegroundColor White
Write-Host "   2. 标准测试 (websocket-test.yaml) - 10并发, 100请求" -ForegroundColor White
Write-Host "   3. 回声测试 (websocket-echo.yaml) - 20并发, 500请求" -ForegroundColor White
Write-Host "   4. 聊天室测试 (websocket-chat.yaml) - 50并发, 1000请求" -ForegroundColor White

# 选择测试场景
Write-Host "`n请选择测试场景 (1-4, 默认: 1): " -ForegroundColor Yellow -NoNewline
$choice = Read-Host

switch ($choice) {
    "2" { $configFile = "testserver/websocket-test.yaml" }
    "3" { $configFile = "testserver/websocket-echo.yaml" }
    "4" { $configFile = "testserver/websocket-chat.yaml" }
    default { $configFile = "testserver/websocket-quick.yaml" }
}

# 运行压测
Write-Host "`n🚀 开始压测: $configFile" -ForegroundColor Green
Write-Host "📊 实时监控: http://localhost:$REALTIME_PORT" -ForegroundColor Yellow
Write-Host "=================================" -ForegroundColor Cyan

.\go-stress.exe -config $configFile

# 显示结果
Write-Host "`n=================================" -ForegroundColor Cyan
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ 压测完成!" -ForegroundColor Green
    Write-Host "`n📈 查看报告:" -ForegroundColor Cyan
    $reportFiles = Get-ChildItem -Path "stress-reports" -Filter "*.html" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($reportFiles) {
        Write-Host "   文件: $($reportFiles.FullName)" -ForegroundColor Yellow
        Write-Host "   提示: 用浏览器打开查看详细报告" -ForegroundColor White
    }
} else {
    Write-Host "❌ 压测失败!" -ForegroundColor Red
}

# 清理
Write-Host "`n🧹 清理进程..." -ForegroundColor Yellow
Stop-Job $serverJob -ErrorAction SilentlyContinue
Remove-Job $serverJob -ErrorAction SilentlyContinue
Write-Host "✅ 完成" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Cyan
