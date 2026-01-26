# 分布式压测

使用 Master/Slave 架构实现大规模分布式压测

## 架构概述

```
        Master (主节点)
            │
  ┌─────────┼─────────┐
  │         │         │
Slave 1  Slave 2  Slave 3
(肉鸡1)  (肉鸡2)  (肉鸡3)
```

### 角色说明

- **Master**：分发配置、统一调度、收集统计、生成报告
- **Slave**：执行压测、上报统计、接收控制

### 适用场景

- 超大规模压测（单机 QPS 不足）
- 分布式环境模拟（不同地域、网络）
- 多目标并发压测
- 长时间稳定性测试

## 快速开始

### 1. 启动 Master 节点

```bash
./go-stress -mode master \
  -config config.yaml \
  -grpc-port 9090 \
  -http-port 8080
```

**参数说明**：
- `-mode master`：Master 模式
- `-config`：压测配置文件
- `-grpc-port`：gRPC 端口（Slave 连接）
- `-http-port`：HTTP 管理端口（查看状态）

### 2. 启动 Slave 节点

在多台机器上启动：

```bash
# 机器 1
./go-stress -mode slave \
  -master 192.168.1.100:9090 \
  -region beijing \
  -slave-id slave-bj-01 \
  -realtime-port 8088

# 机器 2
./go-stress -mode slave \
  -master 192.168.1.100:9090 \
  -region shanghai \
  -slave-id slave-sh-01 \
  -realtime-port 8089

# 机器 3
./go-stress -mode slave \
  -master 192.168.1.100:9090 \
  -region guangzhou \
  -slave-id slave-gz-01 \
  -realtime-port 8090
```

**参数说明**：
- `-mode slave`：Slave 模式
- `-master`：Master 地址（IP:端口）
- `-region`：区域标签（可选）
- `-slave-id`：节点 ID（可选，自动生成）
- `-realtime-port`：实时报告端口（可选，默认 8088）

### 3. 查看状态

访问 Master 管理界面：

```
http://master-ip:8080
```

## 工作流程

### 任务创建和执行流程

1. **Slave 注册**：Slave 连接 Master 并注册
2. **创建任务**：通过 Web 界面或 API 创建任务（状态：pending）
3. **启动任务**：手动启动任务，可选择指定 Slave 节点或区域
4. **任务分配**：Master 将任务分发给选定的 Slave
5. **执行压测**：所有 Slave 并行执行压测
6. **实时上报**：Slave 定期向 Master 上报统计
7. **汇总报告**：Master 汇总所有 Slave 的数据并生成报告

### 任务状态

- **pending**：待执行（已创建但未启动）
- **running**：运行中（正在执行）
- **completed**：已完成（执行成功）
- **failed**：失败（执行出错）
- **stopped**：已停止（用户中断）
- **cancelled**：已取消（用户取消）

### Slave 选择策略

启动任务时可选择 3 种 Slave 分配策略：

#### 1. 全部节点（默认）

使用所有可用的 Slave 节点。

```bash
curl -X POST http://master:8080/api/v1/tasks/{task_id}/start
```

Web 界面：选择"使用所有可用 Slave" → 点击"启动任务"

#### 2. 指定节点

手动选择特定的 Slave 节点。

```bash
curl -X POST http://master:8080/api/v1/tasks/{task_id}/start \
  -H "Content-Type: application/json" \
  -d '{"slave_ids": ["slave-bj-01", "slave-sh-01"]}'
```

Web 界面：选择"指定 Slave 节点" → 勾选 Slave → 点击"启动任务"

#### 3. 按区域选择

使用指定区域的所有 Slave 节点。

```bash
curl -X POST http://master:8080/api/v1/tasks/{task_id}/start \
  -H "Content-Type: application/json" \
  -d '{"slave_region": "beijing"}'
```

Web 界面：选择"按区域选择" → 选择区域 → 点击"启动任务"

### 任务重试

对于完成、失败或停止的任务，可以一键重试：

```bash
# API 方式
curl -X POST http://master:8080/api/v1/tasks/{task_id}/retry

# 响应
{
  "message": "Task retry submitted successfully",
  "new_task_id": "538393102122094592",
  "original_task_id": "538392929836863488",
  "state": "pending"
}
```

**Web 界面**：在任务列表点击"🔁 重试"按钮

**重试特性**：
- 保留原任务配置（协议、URL、并发数等）
- 自动记录重试元数据（`retry_from`、`retry_reason`）
- 支持多次重试，保持完整审计跟踪

## 任务分配策略

### 均匀分配（默认）

```yaml
concurrency: 1000
requests: 100000
```

如果有 4 个 Slave：
- 每个 Slave：250 并发，25000 请求

## Master 配置

```yaml
# master-config.yaml
protocol: http
concurrency: 1000
requests: 100000
timeout: 10s

url: https://api.example.com/users
method: POST
headers:
  Content-Type: application/json
body: '{"test":"data"}'

advanced:
  enable_breaker: true
  max_failures: 100
  ramp_up: 60s
```

## 监控和管理

### Master Web 界面

