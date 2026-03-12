/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 16:19:52
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 16:31:06
 * @FilePath: \go-stress\executor\trace_e2e_test.go
 * @Description: 追踪头端到端测试——发出的请求头与回捞的响应头都要落到结果上
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/statistics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seenHeaders 记录服务端实际收到的请求头，供断言「确实发出去了」
type seenHeaders struct {
	mu   sync.Mutex
	list []http.Header
}

func (s *seenHeaders) add(h http.Header) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.list = append(s.list, h.Clone())
}

func (s *seenHeaders) snapshot() []http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]http.Header, len(s.list))
	copy(out, s.list)
	return out
}

// traceEchoServer 回带一个服务端自造的追踪 ID，并记下收到的请求头
//
// respHeader 为空时什么都不回带，用于验证没捞到时不误报
func traceEchoServer(t *testing.T, respHeader, respValue string) (string, *seenHeaders) {
	t.Helper()
	seen := &seenHeaders{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.add(r.Header)
		if respHeader != "" {
			// 直接写 map 而非 Set，避免被规范化，以便按原样大小写回带
			w.Header()[respHeader] = []string{respValue}
		}
		fmt.Fprint(w, `{"code":0}`)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, seen
}

// runTrace 以给定的 advanced 配置跑一轮，返回请求明细
//
// 明细从 Collector 取而不是从 Report 取：NoReport 模式下报告本身不带明细
func runTrace(t *testing.T, url string, tune func(*config.AdvancedConfig)) []*RequestResult {
	t.Helper()
	result := RunTask(RunOptions{
		Logger:      logger.NewEmptyLogger(),
		StorageMode: StorageModeMemory,
		NoReport:    true,
		NoPrint:     true,
		NoWait:      true,
		ConfigFunc: func() *config.Config {
			cfg := config.DefaultConfig()
			cfg.Concurrency = 1
			cfg.Requests = 2
			cfg.Timeout = 5 * time.Second
			cfg.Advanced.DisableRealtime = true
			if tune != nil {
				tune(cfg.Advanced)
			}
			cfg.APIs = []config.APIConfig{{
				Name:   "traced",
				URL:    url,
				Method: http.MethodGet,
			}}
			return cfg
		},
	})
	require.NoError(t, result.Error)
	require.NotNil(t, result.Executor)

	details := result.Executor.GetCollector().GetRequestDetails(
		0, 100, statistics.StatusFilterAll, "", "",
	)
	require.NotEmpty(t, details, "应采集到请求明细")
	return details
}

// TestTraceHeadersInjected 默认同时带上 traceparent 与 X-Request-Id，且两者同值
//
// 同值是这套设计的关键：对端记的是哪个头并不确定，取同一个字符串才能
// 保证无论它记了哪个，回头都能在报告里反查到同一次请求
func TestTraceHeadersInjected(t *testing.T) {
	url, seen := traceEchoServer(t, "", "")
	details := runTrace(t, url, nil)

	got := seen.snapshot()
	require.Len(t, got, 2, "服务端应收到 2 个请求")

	for i, h := range got {
		tp := h.Get(HeaderTraceparent)
		rid := h.Get(HeaderRequestID)
		require.NotEmpty(t, tp, "第 %d 个请求缺少 traceparent", i)
		require.NotEmpty(t, rid, "第 %d 个请求缺少 %s", i, HeaderRequestID)
		assert.Len(t, rid, 32, "请求 ID 应是 32 位十六进制的 trace-id")
		assert.Equal(t, "00-"+rid+"-", tp[:3+32+1],
			"X-Request-Id 必须与 traceparent 里的 trace-id 同值")
	}

	// 两次请求的标识不应相同，否则无法逐条定位
	assert.NotEqual(t, got[0].Get(HeaderRequestID), got[1].Get(HeaderRequestID))

	// 对端什么都没回带时不该凭空造出一个服务端 ID
	for _, d := range details {
		assert.Empty(t, d.ServerTraceID, "对端未回带时 ServerTraceID 应为空")
		assert.Empty(t, d.ServerTraceHeader)
	}
}

// TestTraceCapturedFromResponse 响应头里的追踪 ID 要落到结果上，大小写随对端
func TestTraceCapturedFromResponse(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{"规范大小写", "X-Trace-Id"},
		{"全小写", "x-trace-id"},
		{"全大写", "X-TRACE-ID"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url, _ := traceEchoServer(t, tc.header, "srv-abc-123")
			details := runTrace(t, url, nil)

			for _, d := range details {
				assert.Equal(t, "srv-abc-123", d.ServerTraceID)
				// Go 的 HTTP 客户端会把响应头规范化，故这里只断言「对得上」
				assert.True(t, len(d.ServerTraceHeader) == len(tc.header),
					"记下的头名 %q 与对端发出的 %q 长度应一致", d.ServerTraceHeader, tc.header)
			}
		})
	}
}

