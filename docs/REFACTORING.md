# go-stress 重构计划

> **文档版本**: v1.0  
> **创建日期**: 2026年1月23日  
> **作者**: kamalyes  
> **状态**: 📋 规划中

---

## 📋 目录

- [1. 重构概述](#1-重构概述)
- [2. go-toolbox 模块集成](#2-go-toolbox-模块集成)
- [3. 报告系统重构](#3-报告系统重构)
- [4. 代码优化清单](#4-代码优化清单)
- [5. go-toolbox 新增功能](#5-go-toolbox-新增功能)
- [6. 实施步骤](#6-实施步骤)

---

## 1. 重构概述

### 1.1 重构目标

- ✅ **统一数据结构**: 报告系统使用统一的数据模型，消除静态/实时模式的重复代码
- ✅ **模块化增强**: 充分利用 go-toolbox 的 mathx、syncx、convert、retry 模块
- ✅ **代码精简**: 移除重复代码，提高代码复用率
- ✅ **性能优化**: 使用 go-toolbox 的高性能组件替换标准库实现
- ✅ **类型安全**: 利用泛型提供编译时类型检查

### 1.2 涉及模块

| 模块 | 当前状态 | 重构方向 |
|:-----|:---------|:---------|
| **statistics** | 报告数据结构冗余 | 统一数据模型，简化代码 |
| **config/variable** | 手动类型转换 | 使用 convert 模块 |
| **executor** | 使用标准库 sync/atomic | 替换为 syncx 原子操作 |
| **statistics/collector** | 自定义数学计算 | 使用 mathx 模块 |
| **protocol** | 类型转换分散 | 集中使用 convert 模块 |

---

## 2. 详细文件分析

### 2.1 config/variable.go (473行)

#### 📊 文件概况
- **当前状态**: 包含大量手动实现的工具函数
- **可优化空间**: ⭐⭐⭐⭐⭐ (非常高)
- **预计减少代码**: ~80行 (17%)

#### 🔍 详细分析

**问题 1: 手动实现的数学函数 (行 298-330)**
```go
// 当前实现 - 重复造轮子
"max": func(a, b int) int {
    if a > b {
        return a
    }
    return b
},
"min": func(a, b int) int {
    if a < b {
        return a
    }
    return b
},
"abs": func(n int) int {
    if n < 0 {
        return -n
    }
    return n
},
"pow": func(x, y float64) float64 {
    return math.Pow(x, y)
},
// ... 更多数学函数
```

**解决方案: 使用 go-toolbox/pkg/mathx**
```go
import "github.com/kamalyes/go-toolbox/pkg/mathx"

// 直接使用泛型版本
"max": mathx.AtMost[int],           // 支持任意数值类型
"min": mathx.AtLeast[int],          // 支持任意数值类型  
"abs": mathx.Abs[int],              // 泛型实现，更安全
"between": mathx.Between[int],      // 新增：限制在范围内
"clamp": mathx.Between[float64],   // 新增：浮点数限制
```

**问题 2: 手动实现的类型转换 (行 362-369)**
```go
// 当前实现 - 容易出错，没有错误处理
"toInt": func(s string) int {
    i, _ := strconv.Atoi(s)  // 忽略错误❌
    return i
},
"toFloat": func(s string) float64 {
    f, _ := strconv.ParseFloat(s, 64)  // 忽略错误❌
    return f
},
```

**解决方案: 使用 go-toolbox/pkg/convert**
```go
import "github.com/kamalyes/go-toolbox/pkg/convert"

// 更安全、功能更强大
"toInt": func(s string) int {
    v, _ := convert.MustIntT[int](s, nil)
    return v
},
"toInt64": func(s string) int64 {
    v, _ := convert.MustIntT[int64](s, nil)
    return v
},
"toFloat": func(s string) float64 {
    v, _ := convert.MustIntT[float64](s, nil)
    return v
},
"toString": convert.MustString[any],

// 新增：四舍五入模式
"roundUp": func(s string) int {
    mode := convert.RoundUp
    v, _ := convert.MustIntT[int](s, &mode)
    return v
},
"roundDown": func(s string) int {
    mode := convert.RoundDown
    v, _ := convert.MustIntT[int](s, &mode)
    return v
},
"roundNearest": func(s string) int {
    mode := convert.RoundNearest
    v, _ := convert.MustIntT[int](s, &mode)
    return v
},
```

**问题 3: 使用 sync/atomic 操作 (行 27, 58, 427)**
```go
import "sync/atomic"

// 直接使用标准库
sequence  uint64

"seq": func() uint64 {
    return atomic.AddUint64(&v.sequence, 1)
},
```

**解决方案: 使用 go-toolbox/pkg/syncx**
```go
import "github.com/kamalyes/go-toolbox/pkg/syncx"

type VariableResolver struct {
    variables map[string]any
    sequence  *syncx.Uint64  // 更优雅的原子类型
    funcMap   template.FuncMap
}

func NewVariableResolver() *VariableResolver {
    v := &VariableResolver{
        variables: make(map[string]any),
        sequence:  syncx.NewUint64(0),  // 初始化
    }
    
    v.funcMap = template.FuncMap{
        "seq": func() uint64 {
            return v.sequence.Add(1)  // 更清晰的API
        },
        // ...
    }
}
```

**可抽离到 go-toolbox 的功能**

这些业务特定的随机函数可以抽离到 `go-toolbox/pkg/random/business.go`：

```go
// go-toolbox/pkg/random/business.go - 新建
package random

// RandomEmail 生成随机邮箱
func RandomEmail() string {
    return fmt.Sprintf("user_%s@example.com", RandString(8, LOWERCASE|NUMBER))
}

// RandomPhone 生成随机手机号（中国）
func RandomPhone() string {
    return fmt.Sprintf("1%s", RandString(10, NUMBER))
}

// RandomIP 生成随机IP地址
func RandomIP() string {
    return fmt.Sprintf("%d.%d.%d.%d",
        RandInt(1, 255), RandInt(0, 255),
        RandInt(0, 255), RandInt(1, 255))
}

// RandomMAC 生成随机MAC地址
func RandomMAC() string {
    mac := make([]byte, 6)
    for i := range mac {
        mac[i] = byte(RandInt(0, 255))
    }
    return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
        mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

// RandomChineseName 生成随机中文名（拼音）
func RandomChineseName() string {
    firstNames := []string{"Zhang", "Wang", "Li", "Liu", "Chen", "Yang"}
    lastNames := []string{"Wei", "Fang", "Lei", "Na", "Ming"}
    return firstNames[RandInt(0, len(firstNames)-1)] + 
           lastNames[RandInt(0, len(lastNames)-1)]
}

// RandomCity 生成随机城市
func RandomCity() string {
    cities := []string{"Beijing", "Shanghai", "Guangzhou", "Shenzhen"}
    return cities[RandInt(0, len(cities)-1)]
}

// RandomHexColor 生成随机十六进制颜色
func RandomHexColor() string {
    return fmt.Sprintf("#%02x%02x%02x",
        RandInt(0, 255), RandInt(0, 255), RandInt(0, 255))
}

// RandomPrice 生成随机价格
func RandomPrice(min, max int) string {
    price := RandInt(min*100, max*100)
    return fmt.Sprintf("%.2f", float64(price)/100)
}

// RandomIDCard 生成随机身份证号（简化版）
func RandomIDCard() string {
    area := fmt.Sprintf("%06d", RandInt(110000, 659000))
    birth := time.Now().AddDate(-RandInt(18, 60), 0, -RandInt(0, 365)).Format("20060102")
    seq := fmt.Sprintf("%03d", RandInt(0, 999))
    return area + birth + seq + "X"
}
```

#### 📝 重构清单

- [ ] 替换 `max/min/abs` 为 `mathx.AtMost/AtLeast/Abs`
- [ ] 替换所有 `strconv` 类型转换为 `convert.MustIntT`
- [ ] 使用 `syncx.Uint64` 替换 `atomic` 操作
- [ ] 移除 `pow/sqrt/ceil/floor/round`，直接使用 `math` 包
- [ ] 将业务随机函数迁移到 `go-toolbox/pkg/random/business.go`
- [ ] 添加更多 mathx 函数：`between`, `clamp`

#### 💡 优化收益
- **代码减少**: ~80行
- **类型安全**: 泛型支持，编译时检查
- **错误处理**: convert 模块有完善的错误处理
- **可维护性**: 使用标准工具包，减少bug

---

### 2.2 statistics/collector.go (321行)

#### 📊 文件概况
- **当前状态**: 使用标准库 atomic 和 sync
- **可优化空间**: ⭐⭐⭐⭐ (高)
- **预计减少代码**: ~50行 (15%)

#### 🔍 详细分析

**问题 1: 使用标准库 atomic 操作 (行 90-95, 213-227)**
```go
import "sync/atomic"

type Collector struct {
    mu sync.Mutex
    
    totalRequests   uint64  // 使用 atomic 操作
    successRequests uint64
    failedRequests  uint64
    // ...
}

func (c *Collector) Collect(result *types.RequestResult) {
    atomic.AddUint64(&c.totalRequests, 1)
    
    if result.Success {
        atomic.AddUint64(&c.successRequests, 1)
    } else {
        atomic.AddUint64(&c.failedRequests, 1)
    }
    // ...
}

func (c *Collector) GetMetrics() *Metrics {
    return &Metrics{
        TotalRequests:   atomic.LoadUint64(&c.totalRequests),
        SuccessRequests: atomic.LoadUint64(&c.successRequests),
        FailedRequests:  atomic.LoadUint64(&c.failedRequests),
    }
}
```

**解决方案: 使用 go-toolbox/pkg/syncx 原子类型**
```go
import "github.com/kamalyes/go-toolbox/pkg/syncx"

type Collector struct {
    mu sync.Mutex
    
    // 使用 syncx 原子类型 - 更优雅的API
    totalRequests   *syncx.Uint64
    successRequests *syncx.Uint64
    failedRequests  *syncx.Uint64
    
    // 其他字段...
}

func NewCollector() *Collector {
    return &Collector{
        totalRequests:   syncx.NewUint64(0),
        successRequests: syncx.NewUint64(0),
        failedRequests:  syncx.NewUint64(0),
        durations:       make([]time.Duration, 0, 10000),
        errors:          make(map[string]uint64),
        statusCodes:     make(map[int]uint64),
        requestDetails:  make([]RequestDetail, 0, 10000),
        maxDetails:      10000,
        minDuration:     time.Hour,
    }
}

func (c *Collector) Collect(result *types.RequestResult) {
    c.totalRequests.Add(1)  // 更清晰的API
    
    if result.Success {
        c.successRequests.Add(1)
    } else {
        c.failedRequests.Add(1)
    }
    // ...
}

func (c *Collector) GetMetrics() *Metrics {
    return &Metrics{
        TotalRequests:   c.totalRequests.Load(),   // 更简洁
        SuccessRequests: c.successRequests.Load(),
        FailedRequests:  c.failedRequests.Load(),
    }
}
```

**问题 2: 百分位计算 (行 204-209)**
```go
// 简单实现，没有边界检查
func (c *Collector) percentile(p float64) time.Duration {
    if len(c.durations) == 0 {
        return 0
    }
    
    index := int(float64(len(c.durations)-1) * p)
    return c.durations[index]
}
```

**解决方案: 使用 go-toolbox/pkg/mathx 统计函数**
```go
import "github.com/kamalyes/go-toolbox/pkg/mathx"

func (c *Collector) percentile(p float64) time.Duration {
    if len(c.durations) == 0 {
        return 0
    }
    
    // 使用 mathx.Between 确保索引安全
    index := mathx.Between(
        int(float64(len(c.durations))*p),
        0,
        len(c.durations)-1,
    )
    return c.durations[index]
}

// 或者直接使用 mathx.Percentile (需要在 go-toolbox 中实现)
func (c *Collector) percentile(p float64) time.Duration {
    if len(c.durations) == 0 {
        return 0
    }
    return mathx.Percentile(c.durations, p)
}
```

**问题 3: RequestDetail 对象频繁创建 (行 127-154)**
```go
// 每次请求都创建新对象，GC压力大
detail := RequestDetail{
    ID:              c.totalRequests,
    Timestamp:       time.Now(),
    Duration:        result.Duration,
    // ... 20多个字段
}
c.requestDetails = append(c.requestDetails, detail)
```

**解决方案: 使用 go-toolbox/pkg/syncx 对象池**
```go
type Collector struct {
    // ... 其他字段
    
    // 使用对象池复用 RequestDetail
    detailPool *syncx.Pool[*RequestDetail]
}

func NewCollector() *Collector {
    return &Collector{
        // ... 其他初始化
        
        detailPool: syncx.NewPool(func() *RequestDetail {
            return &RequestDetail{}
        }),
    }
}

func (c *Collector) Collect(result *types.RequestResult) {
    c.totalRequests.Add(1)
    
    // 从池中获取对象
    detail := c.detailPool.Get()
    
    // 填充数据
    detail.ID = c.totalRequests.Load()
    detail.Timestamp = time.Now()
    detail.Duration = result.Duration
    // ... 填充其他字段
    
    // 保存副本到列表（不影响池中对象）
    c.mu.Lock()
    if len(c.requestDetails) >= c.maxDetails {
        c.requestDetails = c.requestDetails[1000:]
    }
    // 创建副本保存
    detailCopy := *detail
    c.requestDetails = append(c.requestDetails, detailCopy)
    c.mu.Unlock()
    
    // 归还到池中
    c.detailPool.Put(detail)
}
```

**可抽离到 go-toolbox 的功能**

统计函数可以抽离到 `go-toolbox/pkg/mathx/stats.go`：

```go
// go-toolbox/pkg/mathx/stats.go - 新建或扩展
package mathx

import "github.com/kamalyes/go-toolbox/pkg/types"

// Percentile 计算百分位值
// 注意：切片必须已排序
func Percentile[T types.Numerical](values []T, p float64) T {
    if len(values) == 0 {
        return ZeroValue[T]()
    }
    
    p = Between(p, 0.0, 1.0)
    index := Between(
        int(float64(len(values))*p),
        0,
        len(values)-1,
    )
    return values[index]
}

// Mean 计算平均值
func Mean[T types.Numerical](values []T) float64 {
    if len(values) == 0 {
        return 0
    }
    
    var sum T
    for _, v := range values {
        sum += v
    }
    return float64(sum) / float64(len(values))
}

// Median 计算中位数
func Median[T types.Numerical](values []T) T {
    if len(values) == 0 {
        return ZeroValue[T]()
    }
    
    mid := len(values) / 2
    if len(values)%2 == 0 {
        return (values[mid-1] + values[mid]) / 2
    }
    return values[mid]
}

// StandardDeviation 计算标准差
func StandardDeviation[T types.Numerical](values []T) float64 {
    if len(values) == 0 {
        return 0
    }
    
    mean := Mean(values)
    var variance float64
    for _, v := range values {
        diff := float64(v) - mean
        variance += diff * diff
    }
    variance /= float64(len(values))
    
    return math.Sqrt(variance)
}
```

#### 📝 重构清单

- [ ] 替换 `atomic.AddUint64/LoadUint64` 为 `syncx.Uint64`
- [ ] 使用 `syncx.Pool` 优化 RequestDetail 对象创建
- [ ] 使用 `mathx.Between` 优化百分位计算
- [ ] 使用 `mathx.Percentile` 替换自定义实现
- [ ] 考虑使用 `syncx.Map` 优化 errors 和 statusCodes map
- [ ] 将统计函数移到 `go-toolbox/pkg/mathx/stats.go`

#### 💡 优化收益
- **代码减少**: ~50行
- **性能提升**: 对象池减少GC压力 30-40%
- **API优雅**: syncx 原子类型更清晰
- **类型安全**: 泛型统计函数更安全

---

---

### 2.3 statistics/report.go (200行)

#### 📊 文件概况
- **当前状态**: 数据结构与HTML报告重复
- **可优化空间**: ⭐⭐⭐⭐⭐ (非常高)
- **预计减少代码**: ~120行 (60%)

#### 🔍 详细分析

**问题 1: 数据结构重复**
```go
// Report 和 HTMLReportData 功能重叠
type Report struct {
    TotalRequests   uint64
    SuccessRate     float64  // 原始数值
    // ...
}

type HTMLReportData struct {
    TotalRequests   uint64
    SuccessRate     string  // 格式化后的字符串
    // ...
}
```

**解决方案: 统一数据模型 + 格式化器**
详见 [3. 报告系统重构](#3-报告系统重构)

**问题 2: 使用 units.BytesSize 格式化**
```go
// 已经在使用 go-toolbox/pkg/units
data.TotalSize = units.BytesSize(float64(c.totalSize))
```

✅ 这部分已经正确使用了 go-toolbox

#### 📝 重构清单

- [ ] 与 HTMLReportData 合并为统一的 ReportData
- [ ] 创建 ReportFormatter 处理格式化
- [ ] 移除重复的 ToJSON 方法（使用 serializer）
- [ ] 统一 JSON 序列化（见 serializer 模块）

---

### 2.4 statistics/html_report.go (292行)

#### 📊 文件概况
- **当前状态**: 大量数据转换和格式化代码
- **可优化空间**: ⭐⭐⭐⭐⭐ (非常高)
- **预计减少代码**: ~180行 (62%)

#### 🔍 详细分析

**问题 1: HTMLReportData 与 Report 重复**

当前有三套数据结构：
1. `RequestDetail` - 原始数据
2. `RequestDetailDisplay` - HTML显示数据（字符串格式）
3. `Report` - 报告数据

**解决方案: 统一为 ReportData**
```go
// 新的统一数据结构
type ReportData struct {
    Mode         ReportMode    `json:"mode"`  // static | realtime
    GenerateTime time.Time     `json:"generate_time"`
    
    // 所有字段都是原始类型
    TotalRequests   uint64        `json:"total_requests"`
    SuccessRate     float64       `json:"success_rate"`  // 0-100
    QPS             float64       `json:"qps"`
    TotalSize       float64       `json:"total_size"`
    
    // 时间类型
    MinDuration     time.Duration `json:"min_duration"`
    // ...
}

// 格式化器负责展示层转换
type ReportFormatter struct {
    data *ReportData
}

func (f *ReportFormatter) FormatSuccessRate() string {
    return fmt.Sprintf("%.2f%%", f.data.SuccessRate)
}

func (f *ReportFormatter) FormatSize() string {
    return units.BytesSize(f.data.TotalSize)
}
```

**问题 2: 重复的百分比计算和排序逻辑**
```go
// 错误统计 - 重复逻辑
for err, count := range c.errors {
    percentage := float64(count) / float64(c.totalRequests) * 100
    data.ErrorStats = append(data.ErrorStats, ErrorStat{
        Error:      err,
        Count:      count,
        Percentage: fmt.Sprintf("%.2f%%", percentage),
    })
}
sort.Slice(data.ErrorStats, func(i, j int) bool {
    return data.ErrorStats[i].Count > data.ErrorStats[j].Count
})

// 状态码统计 - 完全相同的逻辑
for code, count := range c.statusCodes {
    percentage := float64(count) / float64(c.totalRequests) * 100
    data.StatusCodeStats = append(data.StatusCodeStats, StatusCodeStat{
        StatusCode: code,
        Count:      count,
        Percentage: fmt.Sprintf("%.2f%%", percentage),
    })
}
sort.Slice(data.StatusCodeStats, func(i, j int) bool {
    return data.StatusCodeStats[i].StatusCode < data.StatusCodeStats[j].StatusCode
})
```

**可抽离到 go-toolbox 的公共统计函数**

```go
// go-toolbox/pkg/mathx/stats.go - 扩展
package mathx

// Percentage 计算百分比
func Percentage(part, total uint64) float64 {
    if total == 0 {
        return 0
    }
    return float64(part) / float64(total) * 100
}

// FormatPercentage 格式化百分比
func FormatPercentage(part, total uint64, precision int) string {
    return fmt.Sprintf("%.*f%%", precision, Percentage(part, total))
}

// SortByCount 按计数排序统计数据（降序）
func SortByCount[T any](items []T, getCount func(T) uint64) {
    sort.Slice(items, func(i, j int) bool {
        return getCount(items[i]) > getCount(items[j])
    })
}
```

使用后：
```go
import "github.com/kamalyes/go-toolbox/pkg/mathx"

// 错误统计 - 简化后
for err, count := range c.errors {
    data.ErrorStats = append(data.ErrorStats, ErrorStat{
        Error:      err,
        Count:      count,
        Percentage: mathx.Percentage(count, c.totalRequests),
    })
}
mathx.SortByCount(data.ErrorStats, func(e ErrorStat) uint64 {
    return e.Count
})
```

#### 📝 重构清单

- [ ] 移除 `HTMLReportData`，使用统一的 `ReportData`
- [ ] 移除 `RequestDetailDisplay`，使用统一的 `RequestDetail`
- [ ] 创建 `ReportFormatter` 处理格式化
- [ ] 使用 `mathx.Percentage` 计算百分比
- [ ] 使用 `mathx.SortByCount` 排序统计数据
- [ ] 简化 GenerateHTMLReport 方法

---

### 2.5 statistics/realtime_server.go (479行)

#### 📊 文件概况
- **当前状态**: 与 HTML 报告大量重复代码
- **可优化空间**: ⭐⭐⭐⭐⭐ (非常高)
- **预计减少代码**: ~200行 (42%)

#### 🔍 详细分析

**问题 1: RealtimeData 与其他数据结构重复**
```go
type RealtimeData struct {
    Timestamp       int64   `json:"timestamp"`
    TotalRequests   uint64  `json:"total_requests"`
    SuccessRate     float64 `json:"success_rate"`
    // ... 与 Report 重复
}
```

**解决方案: 使用统一的 ReportData**
```go
func (s *RealtimeServer) handleData(w http.ResponseWriter, r *http.Request) {
    elapsed := time.Since(s.startTime)
    
    // 直接生成统一数据结构
    data := s.collector.GenerateReportData(elapsed, ReportModeRealtime)
    
    // 直接序列化
    json.NewEncoder(w).Encode(data)
}
```

**问题 2: 使用 sync.Mutex 和 sync.Once**
```go
mu          sync.RWMutex
var closeOnce sync.Once
```

**解决方案: 使用 syncx 模块**
```go
import "github.com/kamalyes/go-toolbox/pkg/syncx"

type RealtimeServer struct {
    // 使用 syncx 原子类型管理状态
    isCompleted *syncx.Bool
    isPaused    *syncx.Bool
    isStopped   *syncx.Bool
    
    // 使用 syncx.Map 管理客户端连接
    clients *syncx.Map[chan []byte, bool]
}

func (s *RealtimeServer) MarkCompleted() {
    if s.isCompleted.CAS(false, true) {
        s.endTime = time.Now()
    }
}
```

#### 📝 重构清单

- [ ] 移除 `RealtimeData`，使用统一的 `ReportData`
- [ ] 使用 `syncx.Bool` 管理状态标志
- [ ] 使用 `syncx.Map` 管理客户端连接
- [ ] 简化 collectData 方法
- [ ] 统一 JSON 序列化

---

### 2.6 executor/pool.go (72行)

#### 📊 文件概况
- **当前状态**: 自定义实现的连接池
- **可优化空间**: ⭐⭐⭐⭐ (高)
- **预计减少代码**: ~50行 (69%)

#### 🔍 详细分析

**问题: 完全可以用 syncx.Pool 替换**

当前实现：
```go
type ClientPool struct {
    factory ClientFactory
    pool    chan Client
    maxSize int
    created int
    mu      sync.Mutex
}

func (cp *ClientPool) Get() (Client, error) {
    select {
    case client := <-cp.pool:
        return client, nil
    default:
        cp.mu.Lock()
        defer cp.mu.Unlock()
        if cp.created < cp.maxSize {
            client, err := cp.factory()
            if err != nil {
                return nil, fmt.Errorf("创建客户端失败: %w", err)
            }
            cp.created++
            return client, nil
        }
        return <-cp.pool, nil
    }
}
```

**解决方案: 直接使用 syncx.Pool**
```go
import "github.com/kamalyes/go-toolbox/pkg/syncx"

type ClientPool struct {
    pool *syncx.Pool[types.Client]
}

func NewClientPool(factory ClientFactory, maxSize int) *ClientPool {
    return &ClientPool{
        pool: syncx.NewPool(func() types.Client {
            client, _ := factory()
            return client
        }),
    }
}

func (cp *ClientPool) Get() (types.Client, error) {
    return cp.pool.Get(), nil
}

func (cp *ClientPool) Put(client types.Client) {
    cp.pool.Put(client)
}

func (cp *ClientPool) Close() {
    // syncx.Pool 会自动处理清理
}
```

#### 📝 重构清单

- [ ] 移除自定义 ClientPool 实现
- [ ] 使用 `syncx.Pool[types.Client]`
- [ ] 简化 Get/Put/Close 方法
- [ ] 移除 `mu sync.Mutex` 和 `created` 计数

---

### 2.7 protocol/http_verify.go (336行)

#### 📊 文件概况
- **当前状态**: 大量验证和比较逻辑
- **可优化空间**: ⭐⭐⭐⭐⭐ (非常高)
- **预计减少代码**: ~150行 (45%)

#### 🔍 详细分析

**问题 1: 手动类型转换和数值比较**
```go
func (v *HTTPVerifier) compareNumeric(actualStr, expectStr string, op ExpectOperator) (bool, string) {
    actualNum, err1 := strconv.ParseFloat(actualStr, 64)
    expectNum, err2 := strconv.ParseFloat(expectStr, 64)
    
    if err1 != nil || err2 != nil {
        return false, "数值比较失败: 无法解析为数字"
    }
    
    switch op {
    case OpGT:
        return actualNum > expectNum, ""
    case OpGTE:
        return actualNum >= expectNum, ""
    // ...
    }
}
```

**解决方案: 使用 convert 模块**
```go
import "github.com/kamalyes/go-toolbox/pkg/convert"

func (v *HTTPVerifier) compareNumeric(actualStr, expectStr string, op ExpectOperator) (bool, string) {
    actualNum, err1 := convert.MustIntT[float64](actualStr, nil)
    expectNum, err2 := convert.MustIntT[float64](expectStr, nil)
    
    if err1 != nil || err2 != nil {
        return false, "数值比较失败: 无法解析为数字"
    }
    
    // 使用 mathx 比较
    return v.compareWithOperator(actualNum, expectNum, op), ""
}
```

**问题 2: 大量的字符串验证逻辑**
```go
case OpContains:
    return strings.Contains(actualStr, expectStr), ""
case OpNotContains:
    return !strings.Contains(actualStr, expectStr), ""
case OpHasPrefix:
    return strings.HasPrefix(actualStr, expectStr), ""
case OpHasSuffix:
    return strings.HasSuffix(actualStr, expectStr), ""
case OpEmpty:
    return actualStr == "", ""
case OpNotEmpty:
    return actualStr != "", ""
```

**可抽离到 go-toolbox/pkg/validator 的公共验证器**

```go
// go-toolbox/pkg/validator/compare.go - 新建
package validator

// CompareOperator 比较操作符
type CompareOperator string

const (
    OpEqual              CompareOperator = "eq"
    OpNotEqual           CompareOperator = "ne"
    OpGreaterThan        CompareOperator = "gt"
    OpGreaterThanOrEqual CompareOperator = "gte"
    OpLessThan           CompareOperator = "lt"
    OpLessThanOrEqual    CompareOperator = "lte"
    OpContains           CompareOperator = "contains"
    OpNotContains        CompareOperator = "not_contains"
    OpHasPrefix          CompareOperator = "has_prefix"
    OpHasSuffix          CompareOperator = "has_suffix"
    OpRegex              CompareOperator = "regex"
    OpEmpty              CompareOperator = "empty"
    OpNotEmpty           CompareOperator = "not_empty"
)

// CompareResult 比较结果
type CompareResult struct {
    Success bool
    Message string
    Actual  string
    Expect  string
}

// CompareStrings 比较两个字符串
func CompareStrings(actual, expect string, op CompareOperator) CompareResult {
    result := CompareResult{
        Actual: actual,
        Expect: expect,
    }
    
    switch op {
    case OpEqual:
        result.Success = actual == expect
    case OpNotEqual:
        result.Success = actual != expect
    case OpContains:
        result.Success = strings.Contains(actual, expect)
    case OpNotContains:
        result.Success = !strings.Contains(actual, expect)
    case OpHasPrefix:
        result.Success = strings.HasPrefix(actual, expect)
    case OpHasSuffix:
        result.Success = strings.HasSuffix(actual, expect)
    case OpEmpty:
        result.Success = actual == ""
    case OpNotEmpty:
        result.Success = actual != ""
    case OpRegex:
        matched, err := regexp.MatchString(expect, actual)
        if err != nil {
            result.Message = fmt.Sprintf("正则表达式错误: %v", err)
            return result
        }
        result.Success = matched
    default:
        result.Message = "不支持的操作符"
    }
    
    if !result.Success && result.Message == "" {
        result.Message = fmt.Sprintf("比较失败: 期望 %s %s, 实际 %s", 
            expect, op, actual)
    }
    
    return result
}

// CompareNumbers 比较两个数值
func CompareNumbers[T types.Numerical](actual, expect T, op CompareOperator) CompareResult {
    result := CompareResult{
        Actual: fmt.Sprintf("%v", actual),
        Expect: fmt.Sprintf("%v", expect),
    }
    
    switch op {
    case OpEqual:
        result.Success = actual == expect
    case OpNotEqual:
        result.Success = actual != expect
    case OpGreaterThan:
        result.Success = actual > expect
    case OpGreaterThanOrEqual:
        result.Success = actual >= expect
    case OpLessThan:
        result.Success = actual < expect
    case OpLessThanOrEqual:
        result.Success = actual <= expect
    default:
        result.Message = "不支持的数值操作符"
    }
    
    if !result.Success && result.Message == "" {
        result.Message = fmt.Sprintf("数值比较失败: 期望 %v %s %v, 实际 %v", 
            expect, op, expect, actual)
    }
    
    return result
}

// ValidateJSON 验证JSON结构
func ValidateJSON(data []byte) error {
    var v interface{}
    return json.Unmarshal(data, &v)
}

// ValidateJSONPath 验证JSONPath表达式
func ValidateJSONPath(data []byte, path string, expect interface{}, op CompareOperator) CompareResult {
    result := CompareResult{}
    
    var jsonData interface{}
    if err := json.Unmarshal(data, &jsonData); err != nil {
        result.Message = fmt.Sprintf("解析JSON失败: %v", err)
        return result
    }
    
    // 使用 jsonpath 库查询
    value, err := jsonpath.JsonPathLookup(jsonData, path)
    if err != nil {
        result.Message = fmt.Sprintf("JSONPath查询失败: %v", err)
        return result
    }
    
    result.Actual = fmt.Sprintf("%v", value)
    result.Expect = fmt.Sprintf("%v", expect)
    
    // 根据类型选择比较方式
    actualStr := fmt.Sprintf("%v", value)
    expectStr := fmt.Sprintf("%v", expect)
    
    return CompareStrings(actualStr, expectStr, op)
}

// ValidateStatusCode 验证HTTP状态码
func ValidateStatusCode(actual, expect int) CompareResult {
    return CompareNumbers(actual, expect, OpEqual)
}

// ValidateStatusCodeRange 验证HTTP状态码在范围内
func ValidateStatusCodeRange(actual, min, max int) CompareResult {
    result := CompareResult{
        Actual: fmt.Sprintf("%d", actual),
        Expect: fmt.Sprintf("%d-%d", min, max),
    }
    
    result.Success = actual >= min && actual <= max
    if !result.Success {
        result.Message = fmt.Sprintf("状态码 %d 不在范围 [%d, %d] 内", 
            actual, min, max)
    }
    
    return result
}
```

使用后：
```go
import "github.com/kamalyes/go-toolbox/pkg/validator"

func (v *HTTPVerifier) compareValues(actual, expect interface{}, operator ExpectOperator) (bool, string) {
    actualStr := fmt.Sprintf("%v", actual)
    expectStr := fmt.Sprintf("%v", expect)
    
    // 使用 validator 公共比较器
    result := validator.CompareStrings(actualStr, expectStr, 
        validator.CompareOperator(operator))
    
    return result.Success, result.Message
}

func (v *HTTPVerifier) verifyStatusCode(resp *Response) (bool, error) {
    expectedCode := 200
    // ... 解析 expect
    
    // 使用 validator 验证状态码
    result := validator.ValidateStatusCode(resp.StatusCode, expectedCode)
    
    // 记录验证结果
    resp.Verifications = append(resp.Verifications, VerificationResult{
        Type:    v.config.Type,
        Success: result.Success,
        Message: result.Message,
        Expect:  result.Expect,
        Actual:  result.Actual,
    })
    
    if !result.Success {
        return false, fmt.Errorf(result.Message)
    }
    return true, nil
}
```

#### 📝 重构清单

- [ ] 使用 `convert.MustIntT` 替换 `strconv.ParseFloat`
- [ ] 使用 `validator.CompareStrings` 替换手动比较
- [ ] 使用 `validator.CompareNumbers` 替换数值比较
- [ ] 使用 `validator.ValidateJSONPath` 简化 JSON 验证
- [ ] 使用 `validator.ValidateStatusCode` 验证状态码
- [ ] 抽离通用验证逻辑到 `go-toolbox/pkg/validator/compare.go`

---

### 2.8 config/config.go (200行)

#### 📊 文件概况
- **当前状态**: 配置结构定义
- **可优化空间**: ⭐⭐ (中等)
- **预计减少代码**: ~20行 (10%)

#### 🔍 详细分析

**问题: 配置验证逻辑可以使用 validator 模块**

当前可能需要手动验证配置：
```go
func (c *Config) Validate() error {
    if c.Concurrency == 0 {
        return errors.New("并发数不能为0")
    }
    if c.Requests == 0 && c.Duration == 0 {
        return errors.New("请求数和持续时间不能同时为0")
    }
    if c.URL == "" && len(c.APIs) == 0 {
        return errors.New("必须指定URL或APIs")
    }
    return nil
}
```

**解决方案: 使用 validator 模块**
```go
import "github.com/kamalyes/go-toolbox/pkg/validator"

func (c *Config) Validate() error {
    // 验证并发数
    if c.Concurrency == 0 {
        return fmt.Errorf("并发数不能为0")
    }
    
    // 使用 validator 验证空值
    if c.Requests == 0 && c.Duration == 0 {
        return fmt.Errorf("请求数和持续时间不能同时为0")
    }
    
    // 验证URL或APIs
    urlEmpty := validator.IsEmptyValue(reflect.ValueOf(c.URL))
    apisEmpty := len(c.APIs) == 0
    if urlEmpty && apisEmpty {
        return fmt.Errorf("必须指定URL或APIs")
    }
    
    // 验证超时
    if c.Timeout <= 0 {
        return fmt.Errorf("超时时间必须大于0")
    }
    
    return nil
}
```

#### 📝 重构清单

- [ ] 添加 `Validate()` 方法使用 validator 模块
- [ ] 使用 `validator.IsEmptyValue` 验证空值
- [ ] 添加配置完整性检查

---

## 3. syncx 模块深度使用分析

### 3.1 syncx 可用组件清单

根据分析，go-toolbox/pkg/syncx 提供以下组件：

| 组件 | 说明 | go-stress 使用场景 |
|:-----|:-----|:------------------|
| **Map[K, V]** | 线程安全的泛型Map | errors map, statusCodes map |
| **Set[K]** | 线程安全的集合 | API依赖关系管理 |
| **Pool[T]** | 泛型对象池 | RequestDetail对象复用, Client连接池 |
| **Atomic类型** | Uint64, Int64, Bool等 | 计数器, 状态标志 |
| **Parallel执行器** | 并发执行工具 | Worker并发执行, 批量操作 |
| **StateMachine** | 状态机 | 压测状态管理 |
| **Task** | 任务管理 | 后台任务 |

### 3.2 具体使用方案

#### 3.2.1 使用 syncx.Map 替换标准 map

**当前实现 (statistics/collector.go)**:
```go
type Collector struct {
    mu sync.Mutex
    
    errors      map[string]uint64
    statusCodes map[int]uint64
}

func (c *Collector) Collect(result *types.RequestResult) {
    if result.Error != nil {
        c.mu.Lock()
        c.errors[result.Error.Error()]++
        c.mu.Unlock()
    }
    
    c.mu.Lock()
    if result.StatusCode > 0 {
        c.statusCodes[result.StatusCode]++
    }
    c.mu.Unlock()
}
```

**优化后**:
```go
import "github.com/kamalyes/go-toolbox/pkg/syncx"

type Collector struct {
    // 移除 mu sync.Mutex，使用线程安全的 Map
    errors      *syncx.Map[string, uint64]
    statusCodes *syncx.Map[int, uint64]
}

func NewCollector() *Collector {
    return &Collector{
        errors:      syncx.NewMap[string, uint64](),
        statusCodes: syncx.NewMap[int, uint64](),
        // ...
    }
}

func (c *Collector) Collect(result *types.RequestResult) {
    if result.Error != nil {
        // 原子操作，无需加锁
        errMsg := result.Error.Error()
        old, _ := c.errors.LoadOrStore(errMsg, 0)
        c.errors.Store(errMsg, old+1)
    }
    
    if result.StatusCode > 0 {
        old, _ := c.statusCodes.LoadOrStore(result.StatusCode, 0)
        c.statusCodes.Store(result.StatusCode, old+1)
    }
}

// 遍历也更简单
func (c *Collector) GetErrors() map[string]uint64 {
    result := make(map[string]uint64)
    c.errors.Range(func(k string, v uint64) bool {
        result[k] = v
        return true
    })
    return result
}
```

#### 3.2.2 使用 syncx.Set 管理API依赖

**新增功能 (executor/dependency.go)**:
```go
import "github.com/kamalyes/go-toolbox/pkg/syncx"

type DependencyManager struct {
    // 记录已成功执行的API
    completedAPIs *syncx.Set[string]
    
    // 记录失败的API
    failedAPIs *syncx.Set[string]
}

func NewDependencyManager() *DependencyManager {
    return &DependencyManager{
        completedAPIs: syncx.NewSet[string](),
        failedAPIs:    syncx.NewSet[string](),
    }
}

func (dm *DependencyManager) MarkCompleted(apiName string) {
    dm.completedAPIs.Add(apiName)
}

func (dm *DependencyManager) MarkFailed(apiName string) {
    dm.failedAPIs.Add(apiName)
}

func (dm *DependencyManager) CanExecute(apiName string, dependencies []string) bool {
    // 检查所有依赖是否都已成功
    existing, all := dm.completedAPIs.HasAll(dependencies...)
    if !all {
        return false
    }
    
    // 检查依赖中是否有失败的
    for _, dep := range dependencies {
        if dm.failedAPIs.Has(dep) {
            return false
        }
    }
    
    return true
}
```

#### 3.2.3 使用 syncx.Parallel 并发执行

**当前实现 (executor/scheduler.go)**:
```go
func (s *Scheduler) Run(ctx context.Context) error {
    var wg sync.WaitGroup
    errChan := make(chan error, s.workerCount)
    
    for i := uint64(0); i < s.workerCount; i++ {
        wg.Add(1)
        go func(workerID uint64) {
            defer wg.Done()
            if err := s.runWorker(ctx, workerID); err != nil {
                select {
                case errChan <- err:
                default:
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errChan)
    
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    return nil
}
```

**优化后**:
```go
import "github.com/kamalyes/go-toolbox/pkg/syncx"

func (s *Scheduler) Run(ctx context.Context) error {
    // 创建 worker ID 切片
    workerIDs := make([]uint64, s.workerCount)
    for i := range workerIDs {
        workerIDs[i] = uint64(i)
    }
    
    // 使用 Parallel 执行器
    var firstError error
    syncx.NewParallelSliceExecutor(workerIDs).
        OnError(func(index int, workerID uint64, err error) {
            if firstError == nil {
                firstError = err
            }
            logger.Default.Errorf("Worker %d 失败: %v", workerID, err)
        }).
        OnComplete(func(results []interface{}, errors []error) {
            logger.Default.Info("所有 Worker 完成: 成功 %d, 失败 %d", 
                len(results), len(errors))
        }).
        Execute(func(index int, workerID uint64) (interface{}, error) {
            return nil, s.runWorker(ctx, workerID)
        })
    
    return firstError
}
```

---

## 4. stringx 模块使用分析

### 4.1 当前使用情况

go-stress 已经在使用 stringx 的部分功能：

```go
import "github.com/kamalyes/go-toolbox/pkg/stringx"

// config/variable.go
"upper": func(s string) string {
    return stringx.ToUpper(s)
},
"lower": func(s string) string {
    return stringx.ToLower(s)
},
"reverse": func(s string) string {
    return stringx.Reverse(s)
},

// protocol/http_verify.go
success := stringx.Contains(bodyStr, containsStr)
```

### 4.2 可以增强使用的场景

#### 4.2.1 字符串验证

```go
// 当前手动实现
if len(str) > 80 {
    str = str[:77] + "..."
}

// 使用 stringx
import "github.com/kamalyes/go-toolbox/pkg/stringx"

truncated := stringx.SubString(str, 0, 77) + "..."
```

#### 4.2.2 链式操作

```go
// 可以使用 stringx 的链式调用
result := stringx.New(input).
    ToLowerChain().
    TrimChain().
    ReverseChain().
    Value()
```

---

## 5. serializer 模块使用分析

### 5.1 当前 JSON 序列化问题

**当前实现 (statistics/report.go)**:
```go
func (r *Report) ToJSON() string {
    data := map[string]interface{}{
        "total_requests":   r.TotalRequests,
        "success_requests": r.SuccessRequests,
        // ... 手动构建 map
    }
    
    bytes, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return "{}"
    }
    return string(bytes)
}

func (r *Report) SaveToFile(filename string) error {
    content := r.ToJSON()
    return os.WriteFile(filename, []byte(content), 0644)
}
```

### 5.2 使用 serializer 模块优化

```go
import "github.com/kamalyes/go-toolbox/pkg/serializer"

// 创建统一的序列化器
var reportSerializer = serializer.New[*ReportData]().
    WithType(serializer.TypeJSON).
    WithCompression(serializer.CompressionNone).  // HTML报告不压缩
    WithBase64(false)

// 简化的方法
func (r *ReportData) ToJSON() (string, error) {
    return reportSerializer.EncodeToString(r)
}

func (r *ReportData) SaveToFile(filename string) error {
    data, err := reportSerializer.Encode(r)
    if err != nil {
        return err
    }
    return os.WriteFile(filename, data, 0644)
}

func LoadReportFromFile(filename string) (*ReportData, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return reportSerializer.Decode(data)
}
```

### 5.3 压缩大数据报告

```go
// 对于大量请求明细，可以使用压缩
var compressedSerializer = serializer.New[*ReportData]().
    WithType(serializer.TypeJSON).
    WithCompression(serializer.CompressionGzip).  // 启用压缩
    WithBase64(true)  // Base64编码

// 压缩后保存
func (r *ReportData) SaveCompressed(filename string) error {
    compressed, err := compressedSerializer.Encode(r)
    if err != nil {
        return err
    }
    return os.WriteFile(filename+".gz", compressed, 0644)
}
```

---

## 6. 完整的重构清单

### 6.1 go-toolbox 新增功能清单

#### go-toolbox/pkg/mathx/stats.go
```go
package mathx

import (
    "fmt"
    "math"
    "sort"
)

// Percentile 计算百分位数（支持50, 90, 95, 99）
func Percentile(values []float64, p float64) float64 {
    if len(values) == 0 {
        return 0
    }
    
    sorted := make([]float64, len(values))
    copy(sorted, values)
    sort.Float64s(sorted)
    
    index := int(math.Ceil(float64(len(sorted)) * p / 100.0))
    if index >= len(sorted) {
        index = len(sorted) - 1
    }
    
    return sorted[index]
}

// Percentiles 批量计算多个百分位数
func Percentiles(values []float64, percentiles ...float64) map[float64]float64 {
    result := make(map[float64]float64, len(percentiles))
    
    if len(values) == 0 {
        for _, p := range percentiles {
            result[p] = 0
        }
        return result
    }
    
    // 只排序一次
    sorted := make([]float64, len(values))
    copy(sorted, values)
    sort.Float64s(sorted)
    
    for _, p := range percentiles {
        index := int(math.Ceil(float64(len(sorted)) * p / 100.0))
        if index >= len(sorted) {
            index = len(sorted) - 1
        }
        result[p] = sorted[index]
    }
    
    return result
}

// Percentage 计算百分比
func Percentage(part, total uint64) float64 {
    if total == 0 {
        return 0
    }
    return float64(part) / float64(total) * 100
}

// FormatPercentage 格式化百分比
func FormatPercentage(part, total uint64, precision int) string {
    return fmt.Sprintf("%.*f%%", precision, Percentage(part, total))
}

// Mean 计算平均值
func Mean(values []float64) float64 {
    if len(values) == 0 {
        return 0
    }
    
    sum := 0.0
    for _, v := range values {
        sum += v
    }
    return sum / float64(len(values))
}

// StdDev 计算标准差
func StdDev(values []float64) float64 {
    if len(values) == 0 {
        return 0
    }
    
    mean := Mean(values)
    sumSquares := 0.0
    for _, v := range values {
        diff := v - mean
        sumSquares += diff * diff
    }
    
    return math.Sqrt(sumSquares / float64(len(values)))
}

// SortByCount 按计数排序统计数据（降序）
func SortByCount[T any](items []T, getCount func(T) uint64) {
    sort.Slice(items, func(i, j int) bool {
        return getCount(items[i]) > getCount(items[j])
    })
}

// StatsSummary 统计摘要
type StatsSummary struct {
    Count  int
    Min    float64
    Max    float64
    Mean   float64
    StdDev float64
    P50    float64
    P90    float64
    P95    float64
    P99    float64
}

// SummarizeStats 生成统计摘要
func SummarizeStats(values []float64) StatsSummary {
    if len(values) == 0 {
        return StatsSummary{}
    }
    
    percentiles := Percentiles(values, 50, 90, 95, 99)
    
    return StatsSummary{
        Count:  len(values),
        Min:    Min(values...),
        Max:    Max(values...),
        Mean:   Mean(values),
        StdDev: StdDev(values),
        P50:    percentiles[50],
        P90:    percentiles[90],
        P95:    percentiles[95],
        P99:    percentiles[99],
    }
}
```

#### go-toolbox/pkg/validator/compare.go
```go
package validator

import (
    "encoding/json"
    "fmt"
    "regexp"
    "strings"
    
    "github.com/kamalyes/go-toolbox/pkg/types"
)

// CompareOperator 比较操作符
type CompareOperator string

const (
    OpEqual              CompareOperator = "eq"
    OpNotEqual           CompareOperator = "ne"
    OpGreaterThan        CompareOperator = "gt"
    OpGreaterThanOrEqual CompareOperator = "gte"
    OpLessThan           CompareOperator = "lt"
    OpLessThanOrEqual    CompareOperator = "lte"
    OpContains           CompareOperator = "contains"
    OpNotContains        CompareOperator = "not_contains"
    OpHasPrefix          CompareOperator = "has_prefix"
    OpHasSuffix          CompareOperator = "has_suffix"
    OpRegex              CompareOperator = "regex"
    OpEmpty              CompareOperator = "empty"
    OpNotEmpty           CompareOperator = "not_empty"
)

// CompareResult 比较结果
type CompareResult struct {
    Success bool
    Message string
    Actual  string
    Expect  string
}

// CompareStrings 比较两个字符串
func CompareStrings(actual, expect string, op CompareOperator) CompareResult {
    result := CompareResult{
        Actual: actual,
        Expect: expect,
    }
    
    switch op {
    case OpEqual:
        result.Success = actual == expect
    case OpNotEqual:
        result.Success = actual != expect
    case OpContains:
        result.Success = strings.Contains(actual, expect)
    case OpNotContains:
        result.Success = !strings.Contains(actual, expect)
    case OpHasPrefix:
        result.Success = strings.HasPrefix(actual, expect)
    case OpHasSuffix:
        result.Success = strings.HasSuffix(actual, expect)
    case OpEmpty:
        result.Success = actual == ""
    case OpNotEmpty:
        result.Success = actual != ""
    case OpRegex:
        matched, err := regexp.MatchString(expect, actual)
        if err != nil {
            result.Message = fmt.Sprintf("正则表达式错误: %v", err)
            return result
        }
        result.Success = matched
    default:
        result.Message = "不支持的操作符"
    }
    
    if !result.Success && result.Message == "" {
        result.Message = fmt.Sprintf("比较失败: 期望 %s %s, 实际 %s", 
            expect, op, actual)
    }
    
    return result
}

// CompareNumbers 比较两个数值
func CompareNumbers[T types.Numerical](actual, expect T, op CompareOperator) CompareResult {
    result := CompareResult{
        Actual: fmt.Sprintf("%v", actual),
        Expect: fmt.Sprintf("%v", expect),
    }
    
    switch op {
    case OpEqual:
        result.Success = actual == expect
    case OpNotEqual:
        result.Success = actual != expect
    case OpGreaterThan:
        result.Success = actual > expect
    case OpGreaterThanOrEqual:
        result.Success = actual >= expect
    case OpLessThan:
        result.Success = actual < expect
    case OpLessThanOrEqual:
        result.Success = actual <= expect
    default:
        result.Message = "不支持的数值操作符"
    }
    
    if !result.Success && result.Message == "" {
        result.Message = fmt.Sprintf("数值比较失败: 期望 %v %s %v, 实际 %v", 
            expect, op, expect, actual)
    }
    
    return result
}

// ValidateJSON 验证JSON结构
func ValidateJSON(data []byte) error {
    var v interface{}
    return json.Unmarshal(data, &v)
}

// ValidateStatusCode 验证HTTP状态码
func ValidateStatusCode(actual, expect int) CompareResult {
    return CompareNumbers(actual, expect, OpEqual)
}

// ValidateStatusCodeRange 验证HTTP状态码在范围内
func ValidateStatusCodeRange(actual, min, max int) CompareResult {
    result := CompareResult{
        Actual: fmt.Sprintf("%d", actual),
        Expect: fmt.Sprintf("%d-%d", min, max),
    }
    
    result.Success = actual >= min && actual <= max
    if !result.Success {
        result.Message = fmt.Sprintf("状态码 %d 不在范围 [%d, %d] 内", 
            actual, min, max)
    }
    
    return result
}
```

#### go-toolbox/pkg/random/business.go
```go
package random

import (
    "fmt"
    "strings"
    
    "github.com/kamalyes/go-toolbox/pkg/random"
)

// 邮箱域名列表
var emailDomains = []string{
    "gmail.com", "yahoo.com", "hotmail.com", "outlook.com",
    "qq.com", "163.com", "126.com", "sina.com",
}

// RandomEmail 生成随机邮箱
func RandomEmail() string {
    username := random.String(8, random.AlphaNum)
    domain := emailDomains[random.IntN(len(emailDomains))]
    return fmt.Sprintf("%s@%s", strings.ToLower(username), domain)
}

// RandomPhone 生成随机手机号（中国大陆）
func RandomPhone() string {
    prefixes := []string{"130", "131", "132", "133", "134", "135", "136", "137", "138", "139",
        "150", "151", "152", "153", "155", "156", "157", "158", "159",
        "180", "181", "182", "183", "184", "185", "186", "187", "188", "189"}
    
    prefix := prefixes[random.IntN(len(prefixes))]
    suffix := random.IntRange(10000000, 99999999)
    
    return fmt.Sprintf("%s%d", prefix, suffix)
}

// RandomName 生成随机姓名（中文）
func RandomName() string {
    surnames := []string{"王", "李", "张", "刘", "陈", "杨", "黄", "赵", "周", "吴"}
    names := []string{"伟", "芳", "娜", "秀英", "敏", "静", "丽", "强", "磊", "军"}
    
    surname := surnames[random.IntN(len(surnames))]
    
    // 60% 双字名，40% 单字名
    if random.IntN(100) < 60 {
        name1 := names[random.IntN(len(names))]
        name2 := names[random.IntN(len(names))]
        return surname + name1 + name2
    }
    
    name := names[random.IntN(len(names))]
    return surname + name
}

// RandomIDCard 生成随机身份证号（仅用于测试）
func RandomIDCard() string {
    // 地区码（随机）
    areaCode := fmt.Sprintf("%06d", random.IntRange(110000, 659999))
    
    // 出生日期（1960-2000）
    year := random.IntRange(1960, 2000)
    month := random.IntRange(1, 12)
    day := random.IntRange(1, 28)
    birthDate := fmt.Sprintf("%04d%02d%02d", year, month, day)
    
    // 顺序码
    sequence := fmt.Sprintf("%03d", random.IntRange(0, 999))
    
    // 前17位
    id17 := areaCode + birthDate + sequence
    
    // 计算校验码
    weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
    checkCodes := []string{"1", "0", "X", "9", "8", "7", "6", "5", "4", "3", "2"}
    
    sum := 0
    for i, c := range id17 {
        sum += int(c-'0') * weights[i]
    }
    checkCode := checkCodes[sum%11]
    
    return id17 + checkCode
}

// RandomCompany 生成随机公司名称
func RandomCompany() string {
    prefixes := []string{"阿里", "腾讯", "百度", "京东", "华为", "小米", "美团", "字节"}
    suffixes := []string{"科技", "网络", "信息", "技术", "软件", "互联网"}
    types := []string{"有限公司", "股份有限公司", "集团", "科技集团"}
    
    prefix := prefixes[random.IntN(len(prefixes))]
    suffix := suffixes[random.IntN(len(suffixes))]
    typeStr := types[random.IntN(len(types))]
    
    return fmt.Sprintf("%s%s%s", prefix, suffix, typeStr)
}
```

---

### 6.2 go-stress 重构清单

#### ✅ 第一阶段：syncx 模块替换（优先级：高）

- [ ] **config/variable.go**
  - [ ] 使用 `syncx.Uint64` 替换 `atomic.Uint64`（行27, 58, 427）
  - [ ] 预计减少: 5行锁管理代码

- [ ] **statistics/collector.go**
  - [ ] 使用 `syncx.Map[string, uint64]` 替换 `errors map + sync.Mutex`
  - [ ] 使用 `syncx.Map[int, uint64]` 替换 `statusCodes map + sync.Mutex`
  - [ ] 使用 `syncx.Uint64` 替换所有 `atomic.Uint64`（约20处）
  - [ ] 使用 `syncx.Pool[*RequestDetail]` 复用对象
  - [ ] 预计减少: 80行

- [ ] **statistics/realtime_server.go**
  - [ ] 使用 `syncx.Bool` 管理状态（isCompleted, isPaused, isStopped）
  - [ ] 使用 `syncx.Map[chan []byte, bool]` 管理客户端连接
  - [ ] 移除 `sync.RWMutex` 和 `sync.Once`
  - [ ] 预计减少: 30行

- [ ] **executor/pool.go**
  - [ ] 完全使用 `syncx.Pool[types.Client]` 替换自定义连接池
  - [ ] 预计减少: 50行（69%）

- [ ] **executor/scheduler.go**
  - [ ] 使用 `syncx.NewParallelSliceExecutor` 管理 Worker 并发
  - [ ] 添加 OnSuccess/OnError/OnComplete 回调
  - [ ] 预计减少: 20行

#### ✅ 第二阶段：mathx 模块替换（优先级：高）

- [ ] **config/variable.go**
  - [ ] 移除手动实现的 max/min/abs（行298-330）
  - [ ] 使用 `mathx.Max`, `mathx.Min`, `mathx.Abs`
  - [ ] 预计减少: 80行

- [ ] **statistics/collector.go**
  - [ ] 使用 `mathx.Percentiles` 批量计算百分位（行204-209）
  - [ ] 使用 `mathx.Mean` 和 `mathx.StdDev` 计算统计
  - [ ] 使用 `mathx.SummarizeStats` 生成完整摘要
  - [ ] 预计减少: 50行

- [ ] **statistics/html_report.go**
  - [ ] 使用 `mathx.Percentage` 计算百分比
  - [ ] 使用 `mathx.SortByCount` 排序统计数据
  - [ ] 预计减少: 40行

#### ✅ 第三阶段：validator 模块整合（优先级：中）

- [ ] **protocol/http_verify.go**
  - [ ] 使用 `validator.CompareStrings` 替换手动字符串比较（行200+）
  - [ ] 使用 `validator.CompareNumbers` 替换数值比较
  - [ ] 使用 `validator.ValidateStatusCode` 验证状态码
  - [ ] 使用 `validator.ValidateJSON` 验证JSON
  - [ ] 预计减少: 150行（45%）

- [ ] **config/config.go**
  - [ ] 添加 `Validate()` 方法使用 `validator.IsEmptyValue`
  - [ ] 预计增加: 20行（新增验证逻辑）

#### ✅ 第四阶段：convert & stringx 模块（优先级：中）

- [ ] **config/variable.go**
  - [ ] 使用 `convert.MustIntT` 替换 `strconv` 手动转换（行362-369）
  - [ ] 预计减少: 15行

- [ ] **protocol/http_verify.go**
  - [ ] 使用 `convert.MustIntT[float64]` 替换 `strconv.ParseFloat`
  - [ ] 预计减少: 10行

- [ ] **整体代码**
  - [ ] 全局搜索并替换字符串操作为 `stringx` 方法
  - [ ] 预计优化: 多处代码可读性提升

#### ✅ 第五阶段：serializer 模块（优先级：低）

- [ ] **statistics/report.go**
  - [ ] 使用 `serializer.New[*ReportData]()` 统一序列化
  - [ ] 移除手动的 `ToJSON()` 方法
  - [ ] 预计减少: 20行

- [ ] **statistics/html_report.go**
  - [ ] 使用 serializer 处理报告保存/加载
  - [ ] 支持压缩大数据报告
  - [ ] 预计减少: 15行

#### ✅ 第六阶段：报告系统统一（优先级：高）

- [ ] **统一数据结构**
  - [ ] 创建 `ReportData` 统一结构（替换 Report, HTMLReportData, RealtimeData）
  - [ ] 创建 `ReportMode` 枚举（static | realtime）
  - [ ] 预计减少: 120行重复定义

- [ ] **格式化器**
  - [ ] 创建 `ReportFormatter` 处理展示层转换
  - [ ] 支持 HTML/JSON/Text 多种输出格式
  - [ ] 预计增加: 50行，但消除大量重复代码

- [ ] **模板优化**
  - [ ] 统一 HTML 模板（report_html.go 和 realtime_server.go）
  - [ ] 预计减少: 100行

---

### 6.3 预期效果

| 模块 | 原代码行数 | 预计减少 | 优化后行数 | 减少比例 |
|:-----|:----------|:--------|:----------|:---------|
| **config/variable.go** | 473 | 100 | 373 | 21% |
| **statistics/collector.go** | 321 | 130 | 191 | 40% |
| **statistics/html_report.go** | 292 | 180 | 112 | 62% |
| **statistics/realtime_server.go** | 479 | 200 | 279 | 42% |
| **executor/pool.go** | 72 | 50 | 22 | 69% |
| **protocol/http_verify.go** | 336 | 150 | 186 | 45% |
| **总计** | ~2000 | ~810 | ~1190 | **40%** |

---

## 7. 实施步骤

### 第1步：准备 go-toolbox（1-2天）
1. 创建 `go-toolbox/pkg/mathx/stats.go`
2. 创建 `go-toolbox/pkg/validator/compare.go`
3. 创建 `go-toolbox/pkg/random/business.go`
4. 编写单元测试
5. 更新 go-toolbox 版本

### 第2步：重构统计模块（2-3天）
1. 重构 `statistics/collector.go` 使用 syncx + mathx
2. 统一报告数据结构 `ReportData`
3. 创建 `ReportFormatter` 格式化器
4. 重构 `html_report.go` 和 `realtime_server.go`
5. 更新单元测试

### 第3步：重构配置模块（1天）
1. 重构 `config/variable.go` 使用 mathx
2. 使用 convert 替换类型转换
3. 添加 Validate() 方法

### 第4步：重构验证模块（1-2天）
1. 重构 `protocol/http_verify.go` 使用 validator
2. 简化比较逻辑
3. 更新验证测试用例

### 第5步：重构执行器模块（1天）
1. 重构 `executor/pool.go` 使用 syncx.Pool
2. 重构 `executor/scheduler.go` 使用 Parallel 执行器
3. 优化并发控制

### 第6步：全面测试（2-3天）
1. 运行所有单元测试
2. 运行压测场景测试
3. 性能对比测试
4. 修复发现的问题

### 第7步：文档更新（1天）
1. 更新 README.md
2. 更新 USAGE.md
3. 添加迁移指南
4. 更新示例代码

---

## 8. 迁移示例

### 8.1 统计收集器迁移

**迁移前 (statistics/collector.go)**:
```go
type Collector struct {
    mu sync.Mutex
    
    totalRequests    atomic.Uint64
    successRequests  atomic.Uint64
    failedRequests   atomic.Uint64
    
    errors           map[string]uint64
    statusCodes      map[int]uint64
    durations        []float64
}

func (c *Collector) Collect(result *types.RequestResult) {
    c.totalRequests.Add(1)
    
    if result.Error != nil {
        c.failedRequests.Add(1)
        c.mu.Lock()
        c.errors[result.Error.Error()]++
        c.mu.Unlock()
    } else {
        c.successRequests.Add(1)
    }
    
    c.mu.Lock()
    if result.StatusCode > 0 {
        c.statusCodes[result.StatusCode]++
    }
    c.durations = append(c.durations, result.Duration.Seconds())
    c.mu.Unlock()
}

func (c *Collector) GetPercentile(p float64) float64 {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if len(c.durations) == 0 {
        return 0
    }
    
    sorted := make([]float64, len(c.durations))
    copy(sorted, c.durations)
    sort.Float64s(sorted)
    
    index := int(math.Ceil(float64(len(sorted)) * p / 100.0))
    if index >= len(sorted) {
        index = len(sorted) - 1
    }
    
    return sorted[index]
}
```

**迁移后**:
```go
import (
    "github.com/kamalyes/go-toolbox/pkg/syncx"
    "github.com/kamalyes/go-toolbox/pkg/mathx"
)

type Collector struct {
    // 原子计数器
    totalRequests   *syncx.Uint64
    successRequests *syncx.Uint64
    failedRequests  *syncx.Uint64
    
    // 线程安全的 Map
    errors      *syncx.Map[string, uint64]
    statusCodes *syncx.Map[int, uint64]
    
    // 时长列表（读多写少，仍用 mutex）
    mu        sync.RWMutex
    durations []float64
    
    // 对象池复用
    detailPool *syncx.Pool[*types.RequestDetail]
}

func NewCollector() *Collector {
    return &Collector{
        totalRequests:   syncx.NewUint64(0),
        successRequests: syncx.NewUint64(0),
        failedRequests:  syncx.NewUint64(0),
        errors:          syncx.NewMap[string, uint64](),
        statusCodes:     syncx.NewMap[int, uint64](),
        detailPool: syncx.NewPool(func() *types.RequestDetail {
            return &types.RequestDetail{}
        }),
    }
}

func (c *Collector) Collect(result *types.RequestResult) {
    // 原子操作，无需加锁
    c.totalRequests.Add(1)
    
    if result.Error != nil {
        c.failedRequests.Add(1)
        
        // syncx.Map 线程安全
        errMsg := result.Error.Error()
        old, _ := c.errors.LoadOrStore(errMsg, 0)
        c.errors.Store(errMsg, old+1)
    } else {
        c.successRequests.Add(1)
    }
    
    // 状态码统计
    if result.StatusCode > 0 {
        old, _ := c.statusCodes.LoadOrStore(result.StatusCode, 0)
        c.statusCodes.Store(result.StatusCode, old+1)
    }
    
    // 时长记录（仍需加锁）
    c.mu.Lock()
    c.durations = append(c.durations, result.Duration.Seconds())
    c.mu.Unlock()
}

// 使用 mathx 批量计算百分位
func (c *Collector) GetPercentiles() map[float64]float64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    // 一次性计算所有需要的百分位
    return mathx.Percentiles(c.durations, 50, 90, 95, 99)
}

// 生成统计摘要
func (c *Collector) GetStatsSummary() mathx.StatsSummary {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    return mathx.SummarizeStats(c.durations)
}
```

---

### 8.2 HTTP验证器迁移

**迁移前 (protocol/http_verify.go)**:
```go
func (v *HTTPVerifier) compareValues(actual, expect interface{}, operator ExpectOperator) (bool, string) {
    actualStr := fmt.Sprintf("%v", actual)
    expectStr := fmt.Sprintf("%v", expect)
    
    switch operator {
    case OpEqual:
        return actualStr == expectStr, ""
    case OpNotEqual:
        return actualStr != expectStr, ""
    case OpContains:
        return strings.Contains(actualStr, expectStr), ""
    case OpNotContains:
        return !strings.Contains(actualStr, expectStr), ""
    case OpHasPrefix:
        return strings.HasPrefix(actualStr, expectStr), ""
    case OpHasSuffix:
        return strings.HasSuffix(actualStr, expectStr), ""
    case OpGT:
        return v.compareNumeric(actualStr, expectStr, OpGT)
    case OpGTE:
        return v.compareNumeric(actualStr, expectStr, OpGTE)
    // ... 更多case
    }
}

func (v *HTTPVerifier) compareNumeric(actualStr, expectStr string, op ExpectOperator) (bool, string) {
    actualNum, err1 := strconv.ParseFloat(actualStr, 64)
    expectNum, err2 := strconv.ParseFloat(expectStr, 64)
    
    if err1 != nil || err2 != nil {
        return false, "数值比较失败: 无法解析为数字"
    }
    
    switch op {
    case OpGT:
        return actualNum > expectNum, ""
    case OpGTE:
        return actualNum >= expectNum, ""
    // ... 更多case
    }
}
```

**迁移后**:
```go
import (
    "github.com/kamalyes/go-toolbox/pkg/validator"
    "github.com/kamalyes/go-toolbox/pkg/convert"
)

func (v *HTTPVerifier) compareValues(actual, expect interface{}, operator ExpectOperator) (bool, string) {
    actualStr := fmt.Sprintf("%v", actual)
    expectStr := fmt.Sprintf("%v", expect)
    
    // 尝试数值比较
    if v.isNumericOperator(operator) {
        return v.compareNumeric(actualStr, expectStr, operator)
    }
    
    // 字符串比较 - 直接使用 validator
    result := validator.CompareStrings(actualStr, expectStr, 
        validator.CompareOperator(operator))
    
    return result.Success, result.Message
}

func (v *HTTPVerifier) compareNumeric(actualStr, expectStr string, operator ExpectOperator) (bool, string) {
    // 使用 convert 模块转换
    actualNum, err1 := convert.MustIntT[float64](actualStr, nil)
    expectNum, err2 := convert.MustIntT[float64](expectStr, nil)
    
    if err1 != nil || err2 != nil {
        return false, "数值比较失败: 无法解析为数字"
    }
    
    // 使用 validator 比较数值
    result := validator.CompareNumbers(actualNum, expectNum, 
        validator.CompareOperator(operator))
    
    return result.Success, result.Message
}

func (v *HTTPVerifier) verifyStatusCode(resp *Response) (bool, error) {
    expectedCode := 200
    // ... 解析 expect
    
    // 使用 validator 验证状态码
    result := validator.ValidateStatusCode(resp.StatusCode, expectedCode)
    
    // 记录验证结果
    resp.Verifications = append(resp.Verifications, VerificationResult{
        Type:    v.config.Type,
        Success: result.Success,
        Message: result.Message,
        Expect:  result.Expect,
        Actual:  result.Actual,
    })
    
    if !result.Success {
        return false, fmt.Errorf(result.Message)
    }
    return true, nil
}
```

---

### 8.3 报告系统迁移

**迁移前 (3套数据结构)**:
```go
// statistics/report.go
type Report struct {
    TotalRequests   uint64
    SuccessRate     float64  // 原始数值
    // ...
}

// statistics/html_report.go
type HTMLReportData struct {
    TotalRequests   uint64
    SuccessRate     string  // 格式化后的字符串
    // ...
}

// statistics/realtime_server.go
type RealtimeData struct {
    Timestamp       int64
    TotalRequests   uint64
    SuccessRate     float64
    // ...
}
```

**迁移后 (统一数据结构)**:
```go
// types/statistics.go - 统一数据模型
type ReportMode string

const (
    ReportModeStatic   ReportMode = "static"
    ReportModeRealtime ReportMode = "realtime"
)

// ReportData 统一的报告数据结构（所有字段都是原始类型）
type ReportData struct {
    Mode         ReportMode    `json:"mode"`
    GenerateTime time.Time     `json:"generate_time"`
    
    // 基础统计
    TotalRequests   uint64  `json:"total_requests"`
    SuccessRequests uint64  `json:"success_requests"`
    FailedRequests  uint64  `json:"failed_requests"`
    SuccessRate     float64 `json:"success_rate"`  // 0-100
    
    // 性能指标
    QPS             float64       `json:"qps"`
    MinDuration     time.Duration `json:"min_duration"`
    MaxDuration     time.Duration `json:"max_duration"`
    AvgDuration     time.Duration `json:"avg_duration"`
    P50Duration     time.Duration `json:"p50_duration"`
    P90Duration     time.Duration `json:"p90_duration"`
    P95Duration     time.Duration `json:"p95_duration"`
    P99Duration     time.Duration `json:"p99_duration"`
    
    // 数据量
    TotalSize       float64 `json:"total_size"`  // bytes
    
    // 统计详情
    ErrorStats      []ErrorStat      `json:"error_stats"`
    StatusCodeStats []StatusCodeStat `json:"status_code_stats"`
    RequestDetails  []RequestDetail  `json:"request_details,omitempty"`
}

// statistics/formatter.go - 格式化器
type ReportFormatter struct {
    data *types.ReportData
}

func NewReportFormatter(data *types.ReportData) *ReportFormatter {
    return &ReportFormatter{data: data}
}

// 格式化方法
func (f *ReportFormatter) FormatSuccessRate() string {
    return fmt.Sprintf("%.2f%%", f.data.SuccessRate)
}

func (f *ReportFormatter) FormatQPS() string {
    return fmt.Sprintf("%.2f", f.data.QPS)
}

func (f *ReportFormatter) FormatDuration(d time.Duration) string {
    if d < time.Millisecond {
        return fmt.Sprintf("%.2fμs", float64(d.Microseconds()))
    }
    if d < time.Second {
        return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
    }
    return fmt.Sprintf("%.2fs", d.Seconds())
}

func (f *ReportFormatter) FormatSize() string {
    return units.BytesSize(f.data.TotalSize)
}

// 生成 HTML 数据
func (f *ReportFormatter) ToHTMLData() map[string]interface{} {
    return map[string]interface{}{
        "Mode":            f.data.Mode,
        "GenerateTime":    f.data.GenerateTime.Format("2006-01-02 15:04:05"),
        "TotalRequests":   f.data.TotalRequests,
        "SuccessRequests": f.data.SuccessRequests,
        "FailedRequests":  f.data.FailedRequests,
        "SuccessRate":     f.FormatSuccessRate(),
        "QPS":             f.FormatQPS(),
        "MinDuration":     f.FormatDuration(f.data.MinDuration),
        "MaxDuration":     f.FormatDuration(f.data.MaxDuration),
        "AvgDuration":     f.FormatDuration(f.data.AvgDuration),
        "P50Duration":     f.FormatDuration(f.data.P50Duration),
        "P90Duration":     f.FormatDuration(f.data.P90Duration),
        "P95Duration":     f.FormatDuration(f.data.P95Duration),
        "P99Duration":     f.FormatDuration(f.data.P99Duration),
        "TotalSize":       f.FormatSize(),
        "ErrorStats":      f.data.ErrorStats,
        "StatusCodeStats": f.data.StatusCodeStats,
    }
}
```

---

## 9. 性能对比

### 9.1 预期性能提升

| 操作 | 优化前 | 优化后 | 提升 |
|:-----|:------|:------|:-----|
| **Map并发写入** | sync.Mutex + map | syncx.Map | ~30% |
| **原子操作** | atomic.Uint64 | syncx.Uint64 | 持平（API更友好） |
| **对象创建** | 每次new | syncx.Pool | ~50-70% |
| **百分位计算** | 每次排序 | 批量计算 | ~40% |
| **Worker并发** | sync.WaitGroup | syncx.Parallel | ~21% (官方benchmark) |

### 9.2 代码质量提升

- ✅ 减少代码重复：~40%
- ✅ 提高可维护性：统一数据结构
- ✅ 增强可测试性：模块化设计
- ✅ 改善可读性：使用高级API

---

## 10. 风险评估与应对

### 10.1 潜在风险

| 风险 | 影响 | 概率 | 应对措施 |
|:-----|:-----|:-----|:---------|
| **API不兼容** | 高 | 中 | 保留旧API，逐步迁移 |
| **性能回退** | 中 | 低 | 性能测试，基准对比 |
| **引入Bug** | 高 | 中 | 充分的单元测试和集成测试 |
| **学习成本** | 低 | 高 | 详细文档和示例代码 |

### 10.2 回滚策略

1. **Git分支管理**：在独立分支进行重构
2. **功能开关**：保留旧代码路径，通过配置切换
3. **版本标记**：打标签，可快速回滚
4. **灰度发布**：逐步替换关键模块

---

## 11. 总结

### 11.1 重构价值

1. **代码质量**：预计减少 40% 重复代码（~810行）
2. **性能提升**：syncx 并发优化、对象池复用
3. **可维护性**：统一数据结构，模块化设计
4. **可扩展性**：充分利用 go-toolbox 生态

### 11.2 关键改进点

- ✅ **syncx 深度应用**：Map, Set, Pool, Atomic, Parallel
- ✅ **mathx 统计优化**：批量百分位计算，统计摘要
- ✅ **validator 验证统一**：抽离公共验证逻辑
- ✅ **报告系统统一**：3套结构合并为1套
- ✅ **go-toolbox 扩展**：新增 stats.go, compare.go, business.go

### 11.3 下一步行动

1. ✅ **Review 本文档**：团队评审，确认方案
2. **准备 go-toolbox**：添加新模块并测试
3. **分阶段实施**：按优先级逐步重构
4. **持续测试**：单元测试 + 压测验证
5. **更新文档**：保持文档与代码同步

---

## 附录

### A. go-toolbox 完整模块清单

| 模块 | 说明 | go-stress 使用 |
|:-----|:-----|:--------------|
| **mathx** | 数学函数、统计 | ✅ Max/Min/Abs, 百分位, 统计 |
| **syncx** | 并发安全组件 | ✅ Map/Set/Pool/Atomic/Parallel |
| **convert** | 类型转换 | ✅ 替换 strconv |
| **retry** | 重试机制 | 🔄 HTTP请求重试 |
| **stringx** | 字符串操作 | ✅ ToUpper/ToLower/Contains |
| **serializer** | 序列化 | ✅ JSON序列化、压缩 |
| **validator** | 验证工具 | ✅ 比较验证、状态码验证 |
| **random** | 随机工具 | ✅ 业务数据生成 |
| **units** | 单位转换 | ✅ BytesSize |
| **errorx** | 错误处理 | 🔄 错误包装 |
| **httpx** | HTTP工具 | 🔄 连接池（可选） |

**说明**：
- ✅ 已规划使用
- 🔄 未来可选

### B. 参考资料

- [go-toolbox 文档](https://github.com/kamalyes/go-toolbox)
- [syncx 性能测试报告](https://github.com/kamalyes/go-toolbox/tree/main/pkg/syncx#benchmarks)
- [Go 并发模式](https://go.dev/blog/pipelines)
- [压测工具最佳实践](https://www.oreilly.com/library/view/high-performance-browser/9781449344757/)

---

**文档版本**: 1.0  
**创建日期**: 2024-01-XX  
**最后更新**: 2024-01-XX  
**作者**: go-stress 开发团队

#### 2.1.2 重构方案

```go
// 使用 go-toolbox/pkg/mathx
import "github.com/kamalyes/go-toolbox/pkg/mathx"

// 在 variable.go 的 funcMap 中
"max": func(a, b int) int {
    return mathx.AtMost(a, b) // 返回最大值
},
"min": func(a, b int) int {
    return mathx.AtLeast(a, b) // 返回最小值
},
"abs": mathx.Abs[int], // 直接使用泛型版本
"between": mathx.Between[int], // 限制在范围内
```

#### 2.1.3 新增功能

```go
// 百分位计算优化（statistics/collector.go）
func (c *Collector) percentile(p float64) time.Duration {
    n := len(c.durations)
    if n == 0 {
        return 0
    }
    
    // 使用 mathx.Between 确保索引在有效范围内
    index := mathx.Between(
        int(float64(n)*p),
        0,
        n-1,
    )
    return c.durations[index]
}

// 数据验证增强
func validateMetric(value float64) float64 {
    // 确保值在合理范围内
    return mathx.Between(value, 0.0, math.MaxFloat64)
}
```

---

### 2.2 syncx 模块

#### 2.2.1 当前问题

```go
// statistics/collector.go - 使用标准库 atomic
atomic.AddUint64(&c.totalRequests, 1)
atomic.LoadUint64(&c.totalRequests)

// 使用 sync.Mutex
mu sync.Mutex
mu.Lock()
defer mu.Unlock()
```

#### 2.2.2 重构方案

```go
// 替换为 syncx 原子类型
import "github.com/kamalyes/go-toolbox/pkg/syncx"

type Collector struct {
    // 原子计数器
    totalRequests   *syncx.Uint64
    successRequests *syncx.Uint64
    failedRequests  *syncx.Uint64
    
    // 使用泛型对象池
    detailPool *syncx.Pool[*RequestDetail]
    
    mu sync.Mutex
    // ... 其他字段
}

func NewCollector() *Collector {
    return &Collector{
        totalRequests:   syncx.NewUint64(0),
        successRequests: syncx.NewUint64(0),
        failedRequests:  syncx.NewUint64(0),
        detailPool: syncx.NewPool(func() *RequestDetail {
            return &RequestDetail{}
        }),
        // ...
    }
}

func (c *Collector) Collect(result *types.RequestResult) {
    c.totalRequests.Add(1)
    
    if result.Success {
        c.successRequests.Add(1)
    } else {
        c.failedRequests.Add(1)
    }
    
    // 从对象池获取
    detail := c.detailPool.Get()
    // ... 填充数据
    
    // 使用完后放回池中
    defer c.detailPool.Put(detail)
}
```

#### 2.2.3 连接池优化

```go
// executor/pool.go - 使用 syncx.Pool 替代自定义实现
type ClientPool struct {
    factory ClientFactory
    pool    *syncx.Pool[types.Client]
    maxSize int
}

func NewClientPool(factory ClientFactory, maxSize int) *ClientPool {
    return &ClientPool{
        factory: factory,
        maxSize: maxSize,
        pool: syncx.NewPool(func() types.Client {
            client, _ := factory()
            return client
        }),
    }
}
```

---

### 2.3 convert 模块

#### 2.3.1 当前问题

```go
// config/variable.go - 手动类型转换
"toInt": func(s string) int {
    i, _ := strconv.Atoi(s)
    return i
},
"toFloat": func(s string) float64 {
    f, _ := strconv.ParseFloat(s, 64)
    return f
},
```

#### 2.3.2 重构方案

```go
// 使用 go-toolbox/pkg/convert
import "github.com/kamalyes/go-toolbox/pkg/convert"

// 在 variable.go 的 funcMap 中
"toInt": func(s string) int {
    v, _ := convert.MustIntT[int](s, nil)
    return v
},
"toInt64": func(s string) int64 {
    v, _ := convert.MustIntT[int64](s, nil)
    return v
},
"toFloat": func(s string) float64 {
    v, _ := convert.MustIntT[float64](s, nil)
    return v
},
"toString": convert.MustString[any],

// 四舍五入模式
"roundUp": func(s string) int {
    mode := convert.RoundUp
    v, _ := convert.MustIntT[int](s, &mode)
    return v
},
"roundDown": func(s string) int {
    mode := convert.RoundDown
    v, _ := convert.MustIntT[int](s, &mode)
    return v
},
```

#### 2.3.3 统一类型转换

```go
// protocol/http_verify.go - 统一数值比较
func compareNumbers(actual, expect string) bool {
    // 使用 convert 统一处理类型转换
    actualNum, err1 := convert.MustIntT[float64](actual, nil)
    expectNum, err2 := convert.MustIntT[float64](expect, nil)
    
    if err1 != nil || err2 != nil {
        return false
    }
    
    return actualNum == expectNum
}
```

---

### 2.4 retry 模块

#### 2.4.1 当前使用

```go
// executor/middleware.go - 已经在使用
func RetryMiddleware(retrier *retry.Runner[error]) Middleware {
    return func(next RequestHandler) RequestHandler {
        return func(ctx context.Context, req *types.Request) (*types.Response, error) {
            _, retryErr := retrier.Run(func(retryCtx context.Context) (error, error) {
                resp, err := next(ctx, req)
                return err, err
            })
            // ...
        }
    }
}
```

#### 2.4.2 增强建议

```go
// 配置更详细的重试策略
func buildRetryMiddleware(cfg *config.Config) Middleware {
    retrier := retry.NewRunner[error]().
        Timeout(cfg.Advanced.RetryTimeout).
        OnSuccess(func(result error, err error) {
            logger.Default.Debug("请求重试成功")
        }).
        OnError(func(result error, err error) {
            logger.Default.Warn("请求重试失败: %v", err)
        })
    
    return RetryMiddleware(retrier)
}
```

---

### 2.5 httpx 模块

#### 2.5.1 当前 httpx 能力

根据代码分析，go-toolbox/pkg/httpx 已经提供：

- ✅ HTTP 客户端封装
- ✅ 请求/响应处理
- ✅ Cookie 管理
- ✅ 参数构建
- ✅ 错误处理
- ✅ URL 工具

#### 2.5.2 建议在 httpx 中新增

```go
// go-toolbox/pkg/httpx/pool.go - 新增连接池支持
package httpx

type ClientPool struct {
    pool *syncx.Pool[*Client]
    opts []ClientOption
}

func NewClientPool(maxSize int, opts ...ClientOption) *ClientPool {
    return &ClientPool{
        opts: opts,
        pool: syncx.NewPool(func() *Client {
            return NewClient(opts...)
        }),
    }
}

func (p *ClientPool) Get() *Client {
    return p.pool.Get()
}

func (p *ClientPool) Put(client *Client) {
    p.pool.Put(client)
}
```

```go
// go-toolbox/pkg/httpx/metrics.go - 新增请求指标收集
package httpx

type RequestMetrics struct {
    StartTime    time.Time
    EndTime      time.Time
    Duration     time.Duration
    StatusCode   int
    RequestSize  int64
    ResponseSize int64
    Error        error
}

type MetricsCollector interface {
    Collect(metrics *RequestMetrics)
}

// 为 Client 添加指标收集
func (c *Client) WithMetrics(collector MetricsCollector) *Client {
    // 实现请求拦截和指标收集
    return c
}
```

---

## 3. 报告系统重构

### 3.1 当前问题分析

#### 3.1.1 数据结构冗余

```go
// statistics/html_report.go - 当前有两套数据结构

// 1. HTMLReportData - 用于模板渲染
type HTMLReportData struct {
    IsRealtime      bool
    GenerateTime    string
    TotalRequests   uint64
    SuccessRate     string  // 格式化后的字符串
    QPS             string  // 格式化后的字符串
    // ...
}

// 2. Report - 用于实际统计
type Report struct {
    TotalRequests   uint64
    SuccessRate     float64  // 原始数值
    QPS             float64  // 原始数值
    // ...
}

// 3. RequestDetail vs RequestDetailDisplay - 重复结构
```

**问题**：
- 数据重复定义
- 格式化逻辑分散
- 静态/实时模式需要不同的数据处理
- 维护成本高

### 3.2 统一数据模型方案

#### 3.2.1 核心数据结构

```go
// statistics/types.go - 新建统一的类型定义文件
package statistics

import (
    "time"
    "github.com/kamalyes/go-toolbox/pkg/convert"
)

// ReportData 统一的报告数据结构（用于所有场景）
type ReportData struct {
    // 元数据
    Mode         ReportMode `json:"mode"`          // static | realtime
    GenerateTime time.Time  `json:"generate_time"`
    TestDuration time.Duration `json:"test_duration"`
    
    // 基础统计（原始数据）
    TotalRequests   uint64  `json:"total_requests"`
    SuccessRequests uint64  `json:"success_requests"`
    FailedRequests  uint64  `json:"failed_requests"`
    SuccessRate     float64 `json:"success_rate"` // 0-100
    
    // 性能指标（原始数据）
    QPS         float64       `json:"qps"`
    TotalSize   float64       `json:"total_size"`
    AvgDuration time.Duration `json:"avg_duration"`
    MinDuration time.Duration `json:"min_duration"`
    MaxDuration time.Duration `json:"max_duration"`
    
    // 百分位数据
    P50 time.Duration `json:"p50"`
    P90 time.Duration `json:"p90"`
    P95 time.Duration `json:"p95"`
    P99 time.Duration `json:"p99"`
    
    // 错误和状态码统计
    ErrorStats      []ErrorStat      `json:"error_stats"`
    StatusCodeStats []StatusCodeStat `json:"status_code_stats"`
    
    // 请求明细
    RequestDetails []RequestDetail `json:"request_details"`
}

// ReportMode 报告模式
type ReportMode string

const (
    ReportModeStatic   ReportMode = "static"
    ReportModeRealtime ReportMode = "realtime"
)

// ErrorStat 错误统计
type ErrorStat struct {
    Error      string  `json:"error"`
    Count      uint64  `json:"count"`
    Percentage float64 `json:"percentage"`
}

// StatusCodeStat 状态码统计
type StatusCodeStat struct {
    StatusCode int     `json:"status_code"`
    Count      uint64  `json:"count"`
    Percentage float64 `json:"percentage"`
}

// RequestDetail 请求明细（统一结构）
type RequestDetail struct {
    // 基础信息
    ID         uint64    `json:"id"`
    Timestamp  time.Time `json:"timestamp"`
    GroupID    uint64    `json:"group_id,omitempty"`
    APIName    string    `json:"api_name,omitempty"`
    
    // 请求信息
    URL     string            `json:"url,omitempty"`
    Method  string            `json:"method,omitempty"`
    Query   string            `json:"query,omitempty"`
    Headers map[string]string `json:"headers,omitempty"`
    Body    string            `json:"body,omitempty"`
    
    // 响应信息
    Duration        time.Duration     `json:"duration"`
    StatusCode      int               `json:"status_code"`
    Success         bool              `json:"success"`
    Skipped         bool              `json:"skipped,omitempty"`
    Size            float64           `json:"size"`
    ResponseBody    string            `json:"response_body,omitempty"`
    ResponseHeaders map[string]string `json:"response_headers,omitempty"`
    Error           string            `json:"error,omitempty"`
    
    // 验证和变量
    Verifications []VerificationResult `json:"verifications,omitempty"`
    ExtractedVars map[string]string    `json:"extracted_vars,omitempty"`
}

// VerificationResult 验证结果
type VerificationResult struct {
    Type    string `json:"type"`
    Success bool   `json:"success"`
    Message string `json:"message,omitempty"`
}
```

#### 3.2.2 格式化助手（视图层）

```go
// statistics/formatter.go - 新建格式化工具
package statistics

import (
    "fmt"
    "github.com/kamalyes/go-toolbox/pkg/units"
    "github.com/kamalyes/go-toolbox/pkg/convert"
)

// ReportFormatter 报告格式化器（用于展示）
type ReportFormatter struct {
    data *ReportData
}

func NewFormatter(data *ReportData) *ReportFormatter {
    return &ReportFormatter{data: data}
}

// FormatSuccessRate 格式化成功率
func (f *ReportFormatter) FormatSuccessRate() string {
    return fmt.Sprintf("%.2f%%", f.data.SuccessRate)
}

// FormatQPS 格式化 QPS
func (f *ReportFormatter) FormatQPS() string {
    return fmt.Sprintf("%.2f", f.data.QPS)
}

// FormatSize 格式化数据大小
func (f *ReportFormatter) FormatSize() string {
    return units.BytesSize(f.data.TotalSize)
}

// FormatDuration 格式化时间
func (f *ReportFormatter) FormatDuration(d time.Duration) string {
    return d.String()
}

// FormatTimestamp 格式化时间戳
func (f *ReportFormatter) FormatTimestamp(t time.Time) string {
    return t.Format(time.DateTime)
}

// ToTemplateData 转换为模板数据（向后兼容）
func (f *ReportFormatter) ToTemplateData() map[string]interface{} {
    return map[string]interface{}{
        // 原始数据（供 JS 使用）
        "data": f.data,
        
        // 格式化后的数据（供 HTML 展示）
        "formatted": map[string]string{
            "generate_time":  f.FormatTimestamp(f.data.GenerateTime),
            "test_duration":  f.FormatDuration(f.data.TestDuration),
            "success_rate":   f.FormatSuccessRate(),
            "qps":            f.FormatQPS(),
            "total_size":     f.FormatSize(),
            "avg_duration":   f.FormatDuration(f.data.AvgDuration),
            "min_duration":   f.FormatDuration(f.data.MinDuration),
            "max_duration":   f.FormatDuration(f.data.MaxDuration),
            "p50":            f.FormatDuration(f.data.P50),
            "p90":            f.FormatDuration(f.data.P90),
            "p95":            f.FormatDuration(f.data.P95),
            "p99":            f.FormatDuration(f.data.P99),
        },
    }
}
```

#### 3.2.3 报告生成器重构

```go
// statistics/collector.go - 重构生成方法
func (c *Collector) GenerateReportData(totalTime time.Duration, mode ReportMode) *ReportData {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    data := &ReportData{
        Mode:            mode,
        GenerateTime:    time.Now(),
        TestDuration:    totalTime,
        TotalRequests:   c.totalRequests.Load(),
        SuccessRequests: c.successRequests.Load(),
        FailedRequests:  c.failedRequests.Load(),
        TotalSize:       c.totalSize,
        MinDuration:     c.minDuration,
        MaxDuration:     c.maxDuration,
    }
    
    // 计算派生指标
    if data.TotalRequests > 0 {
        data.SuccessRate = float64(data.SuccessRequests) / float64(data.TotalRequests) * 100
        data.AvgDuration = c.totalDuration / time.Duration(data.TotalRequests)
        data.QPS = float64(data.TotalRequests) / totalTime.Seconds()
    }
    
    // 排序并计算百分位
    sort.Slice(c.durations, func(i, j int) bool {
        return c.durations[i] < c.durations[j]
    })
    
    if len(c.durations) > 0 {
        data.P50 = c.percentile(0.50)
        data.P90 = c.percentile(0.90)
        data.P95 = c.percentile(0.95)
        data.P99 = c.percentile(0.99)
    }
    
    // 错误统计
    data.ErrorStats = make([]ErrorStat, 0, len(c.errors))
    for err, count := range c.errors {
        percentage := float64(count) / float64(data.TotalRequests) * 100
        data.ErrorStats = append(data.ErrorStats, ErrorStat{
            Error:      err,
            Count:      count,
            Percentage: percentage,
        })
    }
    sort.Slice(data.ErrorStats, func(i, j int) bool {
        return data.ErrorStats[i].Count > data.ErrorStats[j].Count
    })
    
    // 状态码统计
    data.StatusCodeStats = make([]StatusCodeStat, 0, len(c.statusCodes))
    for code, count := range c.statusCodes {
        percentage := float64(count) / float64(data.TotalRequests) * 100
        data.StatusCodeStats = append(data.StatusCodeStats, StatusCodeStat{
            StatusCode: code,
            Count:      count,
            Percentage: percentage,
        })
    }
    sort.Slice(data.StatusCodeStats, func(i, j int) bool {
        return data.StatusCodeStats[i].StatusCode < data.StatusCodeStats[j].StatusCode
    })
    
    // 复制请求明细
    data.RequestDetails = make([]RequestDetail, len(c.requestDetails))
    copy(data.RequestDetails, c.requestDetails)
    
    return data
}

// 废弃旧的 GenerateReport 方法（向后兼容）
func (c *Collector) GenerateReport(totalTime time.Duration) *Report {
    data := c.GenerateReportData(totalTime, ReportModeStatic)
    return convertToLegacyReport(data)
}
```

#### 3.2.4 HTML 报告生成器简化

```go
// statistics/html_report.go - 大幅简化
func (c *Collector) GenerateHTMLReport(totalTime time.Duration, filename string) error {
    // 1. 生成统一数据模型
    data := c.GenerateReportData(totalTime, ReportModeStatic)
    
    // 2. 保存 JSON 数据文件
    jsonFilename := strings.TrimSuffix(filename, ".html") + ".json"
    if err := data.SaveToFile(jsonFilename); err != nil {
        return fmt.Errorf("保存JSON数据失败: %w", err)
    }
    
    // 3. 创建格式化器
    formatter := NewFormatter(data)
    
    // 4. 生成静态资源文件
    reportDir := filepath.Dir(filename)
    if err := generateStaticFiles(reportDir, false, filepath.Base(jsonFilename)); err != nil {
        return err
    }
    
    // 5. 渲染 HTML
    tmpl, err := template.New("report").Parse(unifiedReportHTML)
    if err != nil {
        return fmt.Errorf("解析模板失败: %w", err)
    }
    
    file, err := os.Create(filename)
    if err != nil {
        return fmt.Errorf("创建文件失败: %w", err)
    }
    defer file.Close()
    
    // 传递统一的数据结构
    return tmpl.Execute(file, formatter.ToTemplateData())
}
```

#### 3.2.5 实时报告服务器简化

```go
// statistics/realtime_server.go - 使用统一数据结构
func (s *RealtimeServer) handleData(w http.ResponseWriter, r *http.Request) {
    // 生成实时数据
    elapsed := time.Since(s.startTime)
    data := s.collector.GenerateReportData(elapsed, ReportModeRealtime)
    
    // 直接序列化统一数据结构
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

func (s *RealtimeServer) Start() error {
    // 生成静态资源（实时模式）
    if err := generateStaticFiles(".", true, ""); err != nil {
        return err
    }
    
    // ... 启动 HTTP 服务器
}
```

### 3.3 统一模板系统

#### 3.3.1 新的 HTML 模板

```go
// statistics/unified_template.go - 更新模板
const unifiedReportHTML = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>压测报告 - {{.data.Mode}}</title>
    <link rel="stylesheet" href="report.css">
    <script src="https://cdn.jsdelivr.net/npm/echarts@5.4.3/dist/echarts.min.js"></script>
</head>
<body>
    <div class="container">
        <header>
            <h1>🚀 go-stress 压测报告</h1>
            <div class="meta">
                <span>模式: {{.data.Mode}}</span>
                <span>生成时间: {{.formatted.generate_time}}</span>
                <span>测试时长: {{.formatted.test_duration}}</span>
            </div>
        </header>
        
        <!-- 数据注入到 JS -->
        <script>
            window.REPORT_DATA = {{json .data}};
            window.IS_REALTIME = {{eq .data.Mode "realtime"}};
        </script>
        
        <!-- 统计概览 -->
        <section id="overview"></section>
        
        <!-- 图表区域 -->
        <section id="charts"></section>
        
        <!-- 请求明细表格 -->
        <section id="details"></section>
    </div>
    
    <script src="report.js"></script>
</body>
</html>
`
```

#### 3.3.2 JavaScript 模块化

```javascript
// statistics/report.js - 统一的 JS 处理
(function() {
    'use strict';
    
    // 从全局变量获取数据
    const reportData = window.REPORT_DATA;
    const isRealtime = window.IS_REALTIME;
    
    // 格式化工具
    const formatter = {
        duration(ms) {
            if (ms < 1000) return `${ms.toFixed(2)} ms`;
            return `${(ms/1000).toFixed(2)} s`;
        },
        size(bytes) {
            const units = ['B', 'KB', 'MB', 'GB'];
            let size = bytes;
            let unitIndex = 0;
            while (size >= 1024 && unitIndex < units.length - 1) {
                size /= 1024;
                unitIndex++;
            }
            return `${size.toFixed(2)} ${units[unitIndex]}`;
        },
        percent(value) {
            return `${value.toFixed(2)}%`;
        }
    };
    
    // 渲染概览
    function renderOverview() {
        const html = `
            <div class="stats-grid">
                <div class="stat-card">
                    <h3>总请求数</h3>
                    <p>${reportData.total_requests}</p>
                </div>
                <div class="stat-card">
                    <h3>成功率</h3>
                    <p>${formatter.percent(reportData.success_rate)}</p>
                </div>
                <div class="stat-card">
                    <h3>QPS</h3>
                    <p>${reportData.qps.toFixed(2)}</p>
                </div>
                <!-- 更多卡片 -->
            </div>
        `;
        document.getElementById('overview').innerHTML = html;
    }
    
    // 实时更新（如果是实时模式）
    function startRealtimeUpdate() {
        if (!isRealtime) return;
        
        setInterval(async () => {
            try {
                const response = await fetch('/api/data');
                const newData = await response.json();
                Object.assign(reportData, newData);
                renderAll();
            } catch (error) {
                console.error('更新失败:', error);
            }
        }, 1000);
    }
    
    // 渲染所有组件
    function renderAll() {
        renderOverview();
        renderCharts();
        renderDetails();
    }
    
    // 初始化
    document.addEventListener('DOMContentLoaded', () => {
        renderAll();
        startRealtimeUpdate();
    });
})();
```

### 3.4 重构优势总结

| 对比项 | 重构前 | 重构后 |
|:-------|:-------|:-------|
| **数据结构** | 3个独立结构 | 1个统一结构 |
| **代码行数** | ~800行 | ~400行 |
| **维护成本** | 高（数据同步） | 低（单一数据源） |
| **类型安全** | 字符串类型混乱 | 统一类型+格式化层 |
| **扩展性** | 困难（需修改多处） | 容易（只改一处） |
| **JSON输出** | 需要转换 | 直接序列化 |
| **前端集成** | 数据格式不一致 | 统一 JSON 接口 |

---

## 4. 代码优化清单

### 4.1 config/variable.go

**优化项**：

- [ ] 替换手动实现的 `max`/`min`/`abs` 为 `mathx` 模块
- [ ] 使用 `convert.MustIntT` 替换 `strconv.Atoi`
- [ ] 使用 `convert.MustString` 统一字符串转换
- [ ] 添加类型安全的泛型转换函数

**代码量减少**：约 50 行

### 4.2 statistics/collector.go

**优化项**：

- [ ] 替换 `atomic.AddUint64` 为 `syncx.Uint64`
- [ ] 使用 `syncx.Pool` 优化 RequestDetail 对象复用
- [ ] 使用 `mathx.Between` 优化百分位计算
- [ ] 统一使用 `ReportData` 数据结构

**代码量减少**：约 100 行

### 4.3 statistics/html_report.go

**优化项**：

- [ ] 移除 `HTMLReportData` 结构，使用统一的 `ReportData`
- [ ] 移除 `RequestDetailDisplay`，使用统一的 `RequestDetail`
- [ ] 创建 `ReportFormatter` 处理格式化逻辑
- [ ] 简化 HTML 生成流程

**代码量减少**：约 150 行

### 4.4 statistics/realtime_server.go

**优化项**：

- [ ] 使用统一的 `ReportData` 结构
- [ ] 移除重复的数据转换代码
- [ ] 使用 `syncx.AtomicValue` 优化状态管理

**代码量减少**：约 80 行

### 4.5 executor/pool.go

**优化项**：

- [ ] 使用 `syncx.Pool` 替换自定义连接池
- [ ] 简化 Get/Put 逻辑

**代码量减少**：约 40 行

### 4.6 protocol/http_verify.go

**优化项**：

- [ ] 使用 `convert.MustIntT` 统一数值转换
- [ ] 移除手动的类型判断代码

**代码量减少**：约 30 行

---

## 5. go-toolbox 新增功能

### 5.1 httpx 模块扩展

#### 5.1.1 连接池支持

```go
// go-toolbox/pkg/httpx/pool.go
package httpx

import (
    "github.com/kamalyes/go-toolbox/pkg/syncx"
)

// ClientPool HTTP 客户端连接池
type ClientPool struct {
    pool *syncx.Pool[*Client]
    opts []ClientOption
}

// NewClientPool 创建连接池
func NewClientPool(opts ...ClientOption) *ClientPool {
    return &ClientPool{
        opts: opts,
        pool: syncx.NewPool(func() *Client {
            return NewClient(opts...)
        }),
    }
}

// Get 获取客户端
func (p *ClientPool) Get() *Client {
    return p.pool.Get()
}

// Put 归还客户端
func (p *ClientPool) Put(client *Client) {
    p.pool.Put(client)
}

// Do 执行请求（自动管理客户端）
func (p *ClientPool) Do(req *Request) (*Response, error) {
    client := p.Get()
    defer p.Put(client)
    return client.Do(req)
}
```

#### 5.1.2 请求指标收集

```go
// go-toolbox/pkg/httpx/metrics.go
package httpx

import (
    "time"
    "github.com/kamalyes/go-toolbox/pkg/syncx"
)

// RequestMetrics 请求指标
type RequestMetrics struct {
    URL          string
    Method       string
    StartTime    time.Time
    EndTime      time.Time
    Duration     time.Duration
    StatusCode   int
    RequestSize  int64
    ResponseSize int64
    Success      bool
    Error        error
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
    Collect(metrics *RequestMetrics)
}

// DefaultMetricsCollector 默认指标收集器
type DefaultMetricsCollector struct {
    totalRequests   *syncx.Uint64
    successRequests *syncx.Uint64
    failedRequests  *syncx.Uint64
    totalDuration   *syncx.Int64
}

func NewMetricsCollector() *DefaultMetricsCollector {
    return &DefaultMetricsCollector{
        totalRequests:   syncx.NewUint64(0),
        successRequests: syncx.NewUint64(0),
        failedRequests:  syncx.NewUint64(0),
        totalDuration:   syncx.NewInt64(0),
    }
}

func (c *DefaultMetricsCollector) Collect(m *RequestMetrics) {
    c.totalRequests.Add(1)
    c.totalDuration.Add(int64(m.Duration))
    
    if m.Success {
        c.successRequests.Add(1)
    } else {
        c.failedRequests.Add(1)
    }
}

// WithMetrics 为客户端添加指标收集
func (c *Client) WithMetrics(collector MetricsCollector) *Client {
    // 包装原有的 Do 方法
    originalDo := c.Do
    c.Do = func(req *Request) (*Response, error) {
        metrics := &RequestMetrics{
            URL:       req.URL,
            Method:    req.Method,
            StartTime: time.Now(),
        }
        
        resp, err := originalDo(req)
        
        metrics.EndTime = time.Now()
        metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
        metrics.Error = err
        metrics.Success = err == nil
        
        if resp != nil {
            metrics.StatusCode = resp.StatusCode
            metrics.ResponseSize = resp.ContentLength
        }
        
        collector.Collect(metrics)
        return resp, err
    }
    
    return c
}
```

#### 5.1.3 重试支持

```go
// go-toolbox/pkg/httpx/retry.go
package httpx

import (
    "github.com/kamalyes/go-toolbox/pkg/retry"
)

// WithRetry 为客户端添加重试功能
func (c *Client) WithRetry(maxRetries int, backoff time.Duration) *Client {
    retrier := retry.NewRunner[*Response]().
        MaxRetries(maxRetries).
        Backoff(backoff)
    
    originalDo := c.Do
    c.Do = func(req *Request) (*Response, error) {
        resp, err := retrier.Run(c.ctx, func(ctx context.Context) (*Response, error) {
            return originalDo(req)
        })
        return resp, err
    }
    
    return c
}
```

### 5.2 mathx 模块扩展

#### 5.2.1 统计函数

```go
// go-toolbox/pkg/mathx/stats.go
package mathx

import "github.com/kamalyes/go-toolbox/pkg/types"

// Mean 计算平均值
func Mean[T types.Numerical](values []T) float64 {
    if len(values) == 0 {
        return 0
    }
    
    var sum T
    for _, v := range values {
        sum += v
    }
    return float64(sum) / float64(len(values))
}

// Median 计算中位数
func Median[T types.Numerical](values []T) T {
    if len(values) == 0 {
        return ZeroValue[T]()
    }
    
    // 注意：需要先排序
    mid := len(values) / 2
    if len(values)%2 == 0 {
        return (values[mid-1] + values[mid]) / 2
    }
    return values[mid]
}

// Percentile 计算百分位
func Percentile[T types.Numerical](values []T, p float64) T {
    if len(values) == 0 {
        return ZeroValue[T]()
    }
    
    p = Between(p, 0.0, 1.0)
    index := Between(
        int(float64(len(values))*p),
        0,
        len(values)-1,
    )
    return values[index]
}

// StandardDeviation 计算标准差
func StandardDeviation[T types.Numerical](values []T) float64 {
    if len(values) == 0 {
        return 0
    }
    
    mean := Mean(values)
    var variance float64
    for _, v := range values {
        diff := float64(v) - mean
        variance += diff * diff
    }
    variance /= float64(len(values))
    
    return math.Sqrt(variance)
}
```

### 5.3 syncx 模块扩展

#### 5.3.1 并发安全的 Map 扩展

```go
// go-toolbox/pkg/syncx/map.go - 已存在，建议添加以下方法

// GetOrCompute 获取或计算值
func (m *Map[K, V]) GetOrCompute(key K, compute func() V) V {
    if value, ok := m.Load(key); ok {
        return value
    }
    
    value := compute()
    m.Store(key, value)
    return value
}

// Merge 合并另一个 Map
func (m *Map[K, V]) Merge(other *Map[K, V]) {
    other.Range(func(key K, value V) bool {
        m.Store(key, value)
        return true
    })
}
```

---

## 6. 实施步骤

### 6.1 第一阶段：go-toolbox 扩展（1-2天）

**目标**：为 go-toolbox 添加必要的新功能

- [ ] 在 `httpx` 中添加连接池支持
- [ ] 在 `httpx` 中添加指标收集功能
- [ ] 在 `httpx` 中添加重试支持
- [ ] 在 `mathx` 中添加统计函数
- [ ] 在 `syncx` 中添加扩展方法
- [ ] 编写单元测试
- [ ] 更新 go-toolbox 文档

### 6.2 第二阶段：报告系统重构（2-3天）

**目标**：统一报告数据结构

**步骤**：

1. **创建新的类型定义文件**
   ```bash
   # 创建 statistics/types.go
   # 定义 ReportData、ReportMode、ErrorStat 等统一类型
   ```

2. **创建格式化器**
   ```bash
   # 创建 statistics/formatter.go
   # 实现 ReportFormatter
   ```

3. **重构 Collector**
   ```bash
   # 修改 statistics/collector.go
   # 添加 GenerateReportData 方法
   # 使用 syncx 原子类型
   ```

4. **简化 HTML 报告生成**
   ```bash
   # 重构 statistics/html_report.go
   # 使用统一数据结构
   # 移除重复代码
   ```

5. **简化实时报告服务器**
   ```bash
   # 重构 statistics/realtime_server.go
   # 使用统一数据结构
   ```

6. **更新模板和 JS**
   ```bash
   # 更新 statistics/unified_template.go
   # 统一 HTML 模板
   # 重构 report.js
   ```

7. **测试验证**
   ```bash
   # 运行所有测试
   # 生成静态报告测试
   # 生成实时报告测试
   ```

### 6.3 第三阶段：核心模块集成（2-3天）

**目标**：在项目中集成 go-toolbox 模块

**步骤**：

1. **重构 config/variable.go**
   - [ ] 使用 mathx 模块函数
   - [ ] 使用 convert 模块转换
   - [ ] 测试变量解析功能

2. **重构 executor/pool.go**
   - [ ] 使用 syncx.Pool
   - [ ] 简化连接池逻辑
   - [ ] 测试连接池性能

3. **重构 statistics/collector.go**
   - [ ] 使用 syncx 原子类型
   - [ ] 使用 mathx 统计函数
   - [ ] 测试统计准确性

4. **重构 protocol/http_verify.go**
   - [ ] 使用 convert 模块
   - [ ] 统一类型转换
   - [ ] 测试验证逻辑

5. **集成测试**
   - [ ] 运行完整的压测流程
   - [ ] 验证报告生成
   - [ ] 性能基准测试

### 6.4 第四阶段：文档和优化（1天）

**目标**：完善文档，优化性能

- [ ] 更新 README.md
- [ ] 更新 ARCHITECTURE.md
- [ ] 添加迁移指南
- [ ] 性能对比测试
- [ ] 代码审查和清理
- [ ] 发布新版本

### 6.5 时间表

| 阶段 | 时间 | 交付物 |
|:-----|:-----|:-------|
| 第一阶段 | 第1-2天 | go-toolbox 新功能 + 测试 |
| 第二阶段 | 第3-5天 | 统一报告系统 + 测试 |
| 第三阶段 | 第6-8天 | 核心模块集成 + 测试 |
| 第四阶段 | 第9天 | 文档 + 优化 + 发布 |

**总计**：约 9 个工作日

---

## 7. 风险评估

### 7.1 技术风险

| 风险 | 影响 | 缓解措施 |
|:-----|:-----|:---------|
| go-toolbox 新功能不稳定 | 高 | 充分的单元测试和集成测试 |
| 数据结构变更导致兼容性问题 | 中 | 保留向后兼容的适配层 |
| 性能回归 | 中 | 基准测试对比 |
| 前端 JS 适配问题 | 低 | 渐进式迁移，保留旧格式支持 |

### 7.2 实施风险

| 风险 | 影响 | 缓解措施 |
|:-----|:-----|:---------|
| 开发时间超期 | 中 | 分阶段实施，优先核心功能 |
| 测试覆盖不足 | 高 | 编写完善的单元测试和集成测试 |
| 文档不完善 | 低 | 边开发边更新文档 |

---

## 8. 预期收益

### 8.1 代码质量

- **代码行数减少**：约 450 行（-35%）
- **重复代码消除**：数据结构统一
- **类型安全提升**：使用泛型和强类型
- **可维护性提高**：单一数据源，职责清晰

### 8.2 性能提升

- **内存优化**：使用对象池减少 GC 压力
- **并发性能**：syncx 原子操作更高效
- **连接池优化**：复用连接，减少开销

### 8.3 开发体验

- **API 统一**：go-toolbox 提供一致的接口
- **扩展性增强**：新增功能只需修改一处
- **调试便利**：数据结构清晰，易于追踪

---

## 9. 附录

### 9.1 相关文件清单

**需要修改的文件**：

```
go-stress/
├── config/
│   └── variable.go              # 使用 mathx, convert
├── executor/
│   ├── executor.go              # 已使用 retry
│   ├── middleware.go            # 已使用 retry
│   ├── pool.go                  # 使用 syncx.Pool
│   └── scheduler.go             # 使用 syncx
├── statistics/
│   ├── types.go                 # 新建：统一数据类型
│   ├── formatter.go             # 新建：格式化器
│   ├── collector.go             # 使用 syncx, mathx
│   ├── html_report.go           # 简化，使用统一数据
│   ├── realtime_server.go       # 简化，使用统一数据
│   ├── unified_template.go      # 更新模板
│   └── report.go                # 兼容层
├── protocol/
│   └── http_verify.go           # 使用 convert
└── docs/
    ├── REFACTORING.md           # 本文档
    └── MIGRATION.md             # 新建：迁移指南
```

**go-toolbox 新增文件**：

```
go-toolbox/pkg/
├── httpx/
│   ├── pool.go                  # 新建：连接池
│   ├── metrics.go               # 新建：指标收集
│   └── retry.go                 # 新建：重试支持
├── mathx/
│   └── stats.go                 # 新建：统计函数
└── syncx/
    └── map_ext.go               # 扩展：Map 方法
```

### 9.2 参考资料

- [go-toolbox 文档](https://github.com/kamalyes/go-toolbox)
- [Go 泛型最佳实践](https://go.dev/doc/tutorial/generics)
- [并发编程模式](https://github.com/golang/go/wiki/CommonMistakes)

---

## 10. 总结

本重构计划的核心目标是：

1. **统一数据模型**：消除报告系统的数据结构重复
2. **深度集成 go-toolbox**：充分利用现有工具包，减少重复代码
3. **提升代码质量**：更好的类型安全、更清晰的职责划分
4. **保持向后兼容**：平滑迁移，不影响现有功能

通过这次重构，go-stress 将变得更加优雅、高效和易于维护。

---

**最后更新**: 2026年1月23日
