/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 10:39:00
 * @FilePath: \go-stress\statistics\report.go
 * @Description: 统计报告
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-toolbox/pkg/units"
)

// Report 统计报告 - 统一的数据结构，同时支持静态和实时报告
type Report struct {
	// 基础统计
	TotalRequests   uint64  `json:"total_requests"`
	SuccessRequests uint64  `json:"success_requests"`
	FailedRequests  uint64  `json:"failed_requests"`
	SkippedRequests uint64  `json:"skipped_requests"` // 跳过请求数
	SuccessRate     float64 `json:"success_rate"`     // 百分比 0-100

	// 时间统计
	TotalTime  time.Duration `json:"total_time"`
	MinLatency time.Duration `json:"min_latency"`
	MaxLatency time.Duration `json:"max_latency"`
	AvgLatency time.Duration `json:"avg_latency"`

	// 百分位统计
	P50Latency time.Duration `json:"p50_latency"`
	P90Latency time.Duration `json:"p90_latency"`
	P95Latency time.Duration `json:"p95_latency"`
	P99Latency time.Duration `json:"p99_latency"`

	// 性能指标
	QPS       float64 `json:"qps"`
	TotalSize float64 `json:"total_size"` // 字节数

	// 错误统计
	Errors map[string]uint64 `json:"errors,omitempty"`

	// 状态码统计
	StatusCodes map[int]uint64 `json:"status_codes,omitempty"`

	// 请求明细（静态报告用，实时报告不加载）
	RequestDetails []*RequestResult `json:"request_details,omitempty"`

	// === 实时报告专用字段 ===
	Timestamp       int64   `json:"timestamp,omitempty"`        // Unix时间戳
	Elapsed         int64   `json:"elapsed_seconds"`            // 已耗时（秒）- 移除omitempty确保始终输出
	IsCompleted     bool    `json:"is_completed,omitempty"`     // 是否完成
	IsPaused        bool    `json:"is_paused,omitempty"`        // 是否暂停
	IsStopped       bool    `json:"is_stopped,omitempty"`       // 是否停止
	RecentDurations []int64 `json:"recent_durations,omitempty"` // 最近响应时间（毫秒）用于实时图表

	// 运行模式标识
	RunMode RunMode `json:"run_mode,omitempty"`

	// 配置信息（用于报告显示）
	Protocol    string `json:"protocol,omitempty"`    // 协议类型: http/grpc/websocket
	Concurrency uint64 `json:"concurrency,omitempty"` // 并发数
	TotalReqs   uint64 `json:"total_reqs,omitempty"`  // 计划请求数
	logger      logger.ILogger
}

// Print 打印报告（使用单个多列表格）
func (r *Report) Print() {
	r.logger.Info("📊 压测统计报告")

	// 构建单个统一表格
	reportData := []map[string]interface{}{
		{
			"分类":  "📈 基础统计",
			"指标":  "总请求数",
			"值":   fmt.Sprintf("%d", r.TotalRequests),
			"分类2": "⏱️  响应时间",
			"指标2": "最小耗时",
			"值2":  r.MinLatency.String(),
		},
		{
			"分类":  "📈 基础统计",
			"指标":  "成功请求",
			"值":   fmt.Sprintf("%d", r.SuccessRequests),
			"分类2": "⏱️  响应时间",
			"指标2": "最大耗时",
			"值2":  r.MaxLatency.String(),
		},
		{
			"分类":  "📈 基础统计",
			"指标":  "失败请求",
			"值":   fmt.Sprintf("%d", r.FailedRequests),
			"分类2": "⏱️  响应时间",
			"指标2": "平均耗时",
			"值2":  r.AvgLatency.String(),
		},
		{
			"分类":  "📈 基础统计",
			"指标":  "成功率",
			"值":   fmt.Sprintf("%.2f%%", r.SuccessRate),
			"分类2": "⏱️  响应时间",
			"指标2": "P50",
			"值2":  r.P50Latency.String(),
		},
		{
			"分类":  "⚡ 性能指标",
			"指标":  "总耗时",
			"值":   r.TotalTime.String(),
			"分类2": "⏱️  响应时间",
			"指标2": "P90",
			"值2":  r.P90Latency.String(),
		},
		{
			"分类":  "⚡ 性能指标",
			"指标":  "QPS",
			"值":   fmt.Sprintf("%.2f", r.QPS),
			"分类2": "⏱️  响应时间",
			"指标2": "P95",
			"值2":  r.P95Latency.String(),
		},
		{
			"分类":  "⚡ 性能指标",
			"指标":  "传输数据",
			"值":   units.BytesSize(float64(r.TotalSize)),
			"分类2": "⏱️  响应时间",
			"指标2": "P99",
			"值2":  r.P99Latency.String(),
		},
	}

	r.logger.ConsoleTable(reportData)

	// 错误统计（如果有）
	if len(r.Errors) > 0 {
		errorStats := make([]map[string]interface{}, 0, len(r.Errors))
		for errMsg, count := range r.Errors {
			// 截断过长的错误信息
			if len(errMsg) > 80 {
				errMsg = errMsg[:77] + "..."
			}
			errorStats = append(errorStats, map[string]interface{}{
				"错误信息": errMsg,
				"次数":   count,
			})
		}
		r.logger.ConsoleTable(errorStats)
	}
}

// Summary 返回简短摘要
func (r *Report) Summary() string {
	return fmt.Sprintf(
		"请求: %d | 成功率: %.2f%% | QPS: %.2f | 平均耗时: %s",
		r.TotalRequests,
		r.SuccessRate,
		r.QPS,
		r.AvgLatency,
	)
}

// MarshalJSON 自定义JSON序列化，将time.Duration转换为毫秒
func (r *Report) MarshalJSON() ([]byte, error) {
	type Alias Report
	return json.Marshal(&struct {
		*Alias
		// 添加毫秒格式的字段供前端使用
		AvgLatency  float64 `json:"avg_latency"`
		MinLatency  float64 `json:"min_latency"`
		MaxLatency  float64 `json:"max_latency"`
		P50Latency  float64 `json:"p50_latency"`
		P90Latency  float64 `json:"p90_latency"`
		P95Latency  float64 `json:"p95_latency"`
		P99Latency  float64 `json:"p99_latency"`
		TotalTimeMs float64 `json:"total_time_ms"`
	}{
		Alias:       (*Alias)(r),
		AvgLatency:  float64(r.AvgLatency.Microseconds()) / 1000.0,
		MinLatency:  float64(r.MinLatency.Microseconds()) / 1000.0,
		MaxLatency:  float64(r.MaxLatency.Microseconds()) / 1000.0,
		P50Latency:  float64(r.P50Latency.Microseconds()) / 1000.0,
		P90Latency:  float64(r.P90Latency.Microseconds()) / 1000.0,
		P95Latency:  float64(r.P95Latency.Microseconds()) / 1000.0,
		P99Latency:  float64(r.P99Latency.Microseconds()) / 1000.0,
		TotalTimeMs: float64(r.TotalTime.Microseconds()) / 1000.0,
	})
}
