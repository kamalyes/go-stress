/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\slave\stats_buffer.go
 * @Description: 统计数据缓冲区
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package slave

import (
	"context"
	"fmt"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/distributed/common"
	pb "github.com/kamalyes/go-stress/distributed/proto"
	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// StatsBuffer 统计数据缓冲区 - 使用 syncx 重构
type StatsBuffer struct {
	slaveID      string
	taskID       string
	buffer       []*types.RequestResult
	bufferMu     syncx.Locker // 使用 syncx.Lock 替代 sync.Mutex
	maxSize      int
	masterClient pb.MasterServiceClient
	reportStream pb.MasterService_ReportStatsClient
	streamMu     syncx.Locker // 使用 syncx.Lock
	logger       logger.ILogger
	taskManager  *syncx.PeriodicTaskManager // 使用 syncx.PeriodicTask
}

// NewStatsBuffer 创建统计缓冲区
func NewStatsBuffer(slaveID string, maxSize int, log logger.ILogger) *StatsBuffer {
	return &StatsBuffer{
		slaveID:     slaveID,
		buffer:      make([]*types.RequestResult, 0, maxSize),
		bufferMu:    syncx.NewLock(),
		maxSize:     maxSize,
		streamMu:    syncx.NewLock(),
		logger:      log,
		taskManager: syncx.NewPeriodicTaskManager(),
	}
}

// Add 添加统计记录
func (sb *StatsBuffer) Add(result *types.RequestResult) {
	sb.bufferMu.Lock()
	defer sb.bufferMu.Unlock()

	sb.buffer = append(sb.buffer, result)

	// 缓冲区满时立即刷新
	if len(sb.buffer) >= sb.maxSize {
		// 使用 syncx.Go 异步刷新
		syncx.Go().Exec(func() {
			sb.Flush()
		})
	}
}

// Start 启动缓冲区 - 使用 syncx.PeriodicTask
func (sb *StatsBuffer) Start(ctx context.Context) {
	task := syncx.NewPeriodicTask("stats-flush", 1*time.Second, func(taskCtx context.Context) error {
		return sb.Flush()
	}).
		SetOnError(func(name string, err error) {
			sb.logger.WarnKV("Stats flush error", "error", err)
		}).
		SetOnStart(func(name string) {
			sb.logger.InfoKV("Stats buffer started", "slave_id", sb.slaveID)
		}).
		SetOnStop(func(name string) {
			sb.Flush() // 最后刷新一次
			sb.logger.InfoKV("Stats buffer stopped", "slave_id", sb.slaveID)
		})

	sb.taskManager.AddTask(task)
	sb.taskManager.Start()
}

// Flush 刷新缓冲区
func (sb *StatsBuffer) Flush() error {
	sb.bufferMu.Lock()
	if len(sb.buffer) == 0 {
		sb.bufferMu.Unlock()
		return nil
	}

	// 复制并清空缓冲区
	toSend := make([]*types.RequestResult, len(sb.buffer))
	copy(toSend, sb.buffer)
	sb.buffer = sb.buffer[:0]
	sb.bufferMu.Unlock()

	// 聚合数据
	stats := sb.aggregate(toSend)

	// 发送到 Master
	if sb.masterClient != nil && sb.taskID != "" {
		if err := sb.sendToMaster(stats); err != nil {
			sb.logger.WarnKV("Failed to send stats to master", "error", err)
			return err
		}
	}

	sb.logger.DebugKV("Stats flushed",
		"slave_id", sb.slaveID,
		"records", len(toSend),
		"total_requests", stats.TotalRequests)

	return nil
}

// sendToMaster 发送统计数据到 Master
func (sb *StatsBuffer) sendToMaster(stats *common.SlaveStats) error {
	sb.streamMu.Lock()
	defer sb.streamMu.Unlock()

	// 检查 masterClient 是否已设置
	if sb.masterClient == nil {
		return fmt.Errorf("master client not set")
	}

	// 如果流不存在，创建新的流
	if sb.reportStream == nil {
		ctx := context.Background()
		stream, err := sb.masterClient.ReportStats(ctx)
		if err != nil {
			return err
		}
		sb.reportStream = stream
	}

	// 转换状态码格式
	statusCodes := make(map[string]int64)
	for code, count := range stats.StatusCodes {
		statusCodes[fmt.Sprintf("%d", code)] = count
	}

	// 构建 proto 消息
	statsData := &pb.StatsData{
		SlaveId:       sb.slaveID,
		TaskId:        sb.taskID,
		Timestamp:     time.Now().Unix(),
		TotalRequests: stats.TotalRequests,
		SuccessCount:  stats.SuccessCount,
		FailedCount:   stats.FailedCount,
		AvgLatency:    stats.AvgLatency,
		P95Latency:    stats.P95Latency,
		P99Latency:    stats.P99Latency,
		Qps:           stats.QPS,
		StatusCodes:   statusCodes,
	}

	// 发送数据
	return sb.reportStream.Send(statsData)
}

// SetMasterClient 设置 Master 客户端
func (sb *StatsBuffer) SetMasterClient(client pb.MasterServiceClient) {
	sb.streamMu.Lock()
	defer sb.streamMu.Unlock()
	sb.masterClient = client
}

// SetTaskID 设置任务 ID
func (sb *StatsBuffer) SetTaskID(taskID string) {
	sb.bufferMu.Lock()
	defer sb.bufferMu.Unlock()
	sb.taskID = taskID
}

// CloseStream 关闭上报流
func (sb *StatsBuffer) CloseStream() error {
	sb.streamMu.Lock()
	defer sb.streamMu.Unlock()

	if sb.reportStream != nil {
		if _, err := sb.reportStream.CloseAndRecv(); err != nil {
			return err
		}
		sb.reportStream = nil
	}
	return nil
}

// aggregate 聚合统计数据
func (sb *StatsBuffer) aggregate(results []*types.RequestResult) *common.SlaveStats {
	stats := &common.SlaveStats{
		SlaveID:     sb.slaveID,
		StatusCodes: make(map[int]int64),
	}

	if len(results) == 0 {
		return stats
	}

	latencies := make([]float64, 0, len(results))

	for _, r := range results {
		stats.TotalRequests++
		if r.Success {
			stats.SuccessCount++
		} else {
			stats.FailedCount++
		}
		latencies = append(latencies, float64(r.Duration.Milliseconds()))
		stats.StatusCodes[r.StatusCode]++
	}

	// 计算延迟统计
	if len(latencies) > 0 {
		stats.AvgLatency = mathx.Mean(latencies)
		stats.P95Latency = mathx.Percentile(latencies, 95)
		stats.P99Latency = mathx.Percentile(latencies, 99)
	}

	// 计算 QPS (1秒窗口)
	stats.QPS = float64(stats.TotalRequests)

	return stats
}
