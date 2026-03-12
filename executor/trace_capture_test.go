/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 16:12:30
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 16:28:11
 * @FilePath: \go-stress\executor\trace_capture_test.go
 * @Description: 响应头追踪 ID 回捞的测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import "testing"

// TestTraceCaptureCaseInsensitive 大小写不敏感是这个功能的立身之本：
// HTTP/1.1 经 Go 规范化成 X-Trace-Id，HTTP/2 在线路上全小写，
// 自建网关还可能回带 X-TRACE-ID，按字面比对三种里只能中一种
func TestTraceCaptureCaseInsensitive(t *testing.T) {
	c := newTraceCapture(nil)

	cases := []struct {
		name    string
		headers map[string]string
		wantKey string
		wantVal string
	}{
		{"规范大小写", map[string]string{"X-Trace-Id": "abc"}, "X-Trace-Id", "abc"},
		{"全小写", map[string]string{"x-trace-id": "abc"}, "x-trace-id", "abc"},
		{"全大写", map[string]string{"X-TRACE-ID": "abc"}, "X-TRACE-ID", "abc"},
		{"混合大小写", map[string]string{"X-tRaCe-Id": "abc"}, "X-tRaCe-Id", "abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotKey, gotVal := c.find(tc.headers)
			if gotKey != tc.wantKey || gotVal != tc.wantVal {
				t.Fatalf("find(%v) = (%q, %q)，期望 (%q, %q)",
					tc.headers, gotKey, gotVal, tc.wantKey, tc.wantVal)
			}
		})
	}
}

// TestTraceCapturePriority 多个字段同时存在时按配置顺序取，而不是看 map 的遍历顺序
func TestTraceCapturePriority(t *testing.T) {
	c := newTraceCapture(nil)
	headers := map[string]string{
		"X-Request-Id": "req-1",
		"X-Trace-Id":   "trace-1",
		"traceparent":  "00-aaaa-bbbb-01",
		"Content-Type": "application/json",
	}

	// 默认列表里 X-Trace-Id 排在 X-Request-Id 之前
	for i := 0; i < 50; i++ { // map 遍历是随机的，重复多次才能证明顺序不是碰巧
		key, val := c.find(headers)
		if key != "X-Trace-Id" || val != "trace-1" {
			t.Fatalf("第 %d 次取到 (%q, %q)，期望 X-Trace-Id/trace-1", i, key, val)
		}
	}
}

// TestTraceCaptureCustomOrder 自定义列表的顺序即优先级，必须盖过默认列表
func TestTraceCaptureCustomOrder(t *testing.T) {
	c := newTraceCapture([]string{"X-Request-Id", "X-Trace-Id"})
	headers := map[string]string{"X-Request-Id": "req-1", "X-Trace-Id": "trace-1"}

	for i := 0; i < 50; i++ {
		key, _ := c.find(headers)
		if key != "X-Request-Id" {
			t.Fatalf("第 %d 次取到 %q，自定义列表里 X-Request-Id 排第一", i, key)
		}
	}
}

// TestTraceCaptureSkipsEmptyValue 字段在但取值为空等于没捞到，应继续找下一个
func TestTraceCaptureSkipsEmptyValue(t *testing.T) {
	c := newTraceCapture(nil)
	key, val := c.find(map[string]string{"X-Trace-Id": "", "X-Request-Id": "req-1"})
	if key != "X-Request-Id" || val != "req-1" {
		t.Fatalf("取到 (%q, %q)，空值的 X-Trace-Id 应被跳过", key, val)
	}
}

// TestTraceCaptureDisabled 填 "-" 表示整体关闭
func TestTraceCaptureDisabled(t *testing.T) {
	c := newTraceCapture([]string{DisableMarker})
	if c.enabled() {
		t.Fatal(`填 "-" 后 enabled() 应为 false`)
	}
	if key, val := c.find(map[string]string{"X-Trace-Id": "abc"}); key != "" || val != "" {
		t.Fatalf("关闭后仍捞到 (%q, %q)", key, val)
	}
}

// TestTraceCaptureFallbackToDefault 空列表与全空白项都回落到默认列表，而不是静默关闭
func TestTraceCaptureFallbackToDefault(t *testing.T) {
	for _, names := range [][]string{nil, {}, {"", "  "}} {
		c := newTraceCapture(names)
		if !c.enabled() {
			t.Fatalf("newTraceCapture(%v) 应回落到默认列表", names)
		}
		if key, _ := c.find(map[string]string{"x-trace-id": "abc"}); key != "x-trace-id" {
			t.Fatalf("newTraceCapture(%v) 未按默认列表匹配", names)
		}
	}
}

// TestTraceCaptureMiss 没有任何候选字段时不应误报
func TestTraceCaptureMiss(t *testing.T) {
	c := newTraceCapture(nil)
	headers := map[string]string{
		"Content-Type":   "application/json",
		"Content-Length": "31",
		// 长度与 Trace-Id 相同但内容不同，用于盯住「先比长度」那一步的剪枝不能放行错值
		"X-Other-Id": "nope",
	}
	if key, val := c.find(headers); key != "" || val != "" {
		t.Fatalf("不该命中却取到 (%q, %q)", key, val)
	}
}

// BenchmarkTraceCaptureMiss 未命中是热路径上的常态：逐请求都要扫一遍响应头，
// 这个数字必须远小于一次网络往返才谈得上「不拖慢压测本身」
func BenchmarkTraceCaptureMiss(b *testing.B) {
	c := newTraceCapture(nil)
	headers := map[string]string{
		"Content-Type":   "application/json",
		"Content-Length": "128",
		"Date":           "Tue, 03 Nov 2026 06:20:23 GMT",
		"Server":         "nginx",
		"Connection":     "keep-alive",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCaptureKey, benchCaptureVal = c.find(headers)
	}
}

// BenchmarkTraceCaptureHit 命中时提前返回，应比未命中更快
func BenchmarkTraceCaptureHit(b *testing.B) {
	c := newTraceCapture(nil)
	headers := map[string]string{
		"Content-Type": "application/json",
		"X-Trace-Id":   "0af7651916cd43dd8448eb211c80319c",
		"Date":         "Tue, 03 Nov 2026 06:20:23 GMT",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCaptureKey, benchCaptureVal = c.find(headers)
	}
}

// benchCaptureKey/Val 汇出基准结果，防止编译器把整个调用消去
var benchCaptureKey, benchCaptureVal string
