# 测试服务器和配置说明

## 测试服务器

### 启动测试服务器

```bash
# 在项目根目录
go run ./testserver
```

服务器将在 `http://localhost:3000` 启动

### 📡 WebSocket 端点

| 端点 | 协议 | 说明 | 配置文件 |
|------|------|------|----------|
| `/ws` | WebSocket | 通用服务 (ping/echo/info) | `websocket-test.yaml` |
| `/ws/echo` | WebSocket | 回声服务器 | `websocket-echo.yaml` |
| `/ws/chat` | WebSocket | 聊天室模拟 | `websocket-chat.yaml` |

#### WebSocket 压测示例

```bash
# 1️⃣ 启动测试服务器（新终端）
go run ./testserver

# 2️⃣ 运行压测（另一个终端，在项目根目录）
# 快速测试（5并发 20请求）
go-stress -config testserver/websocket-quick.yaml

# 通用 WebSocket 服务（10并发 100请求）
go-stress -config testserver/websocket-test.yaml

# 回声服务（20并发 500请求）
go-stress -config testserver/websocket-echo.yaml

# 聊天室压测（50并发 1000请求）
go-stress -config testserver/websocket-chat.yaml

# 命令行方式（-c 是并发数，-n 是总请求数）
go-stress -protocol websocket -url ws://localhost:3000/ws \
  -body '{"action":"ping","message_id":1}' -c 10 -n 100
```

> ⚠️ **注意**：加载配置文件用 `-config` 参数。`-c` 是并发数（整数），
> 写成 `-c testserver/xxx.yaml` 会解析失败。

**💡 提示**: 确保测试服务器在 `http://localhost:3000` 运行

### 🔗 HTTP API 端点

| 端点 | 方法 | 说明 | 认证 |
|------|------|------|------|
| `/api/login` | POST | 用户登录，返回 token | 否 |
| `/api/user/info` | GET | 获取用户信息 | 是 |
| `/api/user/update` | PUT | 更新用户信息 | 是 |
| `/api/health` | GET | 健康检查 | 否 |
| `/v1/tickets` | POST | 创建工单 | 否 |
| `/v1/messages/send` | POST | 发送工单消息 | 否 |

### API 详细说明

#### 1. 登录接口

```bash
curl -X POST http://localhost:3000/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"pass123"}'
```

响应：

```json
{
  "success": true,
  "token": "uuid-token",
  "user_id": "uuid-user-id",
  "message": "登录成功"
}
```

#### 2. 获取用户信息

```bash
curl -X GET http://localhost:3000/api/user/info \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "X-Session-ID: YOUR_SESSION"
```

响应：

```json
{
  "user_id": "uuid",
  "username": "test_user",
  "email": "test@example.com",
  "role": "admin"
}
```

#### 3. 更新用户信息

```bash
curl -X PUT http://localhost:3000/api/user/update \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"new@example.com","role":"super_admin"}'
```

响应：

```json
{
  "success": true,
  "message": "更新成功",
  "data": {
    "user_id": "uuid",
    "email": "new@example.com",
    "role": "super_admin"
  }
}
```

#### 4. 健康检查

```bash
curl http://localhost:3000/api/health
```

响应：

```json
{
  "status": "healthy",
  "timestamp": 1234567890,
  "service": "test-api"
}
```

---

## 测试配置文件

> 以下均为 `testserver/` 目录下**实际存在**的配置文件。

### 1. test-detail.yaml - 验证器全量测试

**用途**：覆盖全部 18 种验证类型（STATUS_CODE / JSONPATH / HEADER / JSON_VALID 等），走完整登录流程

**配置**：

- 并发数：1
- 请求数：1
- 协议：http

**运行**（在项目根目录）：

```bash
go-stress -config testserver/test-detail.yaml
```

**特点**：

- 登录 → 获取信息 → 更新信息的完整流程
- 数据提取器（extractors）提取响应数据
- 使用 `{{.api_name.variable}}` 引用提取的数据
- 支持 `depends_on` 声明依赖关系

**数据流**：

```
login
  ├─ 提取: token, user_id, session_id
  │
  ├─→ get_user_info
  │     └─ 使用: token, session_id
  │
  └─→ update_user
        └─ 使用: token
```

---

### 2. ticket-and-send.yaml - 依赖链压测

**用途**：模拟"创建工单 → 发送消息"业务链路，测试 API 间数据传递

**配置**：

- 并发数：100
- 请求数：100
- 协议：http

**运行**（在项目根目录）：

```bash
go-stress -config testserver/ticket-and-send.yaml
```

