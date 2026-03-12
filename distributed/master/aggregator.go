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
	"math"
	"time"

	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-stress/statistics"
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

	// APIStats 按 API 名跨 Slave、跨上报窗口的累计
	//
	// 用 map 而非切片：每个 Slave 每秒报一个窗口，同一个 API 会反复出现，
	// 必须按名字归并，合并口径见 common.MergeAPIStats
	APIStats map[string]*statistics.APIStat
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
				APIStats:    make(map[string]*statistics.APIStat),
			}
			da.taskData[taskID] = agg
		}

		// 按 Slave 累加而非覆盖
		//
		// 上报是「每秒一个窗口」，覆盖的话 BySlave 里存的永远只是最近一秒；
		// 而分布式面板跳转实时报告时带的正是 ?slave_id=，页面于是把最近一秒
		// 的几十次请求当成总数显示
		if cur, ok := agg.SlaveStats[stats.SlaveID]; ok {
			common.MergeSlaveStats(cur, stats)
		} else {
			agg.SlaveStats[stats.SlaveID] = common.CloneSlaveStats(stats)
		}
		agg.LastUpdate = time.Now()

		// 聚合数据
		agg.TotalRequests += stats.TotalRequests
		agg.SuccessRequests += stats.SuccessRequests
		agg.FailedRequests += stats.FailedRequests

		// 聚合状态码
		for code, count := range stats.StatusCodes {
			agg.StatusCodes[code] += count
		}

		// 聚合错误类型：此前漏了这一步，于是 Master 的错误分布恒为空
		for errType, count := range stats.ErrorTypes {
			agg.ErrorTypes[errType] += count
		}

		// 聚合 per-API
		if agg.APIStats == nil {
			agg.APIStats = make(map[string]*statistics.APIStat)
		}
		common.MergeAPIStats(agg.APIStats, stats.APIStats)
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
		APIStats:        common.FlattenAPIStats(agg.APIStats),
	}

	// 计算成功率
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100
	}

	// 跨 Slave 汇总延迟
	//
	// 原先的做法是把每个 Slave 的 avg/p95/p99 三个数倒进一个切片，再对这个
	// 混合池求百分位 —— MinLatency 于是成了「各 Slave 的 avg/p95/p99 里最小的
	// 那一个」，跟最小延迟毫无关系。改为按各自的语义归并：极值取极值，
	// 均值按请求数加权，百分位取各 Slave 的较大者（宁可保守也不低报）
	var weighted float64
	for _, slaveStats := range agg.SlaveStats {
		if stats.MinLatency == 0 || (slaveStats.MinLatency > 0 && slaveStats.MinLatency < stats.MinLatency) {
			stats.MinLatency = slaveStats.MinLatency
		}
		stats.MaxLatency = math.Max(stats.MaxLatency, slaveStats.MaxLatency)
		stats.P50Latency = math.Max(stats.P50Latency, slaveStats.P50Latency)
		stats.P90Latency = math.Max(stats.P90Latency, slaveStats.P90Latency)
		stats.P95Latency = math.Max(stats.P95Latency, slaveStats.P95Latency)
		stats.P99Latency = math.Max(stats.P99Latency, slaveStats.P99Latency)
		weighted += slaveStats.AvgLatency * float64(slaveStats.TotalRequests)
	}
	if stats.TotalRequests > 0 {
		stats.AvgLatency = weighted / float64(stats.TotalRequests)
	}

	// 速率按「累计请求数 / 实际耗时」折算
	//
	// 累加各窗口的瞬时 QPS 会把 90 秒的速率叠成 90 倍，只取最后一个窗口
	// 又丢掉跨 Slave 的并行度；按总量除以耗时得到整段压测的平均速率，
	// 与单机报告里 QPS 的口径一致
	elapsed := agg.LastUpdate.Sub(agg.StartTime).Seconds()
	if elapsed > 0 {
		stats.TotalQPS = float64(stats.TotalRequests) / elapsed
		for _, slaveStats := range agg.SlaveStats {
			slaveStats.QPS = float64(slaveStats.TotalRequests) / elapsed
			common.RecomputeAPIQPS(slaveStats.APIStats, elapsed)
		}
		common.RecomputeAPIQPS(stats.APIStats, elapsed)
	}

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
