/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 21:12:08
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 21:38:55
 * @FilePath: \go-stress\config\jmeter_roundtrip_test.go
 * @Description: JMeter 双向转换的往返一致性测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package config

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTrip 把配置导出成 .jmx 再导回，返回还原出的配置与两侧报告
//
// 这是检验双向对称的唯一可靠手段：两个方向的元素名、属性名与位掩码只要
// 有一处对不上，失真都是静默的 —— 导出的脚本在 JMeter 里照样能跑，
// 只是跑的是另一种压力模型
func roundTrip(t *testing.T, cfg *Config) (*Config, *JMXReport, *JMXReport) {
	t.Helper()

	data, expReport, err := ExportJMeterPlan(cfg)
	require.NoError(t, err, "导出失败")
	require.NotEmpty(t, data)

	path := filepath.Join(t.TempDir(), "roundtrip.jmx")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	back, impReport, err := LoadJMeterPlan(path, true)
	require.NoError(t, err, "导回失败，导出的脚本解析不了:\n%s", data)
	require.NotNil(t, back)

	if impReport != nil && impReport.HasFatal() {
		t.Fatalf("导回时出现致命丢弃，说明导出的脚本缺了必需元素:\n%s\n--- jmx ---\n%s",
			impReport.String(), data)
	}

	normalizeAPIs(back)
	return back, expReport, impReport
}

// normalizeAPIs 把顶层单接口写法摊平成 APIs，便于统一断言
//
// 导入侧对「单个采样器且无验证/提取/思考时间」会刻意折叠成顶层 url 写法
// （见 jmxConverter.convert 末尾），这是表示形式的差异而非语义差异。
// 往返测试要比的是语义，所以在这里把两种写法归一
func normalizeAPIs(cfg *Config) {
	if cfg == nil || len(cfg.APIs) > 0 || cfg.URL == "" {
		return
	}
	cfg.APIs = []APIConfig{{
		Name:    "request",
		URL:     cfg.URL,
		Method:  cfg.Method,
		Body:    cfg.Body,
		Headers: cfg.Headers,
	}}
}

