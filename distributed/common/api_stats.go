/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 17:05:16
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 17:28:36
 * @FilePath: \go-stress\distributed\common\api_stats.go
 * @Description: 按 API 名聚合的统计在分布式链路上的聚合与转换
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package common

import (
	"math"
	"sort"
	"strconv"

	pb "github.com/kamalyes/go-stress/distributed/proto"
	"github.com/kamalyes/go-stress/statistics"
	"github.com/kamalyes/go-stress/types"
)

// 这一组函数是 per-API 统计在 Slave → Master → 报告前端之间的唯一转换点
//
// 之所以集中在 common 而不是各写各的：报告前端（report.js）读的是
// statistics.APIStat 的 JSON 标签，单机与分布式共用同一份渲染代码
// 转换一旦在两端各写一遍，加字段时漏掉任何一处，分布式这侧的 API 汇总
// 就会静默少一块 —— 这正是此前整个 api_stats 在 Master 上消失的原因

// AggregateAPIStats 把一个上报窗口内的请求结果按 API 名聚合
//
// seconds 为该窗口的秒数，用于算每个 API 的 QPS；为 0 时 QPS 留空而不是除零
func AggregateAPIStats(results []*types.RequestResult, seconds float64) []statistics.APIStat {
	if len(results) == 0 {
		return nil
	}

	type bucket struct {
		stat      statistics.APIStat
		latencies []float64
	}
	buckets := make(map[string]*bucket)

	for _, r := range results {
		name := r.APIName
		if name == "" {
			continue // 未命名的接口归不进任何一行，强行造一行反而误导
		}

		b, ok := buckets[name]
		if !ok {
			b = &bucket{stat: statistics.APIStat{
				Name:        name,
				StatusCodes: make(map[int]uint64),
				Errors:      make(map[string]uint64),
				MinMs:       math.MaxFloat64,
			}}
			buckets[name] = b
		}

		// 方法与地址取首次出现的取值：URL 里常含变量，逐次收集会得到
		// 成千上万条互不相同的地址，对阅读毫无帮助
		if b.stat.Method == "" {
			b.stat.Method = r.Method
		}
		if b.stat.URL == "" {
			b.stat.URL = r.URL
		}

		b.stat.Requests++
		switch {
		case r.Skipped:
			b.stat.Skipped++
		case r.Success:
			b.stat.Success++
		default:
			b.stat.Failed++
		}

		if r.StatusCode > 0 {
			b.stat.StatusCodes[r.StatusCode]++
		}
		if msg := errorTextOf(r); msg != "" {
			b.stat.Errors[msg]++
		}

		// 用纳秒换算而非 Duration.Milliseconds()：后者向下取整，
		// 亚毫秒的请求会全部变成 0，而 APIStat 的口径本就是浮点毫秒
		ms := float64(r.Duration.Nanoseconds()) / 1e6
		b.latencies = append(b.latencies, ms)
		b.stat.MinMs = math.Min(b.stat.MinMs, ms)
		b.stat.MaxMs = math.Max(b.stat.MaxMs, ms)
	}

	if len(buckets) == 0 {
		return nil // 结果全都没有 API 名，与无数据同等对待
	}

	out := make([]statistics.APIStat, 0, len(buckets))
	for _, b := range buckets {
		s := b.stat
		if len(b.latencies) > 0 {
			sort.Float64s(b.latencies)
			s.AvgMs = meanOf(b.latencies)
			s.P50Ms = percentileOf(b.latencies, 50)
			s.P95Ms = percentileOf(b.latencies, 95)
			s.P99Ms = percentileOf(b.latencies, 99)
		}
		if s.MinMs == math.MaxFloat64 {
			s.MinMs = 0
		}
		// 成功率的分母不含跳过：跳过不是该接口的真实结果
		if done := s.Success + s.Failed; done > 0 {
			s.SuccessRate = float64(s.Success) / float64(done) * 100
		}
		if seconds > 0 {
			s.QPS = float64(s.Requests) / seconds
		}
		out = append(out, s)
	}

	sortAPIStats(out)
	return out
}

// errorTextOf 取结果的错误文本，口径与 statistics 侧的聚合键保持一致
func errorTextOf(r *types.RequestResult) string {
	if r.ErrorMsg != "" {
		return r.ErrorMsg
	}
	if r.Error != nil {
		return r.Error.Error()
	}
	return ""
}