// TestTraceCaptureCustomHeader 自定义字段：默认列表里没有的头也要能捞到
func TestTraceCaptureCustomHeader(t *testing.T) {
	url, _ := traceEchoServer(t, "X-Biz-Trace", "biz-777")

	// 不配时捞不到：X-Biz-Trace 不在默认列表里
	details := runTrace(t, url, nil)
	for _, d := range details {
		assert.Empty(t, d.ServerTraceID, "未配置的字段不应被捞到")
	}

	// 配上就捞得到
	details = runTrace(t, url, func(a *config.AdvancedConfig) {
		a.TraceCaptureHeaders = []string{"x-biz-trace"} // 故意用小写，验证不区分大小写
	})
	for _, d := range details {
		assert.Equal(t, "biz-777", d.ServerTraceID)
	}
}

// TestTraceHeadersDisabled 关闭后两个头都不该出现在请求里
func TestTraceHeadersDisabled(t *testing.T) {
	url, seen := traceEchoServer(t, "", "")
	details := runTrace(t, url, func(a *config.AdvancedConfig) {
		a.DisableTraceHeader = true
	})

	for i, h := range seen.snapshot() {
		assert.Empty(t, h.Get(HeaderTraceparent), "第 %d 个请求不该带 traceparent", i)
		assert.Empty(t, h.Get(HeaderRequestID), "第 %d 个请求不该带 %s", i, HeaderRequestID)
	}

	// 关掉注入不影响本地的 trace_id：报告与日志的关联仍然可用
	for _, d := range details {
		assert.NotEmpty(t, d.TraceID, "关闭注入后仍应生成 trace_id 供本地关联")
	}
}

// TestRequestIDHeaderCustomName 换头名后应按新名字发出，traceparent 不受影响
func TestRequestIDHeaderCustomName(t *testing.T) {
	url, seen := traceEchoServer(t, "", "")
	runTrace(t, url, func(a *config.AdvancedConfig) {
		a.RequestIDHeader = "X-Biz-Request-Id"
	})

	for i, h := range seen.snapshot() {
		assert.NotEmpty(t, h.Get("X-Biz-Request-Id"), "第 %d 个请求缺少自定义请求 ID 头", i)
		assert.Empty(t, h.Get(HeaderRequestID), "换名后不该再带默认的 %s", HeaderRequestID)
		assert.NotEmpty(t, h.Get(HeaderTraceparent), "traceparent 不应受影响")
	}
}

// TestRequestIDHeaderDisabledOnly 只关请求 ID 头，traceparent 仍要带
func TestRequestIDHeaderDisabledOnly(t *testing.T) {
	url, seen := traceEchoServer(t, "", "")
	runTrace(t, url, func(a *config.AdvancedConfig) {
		a.RequestIDHeader = DisableMarker
	})

	for i, h := range seen.snapshot() {
		assert.Empty(t, h.Get(HeaderRequestID), "第 %d 个请求不该带请求 ID 头", i)
		assert.NotEmpty(t, h.Get(HeaderTraceparent), "traceparent 不应被一并关掉")
	}
}
