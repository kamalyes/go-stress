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
		sc.logger.Warn("Stats buffer full, dropping data", "slave_id", stats.SlaveID)
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