**特点**：

- `create_ticket` 提取 `ticket_id`、`session_id` 等
- `send_message` 通过 `{{.create_ticket.ticket_id}}` 引用上一步数据
- 适合验证提取器/依赖链在压力下的正确性

---

### 3. websocket-*.yaml - WebSocket 压测

| 文件 | 并发 | 请求数 | 端点 |
|------|------|--------|------|
| `websocket-quick.yaml` | 5 | 20 | `/ws` |
| `websocket-test.yaml` | 10 | 100 | `/ws` |
| `websocket-echo.yaml` | 20 | 500 | `/ws/echo` |
| `websocket-chat.yaml` | 50 | 1000 | `/ws/chat` |

**运行**（在项目根目录）：

```bash
go-stress -config testserver/websocket-chat.yaml
```

---

### 4. example.curl.txt - curl 导入示例

`example.curl.txt` 是一个 curl 命令文件，可直接导入压测：

```bash
go-stress -curl @testserver/example.curl.txt -c 10 -n 100
```

---

## 完整测试流程

### 1. 启动测试服务器

```bash
# 终端 1
go run ./testserver
```

### 2. 运行不同的测试

```bash
# 终端 2（项目根目录）

# 验证器测试
go-stress -config testserver/test-detail.yaml

# 依赖链压测
go-stress -config testserver/ticket-and-send.yaml

# WebSocket 压测
go-stress -config testserver/websocket-chat.yaml
```

### 3. 查看实时监控

运行压测时会输出实时监控地址（默认端口 8088，配置文件中的
`realtime_port` 可修改），在浏览器打开即可看到实时 QPS / 延迟 / 状态码分布。

### 4. 查看测试报告

测试完成后，HTML 报告保存在 **`stress-report/<Unix时间戳>/`** 目录下：

```
stress-report/
└── 1793643781/          # Unix 时间戳目录
    ├── index.html       # 可视化报告（浏览器打开）
    ├── index.json       # 原始数据（含每个 API 的明细）
    ├── report.css
    ├── report.js
    └── report_actions.js
```

---

## 📊 正常测试报告解读（以 ticket-and-send 为例）

以下是一次**真实运行** `ticket-and-send.yaml` 的报告数据
（来自 `testserver/stress-report/1793643781/index.json`）。

### 顶层指标

```json
{
  "total_requests":  649,
  "success_requests": 649,
  "failed_requests":   0,
  "success_rate":      100,
  "qps":               36.99,
  "total_time_ms":     17543.8,
  "total_size":        374473,
  "status_codes":      { "200": 647 },
  "avg_latency":       238.443,
  "min_latency":       185.819,
  "max_latency":       1103.873,
  "p50_latency":       210.746,
  "p90_latency":       293.06,
  "p95_latency":       309.667,
  "p99_latency":       915.221
}
```

### 指标逐项解读

| 指标 | 本次值 | 怎么看 |
|------|--------|--------|
| `success_rate` | 100% | 正常压测的首要门槛；< 99% 时先看 `errors` 字段 |
| `total_requests` | 649 | 100 并发 × (创建工单+发消息) 累计到总量截止 |
| `qps` | 36.99 | 吞吐量 = 总请求 / 总耗时（17.5s）。本例服务端有 200ms 级延迟，属正常水位 |
| `p50_latency` | 210.7ms | 一半请求快于该值，"大多数用户"的体感 |
| `p90 / p95` | 293 / 310ms | 尾部主流区间，判断是否有慢查询的依据 |
| `p99_latency` | 915.2ms | 长尾。`max(1103) ≈ p99` 说明有极少量异常慢请求，但未影响整体成功率 |
| `status_codes` | `{"200": 647}` | 649 次成功、647 个 200：有 2 次成功请求未计入状态码分布（边界现象，不影响成功率统计） |

**判定标准参考**：
- 成功率 100% + P99 < 1s → 服务健康
- P50 正常但 P99 数倍于 P50 → 存在长尾（GC、锁竞争、慢 SQL）
- QPS 上不去 + 延迟正常 → 并发数受限或客户端到瓶颈

### 失败案例对比（DNS 解析失败）

`stress-report/1793643590/index.json` 是一次**失败**压测，教你识别：

```json
{
  "total_requests":   1000,
  "success_requests":    0,
  "failed_requests":  1000,
  "success_rate":        0,
  "errors": {
    "Get \"https://api.example.com\": dial tcp: lookup api.example.com: no such host": 999
  }
}
```

