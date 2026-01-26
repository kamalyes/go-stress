# 配置文件详解

支持 YAML 和 JSON 格式配置文件

## 基础配置

```yaml
# 协议和并发
protocol: http          # http, grpc, websocket
concurrency: 100        # 并发数
requests: 10000         # 每个并发的请求数
duration: 5m            # 持续时间（与requests二选一，优先使用duration）
timeout: 10s            # 单个请求超时

# 目标配置
url: https://api.example.com/users
method: POST
headers:
  Content-Type: application/json
  Authorization: Bearer {{env "API_TOKEN"}}
body: |
  {
    "username": "user_{{randomString 8}}",
    "timestamp": {{unix}}
  }
```

## HTTP 配置

```yaml
http:
  http2: true                # 启用 HTTP/2
  keepalive: true            # 启用 Keep-Alive
  follow_redirects: true     # 跟随重定向
  max_conns_per_host: 100    # 每个 host 的最大连接数
```

## gRPC 配置

```yaml
grpc:
  use_reflection: true       # 使用 gRPC 反射
  service: pb.UserService    # 服务名
  method: GetUser            # 方法名
  proto_file: user.proto     # proto 文件路径（不使用反射时）
  metadata:
    key: value
```

## 高级配置

```yaml
advanced:
  # 熔断器
  enable_breaker: true       # 启用熔断
  max_failures: 50           # 最大失败次数
  reset_timeout: 30s         # 熔断器重置超时
  
  # 重试
  enable_retry: true         # 启用重试
  max_retries: 3             # 最大重试次数
  retry_interval: 1s         # 重试间隔
  
  # 渐进启动
  ramp_up: 60s               # 渐进启动时间
  
  # 实时报告
  realtime_port: 8088        # 实时报告服务器端口
```

## 验证配置

```yaml
# 单个验证
verify:
  type: status_code          # 验证类型
  operator: eq               # 操作符
  expect: 200                # 期望值
  description: "检查状态码"
```

支持的验证类型：

- `status_code` - HTTP 状态码
- `jsonpath` - JSON 路径
- `contains` - 包含字符串
- `regex` - 正则表达式
- `json_valid` - JSON 格式验证

操作符：

- `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `contains`, `regex`

## 多 API 配置

```yaml
protocol: http
concurrency: 50
requests: 1000
host: https://api.example.com

apis:
  # API 1: 登录
  - name: login
    path: /auth/login
    method: POST
    body: '{"username":"test","password":"123456"}'
    extractors:
      - name: token
        type: jsonpath
        jsonpath: "$.data.token"
    verify:
      - type: jsonpath
        jsonpath: "$.code"
        operator: eq
        expect: 0

  # API 2: 获取用户信息（依赖login）
  - name: get_profile
    path: /user/profile
    method: GET
    depends_on: [login]
    headers:
      Authorization: "Bearer {{token}}"
    verify:
      - type: status_code
        expect: 200

  # API 3: 更新信息（依赖get_profile）
  - name: update_profile
    path: /user/profile
    method: PUT
    depends_on: [get_profile]
    headers:
      Authorization: "Bearer {{token}}"
    body: |
      {
        "nickname": "{{randomString 10}}",
        "updated_at": {{timestamp}}
      }
```

### API 配置选项

```yaml
apis:
  - name: api_name           # API 名称
    path: /api/path          # 路径
    url: https://...         # 完整 URL（优先级高于host+path）
    method: POST             # HTTP 方法
    headers:                 # 请求头（与全局合并）
      Custom-Header: value
    body: "request body"     # 请求体
    weight: 1                # 权重（默认1）
    repeat: 1                # 重复次数（默认1）
    depends_on: [api1]       # 依赖的 API
    extractors:              # 数据提取器
      - name: var_name
        type: jsonpath
        jsonpath: "$.path"
    verify:                  # 验证规则（支持多个）
      - type: status_code
        expect: 200
```

## 数据提取器

支持从HTTP请求和响应中提取数据、应用转换，并存储为变量供后续使用。

### 基础提取

#### 从响应提取（默认）

```yaml
extractors:
  - name: token
    source: response  # response(默认) | request
    type: JSONPATH    # JSONPATH(默认) | REGEX | HEADER | EXPRESSION
    jsonpath: $.data.token
    default: ""
```

#### 从请求提取

```yaml
extractors:
  - name: session_id
    source: request
    type: JSONPATH
    jsonpath: $.session_id
```

### 提取器类型

#### JSONPath 提取

```yaml
extractors:
  - name: user_id
    type: JSONPATH
    jsonpath: "$.data.id"
    default: "0"
  
  # 从请求提取
  - name: request_session
    source: request
    type: JSONPATH
    jsonpath: "$.session_id"
```

#### 正则表达式提取

```yaml
extractors:
  - name: session_id
    type: REGEX
    regex: 'session=([a-f0-9]+)'
    source: response  # 可从request或response提取
```

#### 响应头提取

```yaml
extractors:
  - name: csrf_token
    type: HEADER
    header: X-CSRF-Token
    source: response
  
  # 从请求头提取
  - name: auth_token
    source: request
    type: HEADER
    header: Authorization
```

#### 表达式提取（组合变量）

```yaml
extractors:
  - name: full_name
    type: EXPRESSION
    expression: "{{.first_name}} {{.last_name}}"
