# 测试服务器和配置说明

## 测试服务器

### 启动测试服务器

```bash
# 进入 testserver 目录
cd testserver

# 运行测试服务器
go run test_server.go
```

服务器将在 `http://localhost:3000` 启动

### API 端点

| 端点 | 方法 | 说明 | 认证 |
|------|------|------|------|
| `/api/login` | POST | 用户登录，返回 token | 否 |
| `/api/user/info` | GET | 获取用户信息 | 是 |
| `/api/user/update` | PUT | 更新用户信息 | 是 |
| `/api/health` | GET | 健康检查 | 否 |

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

### 1. test-simple.yaml - 简单压测

**用途**：独立测试每个 API，无依赖关系

**配置**：

- 并发数：10
- 请求数：1000
- 实时监控端口：8089

**运行**：

```bash
go run main.go -c testserver/test-simple.yaml
```

**特点**：

- 无 API 间依赖
- 适合单接口性能测试
- 使用变量 `{{.RequestID}}` 生成唯一用户名

---

### 2. test-detail.yaml - 依赖链测试

**用途**：测试 API 间的数据传递和依赖关系

**配置**：

- 并发数：1
- 请求数：2
- 实时监控端口：8088

**运行**：

```bash
go run main.go -c testserver/test-detail.yaml
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

### 3. test-chain.yaml - 完整链式测试

**用途**：测试复杂的多步骤业务流程

**配置**：

- 并发数：3
- 请求数：50
- 实时监控端口：8090

**运行**：

```bash
go run main.go -c testserver/test-chain.yaml
```

**特点**：

- 4 步完整流程：登录 → 查询 → 更新 → 验证
- 多重依赖关系
- 数据验证（verify）
- 响应断言

**执行流程**：

```
user_login (登录)
  ↓
fetch_user_info (查询用户信息)
  ↓
update_user_profile (更新用户)
  ↓
verify_update (验证更新)
```

---

### 4. test-signature.yaml - 签名认证测试

**用途**：测试带签名认证的 API 请求

**配置**：

- 并发数：5
- 请求数：100
- 实时监控端口：8088
- **启用签名**：是

**运行**：

```bash
go run main.go -c testserver/test-signature.yaml
```

**签名配置**：

```yaml
signature:
  enabled: true
  header_name: X-Sign              # 签名 header 名称
  timestamp_header: X-Timestamp    # 时间戳 header
  nonce_header: X-Nonce            # 随机数 header
  secret_key: "your-secret-key-123"
  algorithm: sha256                # sha1/sha256/sha512
  include_body: true               # 签名包含 body
  include_query: true              # 签名包含查询参数
  include_headers:                 # 签名包含的 headers
    - Content-Type
  extra:                           # 额外的 headers
    X-App-ID: "test-app"
    X-Version: "1.0.0"
```

**签名生成规则**（默认格式）：

```
METHOD + "\n" +
PATH + "\n" +
TIMESTAMP + "\n" +
NONCE + "\n" +
[HEADERS] + "\n" +  # 可选
[QUERY] + "\n" +    # 可选
[BODY]              # 可选
```

**自定义格式**（可选）：

```yaml
signature:
  format: "{method}\n{path}\n{timestamp}\n{nonce}\n{query}\n{body}"
```

支持的占位符：

- `{method}` - HTTP 方法
- `{path}` - 请求路径
- `{timestamp}` - 时间戳
- `{nonce}` - 随机数
- `{body}` - 请求体
- `{query}` - 查询参数
- `{header.XXX}` - 指定 header

**特点**：

- 自动生成签名并添加到请求 header
- 支持 HMAC-SHA1/SHA256/SHA512
- 灵活的签名格式配置
- 支持额外的认证 headers

---

## 完整测试流程

### 1. 启动测试服务器

```bash
# 终端 1
cd testserver
go run test_server.go
```

### 2. 运行不同的测试

```bash
# 终端 2

# 简单压测
go run main.go -c testserver/test-simple.yaml

# 依赖链测试
go run main.go -c testserver/test-detail.yaml

# 完整链式测试
go run main.go -c testserver/test-chain.yaml

# 签名认证测试
go run main.go -c testserver/test-signature.yaml
```

### 3. 查看实时监控

在浏览器打开对应的实时监控端口：

- 简单测试：<http://localhost:8089>
- 依赖链测试：<http://localhost:8088>
- 链式测试：<http://localhost:8090>
- 签名测试：<http://localhost:8088>

### 4. 查看测试报告

测试完成后，HTML 报告会保存在 `stress-report/` 目录下，按时间戳命名。

---

## 配置说明

### 基础配置

```yaml
protocol: http          # 协议：http/grpc
concurrency: 10         # 并发数
requests: 1000          # 总请求数
timeout: 10s            # 超时时间
host: http://localhost:3000  # 目标服务器
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

---

## 常见问题

### Q1: 签名验证失败怎么办？

确保服务端和客户端使用相同的：

- 签名算法（sha256/sha512）
- 密钥（secret_key）
- 签名格式（format）
- 参与签名的字段（include_body/include_query）

### Q2: 依赖链执行顺序是什么？

按照 `depends_on` 声明的依赖关系，使用拓扑排序确定执行顺序。没有依赖的 API 可以并行执行。

### Q3: 如何调试数据提取？

1. 查看日志输出，会显示提取的变量值
2. 使用较小的并发数和请求数
3. 检查 jsonpath 表达式是否正确

### Q4: 并发和请求数的关系？

- `concurrency`: 同时执行的并发数
- `requests`: 总请求数
- 每个并发会轮流执行 API 列表，直到达到总请求数

例如：`concurrency=10, requests=100`

- 10 个并发同时工作
- 共执行 100 次 API 调用

---

## 性能优化建议

1. **HTTP 长连接**：

```yaml
http:
  keepalive: true
  max_idle_conns: 100
  idle_conn_timeout: 90s
```

1. **合理设置并发数**：
   - 根据服务器性能调整
   - 避免过高导致系统崩溃

2. **使用实时监控**：
   - 观察 QPS、延迟、错误率
   - 及时发现性能瓶颈

3. **分阶段压测**：
   - 先小并发预热
   - 逐步增加并发数
   - 观察系统表现

---

## 扩展测试

### 添加新的 API

1. 在测试服务器添加新端点
2. 在 YAML 中配置新 API
3. 设置依赖关系和数据提取

### 自定义签名算法

修改 `signature` 配置，支持：

- `sha1`
- `sha256`
- `sha512`
- 自定义 format

### 复杂场景测试

组合使用：

- 多步骤依赖
- 数据提取和引用
- 响应验证
- 签名认证

---

## 测试报告

测试完成后会生成：

1. **控制台输出**：实时统计信息
2. **HTML 报告**：详细的可视化报告
3. **JSON 数据**：原始测试数据

报告包含：

- 总请求数、成功率、失败数
- QPS、平均延迟、P95/P99
- 每个 API 的详细统计
- 错误信息和分布

---

## 许可证

Copyright (c) 2025 by kamalyes, All Rights Reserved.
