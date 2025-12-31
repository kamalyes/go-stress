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
	"os"
	"time"

	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-toolbox/pkg/units"
)

// Report 统计报告
type Report struct {
	// 基础统计
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
	SuccessRate     float64

	// 时间统计
	TotalTime   time.Duration
	MinDuration time.Duration
	MaxDuration time.Duration
	AvgDuration time.Duration

	// 百分位统计
	P50 time.Duration
	P90 time.Duration
	P95 time.Duration
	P99 time.Duration

	// 性能指标
	QPS       float64
	TotalSize float64

	// 错误统计
	Errors map[string]uint64

	// 状态码统计
	StatusCodes map[int]uint64

	// 请求明细
	RequestDetails []RequestDetail
}

// Print 打印报告（使用单个多列表格）
func (r *Report) Print() {
	logger.Default.Info("")
	logger.Default.Info("📊 压测统计报告")
	logger.Default.Info("")

	// 构建单个统一表格
	reportData := []map[string]interface{}{
		{
			"分类":  "📈 基础统计",
			"指标":  "总请求数",
			"值":   fmt.Sprintf("%d", r.TotalRequests),
			"分类2": "⏱️  响应时间",
			"指标2": "最小耗时",
			"值2":  r.MinDuration.String(),
		},
		{
			"分类":  "📈 基础统计",
			"指标":  "成功请求",
			"值":   fmt.Sprintf("%d", r.SuccessRequests),
			"分类2": "⏱️  响应时间",
			"指标2": "最大耗时",
			"值2":  r.MaxDuration.String(),
		},
		{
			"分类":  "📈 基础统计",
			"指标":  "失败请求",
			"值":   fmt.Sprintf("%d", r.FailedRequests),
			"分类2": "⏱️  响应时间",
			"指标2": "平均耗时",
			"值2":  r.AvgDuration.String(),
		},
		{
			"分类":  "📈 基础统计",
			"指标":  "成功率",
			"值":   fmt.Sprintf("%.2f%%", r.SuccessRate),
			"分类2": "⏱️  响应时间",
			"指标2": "P50",
			"值2":  r.P50.String(),
		},
		{
			"分类":  "⚡ 性能指标",
			"指标":  "总耗时",
			"值":   r.TotalTime.String(),
			"分类2": "⏱️  响应时间",
			"指标2": "P90",
			"值2":  r.P90.String(),
		},
		{
			"分类":  "⚡ 性能指标",
			"指标":  "QPS",
			"值":   fmt.Sprintf("%.2f", r.QPS),
			"分类2": "⏱️  响应时间",
			"指标2": "P95",
			"值2":  r.P95.String(),
		},
		{
			"分类":  "⚡ 性能指标",
			"指标":  "传输数据",
			"值":   units.BytesSize(float64(r.TotalSize)),
			"分类2": "⏱️  响应时间",
			"指标2": "P99",
			"值2":  r.P99.String(),
		},
	}

	logger.Default.ConsoleTable(reportData)

	// 错误统计（如果有）
	if len(r.Errors) > 0 {
		logger.Default.Info("")
		logger.Default.Info("❌ 错误统计")
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
		logger.Default.ConsoleTable(errorStats)
	}

	logger.Default.Info("")
}

// ToJSON 导出为JSON
func (r *Report) ToJSON() string {
	data := map[string]interface{}{
		"total_requests":   r.TotalRequests,
		"success_requests": r.SuccessRequests,
		"failed_requests":  r.FailedRequests,
		"success_rate":     r.SuccessRate,
		"qps":              r.QPS,
		"total_size":       r.TotalSize,
		"total_time_ms":    r.TotalTime.Milliseconds(),
		"min_duration_ms":  r.MinDuration.Milliseconds(),
		"max_duration_ms":  r.MaxDuration.Milliseconds(),
		"avg_duration_ms":  r.AvgDuration.Milliseconds(),
		"p50_ms":           r.P50.Milliseconds(),
		"p90_ms":           r.P90.Milliseconds(),
		"p95_ms":           r.P95.Milliseconds(),
		"p99_ms":           r.P99.Milliseconds(),
		"errors":           r.Errors,
		"status_codes":     r.StatusCodes,
		"request_details":  r.RequestDetails,
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

// Summary 返回简短摘要
func (r *Report) Summary() string {
	return fmt.Sprintf(
		"请求: %d | 成功率: %.2f%% | QPS: %.2f | 平均耗时: %s",
		r.TotalRequests,
		r.SuccessRate,
		r.QPS,
		r.AvgDuration,
	)
}

// SaveToFile 保存报告到文件
func (r *Report) SaveToFile(filename string) error {
	content := r.ToJSON()
	return os.WriteFile(filename, []byte(content), 0644)
}