```

### 数据转换

提取后可应用转换管道处理数据：

#### 字符串转换

```yaml
extractors:
  # 单个转换
  - name: username
    jsonpath: $.username
    transforms:
      - function: trim      # 去除空格
  
  # 链式转换
  - name: email
    jsonpath: $.email
    transforms:
      - function: trim
      - function: lower     # 转小写
  
  # 转大写
  - name: status
    jsonpath: $.status
    transforms:
      - function: upper
```

#### 加密和编码

```yaml
extractors:
  # MD5加密
  - name: password_hash
    jsonpath: $.password
    transforms:
      - function: md5
  
  # SHA256
  - name: secure_hash
    jsonpath: $.data
    transforms:
      - function: sha256
  
  # Base64编码
  - name: encoded_data
    jsonpath: $.raw_data
    transforms:
      - function: base64
```

#### 模板转换

```yaml
extractors:
  # 简单模板
  - name: formatted_id
    jsonpath: $.id
    transforms:
      - template: "USER_{{.value}}"
  
  # 带参数的模板
  - name: prefixed_name
    jsonpath: $.name
    transforms:
      - template: "{{.arg0}}_{{.value}}_{{.arg1}}"
        args: ["PREFIX", "SUFFIX"]
```

#### 复杂转换链

```yaml
extractors:
  - name: processed_email
    jsonpath: $.user.email
    transforms:
      - function: trim       # 1. 去空格
      - function: lower      # 2. 转小写
      - function: md5        # 3. MD5加密
```

### 支持的转换函数

#### 字符串处理

- `upper` - 转大写
- `lower` - 转小写
- `trim` - 去除空格
- `title` - 首字母大写
- `trimPrefix` - 去除前缀
- `trimSuffix` - 去除后缀
- `replace` - 替换字符串
- `reverse` - 反转字符串

#### 加密哈希

- `md5` - MD5哈希
- `sha1` - SHA1哈希
- `sha256` - SHA256哈希

#### 编码解码

- `base64` - Base64编码
- `base64Decode` - Base64解码
- `urlEncode` - URL编码
- `urlDecode` - URL解码
- `hexEncode` - 十六进制编码
- `hexDecode` - 十六进制解码

### 完整示例

```yaml
apis:
  - name: login
    path: /api/login
    method: POST
    body: |
      {
        "username": "{{.username}}",
        "password": "{{.password}}"
      }
    
    extractors:
      # 提取并转换用户名
      - name: normalized_username
        source: request
        jsonpath: $.username
        transforms:
          - function: trim
          - function: lower
      
      # 提取token
      - name: access_token
        source: response
        jsonpath: $.data.token
      
      # 提取并加密session
      - name: session_hash
        source: response
        jsonpath: $.data.session_id
        transforms:
          - function: md5
      
      # 组合多个字段
      - name: user_info
        type: EXPRESSION
        expression: "{{.user_id}}_{{.username}}"
```

## 完整示例

### 示例 1：基础 HTTP 压测

```yaml
protocol: http
concurrency: 100
requests: 10000
timeout: 10s

url: https://api.example.com/api
method: GET

http:
  http2: true
  keepalive: true

advanced:
  ramp_up: 30s
  realtime_port: 8088
```

### 示例 2：带验证的 POST 请求

```yaml
protocol: http
concurrency: 50
requests: 5000
timeout: 15s

url: https://api.example.com/users
method: POST
headers:
  Content-Type: application/json
body: |
  {
    "username": "user_{{randomString 8}}",
    "email": "{{randomEmail}}",
    "timestamp": {{unix}}
  }

verify:
  - type: status_code
    expect: 201
  - type: jsonpath
    jsonpath: "$.code"
    operator: eq
    expect: 0

advanced:
  enable_breaker: true
  max_failures: 100
  enable_retry: true
  max_retries: 3
```

### 示例 3：完整业务流程

```yaml
protocol: http
concurrency: 50
requests: 1000
host: https://api.example.com

headers:
  User-Agent: go-stress/1.0

apis:
  - name: login
    path: /auth/login
    method: POST
    body: '{"username":"test","password":"123456"}'
    extractors:
      - name: access_token
        type: jsonpath
        jsonpath: "$.data.access_token"
      - name: user_id
        type: jsonpath
        jsonpath: "$.data.user_id"
    verify:
      - type: status_code
        expect: 200
      - type: jsonpath
        jsonpath: "$.code"
        operator: eq
        expect: 0

  - name: get_user_info
    path: /users/{{user_id}}
    method: GET
    depends_on: [login]
    headers:
      Authorization: "Bearer {{access_token}}"
    verify:
      - type: status_code
        expect: 200

  - name: update_user
    path: /users/{{user_id}}
    method: PUT
    depends_on: [get_user_info]
    headers:
      Authorization: "Bearer {{access_token}}"
    body: |
      {
        "nickname": "User_{{randomString 6}}",
        "updated_at": "{{date \"2006-01-02 15:04:05\"}}"
      }

advanced:
  enable_breaker: true
  max_failures: 50
  enable_retry: true
  max_retries: 3
  ramp_up: 30s
  realtime_port: 8088
```

## 相关文档

- [命令行参考](CLI_REFERENCE.md) - 命令行参数
- [变量和参数化](VARIABLES.md) - 20+ 模板函数详解
- [快速开始](GETTING_STARTED.md) - 基础使用
