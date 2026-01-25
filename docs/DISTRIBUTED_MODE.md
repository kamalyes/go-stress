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
  -slave-id slave-bj-01

# 机器 2
./go-stress -mode slave \
  -master 192.168.1.100:9090 \
  -region shanghai \
  -slave-id slave-sh-01

# 机器 3
./go-stress -mode slave \
  -master 192.168.1.100:9090 \
  -region guangzhou \
  -slave-id slave-gz-01
```

**参数说明**：
- `-mode slave`：Slave 模式
- `-master`：Master 地址（IP:端口）
- `-region`：区域标签（可选）
- `-slave-id`：节点 ID（可选，自动生成）

### 3. 查看状态

访问 Master 管理界面：

```
http://master-ip:8080
```

## 工作流程

1. **Slave 注册**：Slave 连接 Master 并注册
2. **配置分发**：Master 分发任务配置给 Slave
3. **执行压测**：所有 Slave 并行执行压测
4. **实时上报**：Slave 定期向 Master 上报统计
5. **汇总报告**：Master 汇总所有 Slave 的数据并生成报告

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

## 故障处理

### Slave 故障

- Master 通过心跳检测 Slave 健康状态
- Slave 掉线不影响其他 Slave
- 故障节点的任务可选择重新分配

### Master 故障

- Master 定期保存状态
- 重启后从最近状态恢复
- Slave 自动重连 Master

## 实战示例

### 示例 1：跨地域压测

模拟全球用户访问：

```bash
# Master（中心机房）
./go-stress -mode master -config config.yaml -grpc-port 9090

# Slave（美国）
./go-stress -mode slave -master master:9090 -region us-west

# Slave（欧洲）
./go-stress -mode slave -master master:9090 -region eu-west

# Slave（亚洲）
./go-stress -mode slave -master master:9090 -region ap-east
```

### 示例 2：大规模容量测试

10 台机器，每台 500 并发，总计 5000 并发：

```yaml
# config.yaml
protocol: http
concurrency: 5000
requests: 5000000
url: https://api.example.com/api

advanced:
  ramp_up: 120s
  enable_breaker: true
```

```bash
# Master
./go-stress -mode master -config config.yaml -grpc-port 9090

# 10 个 Slave
for i in {1..10}; do
  ssh slave-$i "go-stress -mode slave -master master:9090 -slave-id slave-$i"
done
```

## 故障排查

### Slave 无法连接 Master

```bash
# 检查网络
telnet master-ip 9090

# 检查防火墙
sudo firewall-cmd --list-ports

# 检查 Master 日志
./go-stress -mode master -config config.yaml -log-level debug
```

## 相关文档

- [快速开始](GETTING_STARTED.md) - 基础使用
- [命令行参考](CLI_REFERENCE.md) - 分布式参数
