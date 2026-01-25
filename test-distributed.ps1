# 分布式压测测试脚本 (PowerShell)
# 用途: 在同一台机器上启动 M 个 Master + S 个 Slave 进行测试

Write-Host "🚀 分布式压测系统测试脚本" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan

# 配置参数
$MASTER_GRPC_PORT = 9090
$MASTER_HTTP_PORT = 8080
$SLAVE_COUNT = 3  # Slave 数量
$SLAVE_BASE_PORT = 9091

$MASTER_ADDR = "localhost:$MASTER_GRPC_PORT"
$ZONES = @("zone-a", "zone-b", "zone-c", "zone-d", "zone-e")  # 可用区列表

# 清理旧进程
Write-Host "`n🧹 清理旧进程..." -ForegroundColor Yellow
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

# 启动 Master
Write-Host "`n🎯 启动 Master 节点 (gRPC:$MASTER_GRPC_PORT, HTTP:$MASTER_HTTP_PORT)..." -ForegroundColor Cyan
Start-Process -FilePath ".\go-stress.exe" -ArgumentList @(
    "-mode", "master",
    "-grpc-port", $MASTER_GRPC_PORT,
    "-http-port", $MASTER_HTTP_PORT,
    "-workers-per-slave", "50",
    "-min-slave-count", ([Math]::Max(2, $SLAVE_COUNT - 1)),
    "-log-level", "info"
) -WindowStyle Normal
Start-Sleep -Seconds 2

# 批量启动 Slave 节点
Write-Host "`n🤖 启动 $SLAVE_COUNT 个 Slave 节点..." -ForegroundColor Green
for ($i = 1; $i -le $SLAVE_COUNT; $i++) {
    $slavePort = $SLAVE_BASE_PORT + $i - 1
    $realtimePort = 8088 + $i - 1  # 每个 Slave 使用不同的实时报告端口
    $slaveId = "slave-$i"
    $region = $ZONES[($i - 1) % $ZONES.Count]
    
    Write-Host "   [$i/$SLAVE_COUNT] 启动 $slaveId (gRPC:$slavePort, Realtime:$realtimePort, Region:$region)..." -ForegroundColor Cyan
    
    Start-Process -FilePath ".\go-stress.exe" -ArgumentList @(
        "-mode", "slave",
        "-master", $MASTER_ADDR,
        "-grpc-port", $slavePort,
        "-realtime-port", $realtimePort,
        "-slave-id", $slaveId,
        "-region", $region,
        "-log-level", "info"
    ) -WindowStyle Normal
    
    Start-Sleep -Milliseconds 500
}

Write-Host "   ✅ 所有 Slave 节点启动完成" -ForegroundColor Green
Start-Sleep -Seconds 1

Write-Host "`n✅ 所有节点启动完成!" -ForegroundColor Green
Write-Host "`n📊 访问管理界面:" -ForegroundColor Cyan
Write-Host "   http://localhost:$MASTER_HTTP_PORT" -ForegroundColor Yellow

# 自动打开浏览器
Write-Host "`n🌐 正在打开浏览器..." -ForegroundColor Cyan
Start-Sleep -Seconds 1
Start-Process "http://localhost:$MASTER_HTTP_PORT"

Write-Host "`n💡 测试步骤:" -ForegroundColor Cyan
Write-Host "   1. 打开浏览器访问上述地址" -ForegroundColor White
Write-Host "   2. 查看 Slave 列表 (应该有 $SLAVE_COUNT 个 Slave)" -ForegroundColor White
Write-Host "   3. 提交压测任务:" -ForegroundColor White
Write-Host "      - URL: http://httpbin.org/get" -ForegroundColor Gray
Write-Host "      - 并发数: 100" -ForegroundColor Gray
Write-Host "      - 请求数: 1000" -ForegroundColor Gray
Write-Host "   4. 点击 '启动任务' 按钮" -ForegroundColor White
Write-Host "   5. 观察任务分配和执行情况" -ForegroundColor White
Write-Host "`n⚙️  配置信息:" -ForegroundColor Cyan
Write-Host "   Slave 数量: $SLAVE_COUNT (修改 `$SLAVE_COUNT 变量可调整)" -ForegroundColor Gray
Write-Host "   gRPC 端口范围: $SLAVE_BASE_PORT - $($SLAVE_BASE_PORT + $SLAVE_COUNT - 1)" -ForegroundColor Gray
Write-Host "   实时报告端口: 8088 - $(8088 + $SLAVE_COUNT - 1)" -ForegroundColor Gray
Write-Host "`n🛑 停止测试: 按 Ctrl+C 或运行 Stop-Process -Name 'go-stress'" -ForegroundColor Yellow
Write-Host "=================================" -ForegroundColor Cyan