// baseCfg 一份覆盖面较广的配置
func baseCfg() *Config {
	recycle := true
	return &Config{
		Protocol:    "http",
		Concurrency: 16,
		Requests:    500,
		Timeout:     7 * time.Second,
		Headers:     map[string]string{"X-Env": "staging"},
		Advanced: &AdvancedConfig{
			RampUp:    30 * time.Second,
			TargetQPS: 250,
		},
		CSVData: &CSVDataConfig{
			Path:            "/tmp/users.csv",
			Variables:       []string{"username", "password"},
			Delimiter:       ",",
			Recycle:         &recycle,
			IgnoreFirstLine: true,
			Sharing:         CSVSharingAll,
		},
		APIs: []APIConfig{{
			Name:   "login",
			URL:    "https://api.example.com:8443/v1/login?from=jmx",
			Method: "POST",
			Body:   `{"user":"{{.username}}"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Verify: []VerifyConfig{
				{Type: types.VerifyTypeStatusCode, Operator: types.OpEQ, Expect: 200},
				{Type: types.VerifyTypeContains, Operator: types.OpContains, Expect: `"code":0`},
				{Type: types.VerifyTypeJSONPath, JSONPath: "$.data.token", Operator: types.OpNotEmpty},
			},
			Extractors: []ExtractorConfig{
				{Name: "token", Type: types.ExtractorTypeJSONPath, JSONPath: "$.data.token", Default: "none"},
			},
		}},
	}
}

// TestRoundTripThreadGroup 线程组的三项负载参数必须原样回来
func TestRoundTripThreadGroup(t *testing.T) {
	cfg := baseCfg()
	back, _, _ := roundTrip(t, cfg)

	assert.EqualValues(t, cfg.Concurrency, back.Concurrency, "并发数")
	assert.EqualValues(t, cfg.Requests, back.Requests, "每并发请求数")
	require.NotNil(t, back.Advanced)
	assert.Equal(t, cfg.Advanced.RampUp, back.Advanced.RampUp, "ramp-up")
}

// TestRoundTripDurationMode 时长模式下 JMeter 忽略循环次数，请求数应为 0
func TestRoundTripDurationMode(t *testing.T) {
	cfg := baseCfg()
	cfg.Duration = 90 * time.Second
	cfg.Requests = 0

	back, _, _ := roundTrip(t, cfg)
	assert.Equal(t, 90*time.Second, back.Duration, "持续时长")
	assert.Zero(t, back.Requests, "时长模式下不该还原出请求数")
}

// TestRoundTripTargetQPS 速率经「每分钟」中转后不能漂移
//
// 导出写 qps*60，导入读 perMinute/60，calcMode 必须是 1（共享）——
// 写 0 会被按「每线程各跑这个速率」解释，总压力翻并发倍
func TestRoundTripTargetQPS(t *testing.T) {
	for _, qps := range []float64{250, 1, 33.5, 1000} {
		cfg := baseCfg()
		cfg.Advanced.TargetQPS = qps

		back, _, _ := roundTrip(t, cfg)
		require.NotNil(t, back.Advanced)
		assert.InDelta(t, qps, back.Advanced.TargetQPS, 1e-9, "qps=%v", qps)
	}
}

// TestRoundTripSamplerURL 地址的四个字段拆开再拼回必须是同一个 URL
func TestRoundTripSamplerURL(t *testing.T) {
	cases := []struct {
		raw  string
		want string // 导入侧刻意不把默认端口写进 URL，这里给出归一后的期望
	}{
		{"https://api.example.com:8443/v1/login?from=jmx", "https://api.example.com:8443/v1/login?from=jmx"},
		{"http://127.0.0.1:3210/v1/items", "http://127.0.0.1:3210/v1/items"},
		{"https://api.example.com/v1/plain", "https://api.example.com/v1/plain"},
		{"http://api.example.com/", "http://api.example.com/"},
		// 默认端口被省略，地址语义不变
		{"http://api.example.com:80/default-port", "http://api.example.com/default-port"},
		{"https://api.example.com:443/default-port", "https://api.example.com/default-port"},
		// 多级路径与带编码的查询串
		{"https://api.example.com/a/b/c?q=1&r=2", "https://api.example.com/a/b/c?q=1&r=2"},
	}
	for _, tc := range cases {
		cfg := baseCfg()
		cfg.CSVData = nil
		cfg.APIs = []APIConfig{{Name: "x", URL: tc.raw, Method: "GET"}}

		back, _, _ := roundTrip(t, cfg)
		require.Len(t, back.APIs, 1, "原始地址 %s", tc.raw)
		assert.Equal(t, tc.want, back.APIs[0].URL, "原始地址 %s", tc.raw)
		assert.Equal(t, "GET", back.APIs[0].Method)
	}
}

// TestRoundTripMethodAndBody 方法与原始请求体
func TestRoundTripMethodAndBody(t *testing.T) {
	cfg := baseCfg()
	back, _, _ := roundTrip(t, cfg)

	require.Len(t, back.APIs, 1)
	assert.Equal(t, "POST", back.APIs[0].Method)
	assert.Equal(t, `{"user":"{{.username}}"}`, back.APIs[0].Body,
		"变量引用要能 {{.x}} → ${x} → {{.x}} 地转回来")
}

// TestRoundTripHeaders 全局与接口级请求头都要还原
//
// 导入侧把线程组级的 HeaderManager 合并进每个采样器，故还原后
// 全局头会出现在接口头里，这是既有语义，断言按此写
func TestRoundTripHeaders(t *testing.T) {
	cfg := baseCfg()
	back, _, _ := roundTrip(t, cfg)

	require.Len(t, back.APIs, 1)
	got := back.APIs[0].Headers
	assert.Equal(t, "application/json", got["Content-Type"], "接口级请求头")
	assert.Equal(t, "staging", got["X-Env"], "全局请求头应合并进采样器")
}

// TestRoundTripVerifications 各类断言的类型、操作符与期望值
func TestRoundTripVerifications(t *testing.T) {
	cases := []struct {
		name string
		in   VerifyConfig
		want VerifyConfig
	}{
		{"状态码等于", VerifyConfig{Type: types.VerifyTypeStatusCode, Operator: types.OpEQ, Expect: 200},
			VerifyConfig{Type: types.VerifyTypeStatusCode, Operator: types.OpEQ, Expect: "200"}},
		{"状态码不等于", VerifyConfig{Type: types.VerifyTypeStatusCode, Operator: types.OpNE, Expect: 500},
			VerifyConfig{Type: types.VerifyTypeStatusCode, Operator: types.OpNE, Expect: "500"}},
		{"包含", VerifyConfig{Type: types.VerifyTypeContains, Operator: types.OpContains, Expect: "ok"},
			VerifyConfig{Type: types.VerifyTypeContains, Operator: types.OpContains, Expect: "ok"}},
		{"不包含", VerifyConfig{Type: types.VerifyTypeContains, Operator: types.OpNotContains, Expect: "error"},
			VerifyConfig{Type: types.VerifyTypeContains, Operator: types.OpNotContains, Expect: "error"}},
		{"正则", VerifyConfig{Type: types.VerifyTypeRegex, Expect: `"id":\d+`},
			VerifyConfig{Type: types.VerifyTypeRegex, Expect: `"id":\d+`}},
		{"响应时间", VerifyConfig{Type: types.VerifyTypeResponseTime, Operator: types.OpLTE, Expect: 800},
			VerifyConfig{Type: types.VerifyTypeResponseTime, Operator: types.OpLTE, Expect: 800}},
		{"响应大小等于", VerifyConfig{Type: types.VerifyTypeResponseSize, Operator: types.OpEQ, Expect: 1024},
			VerifyConfig{Type: types.VerifyTypeResponseSize, Operator: types.OpEQ, Expect: 1024}},
		{"响应大小大于", VerifyConfig{Type: types.VerifyTypeResponseSize, Operator: types.OpGT, Expect: 10},
			VerifyConfig{Type: types.VerifyTypeResponseSize, Operator: types.OpGT, Expect: 10}},
		{"响应大小不超过", VerifyConfig{Type: types.VerifyTypeResponseSize, Operator: types.OpLTE, Expect: 2048},
			VerifyConfig{Type: types.VerifyTypeResponseSize, Operator: types.OpLTE, Expect: 2048}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseCfg()
			cfg.CSVData = nil
			cfg.APIs = []APIConfig{{Name: "x", URL: "http://h/p", Method: "GET",
				Verify: []VerifyConfig{tc.in}}}

			back, _, _ := roundTrip(t, cfg)
			require.Len(t, back.APIs, 1)
			require.Len(t, back.APIs[0].Verify, 1, "应还原出恰好一条验证")

			got := back.APIs[0].Verify[0]
			assert.Equal(t, tc.want.Type, got.Type, "类型")
			assert.Equal(t, tc.want.Operator, got.Operator, "操作符")
			assert.EqualValues(t, tc.want.Expect, got.Expect, "期望值")
		})
	}
}

// TestRoundTripJSONPathAssertion JSON 断言的三种形态
func TestRoundTripJSONPathAssertion(t *testing.T) {
	cases := []struct {
		name string
		in   VerifyConfig
	}{
		{"仅校验存在", VerifyConfig{Type: types.VerifyTypeJSONPath, JSONPath: "$.data.token", Operator: types.OpNotEmpty}},
		{"等于", VerifyConfig{Type: types.VerifyTypeJSONPath, JSONPath: "$.code", Operator: types.OpEQ, Expect: "0"}},
		{"不等于", VerifyConfig{Type: types.VerifyTypeJSONPath, JSONPath: "$.code", Operator: types.OpNE, Expect: "1"}},
		{"正则", VerifyConfig{Type: types.VerifyTypeJSONPath, JSONPath: "$.id", Operator: types.OpRegex, Expect: `\d+`}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseCfg()
			cfg.CSVData = nil
			cfg.APIs = []APIConfig{{Name: "x", URL: "http://h/p", Method: "GET",
				Verify: []VerifyConfig{tc.in}}}

			back, _, _ := roundTrip(t, cfg)
			require.Len(t, back.APIs[0].Verify, 1)
			got := back.APIs[0].Verify[0]

			assert.Equal(t, types.VerifyTypeJSONPath, got.Type)
			assert.Equal(t, tc.in.JSONPath, got.JSONPath)
			assert.Equal(t, tc.in.Operator, got.Operator, "操作符")
			if tc.in.Expect != nil {
				assert.EqualValues(t, tc.in.Expect, got.Expect)
			}
		})
	}
}

// TestRoundTripExtractors 两类提取器
func TestRoundTripExtractors(t *testing.T) {
	cfg := baseCfg()
	cfg.CSVData = nil
	cfg.APIs = []APIConfig{{Name: "x", URL: "http://h/p", Method: "GET",
		Extractors: []ExtractorConfig{
			{Name: "token", Type: types.ExtractorTypeJSONPath, JSONPath: "$.data.token", Default: "none"},
			{Name: "sid", Type: types.ExtractorTypeRegex, Regex: `sid=(\w+)`, Default: "x"},
		}}}

	back, _, _ := roundTrip(t, cfg)
	require.Len(t, back.APIs, 1)
	require.Len(t, back.APIs[0].Extractors, 2)

	byName := map[string]ExtractorConfig{}
	for _, e := range back.APIs[0].Extractors {
		byName[e.Name] = e
	}

	tok := byName["token"]
	assert.Equal(t, types.ExtractorTypeJSONPath, tok.Type)
	assert.Equal(t, "$.data.token", tok.JSONPath)
	assert.Equal(t, "none", tok.Default)

	sid := byName["sid"]
	assert.Equal(t, types.ExtractorTypeRegex, sid.Type)
	assert.Equal(t, `sid=(\w+)`, sid.Regex)
	assert.Equal(t, "x", sid.Default)
}

// TestRoundTripThinkTime 三种分布的思考时间
func TestRoundTripThinkTime(t *testing.T) {
	cases := []*ThinkTimeConfig{
		{Type: ThinkTimeConstant, Delay: 500 * time.Millisecond},
		{Type: ThinkTimeUniform, Delay: 200 * time.Millisecond, Range: 800 * time.Millisecond},
		{Type: ThinkTimeGaussian, Delay: 300 * time.Millisecond, Deviation: 100 * time.Millisecond},
	}

	for _, tt := range cases {
		t.Run(string(tt.Type), func(t *testing.T) {
			cfg := baseCfg()
			cfg.CSVData = nil
			cfg.APIs = []APIConfig{{Name: "x", URL: "http://h/p", Method: "GET", ThinkTime: tt}}

			back, _, _ := roundTrip(t, cfg)
			require.Len(t, back.APIs, 1)
			got := back.APIs[0].ThinkTime
			require.NotNil(t, got, "思考时间丢了")

			assert.Equal(t, tt.Type, got.Type, "分布类型")
			assert.Equal(t, tt.Delay, got.Delay, "基础延迟")
			assert.Equal(t, tt.Range, got.Range, "uniform 区间")
			assert.Equal(t, tt.Deviation, got.Deviation, "gaussian 标准差")
		})
	}
}

// TestRoundTripCSVDataSet CSV 数据源的每一项
func TestRoundTripCSVDataSet(t *testing.T) {
	for _, sharing := range []CSVSharing{CSVSharingAll, CSVSharingWorker} {
		recycle := false
		cfg := baseCfg()
		cfg.CSVData = &CSVDataConfig{
			Path:            "/tmp/users.csv",
			Variables:       []string{"username", "password"},
			Delimiter:       ";",
			Recycle:         &recycle,
			IgnoreFirstLine: true,
			Sharing:         sharing,
		}

		back, _, _ := roundTrip(t, cfg)
		require.NotNil(t, back.CSVData, "sharing=%s", sharing)

		assert.Equal(t, "/tmp/users.csv", back.CSVData.Path)
		assert.Equal(t, []string{"username", "password"}, back.CSVData.Variables)
		assert.Equal(t, ";", back.CSVData.Delimiter)
		require.NotNil(t, back.CSVData.Recycle)
		assert.False(t, *back.CSVData.Recycle, "recycle=false 必须保住")
		assert.True(t, back.CSVData.IgnoreFirstLine)
		assert.Equal(t, sharing, back.CSVData.Sharing, "共享方式")
	}
}

// TestRoundTripMultipleAPIs 多接口的顺序与数量
func TestRoundTripMultipleAPIs(t *testing.T) {
	cfg := baseCfg()
	cfg.CSVData = nil
	cfg.APIs = []APIConfig{
		{Name: "login", URL: "http://h/login", Method: "POST", Body: `{"a":1}`},
		{Name: "orders", URL: "http://h/orders", Method: "GET"},
		{Name: "logout", URL: "http://h/logout", Method: "DELETE"},
	}

	back, _, _ := roundTrip(t, cfg)
	require.Len(t, back.APIs, 3)
	assert.Equal(t, []string{"login", "orders", "logout"},
		[]string{back.APIs[0].Name, back.APIs[1].Name, back.APIs[2].Name},
		"采样器顺序即依赖顺序，不能乱")
	assert.Equal(t, "DELETE", back.APIs[2].Method)
}

// TestRoundTripVariableFunctions 动态变量函数的双向翻译
func TestRoundTripVariableFunctions(t *testing.T) {
	cases := []string{
		"{{uuid}}",
		"{{timestamp}}",
		"{{unix}}",
		"{{randomInt 1 100}}",
		"{{randomString 8}}",
		"{{seq}}",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			cfg := baseCfg()
			cfg.CSVData = nil
			cfg.APIs = []APIConfig{{Name: "x", URL: "http://h/p", Method: "POST",
				Body: `{"v":"` + src + `"}`}}

			back, _, _ := roundTrip(t, cfg)
			require.Len(t, back.APIs, 1)
			assert.Equal(t, `{"v":"`+src+`"}`, back.APIs[0].Body,
				"%s 应能翻成 JMeter 函数再翻回来", src)
		})
	}
}

// TestExportReportsUnmappable 没有对应物的配置项必须出现在报告里
func TestExportReportsUnmappable(t *testing.T) {
	cfg := baseCfg()
	cfg.Advanced.EnableRetry = true
	cfg.Advanced.EnableBreaker = true
	cfg.APIs[0].Extractors = append(cfg.APIs[0].Extractors, ExtractorConfig{
		Name: "ct", Type: types.ExtractorTypeHeader, Header: "Content-Type",
	})
	cfg.APIs[0].Verify = append(cfg.APIs[0].Verify, VerifyConfig{
		Type: types.VerifyTypeJSONSchema, Expect: "{}",
	})

	_, rep, _ := roundTrip(t, cfg)
	require.NotNil(t, rep)
	text := rep.String()

	assert.Contains(t, text, "enable_retry", "重试没有对应元素，必须报出来")
	assert.Contains(t, text, "enable_breaker")
	assert.Contains(t, text, "HEADER", "响应头提取器没有对应物")
	assert.Contains(t, text, "JSON_SCHEMA", "该验证类型没有对应断言")
	assert.False(t, rep.HasFatal(), "这些都只是丢失，不该定为致命")
}

// TestExportRejectsNonHTTP 非 HTTP 协议导出不了，必须定为致命
func TestExportRejectsNonHTTP(t *testing.T) {
	cfg := baseCfg()
	cfg.Protocol = "websocket"

	_, rep, err := ExportJMeterPlan(cfg)
	require.NoError(t, err)
	require.NotNil(t, rep)
	assert.True(t, rep.HasFatal(), "WebSocket 没有等价采样器，应为致命")
}

// TestExportDeterministic 同一份配置每次导出的字节必须完全一致
//
// 请求头来自 map，遍历顺序随机；不排序的话两次导出的脚本会不一样，
// 放进版本库就会无谓地产生差异
func TestExportDeterministic(t *testing.T) {
	cfg := baseCfg()
	cfg.Headers = map[string]string{"A": "1", "B": "2", "C": "3", "D": "4", "E": "5"}
	cfg.APIs[0].Headers = map[string]string{"X": "1", "Y": "2", "Z": "3"}

	first, _, err := ExportJMeterPlan(cfg)
	require.NoError(t, err)
	for i := 0; i < 20; i++ {
		again, _, err := ExportJMeterPlan(cfg)
		require.NoError(t, err)
		require.Equal(t, string(first), string(again), "第 %d 次导出与首次不一致", i+1)
	}
}

// TestExportEmptyConfig 没有任何采样器来源时应报错而不是产出空脚本
func TestExportEmptyConfig(t *testing.T) {
	_, _, err := ExportJMeterPlan(&Config{Protocol: "http", Concurrency: 1, Requests: 1})
	require.Error(t, err)

	_, _, err = ExportJMeterPlan(nil)
	require.Error(t, err)
}

// TestJMXToJMXStable 真实脚本 → 配置 → 脚本 → 配置，两次导入必须一致
//
// 与前面的用例方向相反：那些从配置出发，这个从 JMeter 自己的脚本出发。
// 两个方向都要测，因为失真可能只在其中一侧 —— 导出漏写某个属性，
// 只有从真实脚本出发才会暴露
func TestJMXToJMXStable(t *testing.T) {
	for _, name := range []string{"simple.jmx", "defaults.jmx", "lossy.jmx"} {
		t.Run(name, func(t *testing.T) {
			first, _, err := LoadJMeterPlan(filepath.Join("testdata", name), true)
			if err != nil {
				t.Skipf("该脚本本就导入不了，不在本用例范围内: %v", err)
			}
			require.NotNil(t, first)
			normalizeAPIs(first)

			data, _, err := ExportJMeterPlan(first)
			require.NoError(t, err, "把导入结果再导出失败")

			path := filepath.Join(t.TempDir(), "again.jmx")
			require.NoError(t, os.WriteFile(path, data, 0o644))

			second, rep, err := LoadJMeterPlan(path, true)
			require.NoError(t, err, "导出的脚本重新导入失败:\n%s", data)
			require.NotNil(t, second)
			require.False(t, rep.HasFatal(), "重新导入出现致命丢弃:\n%s", rep.String())
			normalizeAPIs(second)

			assertConfigEquivalent(t, first, second)
		})
	}
}

// assertConfigEquivalent 比对两份配置在可映射字段上的等价性
func assertConfigEquivalent(t *testing.T, want, got *Config) {
	t.Helper()

	assert.EqualValues(t, want.Concurrency, got.Concurrency, "并发数")
	assert.EqualValues(t, want.Requests, got.Requests, "每并发请求数")
	assert.Equal(t, want.Duration, got.Duration, "持续时长")

	wantQPS, gotQPS := 0.0, 0.0
	wantRamp, gotRamp := time.Duration(0), time.Duration(0)
	if want.Advanced != nil {
		wantQPS, wantRamp = want.Advanced.TargetQPS, want.Advanced.RampUp
	}
	if got.Advanced != nil {
		gotQPS, gotRamp = got.Advanced.TargetQPS, got.Advanced.RampUp
	}
	assert.InDelta(t, wantQPS, gotQPS, 1e-9, "目标速率")
	assert.Equal(t, wantRamp, gotRamp, "ramp-up")

	require.Equal(t, len(want.APIs), len(got.APIs), "接口数量")
	for i := range want.APIs {
		w, g := want.APIs[i], got.APIs[i]
		assert.Equal(t, w.URL, g.URL, "第 %d 个接口的地址", i)
		assert.Equal(t, strings.ToUpper(w.Method), strings.ToUpper(g.Method), "第 %d 个接口的方法", i)
		assert.Equal(t, w.Body, g.Body, "第 %d 个接口的请求体", i)
		assert.Equal(t, len(w.Verify), len(g.Verify), "第 %d 个接口的验证条数", i)
		assert.Equal(t, len(w.Extractors), len(g.Extractors), "第 %d 个接口的提取器个数", i)

		for j := range w.Verify {
			assert.Equal(t, w.Verify[j].Type, g.Verify[j].Type,
				"接口 %d 第 %d 条验证的类型", i, j)
			assert.Equal(t, w.Verify[j].Operator, g.Verify[j].Operator,
				"接口 %d 第 %d 条验证的操作符", i, j)
			assert.EqualValues(t, w.Verify[j].Expect, g.Verify[j].Expect,
				"接口 %d 第 %d 条验证的期望值", i, j)
		}
		for j := range w.Extractors {
			assert.Equal(t, w.Extractors[j].Name, g.Extractors[j].Name,
				"接口 %d 第 %d 个提取器的变量名", i, j)
			assert.Equal(t, w.Extractors[j].Type, g.Extractors[j].Type,
				"接口 %d 第 %d 个提取器的类型", i, j)
		}

		if w.ThinkTime.IsZero() {
			assert.True(t, g.ThinkTime.IsZero(), "第 %d 个接口不该凭空多出思考时间", i)
		} else {
			require.NotNil(t, g.ThinkTime, "第 %d 个接口的思考时间丢了", i)
			assert.Equal(t, w.ThinkTime.Type, g.ThinkTime.Type, "第 %d 个接口的分布类型", i)
			assert.Equal(t, w.ThinkTime.Delay, g.ThinkTime.Delay, "第 %d 个接口的基础延迟", i)
		}
	}

	if want.CSVData == nil {
		assert.Nil(t, got.CSVData, "不该凭空多出 CSV 数据源")
	} else {
		require.NotNil(t, got.CSVData, "CSV 数据源丢了")
		// 比对解析后的路径：导出会把相对路径转绝对，使脚本换目录后仍指向同一文件，
		// 两者表示形式不同但指的是同一个文件
		assert.Equal(t, mustAbs(t, want.CSVData.Path), mustAbs(t, got.CSVData.Path),
			"CSV 路径应指向同一个文件")
		assert.Equal(t, want.CSVData.Variables, got.CSVData.Variables)
		assert.Equal(t, want.CSVData.Sharing, got.CSVData.Sharing)
	}
}

// mustAbs 取绝对路径，便于比较表示形式不同但指向同一文件的两个路径
func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return abs
}

// TestExportHeaderDeduplication 共有的请求头只写一份
//
// 回归用：配置加载器会把顶层 headers 合并进每个 api，若导出时组级与采样器级
// 各写一遍，同一个头会在脚本里出现 1 + N 次。JMeter 对同名头在父子
// HeaderManager 之间的取舍并不明确，重复写出去是在给使用者埋坑
func TestExportHeaderDeduplication(t *testing.T) {
	cfg := baseCfg()
	cfg.CSVData = nil
	cfg.Headers = map[string]string{"X-Env": "staging"}
	cfg.APIs = []APIConfig{
		// 模拟加载器合并后的形态：顶层头已在每个 api 里
		{Name: "a", URL: "http://h/a", Method: "GET", Headers: map[string]string{
			"X-Env": "staging", "X-Only-A": "1",
		}},
		{Name: "b", URL: "http://h/b", Method: "GET", Headers: map[string]string{
			"X-Env": "staging",
		}},
	}

	data, _, err := ExportJMeterPlan(cfg)
	require.NoError(t, err)

	assert.Equal(t, 1, strings.Count(string(data), "X-Env"),
		"共有的头只该出现一次，实际脚本:\n%s", data)
	assert.Equal(t, 1, strings.Count(string(data), "X-Only-A"),
		"独有的头出现在它自己的采样器下")

	// 语义不能因为去重而改变
	back, _, _ := roundTrip(t, cfg)
	require.Len(t, back.APIs, 2)
	for i := range back.APIs {
		assert.Equal(t, "staging", back.APIs[i].Headers["X-Env"],
			"第 %d 个接口仍应拿到公共头", i)
	}
	assert.Equal(t, "1", back.APIs[0].Headers["X-Only-A"])
	assert.NotContains(t, back.APIs[1].Headers, "X-Only-A",
		"独有的头不该漏给别的接口")
}

// TestExportHeaderConflictStaysLocal 接口用不同值覆盖同名头时，该头不进组级
func TestExportHeaderConflictStaysLocal(t *testing.T) {
	cfg := baseCfg()
	cfg.CSVData = nil
	cfg.Headers = map[string]string{"X-Env": "staging"}
	cfg.APIs = []APIConfig{
		{Name: "a", URL: "http://h/a", Method: "GET", Headers: map[string]string{"X-Env": "prod"}},
		{Name: "b", URL: "http://h/b", Method: "GET", Headers: map[string]string{"X-Env": "staging"}},
	}

	back, _, _ := roundTrip(t, cfg)
	require.Len(t, back.APIs, 2)
	assert.Equal(t, "prod", back.APIs[0].Headers["X-Env"], "各自的取值不能被公共头冲掉")
	assert.Equal(t, "staging", back.APIs[1].Headers["X-Env"])
}

// TestExportCSVPathIsAbsolute 导出的 CSV 路径必须是绝对的
//
// 回归用：相对路径在导入侧是按 .jmx 所在目录解析的，原样写回相对路径的话，
// 脚本一旦另存到别的目录，CSV 就指到不存在的位置，而 JMeter 只在运行时才报错
func TestExportCSVPathIsAbsolute(t *testing.T) {
	cfg := baseCfg()
	cfg.CSVData = &CSVDataConfig{Path: "testdata/users.csv", Variables: []string{"username"}}

	data, rep, err := ExportJMeterPlan(cfg)
	require.NoError(t, err)

	abs, err := filepath.Abs("testdata/users.csv")
	require.NoError(t, err)
	assert.Contains(t, string(data), abs, "脚本里应写绝对路径")
	assert.Contains(t, rep.String(), "解析为", "路径被改写过，报告里要说明")
}

// TestExportStructuralInvariants 导出的脚本必须满足 JMeter 的结构不变量
//
// 本机没有 JMeter，无法断言「JMeter 真的能打开」。但 .jmx 的层级不是靠嵌套
// 表达的：每个测试元素的子元素放在它后面那个兄弟 <hashTree> 里。缺了这个兄弟
// 节点，JMeter 加载时会把后续元素错挂到别的父节点下，甚至直接报错
//
// 这一条是能在无 JMeter 环境下做的最强检查，也正是本项目导入侧 pairsOf 依赖的前提
func TestExportStructuralInvariants(t *testing.T) {
	cfg := baseCfg()
	cfg.APIs = append(cfg.APIs, APIConfig{
		Name: "orders", URL: "http://h/orders", Method: "GET",
		ThinkTime: &ThinkTimeConfig{Type: ThinkTimeConstant, Delay: time.Second},
		Verify: []VerifyConfig{
			{Type: types.VerifyTypeResponseTime, Operator: types.OpLTE, Expect: 800},
			{Type: types.VerifyTypeResponseSize, Operator: types.OpGT, Expect: 10},
		},
		Extractors: []ExtractorConfig{
			{Name: "sid", Type: types.ExtractorTypeRegex, Regex: `sid=(\w+)`},
		},
	})

	data, _, err := ExportJMeterPlan(cfg)
	require.NoError(t, err)

	// 逐层检查「元素 + 紧随其后的 hashTree」配对
	var walk func(nodes []genericNode, path string)
	walk = func(nodes []genericNode, path string) {
		for i := 0; i < len(nodes); i++ {
			tag := nodes[i].XMLName.Local
			if tag == "hashTree" {
				walk(nodes[i].Children, path+"/hashTree")
				continue
			}
			require.Less(t, i+1, len(nodes),
				"%s/%s 后面没有任何节点，缺少配对的 hashTree", path, tag)
			require.Equal(t, "hashTree", nodes[i+1].XMLName.Local,
				"%s/%s 后面跟的是 %s 而不是 hashTree", path, tag, nodes[i+1].XMLName.Local)
			require.NotEmpty(t, nodes[i].TestClass,
				"%s/%s 缺少 testclass 属性，JMeter 据此实例化元素", path, tag)
			i++ // 配对的 hashTree 已消费，由上面的递归分支之外单独处理
			walk(nodes[i].Children, path+"/"+tag)
		}
	}

	var root genericNode
	require.NoError(t, xml.Unmarshal(data, &root), "导出的内容不是合法 XML")
	require.Len(t, root.Children, 1, "jmeterTestPlan 下应恰好一个 hashTree")
	walk(root.Children[0].Children, "")
}

// genericNode 只为结构检查而设的通用节点
type genericNode struct {
	XMLName   xml.Name
	TestClass string        `xml:"testclass,attr"`
	Children  []genericNode `xml:",any"`
}

// TestRoundTripConverges 来回转换必须收敛，而不是每转一次就改写一次脚本
//
// 第一轮导出与第二轮往往不同（导入侧会把组级元素下发给采样器，是既定语义），
// 但从第二轮起必须稳定下来。否则使用者在两个工具间同步时，每次都会看到
// 一堆与自己无关的结构改动，久而久之就不敢再信任这个转换
func TestRoundTripConverges(t *testing.T) {
	cfg := baseCfg()
	cfg.ThinkTime = &ThinkTimeConfig{Type: ThinkTimeUniform,
		Delay: 200 * time.Millisecond, Range: 800 * time.Millisecond}
	cfg.APIs = append(cfg.APIs, APIConfig{
		Name: "orders", URL: "https://api.example.com/v1/orders", Method: "GET",
		Verify: []VerifyConfig{
			{Type: types.VerifyTypeContains, Operator: types.OpContains, Expect: `"code":0`},
		},
	})

	dir := t.TempDir()
	exportTo := func(c *Config, file string) []byte {
		data, _, err := ExportJMeterPlan(c)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, file), data, 0o644))
		return data
	}
	importFrom := func(file string) *Config {
		back, rep, err := LoadJMeterPlan(filepath.Join(dir, file), true)
		require.NoError(t, err)
		require.False(t, rep.HasFatal(), "重新导入出现致命丢弃:\n%s", rep.String())
		normalizeAPIs(back)
		return back
	}

	exportTo(cfg, "r1.jmx")
	second := exportTo(importFrom("r1.jmx"), "r2.jmx")
	third := exportTo(importFrom("r2.jmx"), "r3.jmx")
	fourth := exportTo(importFrom("r3.jmx"), "r4.jmx")

	require.Equal(t, string(second), string(third), "第二轮与第三轮应完全一致")
	require.Equal(t, string(third), string(fourth), "第三轮与第四轮应完全一致")
}

// TestRoundTripTimeout 超时必须能一路还原
//
// 回归用：导入侧原先只从 HTTP 请求默认值里读超时，从不读采样器自己的，
// 而 JMeter 的 HTTP Request 界面上就有 Connect/Response Timeout 两个输入框。
// 结果是脚本里写了 10s，导进来却按 go-stress 的默认 30s 跑 —— 超时阈值
// 直接决定失败计数，这种静默差异最误导人
func TestRoundTripTimeout(t *testing.T) {
	for _, to := range []time.Duration{3 * time.Second, 10 * time.Second, 90 * time.Second} {
		cfg := baseCfg()
		cfg.CSVData = nil
		cfg.Timeout = to

		back, _, _ := roundTrip(t, cfg)
		assert.Equal(t, to, back.Timeout, "超时 %s 应原样还原", to)
	}
}

// TestImportSamplerTimeout 采样器上的超时也要被读到
func TestImportSamplerTimeout(t *testing.T) {
	jmx := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="p" enabled="true"/>
    <hashTree>
      <ThreadGroup guiclass="ThreadGroupGui" testclass="ThreadGroup" testname="g" enabled="true">
        <stringProp name="ThreadGroup.num_threads">2</stringProp>
        <stringProp name="ThreadGroup.ramp_time">0</stringProp>
        <elementProp name="ThreadGroup.main_controller" elementType="LoopController">
          <boolProp name="LoopController.continue_forever">false</boolProp>
          <stringProp name="LoopController.loops">5</stringProp>
        </elementProp>
        <boolProp name="ThreadGroup.scheduler">false</boolProp>
      </ThreadGroup>
      <hashTree>
        <HTTPSamplerProxy guiclass="HttpTestSampleGui" testclass="HTTPSamplerProxy" testname="s" enabled="true">
          <stringProp name="HTTPSampler.domain">api.example.com</stringProp>
          <stringProp name="HTTPSampler.protocol">https</stringProp>
          <stringProp name="HTTPSampler.path">/v1/x</stringProp>
          <stringProp name="HTTPSampler.method">GET</stringProp>
          <stringProp name="HTTPSampler.connect_timeout">2000</stringProp>
          <stringProp name="HTTPSampler.response_timeout">8000</stringProp>
        </HTTPSamplerProxy>
        <hashTree/>
      </hashTree>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`

	path := filepath.Join(t.TempDir(), "t.jmx")
	require.NoError(t, os.WriteFile(path, []byte(jmx), 0o644))

	cfg, rep, err := LoadJMeterPlan(path, true)
	require.NoError(t, err, "报告:\n%s", rep)
	assert.Equal(t, 10*time.Second, cfg.Timeout,
		"采样器上的 connect 2s + response 8s 应折算为整体 10s")
	assert.Contains(t, rep.String(), "两者之和", "折算方式要说明")
}

// TestImportConflictingTimeouts 多处超时不一致时取最大并如实说明
func TestImportConflictingTimeouts(t *testing.T) {
	sampler := func(name, path, respMS string) string {
		return `<HTTPSamplerProxy guiclass="HttpTestSampleGui" testclass="HTTPSamplerProxy" testname="` + name + `" enabled="true">
          <stringProp name="HTTPSampler.domain">h</stringProp>
          <stringProp name="HTTPSampler.protocol">http</stringProp>
          <stringProp name="HTTPSampler.path">` + path + `</stringProp>
          <stringProp name="HTTPSampler.method">GET</stringProp>
          <stringProp name="HTTPSampler.response_timeout">` + respMS + `</stringProp>
        </HTTPSamplerProxy>
        <hashTree/>`
	}

	jmx := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="p" enabled="true"/>
    <hashTree>
      <ThreadGroup guiclass="ThreadGroupGui" testclass="ThreadGroup" testname="g" enabled="true">
        <stringProp name="ThreadGroup.num_threads">2</stringProp>
        <stringProp name="ThreadGroup.ramp_time">0</stringProp>
        <elementProp name="ThreadGroup.main_controller" elementType="LoopController">
          <boolProp name="LoopController.continue_forever">false</boolProp>
          <stringProp name="LoopController.loops">5</stringProp>
        </elementProp>
        <boolProp name="ThreadGroup.scheduler">false</boolProp>
      </ThreadGroup>
      <hashTree>
        ` + sampler("a", "/a", "3000") + `
        ` + sampler("b", "/b", "12000") + `
      </hashTree>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`

	path := filepath.Join(t.TempDir(), "t.jmx")
	require.NoError(t, os.WriteFile(path, []byte(jmx), 0o644))

	cfg, rep, err := LoadJMeterPlan(path, true)
	require.NoError(t, err, "报告:\n%s", rep)
	assert.Equal(t, 12*time.Second, cfg.Timeout,
		"取最大：取最小会让本可跑更久的请求被提前判超时")
	assert.Contains(t, rep.String(), "不同的超时设定")
}