访问 `http://master:8080` 查看：

**任务管理**：
1. 创建任务：提交配置文件或 JSON 配置
2. 查看任务列表：所有任务的状态和进度
3. 启动任务：点击"启动"按钮开始执行
4. 查看详情：点击任务 ID 查看详细信息和 Slave 分配情况
5. 任务重试：失败/完成任务可一键重试

**节点管理**：
- 查看所有 Slave 节点的状态、区域、资源使用情况
- 点击 Slave ID 查看详细信息（CPU、内存、任务历史）
- 自动刷新机制（3/5/10/30秒间隔可配置）

**实时报告**：
- 点击任务列表中的"📊 实时报告"按钮
- 支持跨 Slave 数据查询（通过 `realtime_url` 参数）
- 实时显示 QPS、成功率、延迟等指标
- 支持按节点/任务过滤请求详情

**节点列表**：
```
┌──────────────┬──────────┬────────┬─────────┬────────┐
│ Slave ID     │ Region   │ Status │ QPS     │ Errors │
├──────────────┼──────────┼────────┼─────────┼────────┤
│ slave-bj-01  │ beijing  │ Active │ 1250.32 │ 5      │
│ slave-sh-01  │ shanghai │ Active │ 1180.45 │ 3      │
│ slave-gz-01  │ guangzhou│ Active │ 1200.18 │ 2      │
└──────────────┴──────────┴────────┴─────────┴────────┘
```

**汇总报告**：
```
Total Requests : 100000
Total Success  : 99850 (99.85%)
Total Failed   : 150 (0.15%)
Total Duration : 32.5s
Total QPS      : 3076.92

By Region:
  beijing    : 33500 requests, QPS: 1030.77
  shanghai   : 33200 requests, QPS: 1021.54
  guangzhou  : 33300 requests, QPS: 1024.62
```

### API 接口

#### 1. 创建任务

**请求**：
```bash
curl -X POST http://master:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "config_file": "{\"protocol\":\"http\",\"url\":\"https://api.example.com\",\"concurrency\":100,\"requests\":1000}"
  }'
```

**响应**：
```json
{
  "task_id": "task-123456",
  "message": "Task created successfully, use /api/v1/tasks/{id}/start to start execution"
}
```

#### 2. 启动任务

**请求**（默认分配）：
```bash
curl -X POST http://master:8080/api/v1/tasks/task-123456/start
```

**请求**（指定 Slave）：
```bash
curl -X POST http://master:8080/api/v1/tasks/task-123456/start \
  -H "Content-Type: application/json" \
  -d '{
    "slave_ids": ["slave-1", "slave-2"]
  }'
```

**请求**（指定区域）：
```bash
curl -X POST http://master:8080/api/v1/tasks/task-123456/start \
  -H "Content-Type: application/json" \
  -d '{
    "slave_region": "beijing"
  }'
```

**响应**：
```json
{
  "task_id": "task-123456",
  "message": "Task started successfully",
  "state": "running"
}
```

#### 3. 查询任务详情

```bash
curl http://master:8080/api/v1/tasks/task-123456
```

#### 4. 获取 Slave 请求详情

```bash
curl "http://master:8080/api/details?slave_id=slave-1&status=all&offset=0&limit=100"
```
  shanghai   : 33200 requests, QPS: 1021.54
  guangzhou  : 33300 requests, QPS: 1024.62
```

## 故障处理

### Slave 故障

- Master 通过心跳检测 Slave 健康状态
- Slave 掉线不影响其他 Slave
- 故障节点的任务可选择重新分配

### Master 故障

- Master 定期保存状态
- 重启后从最近状态恢复
- Slave 自动重连 Master

## 自动化测试脚本

为了方便快速搭建本地分布式测试环境，提供了 PowerShell 自动化脚本。

### test-distributed.ps1

自动启动 1 个 Master + N 个 Slave 节点（默认 3 个）。

```powershell
# 一键启动分布式环境
.\test-distributed.ps1
```

**脚本功能**：
1. 清理旧进程
2. 构建项目（`go build`）
3. 启动 Master 节点（gRPC:9090, HTTP:8080）
4. 批量启动 Slave 节点：
   - `slave-1` (gRPC:9091, Realtime:8088, Region:zone-a)
   - `slave-2` (gRPC:9092, Realtime:8089, Region:zone-b)
   - `slave-3` (gRPC:9093, Realtime:8090, Region:zone-c)
5. 自动打开浏览器（http://localhost:8080）

**配置参数**：
```powershell
$SLAVE_COUNT = 3          # Slave 数量（可修改）
$SLAVE_BASE_PORT = 9091  # Slave gRPC 起始端口
$ZONES = @("zone-a", "zone-b", "zone-c", "zone-d", "zone-e")  # 区域列表
```

**使用场景**：
- 本地开发测试
- CI/CD 集成测试
- 快速演示分布式功能

### test-websocket.ps1

自动启动 WebSocket 测试服务器并运行压测。

```powershell
# WebSocket 压测一键测试
.\test-websocket.ps1
```

**脚本功能**：
1. 清理旧进程
2. 构建项目
3. 启动 WebSocket 测试服务器（端口 3000）
4. 提示选择测试场景：
   - 1: 快速测试 (5并发 20请求)
   - 2: 标准测试 (10并发 100请求)
   - 3: 回声测试 (20并发 500请求)
   - 4: 聊天室测试 (50并发 1000请求)
5. 运行压测并生成报告

## 实战示例

### 示例 1：Web 界面操作流程

1. 启动分布式集群：
```bash
# 启动 Master
./go-stress -mode master -grpc-port 9090 -http-port 8080

