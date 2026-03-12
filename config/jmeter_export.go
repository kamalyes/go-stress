/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 20:38:12
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 21:05:36
 * @FilePath: \go-stress\config\jmeter_export.go
 * @Description: 把 go-stress 配置导出为 JMeter .jmx 脚本
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package config

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/kamalyes/go-stress/types"
)

// 导出是导入的逆向，两侧的元素与字段必须严格对称
//
// 每一处 emit 都对着 jmeter.go 里对应的 read/apply 写：导入侧读哪个
// propName、按哪个位掩码判断匹配方式，这里就写哪一个。不对称的后果不是
// 报错而是静默失真 —— 导出的脚本在 JMeter 里跑出另一种压力模型
//
// 对称性由 jmeter_roundtrip_test.go 守住：配置 → .jmx → 配置，逐字段比对

// jmeterExportVersion 导出脚本声明的 JMeter 版本
//
// 写 5.6.3 而非更高：高版本 JMeter 能打开低版本脚本，反之则可能报
// 元素不识别。导出的脚本首要目的是「别人能用 JMeter 打开」
const jmeterExportVersion = "5.6.3"

// ExportJMeterPlan 把配置导出为 .jmx 内容
//
// 返回的 JMXReport 记录哪些配置项在 JMeter 里没有对应物 —— 与导入方向一样，
// 丢失必须被说出来，而不是让使用者以为两边等价
func ExportJMeterPlan(cfg *Config) ([]byte, *JMXReport, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("配置为空")
	}

	apis := cfg.APIs
	if len(apis) == 0 {
		// 单接口模式：顶层 URL 等价于只有一个 API
		if cfg.URL == "" {
			return nil, nil, fmt.Errorf("配置里既没有 apis 也没有 url，无从导出采样器")
		}
		single := APIConfig{
			Name:   "request",
			URL:    cfg.URL,
			Method: cfg.Method,
			Body:   cfg.Body,
		}
		// 顶层 verify 是单条指针，APIConfig 上是切片
		if cfg.Verify != nil {
			single.Verify = []VerifyConfig{*cfg.Verify}
		}
		apis = []APIConfig{single}
	}

	e := &jmxExporter{cfg: cfg, report: &JMXReport{}}
	e.checkUnmappable(apis)

	w := &jmxWriter{}
	w.raw(`<?xml version="1.0" encoding="UTF-8"?>`)
	w.open("jmeterTestPlan", "version", "1.2", "properties", "5.0", "jmeter", jmeterExportVersion)
	w.open("hashTree")

	e.emitTestPlan(w)
	w.open("hashTree")
	e.emitThreadGroup(w, apis)
	w.close("hashTree")

	w.close("hashTree")
	w.close("jmeterTestPlan")

	return w.bytes(), e.report, nil
}

// jmxExporter 导出过程中的状态
type jmxExporter struct {
	cfg    *Config
	report *JMXReport

	// shared 各接口共有且同值的请求头，只在线程组级别写一份
	shared map[string]string
	// sharedThink 各接口共有的思考时间，同样只写一份
	sharedThink *ThinkTimeConfig
}

// groupThinkTime 算出该放在线程组级别的思考时间
//
// 顶层 think_time，或每个接口都配了同一套思考时间时（导入侧把组级定时器
// 下发给采样器后正是这种形态），都提到组级
func (e *jmxExporter) groupThinkTime(apis []APIConfig) *ThinkTimeConfig {
	allSame := len(apis) > 0
	for i := range apis {
		if apis[i].ThinkTime.IsZero() || !sameThinkTime(apis[i].ThinkTime, apis[0].ThinkTime) {
			allSame = false
			break
		}
	}
	if allSame {
		return apis[0].ThinkTime
	}
	return e.cfg.ThinkTime
}

// sameThinkTime 两份思考时间是否完全一致
func sameThinkTime(a, b *ThinkTimeConfig) bool {
	if a.IsZero() || b.IsZero() {
		return a.IsZero() && b.IsZero()
	}
	return a.Type == b.Type && a.Delay == b.Delay &&
		a.Range == b.Range && a.Deviation == b.Deviation
}

