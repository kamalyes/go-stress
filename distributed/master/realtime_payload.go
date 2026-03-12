/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 17:33:50
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 18:02:19
 * @FilePath: \go-stress\distributed\master\realtime_payload.go
 * @Description: Master 实时报告接口的响应契约
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-stress/statistics"
)

// RealtimePayload /api/realtime/stats 的响应体
//
// Master 的 /realtime 页面与单机报告、Slave 实时报告是同一份 report.js，
// 页面读哪些字段由那份 JS 决定。此前这里是两段手写的 map[string]any
// （全局一段、按 Slave 一段），键名靠口头约定与 JS 对齐 —— 于是
// report.js 新增 api_stats 后，分布式这侧的 API 汇总整块静默消失，
// 页面上连个报错都没有，因为 renderAPIStats 拿到 undefined 就把整段隐藏了
//
// 改成一个具名结构：JSON 标签就是契约本身，字段只定义一处，
// 全局与按 Slave 两条路径共用同一个类型，不会再出现只改一边的情况
type RealtimePayload struct {
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	FailedRequests  int64   `json:"failed_requests"`
	SuccessRate     float64 `json:"success_rate"`
	QPS             float64 `json:"qps"`

	AvgLatency float64 `json:"avg_latency"`
	MinLatency float64 `json:"min_latency"`
	MaxLatency float64 `json:"max_latency"`
	P50Latency float64 `json:"p50_latency"`
	P90Latency float64 `json:"p90_latency"`
	P95Latency float64 `json:"p95_latency"`
	P99Latency float64 `json:"p99_latency"`

	StatusCodes map[int]int64    `json:"status_codes"`
	Errors      map[string]int64 `json:"errors"`

	// APIStats 复用 statistics.APIStat，与单机报告同一个结构
	APIStats []statistics.APIStat `json:"api_stats,omitempty"`

	TotalAgents int    `json:"total_agents"`
	SlaveID     string `json:"slave_id,omitempty"`
	TaskID      string `json:"task_id,omitempty"`

	// BySlave 仅聚合视图返回，单 Slave 视图下没有意义
	BySlave map[string]*common.SlaveStats `json:"by_slave,omitempty"`
}

// NewRealtimePayloadFromSlave 单个 Slave 的视图
func NewRealtimePayloadFromSlave(s *common.SlaveStats, slaveID, taskID string) *RealtimePayload {
	return &RealtimePayload{
		TotalRequests:   s.TotalRequests,
		SuccessRequests: s.SuccessRequests,
		FailedRequests:  s.FailedRequests,
		SuccessRate:     s.SuccessRate,
		QPS:             s.QPS,
		AvgLatency:      s.AvgLatency,
		MinLatency:      s.MinLatency,
		MaxLatency:      s.MaxLatency,
		P50Latency:      s.P50Latency,
		P90Latency:      s.P90Latency,
		P95Latency:      s.P95Latency,
		P99Latency:      s.P99Latency,
		StatusCodes:     s.StatusCodes,
		Errors:          s.ErrorTypes,
		APIStats:        s.APIStats,
		TotalAgents:     1,
		SlaveID:         slaveID,
		TaskID:          taskID,
	}
}

// NewRealtimePayloadFromAggregated 全部 Slave 的聚合视图
//
// QPS 取 TotalQPS：聚合视图里这一项是各 Slave 之和，而 report.js
// 读的键统一叫 qps，差异在这里一次性抹平，不留给调用方记
func NewRealtimePayloadFromAggregated(a *common.AggregatedStats, taskID string) *RealtimePayload {
	return &RealtimePayload{
		TotalRequests:   a.TotalRequests,
		SuccessRequests: a.SuccessRequests,
		FailedRequests:  a.FailedRequests,
		SuccessRate:     a.SuccessRate,
		QPS:             a.TotalQPS,
		AvgLatency:      a.AvgLatency,
		MinLatency:      a.MinLatency,
		MaxLatency:      a.MaxLatency,
		P50Latency:      a.P50Latency,
		P90Latency:      a.P90Latency,
		P95Latency:      a.P95Latency,
		P99Latency:      a.P99Latency,
		StatusCodes:     a.StatusCodes,
		Errors:          a.ErrorTypes,
		APIStats:        a.APIStats,
		TotalAgents:     a.TotalAgents,
		BySlave:         a.BySlave,
		TaskID:          taskID,
	}
}