// APIStatsToProto 转为 proto 消息（Slave 上报用）
func APIStatsToProto(stats []statistics.APIStat) []*pb.APIStatsData {
	if len(stats) == 0 {
		return nil
	}
	out := make([]*pb.APIStatsData, 0, len(stats))
	for _, s := range stats {
		out = append(out, &pb.APIStatsData{
			Name:        s.Name,
			Method:      s.Method,
			Url:         s.URL,
			Requests:    int64(s.Requests),
			Success:     int64(s.Success),
			Failed:      int64(s.Failed),
			Skipped:     int64(s.Skipped),
			AvgLatency:  s.AvgMs,
			MinLatency:  s.MinMs,
			MaxLatency:  s.MaxMs,
			P50Latency:  s.P50Ms,
			P95Latency:  s.P95Ms,
			P99Latency:  s.P99Ms,
			Qps:         s.QPS,
			StatusCodes: statusCodesToProto(s.StatusCodes),
			Errors:      countsToProto(s.Errors),
		})
	}
	return out
}

// APIStatsFromProto 从 proto 消息还原（Master 收数用）
func APIStatsFromProto(data []*pb.APIStatsData) []statistics.APIStat {
	if len(data) == 0 {
		return nil
	}
	out := make([]statistics.APIStat, 0, len(data))
	for _, d := range data {
		if d == nil {
			continue
		}
		s := statistics.APIStat{
			Name:        d.Name,
			Method:      d.Method,
			URL:         d.Url,
			Requests:    uint64(d.Requests),
			Success:     uint64(d.Success),
			Failed:      uint64(d.Failed),
			Skipped:     uint64(d.Skipped),
			AvgMs:       d.AvgLatency,
			MinMs:       d.MinLatency,
			MaxMs:       d.MaxLatency,
			P50Ms:       d.P50Latency,
			P95Ms:       d.P95Latency,
			P99Ms:       d.P99Latency,
			QPS:         d.Qps,
			StatusCodes: statusCodesFromProto(d.StatusCodes),
			Errors:      countsFromProto(d.Errors),
		}
		if done := s.Success + s.Failed; done > 0 {
			s.SuccessRate = float64(s.Success) / float64(done) * 100
		}
		out = append(out, s)
	}
	return out
}

// MergeAPIStats 把一批新上报的 per-API 统计并入累计表
//
// 计数类字段直接相加；延迟没法相加，取舍如下：
//   - Min / Max 取各窗口的极值，这是精确的
//   - Avg 按请求数加权，这也是精确的
//   - P50 / P95 / P99 取各窗口的较大值 —— 合并窗口的真实百分位介于各窗口之间，
//     宁可保守也不低报，与 statistics 侧时序降采样的处理同理
//
// 真正精确的做法是把直方图本身传过来合并，但那要在每次上报里多带 8704 个桶，
// 对一个汇总视图不值当
func MergeAPIStats(acc map[string]*statistics.APIStat, incoming []statistics.APIStat) {
	for i := range incoming {
		in := incoming[i]
		if in.Name == "" {
			continue
		}

		cur, ok := acc[in.Name]
		if !ok {
			cp := in
			cp.StatusCodes = copyUint64Map(in.StatusCodes)
			cp.Errors = copyStringMap(in.Errors)
			cp.QPS = 0 // 理由同下：累计视图的 QPS 由 RecomputeAPIQPS 统一给出
			acc[in.Name] = &cp
			continue
		}

		// 加权平均要在计数更新之前算，否则分母已经变了
		totalReqs := cur.Requests + in.Requests
		if totalReqs > 0 {
			cur.AvgMs = (cur.AvgMs*float64(cur.Requests) + in.AvgMs*float64(in.Requests)) / float64(totalReqs)
		}

		cur.Requests = totalReqs
		cur.Success += in.Success
		cur.Failed += in.Failed
		cur.Skipped += in.Skipped
		// QPS 刻意不累加：来源既有「同一窗口的不同 Slave」也有「同一 Slave 的
		// 不同窗口」，前者该相加、后者该取最新，一个加法无法同时表达两者
		// 累计视图下由 RecomputeAPIQPS 按「累计请求数 / 实际耗时」统一重算

		if cur.Method == "" {
			cur.Method = in.Method
		}
		if cur.URL == "" {
			cur.URL = in.URL
		}

		// MinMs 为 0 可能是「还没有数据」也可能是「真的快到 0.00ms」，
		// 用请求数判断是不是首个窗口，而不是拿 0 当哨兵
		if cur.MinMs == 0 || (in.MinMs > 0 && in.MinMs < cur.MinMs) {
			cur.MinMs = in.MinMs
		}
		cur.MaxMs = math.Max(cur.MaxMs, in.MaxMs)
		cur.P50Ms = math.Max(cur.P50Ms, in.P50Ms)
		cur.P95Ms = math.Max(cur.P95Ms, in.P95Ms)
		cur.P99Ms = math.Max(cur.P99Ms, in.P99Ms)

		if cur.StatusCodes == nil {
			cur.StatusCodes = make(map[int]uint64, len(in.StatusCodes))
		}
		for code, n := range in.StatusCodes {
			cur.StatusCodes[code] += n
		}
		if cur.Errors == nil {
			cur.Errors = make(map[string]uint64, len(in.Errors))
		}
		for msg, n := range in.Errors {
			cur.Errors[msg] += n
		}

		if done := cur.Success + cur.Failed; done > 0 {
			cur.SuccessRate = float64(cur.Success) / float64(done) * 100
		}
	}
}