// groupHeaders 算出该放在线程组级别的请求头
//
// 两个来源：顶层 headers，以及各接口共有且同值的头
//
// 后者不可省：配置加载器会把顶层 headers 合并进每个 api
// （mergeAPIsWithCommon），经它处理过的配置顶层与接口级都有同一批头，
// 只看顶层会漏掉；而直接构造的配置（嵌入式调用、测试）只有顶层有，
// 只看接口级又会把它整个丢掉。两个来源都要取
//
// 某个接口用不同的值覆盖了顶层的同名头时，该头不进组级 —— JMeter 对
// 同名头在父子 HeaderManager 间的取舍并不明确，留在各自采样器里才无歧义
func (e *jmxExporter) groupHeaders(apis []APIConfig) map[string]string {
	out := sharedHeaders(apis)
	if out == nil {
		out = map[string]string{}
	}

	for k, v := range e.cfg.Headers {
		if _, ok := out[k]; ok {
			continue // 接口级已有同名头，以接口级的取值为准
		}
		conflict := false
		for i := range apis {
			if got, ok := apis[i].Headers[k]; ok && got != v {
				conflict = true
				break
			}
		}
		if !conflict {
			out[k] = v
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// sharedHeaders 取所有接口共有且取值相同的请求头
//
// 只有「每个接口都有、且值完全一样」的头才算共有：值不同的头留在各自采样器里，
// 避免同名头在父子 HeaderManager 之间产生取舍歧义
func sharedHeaders(apis []APIConfig) map[string]string {
	if len(apis) == 0 {
		return nil
	}

	shared := map[string]string{}
	for k, v := range apis[0].Headers {
		shared[k] = v
	}
	for i := 1; i < len(apis) && len(shared) > 0; i++ {
		for k, v := range shared {
			if got, ok := apis[i].Headers[k]; !ok || got != v {
				delete(shared, k)
			}
		}
	}
	if len(shared) == 0 {
		return nil
	}
	return shared
}

// subtractHeaders 取 headers 里不在 shared 中（或取值不同）的部分
func subtractHeaders(headers, shared map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range headers {
		if got, ok := shared[k]; ok && got == v {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// checkUnmappable 先把没有对应物的配置项一次性报出来
//
// 放在生成之前而不是边生成边报：使用者一眼能看到「这份脚本比原配置少了什么」，
// 不必在一长串 mapped 记录里找
func (e *jmxExporter) checkUnmappable(apis []APIConfig) {
	c := e.cfg

	if c.Protocol != "" && !strings.EqualFold(string(c.Protocol), "http") {
		e.report.drop(JMXDropFatal, "protocol", string(c.Protocol),
			"JMeter 的 .jmx 只覆盖 HTTP 采样器，gRPC/WebSocket 没有等价元素")
	}

	if adv := c.Advanced; adv != nil {
		if adv.EnableBreaker {
			e.report.drop(JMXDropWarn, "advanced.enable_breaker", "",
				"熔断器是 go-stress 独有的，JMeter 没有对应元素，已忽略")
		}
		if adv.EnableRetry {
			e.report.drop(JMXDropWarn, "advanced.enable_retry", "",
				"失败重试没有对应元素，已忽略")
		}
	}

	for i := range apis {
		api := &apis[i]
		for _, ex := range api.Extractors {
			switch ex.Type {
			case types.ExtractorTypeJSONPath, types.ExtractorTypeRegex, "":
				// 有对应元素
			default:
				e.report.drop(JMXDropWarn, "extractor", api.Name,
					fmt.Sprintf("提取器 %q 的类型 %s 在 JMeter 里没有直接对应的后置处理器，已忽略",
						ex.Name, ex.Type))
			}
		}
		for _, v := range api.Verify {
			if !exportableVerify(v.Type) {
				e.report.drop(JMXDropWarn, "verify", api.Name,
					fmt.Sprintf("验证类型 %s 在 JMeter 里没有对应断言，已忽略", v.Type))
			}
		}
		if len(api.DependsOn) > 0 {
			e.report.drop(JMXDropWarn, "depends_on", api.Name,
				"依赖关系靠采样器在线程组里的先后顺序表达；"+
					"JMeter 没有「依赖失败则跳过」的语义，前置失败时后续仍会发出")
		}
	}
}

// exportableVerify 该验证类型是否有对应的 JMeter 断言
func exportableVerify(t VerifyType) bool {
	switch t {
	case types.VerifyTypeStatusCode, types.VerifyTypeContains,
		types.VerifyTypeRegex, types.VerifyTypeJSONPath,
		types.VerifyTypeResponseTime, types.VerifyTypeResponseSize:
		return true
	}
	return false
}

// emitTestPlan 写 TestPlan 元素
func (e *jmxExporter) emitTestPlan(w *jmxWriter) {
	w.open("TestPlan",
		"guiclass", "TestPlanGui", "testclass", "TestPlan",
		"testname", "go-stress 导出的测试计划", "enabled", "true")
	w.prop("boolProp", "TestPlan.functional_mode", "false")
	w.prop("boolProp", "TestPlan.serialize_threadgroups", "false")
	w.prop("boolProp", "TestPlan.tearDown_on_shutdown", "true")
	w.prop("stringProp", "TestPlan.comments",
		"由 go-stress 导出；仅包含两边都有对应物的元素")

	// 用户定义变量留空：go-stress 的动态变量函数在 JMeter 里对应的是
	// 内置函数调用而非变量，已在各字段里就地翻译
	w.open("elementProp", "name", "TestPlan.user_defined_variables",
		"elementType", "Arguments", "guiclass", "ArgumentsPanel",
		"testclass", "Arguments", "testname", "用户定义的变量", "enabled", "true")
	w.selfClose("collectionProp", "name", "Arguments.arguments")
	w.close("elementProp")

	w.close("TestPlan")
}

// emitThreadGroup 写线程组及其下属元素
func (e *jmxExporter) emitThreadGroup(w *jmxWriter, apis []APIConfig) {
	c := e.cfg

	threads := c.Concurrency
	if threads < 1 {
		threads = 1
	}

	w.open("ThreadGroup",
		"guiclass", "ThreadGroupGui", "testclass", "ThreadGroup",
		"testname", "go-stress 线程组", "enabled", "true")
	w.prop("stringProp", "ThreadGroup.on_sample_error", "continue")
	w.prop("stringProp", "ThreadGroup.num_threads", strconv.FormatUint(threads, 10))

	rampUp := 0
	if c.Advanced != nil && c.Advanced.RampUp > 0 {
		rampUp = int(c.Advanced.RampUp.Seconds())
	}
	w.prop("stringProp", "ThreadGroup.ramp_time", strconv.Itoa(rampUp))

	// 时长模式与次数模式二选一，与导入侧 applyThreadGroup 的判断对称：
	// scheduler=true 时 JMeter 忽略循环次数，故此时循环写 -1（永远）
	byDuration := c.Duration > 0
	loops := "-1"
	if !byDuration {
		n := c.Requests
		if n < 1 {
			n = 1
		}
		loops = strconv.FormatUint(n, 10)
	}

	w.open("elementProp", "name", "ThreadGroup.main_controller",
		"elementType", "LoopController", "guiclass", "LoopControlPanel",
		"testclass", "LoopController", "testname", "循环控制器", "enabled", "true")
	w.prop("boolProp", "LoopController.continue_forever", "false")
	w.prop("stringProp", "LoopController.loops", loops)
	w.close("elementProp")

	w.prop("boolProp", "ThreadGroup.scheduler", strconv.FormatBool(byDuration))
	if byDuration {
		w.prop("stringProp", "ThreadGroup.duration",
			strconv.Itoa(int(c.Duration.Seconds())))
		w.prop("stringProp", "ThreadGroup.delay", "0")
	}
	w.close("ThreadGroup")

	// 线程组的子树
	w.open("hashTree")

	// 共有的请求头只在组级写一份，各采样器只写自己多出来的那些
	//
	// 不能把 cfg.Headers 原样放组级再让各采样器照抄一遍：配置加载器已经把
	// 全局头合并进了每个 api（mergeAPIsWithCommon），照抄会让同一个头在脚本里
	// 出现 1 + N 次。JMeter 对同名头在父子 HeaderManager 间的取舍并不明确，
	// 重复写出去是在给使用者埋坑
	e.shared = e.groupHeaders(apis)
	if len(e.shared) > 0 {
		e.emitHeaderManager(w, "公共请求头", e.shared)
	}
	if c.CSVData != nil {
		e.emitCSVDataSet(w, c.CSVData)
	}
	if c.Advanced != nil && c.Advanced.TargetQPS > 0 {
		e.emitThroughputTimer(w, c.Advanced.TargetQPS)
	}
	// 共有的思考时间上提到组级，与请求头同理
	//
	// 导入侧会把组级定时器下发给每个采样器（JMeter 的作用域规则），
	// 若导出时逐个采样器各写一份，来回转换一次脚本的结构就变一次 ——
	// 使用者在两个工具间同步时会看到无谓的改动
	e.sharedThink = e.groupThinkTime(apis)
	if !e.sharedThink.IsZero() {
		e.emitTimer(w, e.sharedThink, "公共思考时间")
	}

	for i := range apis {
		e.emitSampler(w, &apis[i])
	}

	w.close("hashTree")
}

// emitHeaderManager 写请求头管理器
func (e *jmxExporter) emitHeaderManager(w *jmxWriter, name string, headers map[string]string) {
	w.open("HeaderManager",
		"guiclass", "HeaderPanel", "testclass", "HeaderManager",
		"testname", name, "enabled", "true")
	w.open("collectionProp", "name", "HeaderManager.headers")
	for _, k := range sortedKeys(headers) {
		w.open("elementProp", "name", "", "elementType", "Header")
		w.prop("stringProp", "Header.name", toJMeterVars(k))
		w.prop("stringProp", "Header.value", toJMeterVars(headers[k]))
		w.close("elementProp")
	}
	w.close("collectionProp")
	w.close("HeaderManager")
	w.emptyTree()
	e.report.addMapped("请求头 %d 项 → HeaderManager「%s」", len(headers), name)
}

// emitCSVDataSet 写 CSV 数据源
func (e *jmxExporter) emitCSVDataSet(w *jmxWriter, csv *CSVDataConfig) {
	recycle := true
	if csv.Recycle != nil {
		recycle = *csv.Recycle
	}
	share := "shareMode.all"
	if csv.Sharing == CSVSharingWorker {
		share = "shareMode.thread"
	}
	delim := csv.Delimiter
	if delim == "" {
		delim = ","
	}

	// 相对路径必须在这里解析成绝对路径
	//
	// 导入侧把 .jmx 里的相对 filename 解析为「相对该 .jmx 所在目录」，
	// 若原样写回相对路径，导出的脚本一旦放到别的目录（JMeter 用户几乎一定会
	// 另存），CSV 就指到不存在的位置 —— 而 JMeter 只会在运行时报找不到文件。
	// go-stress 自己的配置把相对路径按当前工作目录解析，故按此转绝对
	path := csv.Path
	if path != "" && !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			e.report.addMapped("csv_data 的相对路径 %q 已按当前目录解析为 %q，"+
				"使导出的脚本换目录后依然指向同一个文件", csv.Path, abs)
			path = abs
		} else {
			e.report.drop(JMXDropWarn, "CSVDataSet", csv.Path,
				fmt.Sprintf("相对路径无法解析为绝对路径（%v），已原样写出；"+
					"脚本换目录后可能找不到该文件", err))
		}
	}

	w.open("CSVDataSet",
		"guiclass", "TestBeanGUI", "testclass", "CSVDataSet",
		"testname", "CSV 数据源", "enabled", "true")
	w.prop("stringProp", "filename", path)
	w.prop("stringProp", "fileEncoding", "UTF-8")
	w.prop("stringProp", "variableNames", strings.Join(csv.Variables, ","))
	w.prop("stringProp", "delimiter", delim)
	w.prop("boolProp", "quotedData", "false")
	w.prop("boolProp", "recycle", strconv.FormatBool(recycle))
	w.prop("boolProp", "stopThread", "false")
	w.prop("boolProp", "ignoreFirstLine", strconv.FormatBool(csv.IgnoreFirstLine))
	w.prop("stringProp", "shareMode", share)
	w.close("CSVDataSet")
	w.emptyTree()
	e.report.addMapped("csv_data → CSVDataSet（%s，shareMode=%s，recycle=%v）",
		path, share, recycle)
}

// emitThroughputTimer 写吞吐量定时器
//
// calcMode 固定写 1（线程组内共享）：go-stress 的限流器本就是全局的，
// 写 0 会被 JMeter 解释成「每个线程各跑这个速率」，总压力翻并发倍
func (e *jmxExporter) emitThroughputTimer(w *jmxWriter, qps float64) {
	perMinute := qps * 60
	w.open("ConstantThroughputTimer",
		"guiclass", "TestBeanGUI", "testclass", "ConstantThroughputTimer",
		"testname", "固定吞吐量定时器", "enabled", "true")
	w.prop("intProp", "calcMode", "1")
	// throughput 用嵌套式 doubleProp：JMeter 自己就是这么存的
	w.openNested("doubleProp")
	w.rawNested("name", "throughput")
	w.rawNested("value", strconv.FormatFloat(perMinute, 'f', -1, 64))
	w.closeNested("doubleProp")
	w.close("ConstantThroughputTimer")
	w.emptyTree()
	e.report.addMapped("target_qps %.2f/s → ConstantThroughputTimer %.0f/分钟（calcMode=1 共享）",
		qps, perMinute)
}

// emitTimer 按分布类型写定时器
func (e *jmxExporter) emitTimer(w *jmxWriter, tt *ThinkTimeConfig, name string) {
	ms := func(d interface{ Milliseconds() int64 }) string {
		return strconv.FormatInt(d.Milliseconds(), 10)
	}

	switch tt.Type {
	case ThinkTimeUniform:
		w.open("UniformRandomTimer",
			"guiclass", "UniformRandomTimerGui", "testclass", "UniformRandomTimer",
			"testname", name, "enabled", "true")
		w.prop("stringProp", "ConstantDelayOffset", ms(tt.Delay))
		w.prop("stringProp", "RandomTimer.range", ms(tt.Range))
		w.close("UniformRandomTimer")

	case ThinkTimeGaussian:
		w.open("GaussianRandomTimer",
			"guiclass", "GaussianRandomTimerGui", "testclass", "GaussianRandomTimer",
			"testname", name, "enabled", "true")
		w.prop("stringProp", "ConstantDelayOffset", ms(tt.Delay))
		w.prop("stringProp", "RandomTimer.range", ms(tt.Deviation))
		w.close("GaussianRandomTimer")

	default:
		w.open("ConstantTimer",
			"guiclass", "ConstantTimerGui", "testclass", "ConstantTimer",
			"testname", name, "enabled", "true")
		w.prop("stringProp", "ConstantTimer.delay", ms(tt.Delay))
		w.close("ConstantTimer")
	}
	w.emptyTree()
	e.report.addMapped("think_time → %s「%s」", describeThinkTime(tt), name)
}

// emitSampler 写一个 HTTP 采样器及其下属的断言、提取器与定时器
func (e *jmxExporter) emitSampler(w *jmxWriter, api *APIConfig) {
	scheme, host, port, path := splitURLForJMX(api.URL, api.Host, api.Path)

	method := strings.ToUpper(api.Method)
	if method == "" {
		method = "GET"
	}

	name := api.Name
	if name == "" {
		name = path
	}

	w.open("HTTPSamplerProxy",
		"guiclass", "HttpTestSampleGui", "testclass", "HTTPSamplerProxy",
		"testname", name, "enabled", "true")

	if api.Body != "" {
		// postBodyRaw=true 时 Arguments 的首个值即原始报文，
		// 与导入侧 samplerBody 的读法对称
		w.prop("boolProp", "HTTPSampler.postBodyRaw", "true")
		w.open("elementProp", "name", "HTTPsampler.Arguments",
			"elementType", "Arguments")
		w.open("collectionProp", "name", "Arguments.arguments")
		w.open("elementProp", "name", "", "elementType", "HTTPArgument")
		w.prop("boolProp", "HTTPArgument.always_encode", "false")
		w.prop("stringProp", "Argument.value", toJMeterVars(api.Body))
		w.prop("stringProp", "Argument.metadata", "=")
		w.close("elementProp")
		w.close("collectionProp")
		w.close("elementProp")
	} else {
		w.open("elementProp", "name", "HTTPsampler.Arguments",
			"elementType", "Arguments", "guiclass", "HTTPArgumentsPanel",
			"testclass", "Arguments", "testname", "用户定义的变量", "enabled", "true")
		w.selfClose("collectionProp", "name", "Arguments.arguments")
		w.close("elementProp")
	}

	w.prop("stringProp", "HTTPSampler.domain", host)
	w.prop("stringProp", "HTTPSampler.port", port)
	w.prop("stringProp", "HTTPSampler.protocol", scheme)
	w.prop("stringProp", "HTTPSampler.path", path)
	w.prop("stringProp", "HTTPSampler.method", method)
	w.prop("boolProp", "HTTPSampler.follow_redirects", strconv.FormatBool(e.followRedirects()))
	w.prop("boolProp", "HTTPSampler.use_keepalive", strconv.FormatBool(e.keepAlive()))
	w.prop("boolProp", "HTTPSampler.DO_MULTIPART_POST", "false")
	if to := e.cfg.Timeout; to > 0 {
		w.prop("stringProp", "HTTPSampler.response_timeout",
			strconv.FormatInt(to.Milliseconds(), 10))
	}
	w.close("HTTPSamplerProxy")

	// 采样器的子树
	w.open("hashTree")
	if own := subtractHeaders(api.Headers, e.shared); len(own) > 0 {
		e.emitHeaderManager(w, name+" 的请求头", own)
	}
	for _, v := range api.Verify {
		e.emitAssertion(w, v, name)
	}
	for _, ex := range api.Extractors {
		e.emitExtractor(w, ex, name)
	}
	// 与组级相同的思考时间不再重复写
	if !api.ThinkTime.IsZero() && !sameThinkTime(api.ThinkTime, e.sharedThink) {
		e.emitTimer(w, api.ThinkTime, name+" 的思考时间")
	}
	w.close("hashTree")

	e.report.addMapped("api「%s」→ HTTPSamplerProxy（%s %s）", name, method, api.URL)
}

// followRedirects 取 HTTP 配置里的跟随重定向，缺省为 true
func (e *jmxExporter) followRedirects() bool {
	if e.cfg.HTTP != nil {
		return e.cfg.HTTP.FollowRedirects
	}
	return true
}

// keepAlive 取 HTTP 配置里的长连接，缺省为 true
func (e *jmxExporter) keepAlive() bool {
	if e.cfg.HTTP != nil {
		return e.cfg.HTTP.KeepAlive
	}
	return true
}

// emitAssertion 把一条验证写成对应的 JMeter 断言
//
// 位掩码与导入侧 readResponseAssertion 的常量一一对应：
// MATCH=1 CONTAINS=2 NOT=4 EQUALS=8 SUBSTRING=16 OR=32
func (e *jmxExporter) emitAssertion(w *jmxWriter, v VerifyConfig, apiName string) {
	const (
		flagMatch     = 1
		flagNot       = 4
		flagEquals    = 8
		flagSubstring = 16
	)

	switch v.Type {
	case types.VerifyTypeStatusCode:
		testType := flagEquals
		if v.Operator == types.OpNE {
			testType |= flagNot
		} else if v.Operator != "" && v.Operator != types.OpEQ {
			e.report.drop(JMXDropWarn, "verify", apiName,
				fmt.Sprintf("状态码验证的操作符 %s 在 ResponseAssertion 里只有等于/不等于，已按等于导出", v.Operator))
		}
		e.writeResponseAssertion(w, "Assertion.response_code", testType, expectString(v.Expect), apiName)

	case types.VerifyTypeContains:
		testType := flagSubstring
		if v.Operator == types.OpNotContains {
			testType |= flagNot
		}
		e.writeResponseAssertion(w, "Assertion.response_data", testType, expectString(v.Expect), apiName)

	case types.VerifyTypeRegex:
		e.writeResponseAssertion(w, "Assertion.response_data", flagMatch, expectString(v.Expect), apiName)

	case types.VerifyTypeJSONPath:
		e.writeJSONPathAssertion(w, v, apiName)

	case types.VerifyTypeResponseTime:
		ms := expectString(v.Expect)
		if v.Operator != "" && v.Operator != types.OpLTE && v.Operator != types.OpLT {
			e.report.drop(JMXDropWarn, "verify", apiName,
				fmt.Sprintf("DurationAssertion 只能表达「不超过」，操作符 %s 已按不超过导出", v.Operator))
		}
		w.open("DurationAssertion",
			"guiclass", "DurationAssertionGui", "testclass", "DurationAssertion",
			"testname", "响应时间断言", "enabled", "true")
		w.prop("stringProp", "DurationAssertion.duration", ms)
		w.close("DurationAssertion")
		w.emptyTree()

	case types.VerifyTypeResponseSize:
		// SizeAssertion.operator: 1=等于 2=不等于 3=大于 4=小于 5=大于等于 6=小于等于
		op := 1
		switch v.Operator {
		case types.OpNE:
			op = 2
		case types.OpGT:
			op = 3
		case types.OpLT:
			op = 4
		case types.OpGTE:
			op = 5
		case types.OpLTE:
			op = 6
		}
		w.open("SizeAssertion",
			"guiclass", "SizeAssertionGui", "testclass", "SizeAssertion",
			"testname", "响应大小断言", "enabled", "true")
		w.prop("stringProp", "SizeAssertion.size", expectString(v.Expect))
		w.prop("intProp", "SizeAssertion.operator", strconv.Itoa(op))
		w.prop("stringProp", "Assertion.scope", "all")
		w.close("SizeAssertion")
		w.emptyTree()
	}
}

// writeResponseAssertion 写 ResponseAssertion
func (e *jmxExporter) writeResponseAssertion(w *jmxWriter, field string, testType int, value, apiName string) {
	if value == "" {
		e.report.drop(JMXDropWarn, "verify", apiName, "期望值为空，该条断言未导出")
		return
	}
	w.open("ResponseAssertion",
		"guiclass", "AssertionGui", "testclass", "ResponseAssertion",
		"testname", "响应断言", "enabled", "true")
	// 集合名沿用 JMeter 自己拼错的 Asserion.test_strings，与其产物保持一致
	w.open("collectionProp", "name", "Asserion.test_strings")
	w.prop("stringProp", hashOf(value), toJMeterVars(value))
	w.close("collectionProp")
	w.prop("stringProp", "Assertion.test_field", field)
	w.prop("boolProp", "Assertion.assume_success", "false")
	w.prop("intProp", "Assertion.test_type", strconv.Itoa(testType))
	w.close("ResponseAssertion")
	w.emptyTree()
}

// writeJSONPathAssertion 写 JSON 断言
func (e *jmxExporter) writeJSONPathAssertion(w *jmxWriter, v VerifyConfig, apiName string) {
	if v.JSONPath == "" {
		e.report.drop(JMXDropWarn, "verify", apiName, "JSONPath 为空，该条断言未导出")
		return
	}

	// 只校验路径存在（operator=not_empty 且没有期望值）时 JSONVALIDATION=false，
	// 与导入侧 readJSONPathAssertion 对称
	validate := !(v.Operator == types.OpNotEmpty && expectString(v.Expect) == "")
	isRegex := v.Regex || v.Operator == types.OpRegex
	invert := v.Operator == types.OpNE

	if validate && !isRegex && !invert && v.Operator != "" && v.Operator != types.OpEQ {
		e.report.drop(JMXDropWarn, "verify", apiName,
			fmt.Sprintf("JSON 断言的操作符 %s 在 JMeter 里只有等于/不等于/正则，已按等于导出", v.Operator))
	}

	w.open("JSONPathAssertion",
		"guiclass", "JSONPathAssertionGui", "testclass", "JSONPathAssertion",
		"testname", "JSON 断言", "enabled", "true")
	w.prop("stringProp", "JSON_PATH", toJMeterVars(v.JSONPath))
	w.prop("stringProp", "EXPECTED_VALUE", toJMeterVars(expectString(v.Expect)))
	w.prop("boolProp", "JSONVALIDATION", strconv.FormatBool(validate))
	w.prop("boolProp", "EXPECT_NULL", "false")
	w.prop("boolProp", "INVERT", strconv.FormatBool(invert))
	w.prop("boolProp", "ISREGEX", strconv.FormatBool(isRegex))
	w.close("JSONPathAssertion")
	w.emptyTree()
}

// emitExtractor 把提取器写成对应的后置处理器
func (e *jmxExporter) emitExtractor(w *jmxWriter, ex ExtractorConfig, apiName string) {
	switch ex.Type {
	case types.ExtractorTypeJSONPath, "":
		if ex.JSONPath == "" {
			e.report.drop(JMXDropWarn, "extractor", apiName,
				fmt.Sprintf("提取器 %q 没有 JSONPath，未导出", ex.Name))
			return
		}
		w.open("JSONPostProcessor",
			"guiclass", "JSONPostProcessorGui", "testclass", "JSONPostProcessor",
			"testname", "JSON 提取器 "+ex.Name, "enabled", "true")
		w.prop("stringProp", "JSONPostProcessor.referenceNames", ex.Name)
		w.prop("stringProp", "JSONPostProcessor.jsonPathExprs", toJMeterVars(ex.JSONPath))
		w.prop("stringProp", "JSONPostProcessor.match_numbers", "1")
		w.prop("stringProp", "JSONPostProcessor.defaultValues", ex.Default)
		w.close("JSONPostProcessor")
		w.emptyTree()

	case types.ExtractorTypeRegex:
		if ex.Regex == "" {
			e.report.drop(JMXDropWarn, "extractor", apiName,
				fmt.Sprintf("提取器 %q 没有正则表达式，未导出", ex.Name))
			return
		}
		w.open("RegexExtractor",
			"guiclass", "RegexExtractorGui", "testclass", "RegexExtractor",
			"testname", "正则提取器 "+ex.Name, "enabled", "true")
		w.prop("stringProp", "RegexExtractor.useHeaders", "false")
		w.prop("stringProp", "RegexExtractor.refname", ex.Name)
		w.prop("stringProp", "RegexExtractor.regex", ex.Regex)
		w.prop("stringProp", "RegexExtractor.template", "$1$")
		w.prop("stringProp", "RegexExtractor.default", ex.Default)
		w.prop("stringProp", "RegexExtractor.match_number", "1")
		w.close("RegexExtractor")
		w.emptyTree()
	}
}

// ---- 文本与 URL 工具 ----

// goStressVarPattern 匹配 go-stress 的 {{.name}} 变量引用
var goStressVarPattern = regexp.MustCompile(`\{\{\s*\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// goStressFuncRules go-stress 动态变量函数 → JMeter 内置函数
//
// 是 jmeter.go 里 jmeterFuncRules 的逆向，只收录同样语义等价的那几项；
// go-stress 独有的函数（randomEmail 等）没有对应物，原样保留并在报告里说明
var goStressFuncRules = []struct {
	re   *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`\{\{\s*randomInt\s+([+-]?\d+)\s+([+-]?\d+)\s*\}\}`), "${__Random($1,$2)}"},
	{regexp.MustCompile(`\{\{\s*randomString\s+(\d+)\s*\}\}`), "${__RandomString($1)}"},
	{regexp.MustCompile(`\{\{\s*uuid\s*\}\}`), "${__UUID()}"},
	{regexp.MustCompile(`\{\{\s*seq\s*\}\}`), "${__counter(FALSE)}"},
	{regexp.MustCompile(`\{\{\s*unix\s*\}\}`), "${__time(/1000)}"},
	{regexp.MustCompile(`\{\{\s*timestamp\s*\}\}`), "${__time()}"},
}

// toJMeterVars 把 go-stress 的模板语法改写回 JMeter 语法
func toJMeterVars(s string) string {
	if s == "" {
		return s
	}
	for _, r := range goStressFuncRules {
		s = r.re.ReplaceAllString(s, r.repl)
	}
	return goStressVarPattern.ReplaceAllString(s, "${$1}")
}

// splitURLForJMX 把地址拆成 JMeter 采样器需要的四个字段
//
// 解析失败时退回「全部塞进 path」：导入侧的 samplerURL 认得 path 本身就是
// 完整 URL 的写法，所以这条退路仍能还原出同一个地址
func splitURLForJMX(rawURL, hostField, pathField string) (scheme, host, port, path string) {
	if rawURL == "" && hostField != "" {
		rawURL = strings.TrimRight(hostField, "/") + pathField
	}
	if rawURL == "" {
		return "", "", "", pathField
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "", "", "", rawURL
	}

	scheme = u.Scheme
	if scheme == "" {
		scheme = "http"
	}
	host = u.Hostname()
	port = u.Port()
	path = u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	return scheme, host, port, path
}

// expectString 把期望值统一成字符串
func expectString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		// 整数值不要写成 200.000000，JMeter 里状态码必须是整数文本
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

// sortedKeys 取 map 的有序键，保证同一配置每次导出的字节完全一致
func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// hashOf 生成 test_strings 子项的 name
//
// JMeter 自己用的是值的 hashCode，这里给一个稳定的等价物即可：
// 导入侧不读这个 name，它只影响脚本在 JMeter 里的显示
func hashOf(s string) string {
	var h int32
	for _, r := range s {
		h = 31*h + r
	}
	return strconv.FormatInt(int64(h), 10)
}

// ---- 极简 XML 写入器 ----

// jmxWriter 按缩进拼出 .jmx 文本
//
// 不用 encoding/xml：JMeter 的元素属性顺序、空元素写法与缩进都有惯例，
// 脚本还经常被人工编辑，逐字控制比迁就编码器的输出更省事
type jmxWriter struct {
	buf    bytes.Buffer
	indent int
}

func (w *jmxWriter) bytes() []byte { return w.buf.Bytes() }

func (w *jmxWriter) pad() {
	for i := 0; i < w.indent; i++ {
		w.buf.WriteString("  ")
	}
}

func (w *jmxWriter) raw(s string) {
	w.pad()
	w.buf.WriteString(s)
	w.buf.WriteByte('\n')
}

// attrs 拼属性对；值为空的属性照写，JMeter 自己也这么存
func attrs(kv []string) string {
	var b strings.Builder
	for i := 0; i+1 < len(kv); i += 2 {
		b.WriteString(` `)
		b.WriteString(kv[i])
		b.WriteString(`="`)
		b.WriteString(escapeXML(kv[i+1]))
		b.WriteString(`"`)
	}
	return b.String()
}

func (w *jmxWriter) open(tag string, kv ...string) {
	w.raw("<" + tag + attrs(kv) + ">")
	w.indent++
}

func (w *jmxWriter) close(tag string) {
	w.indent--
	w.raw("</" + tag + ">")
}

func (w *jmxWriter) selfClose(tag string, kv ...string) {
	w.raw("<" + tag + attrs(kv) + "/>")
}

func (w *jmxWriter) emptyTree() {
	w.selfClose("hashTree")
}

// prop 写一个属性式的 prop 节点
func (w *jmxWriter) prop(kind, name, value string) {
	w.raw("<" + kind + ` name="` + escapeXML(name) + `">` + escapeXML(value) + "</" + kind + ">")
}

// openNested / rawNested / closeNested 写嵌套式 prop（<name>/<value> 子节点）
func (w *jmxWriter) openNested(tag string) {
	w.raw("<" + tag + ">")
	w.indent++
}

func (w *jmxWriter) rawNested(tag, value string) {
	w.raw("<" + tag + ">" + escapeXML(value) + "</" + tag + ">")
}

func (w *jmxWriter) closeNested(tag string) {
	w.indent--
	w.raw("</" + tag + ">")
}

// escapeXML 转义 XML 文本与属性值
//
// 自己写而非用 xml.EscapeText：后者会把换行转成 &#xA;，而 .jmx 里的
// 请求体经常是多行 JSON，转义后在 JMeter 的编辑框里会糊成一行
func escapeXML(s string) string {
	if !strings.ContainsAny(s, `&<>"'`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
