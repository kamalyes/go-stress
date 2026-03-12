# 链路追踪

每个请求携带一个 W3C trace 标识，使压测结果能与被压服务的日志和链路对齐

压测报告里出现一条失败请求时，原本只能靠时间戳去服务端日志里猜是哪一次调用 — 现在可以直接拿 trace-id 去搜

## 🚀 默认行为

**默认开启**，无需配置：

```bash
go-stress -url https://api.example.com -c 10 -n 100
```

每个请求会：

1. 生成一个 W3C trace-context 标识
2. 以 `traceparent` 与 `X-Request-Id` 两个请求头发往被压服务
3. 写入 `ctx`，使 go-stress 自身的日志带上 `trace_id`
4. 记入请求结果，落入存储与报告 JSON 的 `trace_id` 字段
5. 从响应头里回捞服务端自己的追踪 ID，记入 `server_trace_id`

请求头形如：

```
traceparent:  00-6af1f5def8f7a2d50000000000000001-be4b5bd56dc0024f-01
              │  └─ trace-id（32 位十六进制）──────┘ └─ span-id ──┘ └─ flags
              └─ version
X-Request-Id: 6af1f5def8f7a2d50000000000000001
              └─ 与 traceparent 里的 trace-id 同值
```

两个头一起带，是因为它们覆盖的对端形态不同：

- `traceparent` 要求对方接了 OTel SDK 或框架链路中间件才认
- `X-Request-Id` 是网关、Nginx、各家框架访问日志的事实标准，几乎总会被原样记下

取同一个值，于是无论对方记的是哪个头，回头都能用同一个字符串在报告里反查到这一次请求

## 📥 回捞服务端的追踪 ID

很多服务并不接 W3C 链路，而是自己生成一个 ID 再从响应头回带（`x-trace-id`、`x-request-id`、ALB 的 `x-amzn-trace-id` 等）—— 这种情况下我们发出去的 trace-id 在对方日志里根本不存在，**只有对方回带的那个 ID 才查得到**

go-stress 默认会从响应头里把它捞回来，按以下顺序取第一个命中的：

| 优先级 | 响应头 | 常见来源 |
|:------|:-------|:---------|
| 1 | `X-Trace-Id` | 自建服务 / 自建网关 |
| 2 | `X-Request-Id` | Nginx、Kong、各框架中间件 |
| 3 | `X-Correlation-Id` | .NET / Spring 生态 |
| 4 | `Trace-Id` | 省略 `X-` 前缀的变体 |
| 5 | `Request-Id` | 同上 |
| 6 | `X-Amzn-Trace-Id` | AWS ALB / API Gateway |
| 7 | `X-Cloud-Trace-Context` | GCP Load Balancer |
| 8 | `X-B3-TraceId` | Zipkin / 旧版 Istio |
| 9 | `traceparent` | 对端原样回带（信息量最低，故排末位） |

**匹配不区分大小写**：HTTP/1.1 经 Go 规范化成 `X-Trace-Id`，HTTP/2 在线路上全小写，自建网关回带的还可能是 `X-TRACE-ID` — 按字面比对三种里只能中一种

捞到的值落在请求结果的两个字段上：

| 字段 | 含义 |
|:-----|:-----|
| `server_trace_id` | 捞到的 ID 本身 |
| `server_trace_header` | 它实际来自哪个响应头（保留原始大小写） |

HTML 报告的 API 条目里，点开某个状态码的样本列表，这个 ID 会以**蓝色加粗**显示在每条样本上（对端没回带时才退回显示我们自己发出的 `trace_id`），直接复制去对方日志系统里查即可

### 自定义要捞哪些字段

```bash
# 只捞这两个，顺序即优先级
go-stress -url https://api.example.com -trace-capture "x-biz-trace,x-trace-id"

# 关掉回捞
go-stress -url https://api.example.com -trace-capture -
```

```yaml
advanced:
  # 顺序即优先级；留空取上表的默认列表，填 ["-"] 表示不捞
  trace_capture_headers:
    - X-Biz-Trace
    - X-Trace-Id
```

### 自定义请求 ID 头名

```bash
# 换一个头名
go-stress -url https://api.example.com -request-id-header X-Biz-Request-Id

# 不带这个头（traceparent 仍然带）
go-stress -url https://api.example.com -request-id-header -
```

```yaml
advanced:
  # 留空取默认的 X-Request-Id，填 "-" 表示不带
  request_id_header: X-Biz-Request-Id
```

## 🔌 为什么用 traceparent

