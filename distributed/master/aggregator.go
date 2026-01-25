/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 00:06:27
 * @FilePath: \go-stress\distributed\master\aggregator.go
 * @Description: 数据聚合器
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"sort"
	"time"

	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// DataAggregator 数据聚合器
type DataAggregator struct {
	mu         *syncx.RWLock               // 使用 syncx.RWLock 替代 sync.RWMutex
	taskData   map[string]*TaskAggregation // task_id -> aggregation
	windowSize time.Duration
}

// TaskAggregation 任务聚合数据
type TaskAggregation struct {
	TaskID          string
	StartTime       time.Time
	LastUpdate      time.Time
	SlaveStats      map[string]*common.SlaveStats
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	Latencies       []float64
	StatusCodes     map[int]int64
	ErrorTypes      map[string]int64
}

// NewDataAggregator 创建数据聚合器
func NewDataAggregator() *DataAggregator {
	return &DataAggregator{
		mu:         syncx.NewRWLock(),
		taskData:   make(map[string]*TaskAggregation),
		windowSize: 5 * time.Second,
	}
}

// Add 添加统计数据
func (da *DataAggregator) Add(stats *common.SlaveStats) {
	syncx.WithLock(da.mu, func() {
		// 使用 mathx.IfEmpty 处理空 TaskID
		taskID := mathx.IfEmpty(stats.TaskID, "default")

		agg, exists := da.taskData[taskID]
		if !exists {
			agg = &TaskAggregation{
				TaskID:      taskID,
				StartTime:   time.Now(),
				SlaveStats:  make(map[string]*common.SlaveStats),
				StatusCodes: make(map[int]int64),
				ErrorTypes:  make(map[string]int64),
				Latencies:   make([]float64, 0),
			}
			da.taskData[taskID] = agg
		}

		// 更新 Slave 统计
		agg.SlaveStats[stats.SlaveID] = stats
		agg.LastUpdate = time.Now()

		// 聚合数据
		agg.TotalRequests += stats.TotalRequests
		agg.SuccessRequests += stats.SuccessRequests
		agg.FailedRequests += stats.FailedRequests

		// 聚合状态码
		for code, count := range stats.StatusCodes {
			agg.StatusCodes[code] += count
		}
	})
}

// GetAggregation 获取任务聚合数据
func (da *DataAggregator) GetAggregation(taskID string) (*common.AggregatedStats, bool) {
	return syncx.WithRLockReturnWithE(da.mu, func() (*common.AggregatedStats, bool) {
		agg, exists := da.taskData[taskID]
		if !exists {
			return nil, false
		}
		return da.buildAggregatedStats(agg), true
	})
}

// GetAllAggregations 获取所有任务的聚合数据
func (da *DataAggregator) GetAllAggregations() map[string]*common.AggregatedStats {
	return syncx.WithRLockReturnValue(da.mu, func() map[string]*common.AggregatedStats {
		result := make(map[string]*common.AggregatedStats, len(da.taskData))
		for taskID, agg := range da.taskData {
			result[taskID] = da.buildAggregatedStats(agg)
		}
		return result
	})
}

// buildAggregatedStats 构建聚合统计
func (da *DataAggregator) buildAggregatedStats(agg *TaskAggregation) *common.AggregatedStats {
	stats := &common.AggregatedStats{
		TaskID:          agg.TaskID,
		TimeRange:       common.TimeRange{Start: agg.StartTime, End: agg.LastUpdate},
		TotalAgents:     len(agg.SlaveStats),
		TotalRequests:   agg.TotalRequests,
		SuccessRequests: agg.SuccessRequests,
		FailedRequests:  agg.FailedRequests,
		BySlave:         agg.SlaveStats,
		StatusCodes:     agg.StatusCodes,
		ErrorTypes:      agg.ErrorTypes,
	}

	// 计算成功率
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100
	}

	// 收集所有延迟数据
	latencies := make([]float64, 0)
	totalQPS := 0.0

	for _, slaveStats := range agg.SlaveStats {
		latencies = append(latencies, slaveStats.AvgLatency)
		latencies = append(latencies, slaveStats.P95Latency)
		latencies = append(latencies, slaveStats.P99Latency)
		totalQPS += slaveStats.QPS
	}

	// 计算延迟统计
	if len(latencies) > 0 {
		sort.Float64s(latencies)
		stats.MinLatency = latencies[0]
		stats.MaxLatency = latencies[len(latencies)-1]
		stats.AvgLatency = mathx.Mean(latencies)
		stats.P50Latency = mathx.Percentile(latencies, 50)
		stats.P90Latency = mathx.Percentile(latencies, 90)
		stats.P95Latency = mathx.Percentile(latencies, 95)
		stats.P99Latency = mathx.Percentile(latencies, 99)
	}

	stats.TotalQPS = totalQPS

	return stats
}

// Clear 清空指定任务的数据
func (da *DataAggregator) Clear(taskID string) {
	syncx.WithLock(da.mu, func() {
		delete(da.taskData, taskID)
	})
}

// ClearAll 清空所有数据
func (da *DataAggregator) ClearAll() {
	syncx.WithLock(da.mu, func() {
		da.taskData = make(map[string]*TaskAggregation)
	})
}