// FlattenAPIStats 把累计表摊平成切片，按请求数降序
func FlattenAPIStats(acc map[string]*statistics.APIStat) []statistics.APIStat {
	if len(acc) == 0 {
		return nil
	}
	out := make([]statistics.APIStat, 0, len(acc))
	for _, s := range acc {
		out = append(out, *s)
	}
	sortAPIStats(out)
	return out
}

// sortAPIStats 请求数多的排前面，长尾接口不至于被挤到看不见
// 请求数相同时按名称排，保证同一份数据每次渲染的顺序一致
func sortAPIStats(stats []statistics.APIStat) {
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Requests != stats[j].Requests {
			return stats[i].Requests > stats[j].Requests
		}
		return stats[i].Name < stats[j].Name
	})
}

// ---- 小工具 ----

// meanOf 算术平均
func meanOf(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range v {
		sum += x
	}
	return sum / float64(len(v))
}

// percentileOf 取百分位，入参须已升序
// 取上界而非插值：与 statistics 侧的直方图口径一致，宁可保守也不低报
func percentileOf(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func statusCodesToProto(m map[int]uint64) map[string]int64 {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]int64, len(m))
	for code, n := range m {
		out[strconv.Itoa(code)] = int64(n)
	}
	return out
}

func statusCodesFromProto(m map[string]int64) map[int]uint64 {
	if len(m) == 0 {
		return nil
	}
	out := make(map[int]uint64, len(m))
	for code, n := range m {
		// 解析失败的键直接丢弃：状态码不是数字说明上游数据已经坏了，
		// 塞进 0 这一桶只会在报告里多出一行看不懂的 "0"
		if c, err := strconv.Atoi(code); err == nil {
			out[c] += uint64(n)
		}
	}
	return out
}

func countsToProto(m map[string]uint64) map[string]int64 {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]int64, len(m))
	for k, n := range m {
		out[k] = int64(n)
	}
	return out
}

func countsFromProto(m map[string]int64) map[string]uint64 {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]uint64, len(m))
	for k, n := range m {
		out[k] = uint64(n)
	}
	return out
}

