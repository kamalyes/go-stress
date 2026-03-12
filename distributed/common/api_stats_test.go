/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 18:12:05
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 18:31:50
 * @FilePath: \go-stress\distributed\common\api_stats_test.go
 * @Description: per-API 统计在分布式链路上的聚合与转换测试
 *
 * Copyright (c) 2026 by kamalyes 501893067@qq.com, All Rights Reserved.
 */
package common

import (
	"errors"
	"testing"
	"time"

	"github.com/kamalyes/go-stress/statistics"
	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// res 造一条请求结果
func res(name string, code int, ms float64, success bool) *types.RequestResult {
	return &types.RequestResult{
		APIName:    name,
		Method:     "GET",
		URL:        "http://x/" + name,
		StatusCode: code,
		Success:    success,
		Duration:   time.Duration(ms * float64(time.Millisecond)),
	}
}

// TestAggregateAPIStatsGrouping 按 API 名分组，计数与成功率都要对
func TestAggregateAPIStatsGrouping(t *testing.T) {
	results := []*types.RequestResult{
		res("login", 200, 5, true),
		res("login", 200, 7, true),
		res("orders", 500, 20, false),
		res("orders", 200, 10, true),
	}
	results[2].ErrorMsg = "验证失败: STATUS_CODE"

	stats := AggregateAPIStats(results, 2)
	require.Len(t, stats, 2)

	byName := map[string]statistics.APIStat{}
	for _, s := range stats {
		byName[s.Name] = s
	}

	login := byName["login"]
	assert.EqualValues(t, 2, login.Requests)
	assert.EqualValues(t, 2, login.Success)
	assert.EqualValues(t, 100, login.SuccessRate)
	assert.EqualValues(t, map[int]uint64{200: 2}, login.StatusCodes)
	assert.Equal(t, 1.0, login.QPS, "2 条请求 / 2 秒窗口")
	assert.Equal(t, "GET", login.Method)

	orders := byName["orders"]
	assert.EqualValues(t, 2, orders.Requests)
	assert.EqualValues(t, 1, orders.Failed)
	assert.EqualValues(t, 50, orders.SuccessRate)
	assert.EqualValues(t, map[int]uint64{200: 1, 500: 1}, orders.StatusCodes)
	assert.EqualValues(t, map[string]uint64{"验证失败: STATUS_CODE": 1}, orders.Errors)
}

// TestAggregateAPIStatsSubMillisecond 亚毫秒请求不能被抹成 0
//
// 回归用：全局那条路径走 Duration.Milliseconds() 向下取整，
// 0.3ms 的请求会变成 0ms，per-API 这条必须用纳秒换算
func TestAggregateAPIStatsSubMillisecond(t *testing.T) {
	stats := AggregateAPIStats([]*types.RequestResult{
		res("fast", 200, 0.3, true),
		res("fast", 200, 0.7, true),
	}, 1)
	require.Len(t, stats, 1)

	assert.InDelta(t, 0.5, stats[0].AvgMs, 1e-9)
	assert.InDelta(t, 0.3, stats[0].MinMs, 1e-9)
	assert.InDelta(t, 0.7, stats[0].MaxMs, 1e-9)
}

// TestAggregateAPIStatsSkippedExcludedFromRate 跳过不计入成功率的分母
func TestAggregateAPIStatsSkippedExcludedFromRate(t *testing.T) {
	skipped := res("dep", 0, 0, false)
	skipped.Skipped = true

	stats := AggregateAPIStats([]*types.RequestResult{
		res("dep", 200, 1, true),
		skipped,
	}, 1)
	require.Len(t, stats, 1)

	assert.EqualValues(t, 2, stats[0].Requests)
	assert.EqualValues(t, 1, stats[0].Skipped)
	assert.EqualValues(t, 100, stats[0].SuccessRate, "跳过不该把成功率拉到 50%")
	// 只有那条真实发出的请求贡献了 200；跳过的那条 StatusCode 为 0，
	// 不该在分布里造出一个看不懂的 "0" 桶
	assert.Equal(t, map[int]uint64{200: 1}, stats[0].StatusCodes)
}

// TestAggregateAPIStatsUnnamedDropped 没有 API 名的结果归不进任何一行
func TestAggregateAPIStatsUnnamedDropped(t *testing.T) {
	assert.Nil(t, AggregateAPIStats([]*types.RequestResult{res("", 200, 1, true)}, 1))
	assert.Nil(t, AggregateAPIStats(nil, 1))
}

// TestAggregateAPIStatsErrorFallback ErrorMsg 为空时回落到 Error
func TestAggregateAPIStatsErrorFallback(t *testing.T) {
	r := res("x", 500, 1, false)
	r.Error = errors.New("connection refused")

	stats := AggregateAPIStats([]*types.RequestResult{r}, 1)
	require.Len(t, stats, 1)
	assert.EqualValues(t, map[string]uint64{"connection refused": 1}, stats[0].Errors)
}

// TestMergeAPIStatsWeightedAverage 平均耗时按请求数加权，且必须在更新计数之前算
//
// 这是最容易写错的一步：先把 Requests 加上去再算加权平均，分母就已经变了，
// 结果会悄悄偏向后到的那个窗口
func TestMergeAPIStatsWeightedAverage(t *testing.T) {
	acc := map[string]*statistics.APIStat{}

	// 第一个窗口：90 条，平均 10ms
	MergeAPIStats(acc, []statistics.APIStat{{
		Name: "api", Requests: 90, Success: 90, AvgMs: 10, MinMs: 5, MaxMs: 15,
		P50Ms: 10, P95Ms: 14, P99Ms: 15, QPS: 90,
	}})
	// 第二个窗口：10 条，平均 110ms
	MergeAPIStats(acc, []statistics.APIStat{{
		Name: "api", Requests: 10, Success: 10, AvgMs: 110, MinMs: 100, MaxMs: 120,
		P50Ms: 110, P95Ms: 119, P99Ms: 120, QPS: 10,
	}})

	got := acc["api"]
	require.NotNil(t, got)
	assert.EqualValues(t, 100, got.Requests)
	// (90*10 + 10*110) / 100 = 20；写错成后置更新会算出 (900+1100)/100 以外的值
	assert.InDelta(t, 20.0, got.AvgMs, 1e-9)
	assert.EqualValues(t, 100, got.SuccessRate)

	// QPS 不由合并给出：来源既有「同一窗口的不同 Slave」也有「同一 Slave 的
	// 不同窗口」，相加会把 N 秒的速率叠成 N 倍。累计视图统一由 RecomputeAPIQPS
	// 按「累计请求数 / 实际耗时」重算
	assert.Zero(t, got.QPS, "合并阶段不该给出 QPS")

	flat := FlattenAPIStats(acc)
	RecomputeAPIQPS(flat, 10)
	assert.InDelta(t, 10.0, flat[0].QPS, 1e-9, "100 次 / 10 秒")
}

// TestRecomputeAPIQPSGuardsZero 耗时为 0 或负数时不改动，避免除零造出 Inf
func TestRecomputeAPIQPSGuardsZero(t *testing.T) {
	for _, sec := range []float64{0, -1} {
		stats := []statistics.APIStat{{Name: "api", Requests: 10, QPS: 3}}
		RecomputeAPIQPS(stats, sec)
		assert.InDelta(t, 3.0, stats[0].QPS, 1e-9)
	}
}

// TestMergeSlaveStatsAccumulates 按 Slave 累加而非覆盖
//
// 回归用：聚合器原先是 agg.SlaveStats[id] = stats 直接覆盖，于是分布式面板
// 带着 ?slave_id= 打开实时报告时，页面把最近一秒的几十次请求当成了总数
func TestMergeSlaveStatsAccumulates(t *testing.T) {
	dst := CloneSlaveStats(&SlaveStats{
		SlaveID: "s1", TotalRequests: 100, SuccessRequests: 90, FailedRequests: 10,
		AvgLatency: 10, MinLatency: 2, MaxLatency: 30, P95Latency: 25, P99Latency: 30,
		StatusCodes: map[int]int64{200: 90, 500: 10},
		ErrorTypes:  map[string]int64{"boom": 10},
		APIStats:    []statistics.APIStat{{Name: "login", Requests: 100, Success: 90, Failed: 10}},
	})

	MergeSlaveStats(dst, &SlaveStats{
		SlaveID: "s1", TotalRequests: 50, SuccessRequests: 50,
		AvgLatency: 40, MinLatency: 1, MaxLatency: 20, P95Latency: 18, P99Latency: 20,
		StatusCodes: map[int]int64{200: 50},
		APIStats:    []statistics.APIStat{{Name: "login", Requests: 50, Success: 50}},
	})

	assert.EqualValues(t, 150, dst.TotalRequests)
	assert.EqualValues(t, 140, dst.SuccessRequests)
	assert.EqualValues(t, 10, dst.FailedRequests)
	// (100*10 + 50*40) / 150 = 20
	assert.InDelta(t, 20.0, dst.AvgLatency, 1e-9)
	assert.InDelta(t, 1.0, dst.MinLatency, 1e-9, "Min 取极小")
	assert.InDelta(t, 30.0, dst.MaxLatency, 1e-9, "Max 取极大")
	assert.InDelta(t, 25.0, dst.P95Latency, 1e-9, "百分位取较大者")
	assert.Equal(t, map[int]int64{200: 140, 500: 10}, dst.StatusCodes)
	assert.Equal(t, map[string]int64{"boom": 10}, dst.ErrorTypes)
	assert.InDelta(t, 140.0/150*100, dst.SuccessRate, 1e-9)

	require.Len(t, dst.APIStats, 1)
	assert.EqualValues(t, 150, dst.APIStats[0].Requests, "per-API 也要跟着累加")
}

// TestCloneSlaveStatsDeepCopies 累计表接管首个窗口时必须深拷贝
func TestCloneSlaveStatsDeepCopies(t *testing.T) {
	src := &SlaveStats{
		SlaveID: "s1", TotalRequests: 10,
		StatusCodes: map[int]int64{200: 10},
		ErrorTypes:  map[string]int64{"e": 1},
		APIStats:    []statistics.APIStat{{Name: "a", Requests: 10}},
	}
	cp := CloneSlaveStats(src)
	MergeSlaveStats(cp, &SlaveStats{
		TotalRequests: 5,
		StatusCodes:   map[int]int64{200: 5},
		APIStats:      []statistics.APIStat{{Name: "a", Requests: 5}},
	})

	assert.EqualValues(t, 15, cp.TotalRequests)
	assert.EqualValues(t, 10, src.TotalRequests, "来源不该被改写")
	assert.Equal(t, map[int]int64{200: 10}, src.StatusCodes, "来源 map 不该被改写")
	assert.EqualValues(t, 10, src.APIStats[0].Requests, "来源 APIStats 不该被改写")

	assert.Nil(t, CloneSlaveStats(nil))
}

// TestMergeSlaveStatsNilSafe nil 入参不应 panic
func TestMergeSlaveStatsNilSafe(t *testing.T) {
	assert.NotPanics(t, func() {
		MergeSlaveStats(nil, &SlaveStats{TotalRequests: 1})
		MergeSlaveStats(&SlaveStats{}, nil)
		MergeSlaveStats(nil, nil)
	})
}

// TestMergeAPIStatsLatencyBounds Min 取极小、Max 与百分位取极大
func TestMergeAPIStatsLatencyBounds(t *testing.T) {
	acc := map[string]*statistics.APIStat{}
	MergeAPIStats(acc, []statistics.APIStat{{
		Name: "api", Requests: 1, MinMs: 8, MaxMs: 20, P50Ms: 10, P95Ms: 18, P99Ms: 20,
	}})
	MergeAPIStats(acc, []statistics.APIStat{{
		Name: "api", Requests: 1, MinMs: 3, MaxMs: 12, P50Ms: 6, P95Ms: 11, P99Ms: 12,
	}})

	got := acc["api"]
	assert.InDelta(t, 3.0, got.MinMs, 1e-9, "Min 取各窗口的极小值")
	assert.InDelta(t, 20.0, got.MaxMs, 1e-9, "Max 取各窗口的极大值")
	// 百分位取较大值：合并窗口的真实百分位介于两者之间，宁可保守也不低报
	assert.InDelta(t, 10.0, got.P50Ms, 1e-9)
	assert.InDelta(t, 18.0, got.P95Ms, 1e-9)
	assert.InDelta(t, 20.0, got.P99Ms, 1e-9)
}

// TestMergeAPIStatsZeroMinIsNotSentinel MinMs 为 0 要能被真实的更大值取代
//
// 首个窗口若快到 0.00ms，拿 0 当「还没数据」的哨兵就会把后续所有 Min 都顶掉
func TestMergeAPIStatsZeroMinIsNotSentinel(t *testing.T) {
	acc := map[string]*statistics.APIStat{}
	MergeAPIStats(acc, []statistics.APIStat{{Name: "api", Requests: 1, MinMs: 0}})
	MergeAPIStats(acc, []statistics.APIStat{{Name: "api", Requests: 1, MinMs: 7}})
	assert.InDelta(t, 7.0, acc["api"].MinMs, 1e-9, "0 之后来的 7 应成为新的 Min")

	// 反过来：先 7 后 3，应取 3
	acc2 := map[string]*statistics.APIStat{}
	MergeAPIStats(acc2, []statistics.APIStat{{Name: "api", Requests: 1, MinMs: 7}})
	MergeAPIStats(acc2, []statistics.APIStat{{Name: "api", Requests: 1, MinMs: 3}})
	assert.InDelta(t, 3.0, acc2["api"].MinMs, 1e-9)
}

// TestMergeAPIStatsDoesNotAliasMaps 并入首个窗口时必须复制 map
//
// 直接持有来源 map 的话，下一个窗口的累加会写穿到调用方的数据上
func TestMergeAPIStatsDoesNotAliasMaps(t *testing.T) {
	src := statistics.APIStat{
		Name: "api", Requests: 1,
		StatusCodes: map[int]uint64{200: 1},
		Errors:      map[string]uint64{"boom": 1},
	}
	acc := map[string]*statistics.APIStat{}
	MergeAPIStats(acc, []statistics.APIStat{src})
	MergeAPIStats(acc, []statistics.APIStat{{
		Name: "api", Requests: 1,
		StatusCodes: map[int]uint64{200: 1},
		Errors:      map[string]uint64{"boom": 1},
	}})

	assert.EqualValues(t, 2, acc["api"].StatusCodes[200])
	assert.EqualValues(t, 1, src.StatusCodes[200], "来源 map 不该被改写")
	assert.EqualValues(t, 1, src.Errors["boom"], "来源 map 不该被改写")
}

// TestAPIStatsProtoRoundTrip 过一遍 proto 再回来，字段不能丢
func TestAPIStatsProtoRoundTrip(t *testing.T) {
	in := []statistics.APIStat{{
		Name: "login", Method: "POST", URL: "http://x/login",
		Requests: 10, Success: 8, Failed: 2, Skipped: 0,
		AvgMs: 12.5, MinMs: 3.25, MaxMs: 40.5,
		P50Ms: 11, P95Ms: 38, P99Ms: 40, QPS: 5,
		StatusCodes: map[int]uint64{200: 8, 500: 2},
		Errors:      map[string]uint64{"boom": 2},
	}}

	out := APIStatsFromProto(APIStatsToProto(in))
	require.Len(t, out, 1)
	got := out[0]

	assert.Equal(t, in[0].Name, got.Name)
	assert.Equal(t, in[0].Method, got.Method)
	assert.Equal(t, in[0].URL, got.URL)
	assert.Equal(t, in[0].Requests, got.Requests)
	assert.Equal(t, in[0].Success, got.Success)
	assert.Equal(t, in[0].Failed, got.Failed)
	assert.InDelta(t, in[0].AvgMs, got.AvgMs, 1e-9)
	assert.InDelta(t, in[0].MinMs, got.MinMs, 1e-9)
	assert.InDelta(t, in[0].P99Ms, got.P99Ms, 1e-9)
	assert.Equal(t, in[0].StatusCodes, got.StatusCodes)
	assert.Equal(t, in[0].Errors, got.Errors)
	// 成功率不过线路，由接收侧按计数重算
	assert.InDelta(t, 80.0, got.SuccessRate, 1e-9)
}

// TestStatusCodesFromProtoDropsGarbage 状态码键不是数字说明上游已坏，直接丢
func TestStatusCodesFromProtoDropsGarbage(t *testing.T) {
	got := statusCodesFromProto(map[string]int64{"200": 3, "abc": 9})
	assert.Equal(t, map[int]uint64{200: 3}, got, `"abc" 不该变成 0 这一桶`)
}

// TestFlattenAPIStatsOrdering 请求数降序；相同则按名称，保证渲染顺序稳定
func TestFlattenAPIStatsOrdering(t *testing.T) {
	acc := map[string]*statistics.APIStat{
		"b": {Name: "b", Requests: 5},
		"a": {Name: "a", Requests: 5},
		"c": {Name: "c", Requests: 50},
	}
	got := FlattenAPIStats(acc)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"c", "a", "b"}, []string{got[0].Name, got[1].Name, got[2].Name})

	assert.Nil(t, FlattenAPIStats(nil))
}

// TestVerificationsProtoRoundTrip 验证明细过线路不能丢 expect/actual
//
// Master 的下钻靠这两项给出「期望 200，实际 404」这种一眼能懂的原因
func TestVerificationsProtoRoundTrip(t *testing.T) {
	in := []types.VerificationResult{{
		Type:     types.VerifyTypeStatusCode,
		Success:  false,
		Message:  "number comparison failed: actual 404, operator eq, expected 200",
		Expect:   "200",
		Actual:   "404",
		Operator: "eq",
	}}

	out := VerificationsFromProto(VerificationsToProto(in))
	require.Len(t, out, 1)
	assert.Equal(t, in[0], out[0])

	assert.Nil(t, VerificationsToProto(nil))
	assert.Nil(t, VerificationsFromProto(nil))
}
