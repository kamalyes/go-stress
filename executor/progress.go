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
	"sync/atomic"
	"time"

	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-stress/statistics"
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
}

// NewProgressTracker 创建进度跟踪器
func NewProgressTracker(total uint64) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		completed: 0,
		startTime: time.Now(),
	}
}

// NewProgressTrackerWithCollector 创建带统计收集器的进度跟踪器
func NewProgressTrackerWithCollector(total uint64, collector *statistics.Collector, workerCount uint64) *ProgressTracker {
	return &ProgressTracker{
		total:       total,
		completed:   0,
		startTime:   time.Now(),
		collector:   collector,
		workerCount: workerCount,
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

// Start 启动进度显示
func (pt *ProgressTracker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	logger.Default.Info("")
	logger.Default.Info("🚀 压测进行中...")
	logger.Default.Info("")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(pt.startTime)
			if elapsed < time.Second {
				continue
			}

			pt.printProgress(elapsed)
		}
	}
}

// printProgress 打印进度行
func (pt *ProgressTracker) printProgress(elapsed time.Duration) {
	if pt.collector == nil {
		pt.printSimpleProgress(elapsed)
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
	statusStr := ""
	for code, count := range statusCodes {
		if statusStr != "" {
			statusStr += " "
		}
		statusStr += fmt.Sprintf("%d:%d", code, count)
	}
	if statusStr == "" {
		statusStr = "-"
	}

	// 打印表头（仅第一次）
	if !pt.headerPrinted {
		pt.printTableHeader()
		pt.headerPrinted = true
	}

	// 打印数据行
	minDur := "-"
	maxDur := "-"
	avgDur := "-"
	if stats.MinLatency < time.Hour {
		minDur = fmt.Sprintf("%.2fms", float64(stats.MinLatency.Microseconds())/1000)
	}
	if stats.MaxLatency > 0 {
		maxDur = fmt.Sprintf("%.2fms", float64(stats.MaxLatency.Microseconds())/1000)
	}
	if stats.AvgLatency > 0 {
		avgDur = fmt.Sprintf("%.2fms", float64(stats.AvgLatency.Microseconds())/1000)
	}

	logger.Default.Infof("│ %4ds │ %6d │ %6d │ %6d │ %7.2f │ %8s │ %8s │ %8s │ %9s │ %9s │ %-11s │",
		int(seconds),
		pt.workerCount,
		stats.SuccessRequests,
		stats.FailedRequests,
		qps,
		maxDur,
		minDur,
		avgDur,
		units.BytesSize(float64(stats.TotalSize)),
		units.BytesSize(bytesPerSec),
		statusStr,
	)
}

// printTableHeader 打印表格表头
func (pt *ProgressTracker) printTableHeader() {
	logger.Default.Info("┌──────┬────────┬────────┬────────┬─────────┬──────────┬──────────┬──────────┬───────────┬───────────┬─────────────┐")
	logger.Default.Info("│ 耗时 │ 并发数 │ 成功数 │ 失败数 │   QPS   │ 最长耗时 │ 最短耗时 │ 平均耗时 │  下载字节 │  字节/秒  │   状态码    │")
	logger.Default.Info("├──────┼────────┼────────┼────────┼─────────┼──────────┼──────────┼──────────┼───────────┼───────────┼─────────────┤")
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

	// 打印表头（仅第一次）
	if !pt.headerPrinted {
		logger.Default.Info("┌──────────────────────┬──────────────┬──────────────┬─────────┬────────┐")
		logger.Default.Info("│       进度           │     耗时     │   预计剩余   │   QPS   │ 并发数 │")
		logger.Default.Info("├──────────────────────┼──────────────┼──────────────┼─────────┼────────┤")
		pt.headerPrinted = true
	}

	// 打印数据行
	logger.Default.Infof("│ %6d/%6d (%5.2f%%) │ %12s │ %12s │ %7.2f │ %6d │",
		completed, total, percentage,
		elapsed.Round(time.Second).String(),
		eta.Round(time.Second).String(),
		qps,
		pt.workerCount,
	)
}

// Complete 完成并打印底部边框
func (pt *ProgressTracker) Complete() {
	if !pt.headerPrinted {
		return
	}

	// 根据是否有收集器打印不同的底部边框
	if pt.collector != nil {
		// 完整统计模式
		logger.Default.Info("└──────┴────────┴────────┴────────┴─────────┴──────────┴──────────┴──────────┴───────────┴───────────┴─────────────┘")
	} else {
		// 简单进度模式
		logger.Default.Info("└──────────────────────┴──────────────┴──────────────┴─────────┴────────┘")
	}
	logger.Default.Info("")
}
