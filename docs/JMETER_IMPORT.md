# JMeter 脚本互转

`.jmx` 与 go-stress 配置双向转换，两边都免去照着界面一项项重敲

```bash
# JMeter → go-stress：直接跑
go-stress -jmx plan.jmx

# JMeter → go-stress：转成原生 YAML，便于审阅、手工调整、纳入版本管理
go-stress -jmx plan.jmx -jmx-dump plan.yaml

# go-stress → JMeter：把配置导出成 .jmx，交给只用 JMeter 的同事
go-stress -config plan.yaml -jmx-export plan.jmx
```

三个参数**都只转换不压测** — 转换是一次性的迁移动作，与真正发压混在一条命令里容易误触

`-jmx-export` 的配置来源与压测时完全一致（`-config` / `-curl` / `-jmx` / 命令行参数），这样「导出的脚本」与「真正会发的压力」出自同一份配置，不会各走各的

## ⚠️ 先读这一节

导入必然有损，因此**转换报告是这个功能的主体而非附属品**

压测工具最坏的失败模式不是报错，而是静默产出一个「不是你要的那个测试」，数字却煞有介事 — 一个带 1 秒 ConstantTimer 的脚本描述的是带思考时间的闭环负载，丢掉 timer 后变成全速发压，打给服务端的真实 QPS 会差出一个数量级，而报告看起来完全正常

因此遇到**会改变施加负载**的元素时，默认**直接报错退出**并逐条列明原因：

```
JMeter 脚本转换报告: plan.jmx (JMeter 5.6.3)
────────────────────────────────────────────────────────────
  ✓ 线程组「组A」→ 并发 20，每并发 100 次，ramp-up 5s
  ✓ 请求「下单」POST http://shop.example.com/order

  未能导入:
  ✗ [FATAL] IfController 「仅 VIP」
      条件控制器决定请求是否发出，忽略它会多发请求
  ✗ [FATAL] JSR223PostProcessor 「算签名」
      脚本后置处理器可改写提取结果，无法转换

错误: 脚本中有 2 处无法导入且会改变施加的负载（IfController, JSR223PostProcessor），
      按原样导入会得出错误结论；确认可接受差异后加 -jmx-allow-lossy 继续
```

确认差异可接受后：

```bash
go-stress -jmx plan.jmx -jmx-allow-lossy
```

### 另外两点

**导入只还原「测试意图」，不等于能复现 JMeter 的压测结果** — 两者的连接复用、线程模型、断言开销都不同，同一份脚本两边跑出的 QPS 本来就不该一样，拿它做跨工具对比会得出错误结论

**命令行参数可覆盖脚本** — `-c` / `-n` / `-timeout` 在显式写出时覆盖脚本中的对应值，便于用同一份脚本调整压力

## ✅ 已支持的元素

### 线程组

| JMeter | go-stress | 说明 |
|:-------|:----------|:-----|
| `ThreadGroup.num_threads` | `concurrency` | |
| `LoopController.loops` | `requests` | 每并发请求数 |
| `ThreadGroup.ramp_time` | `advanced.ramp_up` | 单位由秒转为 duration |
| `ThreadGroup.scheduler` + `duration` | `duration` | 开启调度器时以持续时间为准，JMeter 此时也忽略循环次数 |
| `ThreadGroup.on_sample_error` | 见下文「失败传播」 | |

`LoopController.continue_forever` 为真且未设持续时间时无法推出请求总数，报 FATAL — 请在脚本里设定循环次数或持续时间，或导入后用 `-n` / `-duration` 指定

### HTTP 采样器

| JMeter | go-stress |
|:-------|:----------|
| `HTTPSampler.protocol` + `domain` + `port` + `path` | `url`（拼成完整地址，默认端口不写入） |
| `HTTPSampler.method` | `method` |
| `HTTPSampler.postBodyRaw` = true | `body`（取 Arguments 首个值作原始报文） |
| `HTTPSampler.postBodyRaw` = false | `body`（Arguments 拼成 `a=1&b=2` 表单） |
| `HeaderManager` | `headers` |

