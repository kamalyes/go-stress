/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-24 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 22:51:17
 * @FilePath: \go-stress\statistics\report_builder.go
 * @Description: 报告构建器 - 职责分离的核心
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"time"

	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// ReportBuilder 报告构建器 - 负责从 Collector 提取和计算数据
type ReportBuilder struct {
	collector *Collector
}

// NewReportBuilder 创建报告构建器
func NewReportBuilder(collector *Collector) *ReportBuilder {
	return &ReportBuilder{collector: collector}
}

// BuildReport 构建报告 - 使用读锁，最小化锁持有时间
func (rb *ReportBuilder) BuildReport(totalTime time.Duration, includeDetails bool) *Report {
	c := rb.collector

	// 第一步：原子读取计数器（无锁）
	totalReqs := c.totalRequests.Load()
	successReqs := c.successRequests.Load()
	failedReqs := c.failedRequests.Load()
	skippedReqs := c.skippedRequests.Load()

	// 第二步：使用 ToMap() 高级方法获取数据（一行代码搞定）
	errors := c.errors.ToMap()
	statusCodes := c.statusCodes.ToMap()

	// 第三步：读取时长数据（需要锁，但使用深拷贝快速复制）
	report := syncx.WithRLockReturnValue(c.mu, func() *Report {
		var durationsCopy []float64

		// 使用 syncx.DeepCopy 深拷贝切片
		if err := syncx.DeepCopy(&durationsCopy, &c.durations); err == nil {
			// 深拷贝成功
		} else {
			// 降级为手动复制
			durationsCopy = make([]float64, len(c.durations))
			copy(durationsCopy, c.durations)
		}

		// 在锁内快速构建报告
		return &Report{
			TotalRequests:   totalReqs,
			SuccessRequests: successReqs,
			FailedRequests:  failedReqs,
			SkippedRequests: skippedReqs,
			TotalTime:       totalTime,
			MinLatency:      c.minDuration,
			MaxLatency:      c.maxDuration,
			TotalSize:       c.totalSize,
			Errors:          errors,
			StatusCodes:     statusCodes,
			RequestDetails:  nil,       // 详情数据从SQLite按需加载
			RunMode:         c.runMode, // 传递运行模式
			Protocol:        c.protocol,
			Concurrency:     c.concurrency,
			TotalReqs:       c.totalReqs,
			logger:          c.logger,
		}
	})

	// 第三步：如果需要详情数据，从存储加载
	if includeDetails {
		report.RequestDetails = c.GetRequestDetails(0, 10000000, StatusFilterAll, "", "") // 最多取1000万条
	}

	// 第四步：在锁外计算统计数据
	rb.calculateStats(report, totalReqs, totalTime)

	return report
}

// calculateStats 计算统计数据（提取公共逻辑）
func (rb *ReportBuilder) calculateStats(report *Report, totalReqs uint64, totalTime time.Duration) {
	if totalReqs == 0 {
		return
	}

	// 计算基础统计
	report.SuccessRate = mathx.Percentage(report.SuccessRequests, totalReqs)
	report.AvgLatency = rb.collector.totalDuration / time.Duration(totalReqs)
	report.QPS = float64(totalReqs) / totalTime.Seconds()

	// 计算百分位
	if len(rb.collector.durations) > 0 {
		percentiles := mathx.Percentiles(rb.collector.durations, 50, 90, 95, 99)
		report.P50Latency = time.Duration(percentiles[50] * float64(time.Second))
		report.P90Latency = time.Duration(percentiles[90] * float64(time.Second))
		report.P95Latency = time.Duration(percentiles[95] * float64(time.Second))
		report.P99Latency = time.Duration(percentiles[99] * float64(time.Second))
	}
}

// BuildSummary 构建摘要（不包含明细，最快）
func (rb *ReportBuilder) BuildSummary(totalTime time.Duration) *Report {
	return rb.BuildReport(totalTime, false)
}

// BuildFullReport 构建完整报告（包含明细，默认10万条）
func (rb *ReportBuilder) BuildFullReport(totalTime time.Duration) *Report {
	return rb.BuildReport(totalTime, true)
}

// BuildFullReportWithLimit 构建完整报告（指定明细数量限制）
func (rb *ReportBuilder) BuildFullReportWithLimit(totalTime time.Duration, detailsLimit int) *Report {
	report := rb.BuildReport(totalTime, false)

	// -1 表示导出全部，查询一个很大的数字
	if detailsLimit < 0 {
		detailsLimit = 10000000 // 1000万，实际上就是全部
	} else if detailsLimit == 0 {
		// 0 表示不导出详情
		report.RequestDetails = []*RequestResult{}
		return report
	}

	// 单独加载指定数量的详情
	report.RequestDetails = rb.collector.GetRequestDetails(0, detailsLimit, StatusFilterAll, "", "")

	return report
}

// BuildRealtimeReport 构建实时报告 - 不包含明细，包含实时字段
func (rb *ReportBuilder) BuildRealtimeReport(startTime time.Time, endTime time.Time, isCompleted, isPaused, isStopped bool) *Report {
	// 计算实际耗时
	var elapsed time.Duration
	if isCompleted && !endTime.IsZero() {
		// 如果完成且有固定结束时间，使用固定结束时间（避免QPS持续变化）
		elapsed = endTime.Sub(startTime)
	} else {
		// 否则使用当前时间（实时计算）
		elapsed = time.Since(startTime)
	}

	// 构建基础报告（不含明细）
	report := rb.BuildReport(elapsed, false)

	// 添加实时专用字段
	report.Timestamp = time.Now().Unix()
	report.Elapsed = int64(elapsed.Seconds())
	report.IsCompleted = isCompleted
	report.IsPaused = isPaused
	report.IsStopped = isStopped

	// 获取最近20个响应时间用于实时图表
	c := rb.collector
	report.RecentDurations = syncx.WithRLockReturnValue(c.mu, func() []int64 {
		durationsLen := len(c.durations)
		if durationsLen == 0 {
			return nil
		}

		start := 0
		if durationsLen > 20 {
			start = durationsLen - 20
		}

		recent := make([]int64, 0, durationsLen-start)
		for i := start; i < durationsLen; i++ {
			// durations 是 float64 (秒)，转换为毫秒
			recent = append(recent, int64(c.durations[i]*1000))
		}
		return recent
	})

	return report
}