**解读**：`errors` 字段直接给出根因 —— 域名解析失败（目标地址不可达）。
遇到 0% 成功率时**先看 `errors` 再看延迟**，99% 的 0% 成功率都是环境问题
（DNS/防火墙/端口未开放），不是性能问题。

---

## 配置说明

### 基础配置

```yaml
protocol: http          # 协议：http/grpc/websocket
concurrency: 10         # 并发数
requests: 1000          # 总请求数
timeout: 10s            # 超时时间
host: http://localhost:3000  # 目标服务器（API 的 path 会拼接到 host 后）
```

### Headers 配置

```yaml
headers:
  Content-Type: application/json
  User-Agent: my-test
  Authorization: Bearer token
```

### 数据提取器（Extractors）

从响应中提取数据供后续请求使用：

```yaml
extractors:
  # JSON 路径提取
  - name: token
    type: jsonpath
    jsonpath: $.token
    default: ""

  # Header 提取
  - name: session
    type: header
    header: X-Session-ID
    default: ""
```

### 依赖关系（Depends On）

声明 API 的执行顺序：

```yaml
apis:
  - name: api1
    # ...

  - name: api2
    depends_on:
      - api1  # api2 在 api1 之后执行
```

### 数据引用

使用模板语法引用提取的数据：

```yaml
headers:
  Authorization: "Bearer {{.login.token}}"

body: '{"user_id":"{{.login.user_id}}"}'
```

### 响应验证（Verify）

验证响应是否符合预期：

```yaml
verify:
  # 状态码验证
  - type: status
    expect: 200

  # JSON 字段验证
  - type: jsonpath
    jsonpath: $.success
    expect: true

  # Header 验证
  - type: header
    header: Content-Type
    expect: "application/json"
```

完整验证类型清单见 `testserver/test-detail.yaml`（覆盖全部验证器）。

---

## 常见问题

### Q1: 依赖链执行顺序是什么？

按照 `depends_on` 声明的依赖关系，使用拓扑排序确定执行顺序；没有依赖的 API 可以并行执行。

### Q2: 如何调试数据提取？

1. 查看日志输出，会显示提取的变量值
2. 使用较小的并发数和请求数（如 `test-detail.yaml` 的 1 并发 1 请求）
3. 检查 jsonpath 表达式是否正确

### Q3: 并发和请求数的关系？

- `concurrency`: 同时执行的并发数（`-c`）
- `requests`: 总请求数（`-n`）
- 每个并发会轮流执行 API 列表，直到达到总请求数

例如：`concurrency=10, requests=100`

- 10 个并发同时工作
- 共执行 100 次 API 调用

### Q4: `-c` 和 `-config` 是什么关系？

| 参数 | 含义 | 示例 |
|------|------|------|
| `-c` | 并发数（整数） | `-c 10` |
| `-n` | 总请求数（整数） | `-n 100` |
| `-config` | YAML 配置文件路径 | `-config testserver/websocket-chat.yaml` |
| `-curl` | curl 命令文件 | `-curl @testserver/example.curl.txt` |

配置文件里已有 `concurrency`/`requests` 时，命令行的 `-c`/`-n` 会覆盖它们。

---

## 性能优化建议

1. **HTTP 长连接**：

```yaml
http:
  keepalive: true
  max_idle_conns: 100
  idle_conn_timeout: 90s
```

2. **合理设置并发数**：
   - 根据服务器性能调整
   - 避免过高导致系统崩溃

3. **使用实时监控**：
   - 观察 QPS、延迟、错误率
   - 及时发现性能瓶颈

4. **分阶段压测**：
   - 先小并发预热
   - 逐步增加并发数
   - 观察系统表现

---

## 扩展测试

### 添加新的 API

1. 在测试服务器添加新端点（`test_server.go` 中的 `http.HandleFunc`）
2. 在 YAML 中配置新 API
3. 设置依赖关系和数据提取

### 复杂场景测试

组合使用：

- 多步骤依赖（`ticket-and-send.yaml`）
- 数据提取和引用（`{{.api_name.variable}}`）
- 响应验证（`test-detail.yaml` 全量验证器）

---

## 测试报告

测试完成后会生成：

1. **控制台输出**：实时统计信息
2. **HTML 报告**：`stress-report/<时间戳>/index.html`
3. **JSON 数据**：同目录 `index.json`

报告包含：

- 总请求数、成功率、失败数
- QPS、平均延迟、P50/P90/P95/P99
- 每个 API 的详细统计
- 错误信息和分布

---

## 许可证

Copyright (c) 2025 by kamalyes, All Rights Reserved.
