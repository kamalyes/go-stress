# 命令行参数参考

完整的命令行参数说明

## 基础参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-config` | string | - | 配置文件路径（yaml/json） |
| `-curl` | string | - | curl 命令文件路径 |
| `-protocol` | string | `http` | 协议：http, grpc, websocket |
| `-url` | string | - | 目标 URL |
| `-c` | uint64 | `1` | 并发数 |
| `-n` | uint64 | `1` | 每个并发的请求数 |
| `-method` | string | `GET` | HTTP 方法 |
| `-timeout` | duration | `30s` | 请求超时时间 |

## HTTP 参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-http2` | bool | `false` | 启用 HTTP/2 |
| `-keepalive` | bool | `false` | 启用 Keep-Alive 长连接 |
| `-H` | string | - | 请求头（可多次使用） |
| `-data` | string | - | 请求体数据 |

**示例**：
```bash
./go-stress -url https://api.example.com/users \
  -method POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -data '{"name":"test"}' \
  -http2 -keepalive \
  -c 100 -n 10000
```

## gRPC 参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-grpc-reflection` | bool | `false` | 使用 gRPC 反射 |
| `-grpc-service` | string | - | gRPC 服务名 |
| `-grpc-method` | string | - | gRPC 方法名 |

**示例**：
```bash
./go-stress -protocol grpc \
  -url localhost:50051 \
  -grpc-reflection \
  -grpc-service pb.UserService \
  -grpc-method GetUser \
  -data '{"id":123}' \
  -c 50 -n 1000
```

## 日志参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-log-level` | string | `info` | 日志级别：debug, info, warn, error |
| `-log-file` | string | - | 日志文件路径 |
| `-quiet` | bool | `false` | 静默模式（仅错误） |
| `-verbose` | bool | `false` | 详细模式（调试信息） |

**示例**：
```bash
# 静默模式
./go-stress -config config.yaml -quiet

# 详细日志并保存到文件
./go-stress -config config.yaml -verbose -log-file stress.log
```

## 存储和报告参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-storage` | string | `memory` | 存储模式：memory, sqlite |
| `-report-prefix` | string | `stress-report` | 报告文件名前缀 |
| `-max-memory` | string | - | 内存阈值（如：2GB, 512MB） |

**示例**：
```bash
# 使用 SQLite 持久化存储
./go-stress -config config.yaml -storage sqlite

# 内存限制
./go-stress -config config.yaml -max-memory 2GB -storage sqlite
```

## 分布式参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `-mode` | string | `standalone` | 运行模式：standalone, master, slave |
| `-master` | string | - | Master 地址（Slave 模式必需） |
| `-slave-id` | string | - | Slave ID（可选） |
| `-grpc-port` | int | `9090` | gRPC 端口 |
| `-http-port` | int | `8080` | HTTP 端口（Master 模式） |
| `-region` | string | `default` | 节点区域标签 |

**示例**：
```bash
# Master 节点
./go-stress -mode master -config config.yaml -grpc-port 9090 -http-port 8080

# Slave 节点
./go-stress -mode slave -master 192.168.1.100:9090 -region beijing -slave-id slave-01
```

## 参数优先级

1. 命令行参数（最高）
2. 配置文件
3. curl 文件
4. 默认值（最低）

**示例**：
```bash
# config.yaml 中 concurrency: 50，但命令行指定 -c 100，最终使用 100
./go-stress -config config.yaml -c 100
```

## 获取帮助

```bash
./go-stress -help
```