用 [W3C trace-context](https://www.w3.org/TR/trace-context/) 标准头而非自造的 `X-Trace-Id`：被压服务只要接了任意 OTel SDK 或主流框架的链路中间件，就会自动把它作为父 span 接上 — 服务端那条链路与压测结果共享同一个 trace-id，**无需对方改一行代码**

典型用法：压测跑完，从报告里挑出 P99 那几条请求的 trace-id，直接去 Jaeger / Tempo / 自建链路系统里搜，看服务端这一次调用慢在哪个环节

## 🔍 在哪能看到 trace-id

| 位置 | 字段 | 说明 |
|:-----|:-----|:-----|
| 被压服务 | `traceparent` / `X-Request-Id` 请求头 | 服务端据此接上自己的链路或记进访问日志 |
| 报告 JSON | `trace_id`、`server_trace_id`、`server_trace_header` | `stress-report/<时间戳>/index.json` 中每条请求详情 |
| HTML 报告 | API 条目 → 点状态码 → 样本行 | 服务端 ID 蓝色加粗，展开后在「请求信息」页签里两个 ID 并列 |
| SQLite 存储 | `trace_id`、`server_trace_id`、`server_trace_header` 列 | `-storage sqlite` 时可用 SQL 查询 |
| BadgerDB 存储 | 同上三个字段 | `-storage badger` 时随结果 JSON 一并保存 |
| go-stress 日志 | `trace_id` | 走 `*Context*` 系列日志方法的条目自动带上 |

SQL 查询示例：

```sql
-- 找出最慢的 10 条请求及其 trace-id
SELECT trace_id, url, status_code, duration
FROM request_details
ORDER BY duration DESC
LIMIT 10;

-- 按 trace-id 定位某一次调用
SELECT * FROM request_details WHERE trace_id = '6af1f5def8f7a2d50000000000000001';

-- 失败请求配上服务端自己的 ID，拿去对方日志系统里逐条查
SELECT server_trace_header, server_trace_id, status_code, error
FROM request_details
WHERE success = 0 AND server_trace_id != ''
ORDER BY timestamp DESC
LIMIT 20;
```

> `server_trace_id` / `server_trace_header` 两列对旧库由 `ALTER TABLE` 补出，历史行为 `NULL`

## 🔧 关闭注入

`traceparent` 是发往被压服务的真实请求头，需要逐字节复现原始请求时（例如签名校验把全部请求头纳入计算）可以关掉：

```bash
go-stress -url https://api.example.com -no-trace-header
```

```yaml
advanced:
  disable_trace_header: true
```

`disable_trace_header` 同时管住 `traceparent` 与 `X-Request-Id` 两个头：需要逐字节复现原始请求时，多带哪一个都会破坏签名

关闭后仍会生成 trace-id 并记入结果与日志，只是不往外发 — 本地关联仍然可用，只是无法与服务端对齐；响应头的回捞不受影响，对端若自带追踪头照样捞得到

只想关掉其中一个时用 `-request-id-header -` 或 `-trace-capture -`，见上文

## ⚙️ 实现细节

### 标识如何生成

每个 Worker 独占一个生成器，全程无锁、无原子操作：

- **trace-id** = Worker 创建时一次性随机的 8 字节前缀 + 8 字节自增序号
- **span-id** = 前缀与序号的 splitmix64 混淆值，避免与 trace-id 尾部逐位相同
- **flags** 固定 `01`，表示已采样

前缀保证跨 Worker 与跨次压测不撞，序号保证同一 Worker 内唯一

不逐请求调 `crypto/rand`：这里的标识只用于把压测结果与被压服务的日志对上，不承担任何安全语义，而压测工具自身绝不能成为被测系统的瓶颈

实测开销 **37.8 ns/op、96 B/op、2 allocs/op**（Apple M4）— 两次分配躲不掉，Go 字符串不可变，trace-id 没法复用 traceparent 的底层数组

### 回捞如何做到不拖慢压测

回捞走在每个请求的收尾路径上，而「一个候选字段都没命中」才是常态 — 多数服务并不回带追踪头

朴素写法是拿 9 个候选字段逐一与每个响应头 `EqualFold`，实测 **316 ns/op**，全花在注定失败的比较上 —— 改成先过一道 O(1) 预筛：构造时把所有候选的**字段名长度**与**首字母**各压成一个位集，匹配时两次位测试即可排除 `Content-Type`、`Date`、`Server` 这类绝大多数响应头

| 场景 | 朴素实现 | 位集预筛 | |
|:-----|--------:|--------:|:--|
| 未命中（5 个普通响应头） | 316.1 ns/op | **37.7 ns/op** | 8.4× |
| 命中 `X-Trace-Id` | 28.4 ns/op | 30.4 ns/op | 慢 2 ns |

两者都是 **0 B/op、0 allocs/op**；命中路径反而慢了约 2 ns：预筛版外层遍历响应头，不能一命中就返回（map 遍历顺序随机，先遇到的未必优先级最高），改为记住当前最优候选的下标并收窄内层上界 —— 拿命中路径的 2 ns 换未命中路径的 278 ns，这笔账是划算的

候选字段名超过 63 字节或首字符不是 ASCII 字母时编不进位集，此时整体退回逐一比对 — 与其悄悄漏掉用户配的字段，不如慢一点但不漏

### 为什么不直接上报 OTel

go-stress 不做 OTel 上报，这是刻意的取舍：

`traceparent` 注入后，被压服务已会把自己的 span 报到它自己的采集端，且带的是此处记录的同一个 trace-id — **排查目标已经达成**

而压测端若按每请求一个 span 上报，在本机实测的 19k QPS 下就是 19000 spans/秒：上报开销会开始与被测对象争抢资源，污染测量结果，还会压垮采集端

go-stress 只当 trace 的**发起者**，不做生产者

> go.mod 里的 `go.opentelemetry.io/otel*` 条目全是 `// indirect`，经 go-logger 与 grpc 传递进来，go-stress 自身零 OTel 代码

## 🔗 与分布式模式

分布式模式下每个 Slave 的 Worker 各自生成标识，前缀随机因此跨节点不会相撞

请求结果同时带 `node_id` 与 `task_id`，与 `trace_id` 配合即可定位「哪个节点、哪个任务、哪一次调用」：

```sql
SELECT node_id, task_id, trace_id, duration
FROM request_details
WHERE success = 0
ORDER BY duration DESC;
```
