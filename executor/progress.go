/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:00:00
 * @FilePath: \go-stress\executor\progress.go
 * @Description: 进度跟踪器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/statistics"
	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
	"github.com/kamalyes/go-toolbox/pkg/units"
)

// ProgressTracker 进度跟踪器
type ProgressTracker struct {
	total         uint64
	completed     uint64
	startTime     time.Time
	collector     *statistics.Collector
	workerCount   uint64
	headerPrinted bool // 标记是否已打印表头
	logger        logger.ILogger
}

// NewProgressTracker 创建进度跟踪器
func NewProgressTracker(total uint64, log logger.ILogger) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		completed: 0,
		startTime: time.Now(),
		logger:    log,
	}
}

// NewProgressTrackerWithCollector 创建带统计收集器的进度跟踪器
func NewProgressTrackerWithCollector(total uint64, collector *statistics.Collector, workerCount uint64, log logger.ILogger) *ProgressTracker {
	return &ProgressTracker{
		total:       total,
		completed:   0,
		startTime:   time.Now(),
		collector:   collector,
		workerCount: workerCount,
		logger:      log,
	}
}

// Increment 增加完成数
func (pt *ProgressTracker) Increment() uint64 {
	return atomic.AddUint64(&pt.completed, 1)
}

// GetProgress 获取当前进度
func (pt *ProgressTracker) GetProgress() (completed, total uint64, percentage float64) {
	completed = atomic.LoadUint64(&pt.completed)
	total = pt.total
	percentage = float64(completed) / float64(total) * 100
	return
}

// Start 启动进度显示 - 使用 EventLoop
func (pt *ProgressTracker) Start(ctx context.Context) {
	pt.logger.Info("🚀 压测进行中...")
	// 使用 EventLoop 统一管理定时任务
	syncx.NewEventLoop(ctx).
		OnTicker(time.Second, func() {
			elapsed := time.Since(pt.startTime)
			if elapsed >= time.Second {
				pt.printProgress(elapsed)
			}
		}).
		Run()
}

// printProgress 打印进度行
func (pt *ProgressTracker) printProgress(elapsed time.Duration) {
	mathx.When(pt.collector == nil).
		Then(func() { pt.printSimpleProgress(elapsed) }).
		Do()

	if pt.collector == nil {
		return
	}

	// 获取统计数据
	completed := atomic.LoadUint64(&pt.completed)
	stats := pt.collector.GetSnapshot()

	// 计算实时指标
	seconds := elapsed.Seconds()
	qps := float64(completed) / seconds
	bytesPerSec := float64(stats.TotalSize) / seconds

	// 构建状态码统计字符串
	statusCodes := pt.collector.GetStatusCodes()
	statusStr := mathx.IfEmpty(buildStatusStr(statusCodes), "-")

	// 第一次显示时打印表头
	if !pt.headerPrinted {
		pt.logger.Info("")
		pt.logger.Info("┌──────┬────────┬────────┬────────┬──────┬──────────┬──────────┬──────────┬──────────┬─────────┬────────┐")
		pt.logger.Info("│ 耗时 │ 并发数 │ 成功数 │ 失败数 │ QPS  │ 最长耗时 │ 最短耗时 │ 平均耗时 │ 下载字节 │ 字节/秒 │ 状态码 │")
		pt.logger.Info("├──────┼────────┼────────┼────────┼──────┼──────────┼──────────┼──────────┼──────────┼─────────┼────────┤")
		pt.headerPrinted = true
	}

	// 格式化每个字段
	timeStr := fmt.Sprintf("%-4s", fmt.Sprintf("%ds", int(seconds)))
	concurrencyStr := fmt.Sprintf("%-6d", pt.workerCount)
	successStr := fmt.Sprintf("%-6d", stats.SuccessRequests)
	failedStr := fmt.Sprintf("%-6d", stats.FailedRequests)
	qpsStr := fmt.Sprintf("%4.2f", qps)
	maxLatencyStr := fmt.Sprintf("%-8s", formatLatency(stats.MaxLatency))
	minLatencyStr := fmt.Sprintf("%-8s", formatLatency(stats.MinLatency))
	avgLatencyStr := fmt.Sprintf("%-8s", formatLatency(stats.AvgLatency))
	bytesStr := fmt.Sprintf("%-8s", units.BytesSize(float64(stats.TotalSize)))
	bytesPerSecStr := fmt.Sprintf("%-7s", units.BytesSize(bytesPerSec))
	statusCodeStr := fmt.Sprintf("%-6s", statusStr)

	// 只打印数据行，不打印底部边框（底部边框在 Complete() 中打印）
	pt.logger.Info("│ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s │",
		timeStr, concurrencyStr, successStr, failedStr, qpsStr,
		maxLatencyStr, minLatencyStr, avgLatencyStr, bytesStr, bytesPerSecStr, statusCodeStr)
}

// printSimpleProgress 打印简单进度（无收集器模式）
func (pt *ProgressTracker) printSimpleProgress(elapsed time.Duration) {
	completed, total, percentage := pt.GetProgress()

	// 计算预估剩余时间
	var eta time.Duration
	if completed > 0 {
		avgTimePerReq := elapsed / time.Duration(completed)
		remaining := total - completed
		eta = avgTimePerReq * time.Duration(remaining)
	}

	// 计算QPS
	qps := float64(completed) / elapsed.Seconds()

	// 构建表格数据
	tableData := []map[string]interface{}{
		{
			"进度":   fmt.Sprintf("%d/%d (%.2f%%)", completed, total, percentage),
			"耗时":   elapsed.Round(time.Second).String(),
			"预计剩余": eta.Round(time.Second).String(),
			"QPS":  fmt.Sprintf("%.2f", qps),
			"并发数":  pt.workerCount,
		},
	}

	// 使用 ConsoleTable 显示数据
	pt.logger.ConsoleTable(tableData)
}

// Complete 完成并打印底部边框
func (pt *ProgressTracker) Complete() {
	// 如果显示过表头，打印表格底部
	if pt.headerPrinted {
		pt.logger.Info("└──────┴────────┴────────┴────────┴──────┴──────────┴──────────┴──────────┴──────────┴─────────┴────────┘")
	}
	pt.logger.Info("🎉 压测完成！")
}

// buildStatusStr 构建状态码统计字符串
func buildStatusStr(statusCodes map[int]uint64) string {
	if len(statusCodes) == 0 {
		return ""
	}

	var parts []string
	for code, count := range statusCodes {
		parts = append(parts, fmt.Sprintf("%d:%d", code, count))
	}
	return strings.Join(parts, " ")
}

// formatLatency 格式化延迟时间
func formatLatency(latency time.Duration) string {
	return mathx.WhenValue[string](latency > 0 && latency < time.Hour).
		ThenReturn(fmt.Sprintf("%.2fms", float64(latency.Microseconds())/1000)).
		ElseReturn("-").
		Get()
}
