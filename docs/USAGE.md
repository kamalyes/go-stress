# 📖 使用指南

## 📋 目录

- [快速开始](#-快速开始)
- [命令行参数](#-命令行参数)
- [配置文件](#-配置文件)
- [使用示例](#-使用示例)
- [高级用法](#-高级用法)
- [常见问题](#-常见问题)

---

## 🚀 快速开始

### 安装

```bash
# 从源码构建
git clone https://github.com/kamalyes/go-stress.git
cd go-stress
go build -o go-stress

# 或直接运行
go run main.go -help
```

### 基本使用

```bash
# HTTP 压测
./go-stress -url https://example.com -c 10 -n 100

# 使用配置文件
./go-stress -config config.yaml

# 使用curl命令文件
./go-stress -curl curl.txt -c 100 -n 1000

# 查看帮助
./go-stress -help
```

---

## ⚙️ 命令行参数

### 基础参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-config` | string | - | 配置文件路径（yaml/json） |
| `-curl` | string | - | curl命令文件路径（自动解析） |
| `-protocol` | string | `http` | 协议类型（http/grpc/websocket） |
| `-url` | string | - | 目标 URL（必填） |
| `-c` | uint64 | `1` | 并发数 |
| `-n` | uint64 | `1` | 每个并发的请求数 |
| `-method` | string | `GET` | 请求方法 |
| `-timeout` | duration | `30s` | 请求超时时间 |

### HTTP 参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-http2` | bool | `false` | 使用 HTTP/2 协议 |
| `-keepalive` | bool | `false` | 使用长连接 |
| `-H` | string | - | 请求头（可多次使用） |
| `-data` | string | - | 请求体数据 |

### gRPC 参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-grpc-reflection` | bool | `false` | 使用 gRPC 反射 |
| `-grpc-service` | string | - | gRPC 服务名 |
| `-grpc-method` | string | - | gRPC 方法名 |

### 日志参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-log-level` | string | `info` | 日志级别（debug/info/warn/error） |
| `-log-file` | string | - | 日志文件路径 |
| `-quiet` | bool | `false` | 静默模式（仅错误） |
| `-verbose` | bool | `false` | 详细模式（包含调试信息） |

---

## 📝 配置文件

### YAML 配置示例

```yaml
# config.yaml - HTTP 压测配置
protocol: http
concurrency: 100
requests: 1000
timeout: 10s
url: https://api.example.com/users
method: POST

headers:
  Content-Type: application/json
  Authorization: Bearer your-token-here

body: |
  {
    "name": "test",
    "email": "test@example.com"
  }

# HTTP 配置
http:
  http2: true
  keepalive: true
  follow_redirects: true
  max_conns_per_host: 100

# 高级配置
advanced:
  enable_breaker: true      # 启用熔断
  max_failures: 10          # 最大失败次数
  reset_timeout: 30s        # 熔断恢复时间
  enable_retry: true        # 启用重试
  max_retries: 3            # 最大重试次数
  ramp_up: 10s              # 渐进启动时长

# 响应验证
verify:
  type: status_code         # 验证类型
  rules:
    expected: 200           # 期望状态码
```

### gRPC 配置示例

```yaml
# config-grpc.yaml - gRPC 压测配置
protocol: grpc
concurrency: 50
requests: 500
timeout: 5s
url: localhost:50051

grpc:
  use_reflection: true
  service: proto.UserService
  method: GetUser
  metadata:
    authorization: Bearer token
  tls:
    enabled: false

body: |
  {
    "id": "12345"
  }

advanced:
  enable_breaker: true
  max_failures: 5
  reset_timeout: 20s
```

### JSON 配置示例

```json
{
  "protocol": "http",
  "concurrency": 100,
  "requests": 1000,
  "timeout": "10s",
  "url": "https://api.example.com/users",
  "method": "POST",
  "headers": {
    "Content-Type": "application/json"
  },
  "body": "{\"name\":\"test\"}",
  "http": {
    "http2": true,
    "keepalive": true
  },
  "advanced": {
    "enable_breaker": true,
    "max_failures": 10,
    "reset_timeout": "30s"
  }
}
```

### curl 命令文件示例

将curl命令保存到文件（如 `curl.txt`）：

```bash
curl 'http://localhost:8081/v1/messages/send' \
  -H 'Accept: application/json' \
  -H 'Authorization: Bearer your-token-here' \
  -H 'Content-Type: application/json' \
  --data-raw '{"session_id":"test","content":"{{md5 \"text\"}}-{{unixNano}}"}' \
  --insecure
```

然后使用 `-curl` 参数加载：

```bash
# 自动解析curl命令，设置并发数和请求数
./go-stress -curl curl.txt -c 100 -n 1000

# 会自动提取：
# - URL
# - HTTP方法（POST/GET等）
# - 请求头（-H）
# - 请求体（--data-raw）
# - 支持模板变量（{{md5}}、{{unixNano}}等）
```

**优势**：

- 🚀 从浏览器或Postman直接复制curl命令
- 📝 自动解析所有参数（URL、headers、body等）
- 🔧 支持模板变量动态生成数据
- ⚡ 快速开始压测，无需手动配置

---

## 💡 使用示例

### 1. 使用curl文件进行压测

从浏览器开发者工具复制curl命令，保存为 `api-test.txt`：

```bash
curl 'https://api.example.com/users' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer token123' \
  --data-raw '{"name":"user-{{randomInt 1 1000}}","email":"test{{seq}}@example.com"}'
```

执行压测：

```bash
# 100并发，每个1000请求
./go-stress -curl api-test.txt -c 100 -n 1000
```

**输出示例**：

```
📄 解析curl文件: api-test.txt
✅ 解析成功
  URL: https://api.example.com/users
  方法: POST
  请求头: 2个
  请求体: 支持模板变量

🚀 开始压测...
```

---

### 2. 基础 HTTP GET 请求

```bash
# 10 个并发，每个发送 100 个请求
./go-stress -url https://api.example.com/health -c 10 -n 100
```

**输出示例**：

```
╔══════════════════════════════════════════════════════════╗
║     ⚡ Go Stress Testing Tool ⚡                         ║
╚══════════════════════════════════════════════════════════╝

🚀 开始压测...
📊 协议: http
🔢 并发数: 10
📈 每并发请求数: 100
⏱️  超时时间: 30s

⏳ 进度: 1000/1000 (100.00%) | 耗时: 5s | 预计剩余: 0s

✅ 压测完成!

📊 压测报告
═══════════════════════════════════════════════════════════
总请求数:     1000
成功请求:     1000
失败请求:     0
成功率:       100.00%
QPS:          200.00
═══════════════════════════════════════════════════════════
最小延迟:     10ms
最大延迟:     50ms
平均延迟:     25ms
P50 延迟:     24ms
P95 延迟:     45ms
P99 延迟:     49ms
═══════════════════════════════════════════════════════════
```

---

### 3. POST 请求带请求体

```bash
# 使用 JSON 数据
./go-stress \
  -url https://api.example.com/users \
  -method POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -data '{"name":"test","email":"test@example.com"}' \
  -c 20 \
  -n 50
```

---

### 4. HTTP/2 长连接压测

```bash
./go-stress \
  -url https://api.example.com/api \
  -http2 \
  -keepalive \
  -c 50 \
  -n 200 \
  -timeout 5s
```

---

### 5. gRPC 压测（使用反射）

```bash
./go-stress \
  -protocol grpc \
  -url localhost:50051 \
  -grpc-reflection \
  -grpc-service proto.UserService \
  -grpc-method GetUser \
  -data '{"id":"12345"}' \
  -c 10 \
  -n 100
```

---

### 6. 使用配置文件

```bash
# 从 YAML 配置文件加载
./go-stress -config config.yaml

# 从 JSON 配置文件加载
./go-stress -config config.json
```

**config.yaml**:

```yaml
protocol: http
url: https://api.example.com/users
method: GET
concurrency: 100
requests: 1000
timeout: 10s

headers:
  Accept: application/json
  User-Agent: go-stress/1.0

advanced:
  enable_breaker: true
  max_failures: 10
  reset_timeout: 30s
  ramp_up: 5s
```

---

### 7. 启用熔断和重试

```bash
./go-stress \
  -config advanced-config.yaml \
  -verbose
```

**advanced-config.yaml**:

```yaml
protocol: http
url: https://api.example.com/unstable
concurrency: 50
requests: 500

advanced:
  enable_breaker: true      # 启用熔断器
  max_failures: 5           # 5次失败后熔断
  reset_timeout: 30s        # 30秒后尝试恢复
  
  enable_retry: true        # 启用重试
  max_retries: 3            # 最多重试3次
  retry_delay: 1s           # 重试延迟
  
  ramp_up: 10s              # 10秒内渐进启动
```

---

### 8. 响应验证

```bash
./go-stress -config verify-config.yaml
```

**verify-config.yaml**:

```yaml
protocol: http
url: https://api.example.com/users/1
method: GET
concurrency: 10
requests: 100

# 验证状态码
verify:
  type: status_code
  rules:
    expected: 200

# 或验证 JSON 字段
# verify:
#   type: json
#   rules:
#     path: "data.status"
#     expected: "success"

# 或验证响应包含特定内容
# verify:
#   type: contains
#   rules:
#     content: "success"
```

---

### 9. 日志配置

```bash
# 详细模式（调试）
./go-stress -url https://example.com -c 10 -n 100 -verbose

# 静默模式（仅错误）
./go-stress -url https://example.com -c 10 -n 100 -quiet

# 输出到文件
./go-stress -url https://example.com -c 10 -n 100 -log-file stress.log

# 设置日志级别
./go-stress -url https://example.com -c 10 -n 100 -log-level debug
```

---

## 🎯 高级用法

### 1. 渐进式启动（Ramp-up）

平滑增加负载，避免突发流量：

```yaml
protocol: http
url: https://api.example.com
concurrency: 100
requests: 1000

advanced:
  ramp_up: 30s  # 在30秒内逐步启动100个并发
```

**效果**：

- Worker 0: 立即启动
- Worker 1: 0.3秒后启动
- Worker 2: 0.6秒后启动
- ...
- Worker 99: 29.7秒后启动

---

### 2. 连接池优化

```yaml
http:
  keepalive: true           # 启用长连接
  max_conns_per_host: 100   # 每个主机最大连接数
```

**优势**：

- 减少连接创建开销
- 提高吞吐量
- 降低延迟

---

### 3. 动态变量替换

```yaml
protocol: http
url: https://api.example.com/users/{{user_id}}
method: GET

variables:
  user_id: "12345"
```

---

### 4. 多场景测试

创建多个配置文件，依次执行：

```bash
# 场景1：低负载
./go-stress -config scenario1-low.yaml

# 场景2：中负载
./go-stress -config scenario2-medium.yaml

# 场景3：高负载
./go-stress -config scenario3-high.yaml
```

---

### 5. 信号中断

支持 `Ctrl+C` 优雅停止：

```bash
./go-stress -url https://example.com -c 100 -n 10000

# 按 Ctrl+C 中断
⚠️  收到中断信号，正在停止...
✅ 压测完成!
```

---

### 6. 报告保存

压测完成后自动保存 JSON 报告：

```bash
./go-stress -url https://example.com -c 10 -n 100

# 输出
💾 报告已保存: stress-report-1703923200.json
```

**报告格式**：

```json
{
  "total_requests": 1000,
  "success_requests": 1000,
  "failed_requests": 0,
  "success_rate": 100.0,
  "qps": 200.0,
  "avg_duration": "25ms",
  "min_duration": "10ms",
  "max_duration": "50ms",
  "p50_duration": "24ms",
  "p95_duration": "45ms",
  "p99_duration": "49ms",
  "total_duration": "5s"
}
```

---

## ❓ 常见问题

### Q1: 如何提高 QPS？

**A**: 增加并发数和优化配置：

```yaml
concurrency: 200          # 增加并发
requests: 5000

http:
  keepalive: true         # 启用长连接
  http2: true             # 使用 HTTP/2
  max_conns_per_host: 200 # 增加连接池

advanced:
  ramp_up: 0s             # 关闭渐进启动
```

---

### Q2: 熔断器什么时候触发？

**A**: 当失败次数达到 `max_failures` 时触发：

```yaml
advanced:
  enable_breaker: true
  max_failures: 10        # 10次失败后熔断
  reset_timeout: 30s      # 30秒后尝试恢复
```

**熔断状态**：

- **Closed（关闭）**: 正常请求
- **Open（打开）**: 直接拒绝请求
- **Half-Open（半开）**: 尝试恢复

---

### Q3: 如何处理高延迟？

**A**: 调整超时和重试策略：

```yaml
timeout: 30s              # 增加超时时间

advanced:
  enable_retry: true
  max_retries: 3          # 重试3次
  retry_delay: 2s         # 重试延迟2秒
```

---

### Q4: 为什么成功率很低？

**可能原因**：

1. 服务器负载过高
2. 网络问题
3. 超时设置太短
4. 验证规则不正确

**解决方法**：

```bash
# 增加超时时间
-timeout 60s

# 降低并发数
-c 10

# 启用详细日志查看错误
-verbose

# 检查验证规则
verify:
  type: status_code
  rules:
    expected: 200  # 确保期望值正确
```

---

### Q5: 如何压测需要认证的 API？

**A**: 添加认证头：

```bash
# Bearer Token
./go-stress \
  -url https://api.example.com/protected \
  -H "Authorization: Bearer your-token-here" \
  -c 10 -n 100

# Basic Auth
./go-stress \
  -url https://api.example.com/protected \
  -H "Authorization: Basic dXNlcjpwYXNz" \
  -c 10 -n 100
```

**或在配置文件中**：

```yaml
headers:
  Authorization: Bearer your-token-here
```

---

### Q6: 如何压测 HTTPS？

**A**: 直接使用 HTTPS URL：

```bash
./go-stress -url https://secure.example.com -c 10 -n 100
```

**TLS 配置**（gRPC）：

```yaml
protocol: grpc
url: secure.example.com:50051

grpc:
  tls:
    enabled: true
    cert_file: client.crt
    key_file: client.key
    ca_file: ca.crt
```

---

### Q7: 如何模拟真实用户行为？

**A**: 使用渐进启动和随机延迟：

```yaml
advanced:
  ramp_up: 60s            # 60秒内逐步增加负载
  think_time: 1s          # 请求间隔（思考时间）
  think_time_variance: 0.5 # 50%的随机变化
```

---

### Q8: 内存占用过高怎么办？

**A**: 优化配置：

```yaml
# 减少并发数
concurrency: 50

# 启用连接池复用
http:
  keepalive: true
  max_conns_per_host: 50

# 分批执行
# 方案1: 10个并发 × 1000次
# 方案2: 100个并发 × 100次（内存更高）
```

---

## 📚 参考资源

- [架构设计文档](ARCHITECTURE.md) - 详细的架构设计
- [问题反馈](https://github.com/kamalyes/go-stress/issues) - 报告 bug
- [讨论区](https://github.com/kamalyes/go-stress/discussions) - 技术交流

---

## 📄 许可证

MIT License - 详见 [LICENSE](../LICENSE)

## 👨‍💻 作者

Kamal Yang ([@kamalyes](https://github.com/kamalyes))