`path` 本身已是完整 URL 时直接采用

原始请求体按原文保留首尾空白、换行和 CRLF，全空白报文也不会被当成空请求体丢弃。

**HTTP 请求默认值**（`ConfigTestElement` / HttpDefaultsGui）同样支持：真实脚本极常把域名、端口、协议统一写在这里、采样器只填 path，默认值只填补采样器留空的字段，不做前缀拼接，与 JMeter 行为一致；其 `response_timeout` 映射为 `timeout`

### 定时器

| JMeter | go-stress |
|:-------|:----------|
| `ConstantTimer` | `think_time: {type: constant, delay}` |
| `UniformRandomTimer` | `think_time: {type: uniform, delay, range}` |
| `GaussianRandomTimer` | `think_time: {type: gaussian, delay, deviation}` |
| `ConstantThroughputTimer` | `advanced.target_qps` |
| `PreciseThroughputTimer` | `advanced.target_qps` |

吞吐量定时器的 `throughput` 单位是**每分钟**，换算为每秒写入 `target_qps`

`calcMode` 按真实语义换算：`0` 表示每线程各跑该速率，故总到达率为 `速率 × 并发数`，据此换算而非原样照搬 — 照搬会让实际压力小到并发倍数分之一；`1` 及以上为全局共享，与 go-stress 的限流器一致

定时器与思考时间详见 [负载形态](LOAD_PROFILE.md)

### 数据源

| JMeter | go-stress |
|:-------|:----------|
| `CSVDataSet.filename` | `csv_data.path`（相对路径按 `.jmx` 所在目录解析） |
| `CSVDataSet.variableNames` | `csv_data.variables` |
| `CSVDataSet.delimiter` | `csv_data.delimiter` |
| `CSVDataSet.recycle` | `csv_data.recycle` |
| `CSVDataSet.ignoreFirstLine` | `csv_data.ignore_first_line` |
| `CSVDataSet.shareMode` | `csv_data.sharing`（`shareMode.all` → `all`，每线程 → `worker`） |

go-stress 目前只支持一个数据源，脚本里有多个 `CSVDataSet` 时报 FATAL — 多份数据会改变参数组合，请合并成一个 CSV 后再导入

### 断言

| JMeter | go-stress |
|:-------|:----------|
| `ResponseAssertion` + `response_code` | `STATUS_CODE`（含 NOT 位 → `ne`） |
| `ResponseAssertion` + `response_data` + Substring/Contains | `CONTAINS`（含 NOT 位 → `not_contains`） |
| `ResponseAssertion` + `response_data` + Matches | `REGEX` |
| `JSONPathAssertion` | `JSONPATH` |
| `DurationAssertion` | `RESPONSE_TIME`（`lte`） |
| `SizeAssertion` | `RESPONSE_SIZE`（按 `operator` 取 `eq`/`ne`/`gt`/`lt`/`gte`/`lte`） |

`RESPONSE_TIME` 按客户端测得的实际耗时比较，单位为毫秒，保留小数精度；HTTP 耗时包含响应体读取，不只计到收到响应头。

JMeter 的匹配方式是位掩码（`MATCH=1 CONTAINS=2 NOT=4 EQUALS=8 SUBSTRING=16 OR=32`），逐位解析

不支持的组合会记 WARN 并跳过该条：「响应体全文相等」与「不匹配正则」没有对应的验证类型；多个期望值以「或」组合时逐条转换，于是校验会比脚本更严格（go-stress 的多条验证之间是「与」）

### 提取器

| JMeter | go-stress |
|:-------|:----------|
| `JSONPostProcessor` | `extractors` 型 `JSONPATH`（支持一个元素声明多个变量，分号分隔） |
| `RegexExtractor` | `extractors` 型 `REGEX` |
| `BoundaryExtractor` | `extractors` 型 `BOUNDARY`，保留字面左右边界 |