# 启动多个 Slave
./go-stress -mode slave -master localhost:9090 -slave-id slave-1 -region zone-a
./go-stress -mode slave -master localhost:9090 -slave-id slave-2 -region zone-b
```

2. 访问管理界面：`http://localhost:8080`

3. 创建任务：
   - 点击"创建任务"
   - 上传配置文件或粘贴 JSON 配置
   - 提交后任务状态为"待执行"

4. 启动任务：
   - 在任务列表中点击任务 ID
   - 查看可用的 Slave 节点
   - 点击"启动任务"按钮
   - 任务开始执行，状态变为"运行中"

5. 查看实时数据：
   - 实时 QPS、成功率
   - 各 Slave 的执行情况
   - 请求详情列表

### 示例 2：跨地域压测

模拟全球用户访问：

```bash
# Master（中心机房）
./go-stress -mode master -grpc-port 9090 -http-port 8080

# Slave（美国）
./go-stress -mode slave -master master:9090 -region us-west -slave-id us-slave-1

# Slave（欧洲）
./go-stress -mode slave -master master:9090 -region eu-west -slave-id eu-slave-1

# Slave（亚洲）
./go-stress -mode slave -master master:9090 -region ap-east -slave-id ap-slave-1
```

**创建任务**：
```bash
curl -X POST http://master:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "config_file": "{\"protocol\":\"http\",\"url\":\"https://api.example.com\",\"concurrency\":300,\"requests\":10000}"
  }'
```

**指定区域启动**（只在亚洲节点执行）：
```bash
curl -X POST http://master:8080/api/v1/tasks/{task_id}/start \
  -H "Content-Type: application/json" \
  -d '{"slave_region": "ap-east"}'
```

### 示例 3：大规模容量测试

10 台机器，每台 500 并发，总计 5000 并发：

```bash
# Master
./go-stress -mode master -grpc-port 9090 -http-port 8080

# 10 个 Slave
for i in {1..10}; do
  ssh slave-$i "go-stress -mode slave -master master:9090 -slave-id slave-$i"
done
```

**创建任务**：
```bash
curl -X POST http://master:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "config_file": "{\"protocol\":\"http\",\"url\":\"https://api.example.com\",\"concurrency\":5000,\"requests\":5000000,\"advanced\":{\"ramp_up\":\"120s\"}}"
  }'
```

**启动任务**（使用所有 Slave）：
```bash
curl -X POST http://master:8080/api/v1/tasks/{task_id}/start
```

### 示例 4：渐进式压测

分阶段增加压力：

1. **第一阶段**：创建任务，启动 2 个 Slave
```bash
curl -X POST http://master:8080/api/v1/tasks/{task_id}/start \
  -d '{"slave_ids": ["slave-1", "slave-2"]}'
```

2. **第二阶段**：创建新任务，启动 5 个 Slave
```bash
curl -X POST http://master:8080/api/v1/tasks/{task_id}/start \
  -d '{"slave_ids": ["slave-1", "slave-2", "slave-3", "slave-4", "slave-5"]}'
```

3. **第三阶段**：创建新任务，启动所有 Slave
```bash
curl -X POST http://master:8080/api/v1/tasks/{task_id}/start
```

## 故障排查

### Slave 无法连接 Master

```bash
# 检查网络
telnet master-ip 9090

# 检查防火墙
sudo firewall-cmd --list-ports

# 检查 Master 日志
./go-stress -mode master -grpc-port 9090 -log-level debug
```

### 任务无法启动

1. **检查任务状态**：确保任务状态为 `pending`
2. **检查 Slave 数量**：至少需要 `min-slave-count` 个 Slave 在线
3. **查看 Master 日志**：检查任务分配是否成功
4. **验证配置**：确保配置文件格式正确

### 查询详情接口返回空数据

**原因**：
- 任务还未执行（状态为 pending）
- Slave 未收到任务分配
- 任务刚开始执行，还没有详情数据

**解决方案**：
1. 确认任务已启动（状态为 running 或 completed）
2. 等待任务执行一段时间后再查询
3. 检查 Slave 日志确认任务是否真正执行
4. 验证 `slave_id` 参数是否正确

### 查看调试日志

**Master 日志**：
```bash
./go-stress -mode master -grpc-port 9090 -log-level debug
```

**Slave 日志**：
```bash
./go-stress -mode slave -master master:9090 -log-level debug
```

## 相关文档

- [快速开始](GETTING_STARTED.md) - 基础使用
- [命令行参考](CLI_REFERENCE.md) - 分布式参数
