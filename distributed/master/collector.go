/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\master\collector.go
 * @Description: 统计数据收集器
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"context"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// StatsCollector 统计数据收集器
type StatsCollector struct {
	mu         *syncx.RWLock // 使用 syncx.RWLock 替代 sync.RWMutex
	buffer     chan *common.SlaveStats
	cache      map[string]*common.SlaveStats            // slave_id -> stats
	taskStats  map[string]map[string]*common.SlaveStats // task_id -> slave_id -> stats
	bufferSize int
	aggregator *DataAggregator
	logger     logger.ILogger
}

// NewStatsCollector 创建统计收集器
func NewStatsCollector(bufferSize int, log logger.ILogger) *StatsCollector {
	return &StatsCollector{
		mu:         syncx.NewRWLock(),
		buffer:     make(chan *common.SlaveStats, bufferSize),
		cache:      make(map[string]*common.SlaveStats),
		taskStats:  make(map[string]map[string]*common.SlaveStats),
		bufferSize: bufferSize,
		aggregator: NewDataAggregator(),
		logger:     log,
	}
}

// Collect 收集统计数据
func (sc *StatsCollector) Collect(stats *common.SlaveStats) error {
	select {
	case sc.buffer <- stats:
		return nil
	default:
		sc.logger.WarnKV("Stats buffer full, dropping data", "slave_id", stats.SlaveID)
		return nil
	}
}

// Start 启动收集器
func (sc *StatsCollector) Start(ctx context.Context) {
	sc.logger.Info("Stats collector started")

	for {
		select {
		case <-ctx.Done():
			sc.logger.Info("Stats collector stopped")
			return
		case stats := <-sc.buffer:
			sc.processStats(stats)
		}
	}
}

// processStats 处理统计数据
func (sc *StatsCollector) processStats(stats *common.SlaveStats) {
	syncx.WithLock(sc.mu, func() {
		// 更新缓存
		sc.cache[stats.SlaveID] = stats

		// 传递给聚合器
		if sc.aggregator != nil {
			sc.aggregator.Add(stats)
		}
	})
}

// GetSlaveStats 获取指定 Slave 的统计
func (sc *StatsCollector) GetSlaveStats(slaveID string) (*common.SlaveStats, bool) {
	return syncx.WithRLockReturnWithE(sc.mu, func() (*common.SlaveStats, bool) {
		stats, exists := sc.cache[slaveID]
		return stats, exists
	})
}

// GetAllStats 获取所有 Slave 的统计数据
func (sc *StatsCollector) GetAllStats() map[string]*common.SlaveStats {
	return syncx.WithRLockReturnValue(sc.mu, func() map[string]*common.SlaveStats {
		result := make(map[string]*common.SlaveStats, len(sc.cache))
		for k, v := range sc.cache {
			result[k] = v
		}
		return result
	})
}

// GetTaskStats 获取指定任务的统计数据
func (sc *StatsCollector) GetTaskStats(taskID string) map[string]*common.SlaveStats {
	return syncx.WithRLockReturnValue(sc.mu, func() map[string]*common.SlaveStats {
		stats, exists := sc.taskStats[taskID]
		if !exists {
			return make(map[string]*common.SlaveStats)
		}

		result := make(map[string]*common.SlaveStats, len(stats))
		for k, v := range stats {
			result[k] = v
		}
		return result
	})
}

// Clear 清空缓存
func (sc *StatsCollector) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.cache = make(map[string]*common.SlaveStats)
	sc.taskStats = make(map[string]map[string]*common.SlaveStats)
}

// GetAggregator 获取聚合器
func (sc *StatsCollector) GetAggregator() *DataAggregator {
	return sc.aggregator
}

// GetAggregatedStats 获取聚合统计数据（所有任务）
func (sc *StatsCollector) GetAggregatedStats() *common.AggregatedStats {
	if sc.aggregator == nil {
		return &common.AggregatedStats{}
	}

	// 获取所有任务的聚合数据
	allAggs := sc.aggregator.GetAllAggregations()

	// 如果没有数据,返回空统计
	if len(allAggs) == 0 {
		return &common.AggregatedStats{}
	}

	// 如果只有一个任务,直接返回
	if len(allAggs) == 1 {
		for _, agg := range allAggs {
			return agg
		}
	}

	// 多个任务时,合并所有统计
	merged := &common.AggregatedStats{
		StatusCodes: make(map[int]int64),
		ErrorTypes:  make(map[string]int64),
		BySlave:     make(map[string]*common.SlaveStats),
	}

	for _, agg := range allAggs {
		merged.TotalRequests += agg.TotalRequests
		merged.SuccessRequests += agg.SuccessRequests
		merged.FailedRequests += agg.FailedRequests
		merged.TotalQPS += agg.TotalQPS
		merged.TotalAgents += agg.TotalAgents

		// 合并状态码
		for code, count := range agg.StatusCodes {
			merged.StatusCodes[code] += count
		}

		// 合并错误类型
		for errType, count := range agg.ErrorTypes {
			merged.ErrorTypes[errType] += count
		}

		// 合并 Slave 数据
		for slaveID, stats := range agg.BySlave {
			merged.BySlave[slaveID] = stats
		}
	}

	// 重新计算成功率
	if merged.TotalRequests > 0 {
		merged.SuccessRate = float64(merged.SuccessRequests) / float64(merged.TotalRequests) * 100
	}

	return merged
}