`defaultValues` 映射为 `default`；`RegexExtractor.template` 非 `$1$` 时记 WARN，只提取第一个捕获组

正则与边界提取器支持正整数 match_number，指定第 N 次匹配
JMeter 的 0 表示随机匹配，负数表示提取全部，二者与原生配置的默认值语义不同；导入遇到这些值、动态序号、非响应正文提取源或非主样本作用域时记 WARN 并跳过该提取器
边界包含动态变量或函数时同样会提示并跳过；静态边界和默认值的首尾空白、换行原样保留
导出会写明正整数匹配序号；原生 request 来源没有等价的响应后置处理器，会提示并跳过；transforms 管道无法复现时提示导出只保留原始提取
新增边界字段与示例见[配置参考](CONFIG_FILE.md#左右边界提取)，匹配语义参考 [JMeter Boundary Extractor](https://jmeter.apache.org/usermanual/component_reference.html#Boundary_Extractor)

### 变量与参数化

`${name}` 自动改写为 go-stress 的 `{{.name}}`，作用于 URL、请求头、请求体、断言期望值、JSONPath

`TestPlan` 与 `Arguments` 里的用户定义变量映射为 `variables`

#### 内置函数 → 动态变量函数

JMeter 的内置函数按语义等价**直接翻译**为 go-stress 的[动态变量函数](VARIABLES.md)：

| JMeter 函数 | go-stress | 语义 |
|:------------|:----------|:-----|
| `${__Random(min, max)}` | `{{randomInt min max}}` | 区间内随机整数 |
| `${__RandomString(len)}` | `{{randomString len}}` | 定长随机字符串 |
| `${__UUID()}` | `{{uuid}}` | 随机 UUID |
| `${__counter(...)}` | `{{seq}}` | 每次求值递增 |
| `${__time()}` | `{{timestamp}}` | 毫秒级时间戳 |
| `${__time(/1000)}` | `{{unix}}` | 秒级时间戳 |

这些引用在 JMeter 里每次求值都取新值，与 go-stress 动态变量函数的语义一致，翻译前后负载模型不变

**没有等价物的函数原样保留并记 WARN**，例如 `${__time(yyyy/MM/dd)}`（带格式串的时间）、`${__machineName()}`、`${__threadNum()}` — 请改写为 go-stress 的[变量函数](VARIABLES.md)（`{{date "格式串"}}` 可覆盖常见日期需求）

#### 参数化配置元件

| JMeter 元素 | go-stress | 条件 |
|:------------|:----------|:-----|
| `CounterConfig` | 引用处译为 `{{seq}}` | 需有变量名（`CounterConfig.name`） |
| `RandomVariableConfig` | 引用处译为 `{{randomInt 最小 最大}}` | 变量名与上下限（`minimumValue`/`maximumValue`）齐全且为纯数字 |

例如组级放了变量名为 `rnd_page`、范围 1~10 的 `RandomVariableConfig`，采样器 path 里的 `${rnd_page}` 会被译成 `{{randomInt 1 10}}`；计数器变量 `seq_no` 同理译成 `{{seq}}`

映射过程会写进转换报告（`✓ 计数器「seq_no」→ 内置变量函数 {{seq}}`），变量名或上下限不全时记 WARN 并保留原样，便于在 YAML 里手工补全

## 🔗 执行顺序与失败传播

JMeter 的采样器在每轮循环里**全部按序执行**，而 go-stress 在没有 `depends_on` 时走 API 选择器（轮询），**每轮只发其中一个**

因此导入时会把采样器按脚本顺序串成一条依赖链 — 不串的话，`登录 → 查询` 会变成交替发送，请求配比与依赖关系双双不成立

依赖链有个副作用必须说明：**上游失败时下游会被跳过**，而 JMeter 的默认错误处理是 `continue`（某个采样器失败，后面的照发）

- 验证失败：已自动设 `continue_on_failure`，不阻断后续请求
- 传输层失败（连接被拒、超时）：仍会使后续请求按「依赖失败」跳过，统计里会多出跳过计数

这点差异会写进转换报告而非默默带过

## ❌ 不支持的元素

### FATAL — 会改变施加的负载，默认拒绝导入

| 元素 | 原因 |
|:-----|:-----|
| `PoissonRandomTimer` | 泊松分布定时器没有对应的思考时间类型 |
| `SyncTimer` | 集合点会把请求攒成批次再齐发，忽略它会改变请求的时间分布 |
| `ThroughputController` | 按比例裁剪请求，忽略它会改变各请求的占比 |
| `IfController` | 决定请求是否发出，忽略它会多发请求 |
| `WhileController` / `LoopController`（嵌套） / `ForeachController` | 决定请求发几次，忽略它会改变请求总数 |
| `RandomController` / `InterleaveControl` / `SwitchController` | 每轮只执行其中一个子项，忽略它会把全部子项都发出去 |
| `RandomOrderController` | 会改变请求顺序 |
| `RunTime` | 限定子项的执行时长 |
| `JSR223PreProcessor` / `JSR223PostProcessor` / `JSR223Sampler` | 脚本可改写请求内容或提取结果 |
| `BeanShellPreProcessor` / `BeanShellPostProcessor` / `BeanShellSampler` | 同上 |
| `AuthManager` | 自动附加认证头，缺了它请求可能全部被拒 |
| `CookieManager` | 维护会话，缺了它依赖登录态的请求会失败 |
| `TransactionController` | 聚合子请求的耗时统计，指标口径与导入后不同 |
| `ModuleController` / `IncludeController` / `TestFragmentController` | 内容不在本处 |
| `CriticalSectionController` | 会串行化子项的执行 |
| 多个 `ThreadGroup` | go-stress 只有一组全局并发，合并会改变各组的并发比例，请拆成多个脚本分别导入 |
| **任何未知元素** | 无法判断是否影响负载，宁可挡下让人确认 |

### WARN — 只影响校验强度或统计口径

`XPathAssertion`、`XPath2Assertion`、`HTMLAssertion`、`MD5HexAssertion`、`SMIMEAssertion`、`CompareAssertion`、`XPathExtractor`、`XPath2Extractor`、`HtmlExtractor`、`BoundaryExtractor`、`UserParameters`、`DNSCacheManager`、`KeystoreConfig`、`LoginConfig`、`setUp`/`tearDown` 线程组

其中部分有替代写法：`BoundaryExtractor` 可改写成正则提取器，`UserParameters` 可改用 CSV 数据源；`CounterConfig` 与 `RandomVariableConfig` 已能直接映射到内置的 `seq` / `randomInt` 函数（见上文「参数化配置元件」），仅在变量名或参数不全时记 WARN

### INFO — 对负载无影响，静默忽略

各类监听器与结果展示元素（`ResultCollector`、`SummaryReport`、`ViewResultsFullVisualizer`、`BackendListener` 等），以及 `enabled="false"` 的元素 — 后者 JMeter 本身也不会执行，一并算上等于凭空多打一批请求

## 📄 完整示例

`plan.jmx`（节选）：

```xml
<ThreadGroup testname="主线程组">
  <stringProp name="ThreadGroup.num_threads">50</stringProp>
  <stringProp name="ThreadGroup.ramp_time">10</stringProp>
  <elementProp name="ThreadGroup.main_controller" elementType="LoopController">
    <stringProp name="LoopController.loops">200</stringProp>
  </elementProp>
</ThreadGroup>
```

导入结果：

```
  ✓ 用户定义变量 1 项 → variables
  ✓ 线程组「主线程组」→ 并发 50，每并发 200 次，ramp-up 10s
  ✓ CSVDataSet「账号数据」→ csv_data（users.csv，sharing=all，recycle=true）
  ✓ 线程组级请求头 2 条 → 应用到该组所有请求
  ✓ 线程组级思考时间 → 1.5s
  ✓ 请求「登录」POST https://{{.host}}:8443/v1/login（2 条验证、1 个提取器、思考时间 1.5s）
  ✓ 请求「查询订单」GET https://{{.host}}/v1/orders?token={{.token}}（1 条验证、思考时间 1.5s）
  ✓ 2 个请求按脚本顺序串为依赖链，每轮循环依次执行
```

`-jmx-dump` 导出的 YAML 开头带转换说明注释，提醒对照报告确认差异后再用于压测

## 🔁 反方向：导出为 JMeter 脚本

```bash
go-stress -config plan.yaml -jmx-export plan.jmx
```

导出同样会打转换报告，并且**丢失项与导入方向不同** —— 两个工具各有对方没有的能力：

| go-stress 的配置 | 导出结果 |
|:-----------------|:---------|
| `concurrency` / `requests` / `duration` / `ramp_up` | ThreadGroup（时长模式下写 `scheduler=true`，循环次数写 -1，与 JMeter 的取舍一致） |
| `target_qps` | ConstantThroughputTimer，速率 ×60 写成「每分钟」，`calcMode=1`（组内共享） |
| `headers` | HeaderManager；各接口共有且同值的头只在组级写一份 |
| `think_time` | ConstantTimer / UniformRandomTimer / GaussianRandomTimer |
| `csv_data` | CSVDataSet，相对路径会解析成绝对路径 |
| `timeout` | 每个采样器的 `HTTPSampler.response_timeout` |
| `apis[]` | 逐个 HTTPSamplerProxy，按配置顺序排列 |
| `verify` | ResponseAssertion / JSONPathAssertion / DurationAssertion / SizeAssertion |
| `extractors` | JSONPostProcessor / RegexExtractor / BoundaryExtractor |
| `{{.var}}`、内置函数 | `${var}`、`${__UUID()}` 等（见下表） |

**没有对应物、会被报出来的：**

| 配置项 | 原因 |
|:-------|:-----|
| `protocol: grpc` / `websocket` | `.jmx` 只覆盖 HTTP 采样器（定为 FATAL，默认拒绝导出） |
| `advanced.enable_breaker` | 熔断器是 go-stress 独有的 |
| `advanced.enable_retry` | 失败重试没有对应元素 |
| `depends_on` | 依赖关系靠采样器顺序表达，但 JMeter 没有「依赖失败则跳过」的语义，前置失败时后续仍会发出 |
| `extractors` 里的 `HEADER` / `EXPRESSION` | 没有直接对应的后置处理器 |
| `verify` 里 JSON_SCHEMA、EMAIL、UUID 等 | 没有对应断言 |

CSV 的相对路径会在导出时解析成绝对路径：导入侧是按 `.jmx` 所在目录解析相对路径的，原样写回相对路径的话，脚本一旦另存到别的目录，CSV 就指到不存在的位置 —— 而 JMeter 只在运行时才报找不到文件

### 来回转换是收敛的

第一轮导出与第二轮往往不同：导入侧会按 JMeter 的作用域规则把组级的请求头与定时器下发给每个采样器，这是既定语义;但**从第二轮起脚本不再变化**

```bash
go-stress -config a.yaml       -jmx-export r1.jmx
go-stress -jmx r1.jmx          -jmx-dump   b.yaml
go-stress -config b.yaml       -jmx-export r2.jmx
go-stress -jmx r2.jmx          -jmx-dump   c.yaml
go-stress -config c.yaml       -jmx-export r3.jmx
diff r2.jmx r3.jmx   # 无差异
```

这条性质由 `TestRoundTripConverges` 守着 —— 不收敛的话，使用者每次在两个工具间同步都会看到一堆与自己无关的结构改动，久而久之就不敢再信任这个转换

### 动态变量函数的双向翻译

| go-stress | JMeter |
|:----------|:-------|
| `{{uuid}}` | `${__UUID()}` |
| `{{timestamp}}` | `${__time()}` |
| `{{unix}}` | `${__time(/1000)}` |
| `{{randomInt a b}}` | `${__Random(a,b)}` |
| `{{randomString n}}` | `${__RandomString(n)}` |
| `{{seq}}` | `${__counter(FALSE)}` |

只收录语义等价项;go-stress 独有的函数（`randomEmail` 等）在 JMeter 里没有对应物，会原样留在脚本里 —— JMeter 不会求值，需要手工替换

## 🔬 用真实脚本验证过

手写的 fixture 只能覆盖「我想得到的情况」—— 为了验证转换思路本身站不站得住，从 GitHub 公开仓库取了 **345 份真实 `.jmx`**（0.8KB ~ 309KB），对每一份做「导入 → 导出 → 再导入 → 再导出」：

| 指标 | 结果 |
|:-----|:-----|
| XML 解析失败 | 0 |
| panic | 0 |
| 导出失败 | 0 |
| 导出后重新导入失败 | 0 |
| 往返不收敛 | 0 |
| 往返字段不一致 | 0 |
| 成功导入 | 232 / 345 份，共还原 670 个请求 |

未导入的 113 份都是**应当拒绝**的：

| 份数 | 原因 |
|-----:|:-----|
| 64 | 采样器嵌在会改变负载的控制器内（If / Loop / OnceOnly / Random / Foreach / While / Module / TestFragment） |
| 25 | 脚本用的是非 HTTP 采样器（Java / TCP / JDBC / FTP / LDAP / Debug） |
| 24 | 自定义线程组（`UltimateThreadGroup` 等，自带阶梯式加压计划） |

### 这批语料逼出来的缺陷

全都不是手写 fixture 能想到的：

| 影响份数 | 缺陷 |
|---------:|:-----|
| 86 | **采样器嵌在纯分组容器里就整份导不进** —— 真实脚本极少把采样器直接挂在线程组下，`TransactionController`（264 次）与 `GenericController`（189 次）满地都是;这三类纯分组容器现在会被下探 |
| 73 | **请求体里的回车丢失** —— JMeter 把 CR 写成 `&#xd;` 实体 —— XML 规范要求解析器把字面 `
` 归一成 `
`，不用实体就留不住;导出侧原样写字面回车，一个来回 CRLF 报文就变成了 LF |
| 48 | **测试计划层的请求头在运行时消失** —— 挂在 TestPlan 下的 HeaderManager 对全部采样器生效，但导入时只存进 `cfg.Headers`，而把它合并进各 api 的 `mergeAPIsWithCommon` 只在配置文件加载器里调用，`-jmx` 这条路径不走它 —— 脚本里写了 `Authorization`，压出来每个请求都 401 |
| 28 | **地址被重新做百分号编码** —— 走 `url.Parse` 再取 `EscapedPath` 会把 `/items/{{.id}}` 编成 `/items/%7B%7B.id%7D%7D`，JMeter 拿到的是字面编码串而非变量引用，请求直接打到错误的路径;中文路径同样被整段编码 |
| 12 | 裸主机地址被擅自补上尾随斜杠 |
| 64 | **错误信息把人引向错误的方向** —— 采样器被控制器挡住时一律回「没有启用的 HTTP 采样器」，字面没错，但使用者会以为文件损坏;现在会指名是哪些容器 |

每一项都有对应的回归用例（见 `config/jmeter_roundtrip_test.go` 末尾），fixture 是 `config/testdata/nested.jmx` —— 一份浓缩了嵌套容器、计划层请求头、CRLF 报文、路径变量与裸主机地址的脚本

> 语料本身没有入库：10MB 第三方脚本有各自的许可，且会让仓库体积失控

> 本机未安装 JMeter，因此「JMeter 能否打开导出的脚本」没有实测过;能做的最强检查是结构不变量：`.jmx` 的层级靠「元素 + 紧随其后的兄弟 `hashTree`」表达，缺了配对节点 JMeter 加载时会把元素错挂到别的父节点下 —— `TestExportStructuralInvariants` 逐层校验了这一点，也校验了每个元素都带 `testclass` 属性