func copyUint64Map(m map[int]uint64) map[int]uint64 {
	if m == nil {
		return nil
	}
	out := make(map[int]uint64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyStringMap(m map[string]uint64) map[string]uint64 {
	if m == nil {
		return nil
	}
	out := make(map[string]uint64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ---- 验证结果的 proto 转换 ----
//
// 放在这里与 per-API 统计同处：两者都是「Slave 算、Master 显示」的数据，
// 转换点同样必须唯一

// VerificationsToProto 转为 proto 消息（Slave 返回明细用）
func VerificationsToProto(rules []types.VerificationResult) []*pb.VerificationData {
	if len(rules) == 0 {
		return nil
	}
	out := make([]*pb.VerificationData, 0, len(rules))
	for _, v := range rules {
		out = append(out, &pb.VerificationData{
			Type:        string(v.Type),
			Success:     v.Success,
			Skipped:     v.Skipped,
			Message:     v.Message,
			Expect:      v.Expect,
			Actual:      v.Actual,
			Field:       v.Field,
			Operator:    v.Operator,
			Description: v.Description,
		})
	}
	return out
}

// VerificationsFromProto 从 proto 消息还原（Master 透传明细用）
func VerificationsFromProto(data []*pb.VerificationData) []types.VerificationResult {
	if len(data) == 0 {
		return nil
	}
	out := make([]types.VerificationResult, 0, len(data))
	for _, d := range data {
		if d == nil {
			continue
		}
		out = append(out, types.VerificationResult{
			Type:        types.VerifyType(d.Type),
			Success:     d.Success,
			Skipped:     d.Skipped,
			Message:     d.Message,
			Expect:      d.Expect,
			Actual:      d.Actual,
			Field:       d.Field,
			Operator:    d.Operator,
			Description: d.Description,
		})
	}
	return out
}

// RecomputeAPIQPS 按「累计请求数 / 实际耗时」重算每个 API 的 QPS
//
// 累计视图里不能把各窗口的瞬时 QPS 相加（那等于把 90 秒的速率叠成 90 倍），
// 也不能只取最后一个窗口（那丢掉了跨 Slave 的并行度）。按总量除以耗时
// 得到的是整段压测的平均速率，与单机报告里 QPS 的口径完全一致
func RecomputeAPIQPS(stats []statistics.APIStat, seconds float64) {
	if seconds <= 0 {
		return
	}
	for i := range stats {
		stats[i].QPS = float64(stats[i].Requests) / seconds
	}
}

// MergeSlaveStats 把一个上报窗口累加进该 Slave 的累计统计
//
// 此前聚合器是 agg.SlaveStats[id] = stats 直接覆盖，存的只是最近一个窗口。
// 而分布式面板跳转到实时报告时带的正是 ?slave_id=，于是页面把「最近一秒的
// 几十次请求」当成总数显示 —— Slave 自己报了三万次，Master 页面上写着 56
//
// 速率类字段（QPS/TotalQPS）不在这里处理，由 buildAggregatedStats 按耗时重算
func MergeSlaveStats(dst, src *SlaveStats) {
	if dst == nil || src == nil {
		return
	}

	// 加权平均要在计数更新之前算，否则分母已经变了
	total := dst.TotalRequests + src.TotalRequests
	if total > 0 {
		dst.AvgLatency = (dst.AvgLatency*float64(dst.TotalRequests) +
			src.AvgLatency*float64(src.TotalRequests)) / float64(total)
	}

	dst.TotalRequests = total
	dst.SuccessRequests += src.SuccessRequests
	dst.FailedRequests += src.FailedRequests

	// Min 为 0 可能是「还没数据」也可能是「真的快到 0」，用请求数而非 0 当哨兵
	if dst.MinLatency == 0 || (src.MinLatency > 0 && src.MinLatency < dst.MinLatency) {
		dst.MinLatency = src.MinLatency
	}
	dst.MaxLatency = math.Max(dst.MaxLatency, src.MaxLatency)
	// 百分位取各窗口较大值：合并窗口的真实百分位介于各窗口之间，宁可保守也不低报
	dst.P50Latency = math.Max(dst.P50Latency, src.P50Latency)
	dst.P90Latency = math.Max(dst.P90Latency, src.P90Latency)
	dst.P95Latency = math.Max(dst.P95Latency, src.P95Latency)
	dst.P99Latency = math.Max(dst.P99Latency, src.P99Latency)

	if dst.StatusCodes == nil {
		dst.StatusCodes = make(map[int]int64, len(src.StatusCodes))
	}
	for code, n := range src.StatusCodes {
		dst.StatusCodes[code] += n
	}
	if dst.ErrorTypes == nil {
		dst.ErrorTypes = make(map[string]int64, len(src.ErrorTypes))
	}
	for msg, n := range src.ErrorTypes {
		dst.ErrorTypes[msg] += n
	}

	// per-API 走同一套合并口径；切片转 map 再摊平的开销与接口数同阶，
	// 一次压测通常只有个位数到几十个接口，不值得为此再维护一份常驻 map
	if len(src.APIStats) > 0 || len(dst.APIStats) > 0 {
		acc := make(map[string]*statistics.APIStat, len(dst.APIStats)+len(src.APIStats))
		MergeAPIStats(acc, dst.APIStats)
		MergeAPIStats(acc, src.APIStats)
		dst.APIStats = FlattenAPIStats(acc)
	}

	if dst.TotalRequests > 0 {
		dst.SuccessRate = float64(dst.SuccessRequests) / float64(dst.TotalRequests) * 100
	}
}

// CloneSlaveStats 深拷贝，供累计表接管首个窗口时使用
//
// 必须复制：直接持有上报对象的话，后续窗口的累加会写穿到调用方的数据上
func CloneSlaveStats(src *SlaveStats) *SlaveStats {
	if src == nil {
		return nil
	}
	cp := *src
	cp.StatusCodes = copyIntInt64Map(src.StatusCodes)
	cp.ErrorTypes = copyStringInt64Map(src.ErrorTypes)
	if len(src.APIStats) > 0 {
		acc := make(map[string]*statistics.APIStat, len(src.APIStats))
		MergeAPIStats(acc, src.APIStats)
		cp.APIStats = FlattenAPIStats(acc)
	}
	return &cp
}

func copyIntInt64Map(m map[int]int64) map[int]int64 {
	if m == nil {
		return nil
	}
	out := make(map[int]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyStringInt64Map(m map[string]int64) map[string]int64 {
	if m == nil {
		return nil
	}
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
