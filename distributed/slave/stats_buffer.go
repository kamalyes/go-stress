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
	"sort"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/distributed/common"
	pb "github.com/kamalyes/go-stress/distributed/proto"
	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// maxStatsBacklog 断线期间最多缓存的补报数据条数
//
// Master 故障/主备切换窗口内（默认 15s Lease 过期 + 接管）上报会失败，
// 失败的批次进入 backlog，恢复连接后按 FIFO 补报；超过上限丢最旧批次
// 并告警——压测统计可容忍少量批次缺失，但不能无限吃内存
const maxStatsBacklog = 1024

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
	// backlog 断线期间待补报的统计批次（FIFO，上限 maxStatsBacklog）
	backlog   []*pb.StatsData
	backlogMu syncx.Locker
}

// NewStatsBuffer 创建统计缓冲区
func NewStatsBuffer(slaveID string, maxSize int, log logger.ILogger) *StatsBuffer {
	return &StatsBuffer{
		slaveID:   slaveID,
		buffer:    make([]*types.RequestResult, 0, maxSize),
		bufferMu:  syncx.NewLock(),
		maxSize:   maxSize,
		streamMu:  syncx.NewLock(),
		backlogMu: syncx.NewLock(),
		logger:    log,
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
//
// 上报失败（Master 故障/主备切换窗口）时批次进入 backlog，连接恢复后
// 按时序补报——断线期间的统计数据不会静默丢失
func (sb *StatsBuffer) Flush() error {
	sb.bufferMu.Lock()
	if len(sb.buffer) == 0 {
		sb.bufferMu.Unlock()
		// 缓冲区为空也要尝试补报积压（重连后的恢复路径）
		return sb.flushBacklog()
	}

	// 复制并清空缓冲区（taskID 同锁快照，避免与 SetTaskID 竞态）
	toSend := make([]*types.RequestResult, len(sb.buffer))
	copy(toSend, sb.buffer)
	sb.buffer = sb.buffer[:0]
	taskID := sb.taskID
	sb.bufferMu.Unlock()

	// 聚合数据
	stats := sb.aggregate(toSend, taskID)

	// 没有 taskID 的数据无法归属任务，不上报也不进积压
	if taskID == "" {
		sb.logger.DebugKV("Stats flushed locally (no task)",
			"slave_id", sb.slaveID,
			"records", len(toSend))
		return nil
	}

	data := sb.buildStatsData(stats)

	if sb.masterClient == nil {
		sb.enqueueBacklog(data)
		return nil
	}

	// 发送到 Master：失败进积压，成功后补报历史积压
	if err := sb.sendToMaster(data); err != nil {
		sb.enqueueBacklog(data)
		sb.logger.WarnKV("Failed to send stats to master, queued for resend",
			"error", err,
			"backlog", sb.backlogSize())
		return err
	}
	if err := sb.flushBacklog(); err != nil {
		return err
	}

	sb.logger.DebugKV("Stats flushed",
		"slave_id", sb.slaveID,
		"records", len(toSend),
		"total_requests", stats.TotalRequests)

	return nil
}

// backlogSize 当前积压条数
func (sb *StatsBuffer) backlogSize() int {
	sb.backlogMu.Lock()
	defer sb.backlogMu.Unlock()
	return len(sb.backlog)
}

// enqueueBacklog 将发送失败的批次放入补报队列（超限丢最旧）
func (sb *StatsBuffer) enqueueBacklog(data *pb.StatsData) {
	sb.backlogMu.Lock()
	defer sb.backlogMu.Unlock()
	if len(sb.backlog) >= maxStatsBacklog {
		dropped := sb.backlog[0]
		sb.backlog = sb.backlog[1:]
		sb.logger.WarnKV("Stats backlog overflow, dropped oldest batch",
			"dropped_timestamp", dropped.Timestamp)
	}
	sb.backlog = append(sb.backlog, data)
}

// flushBacklog 补报积压批次（FIFO，失败则剩余批次回到队列）
func (sb *StatsBuffer) flushBacklog() error {
	if sb.masterClient == nil {
		return nil
	}

	sb.backlogMu.Lock()
	if len(sb.backlog) == 0 {
		sb.backlogMu.Unlock()
		return nil
	}
	pending := sb.backlog
	sb.backlog = nil
	sb.backlogMu.Unlock()

	for i, data := range pending {
		if err := sb.sendToMaster(data); err != nil {
			sb.backlogMu.Lock()
			sb.backlog = append(pending[i:], sb.backlog...)
			sb.backlogMu.Unlock()
			return err
		}
	}

	if len(pending) > 0 {
		sb.logger.InfoKV("Stats backlog resent to master", "batches", len(pending))
	}
	return nil
}

// buildStatsData 将聚合结果转为 proto 消息
func (sb *StatsBuffer) buildStatsData(stats *common.SlaveStats) *pb.StatsData {
	// 转换状态码格式
	statusCodes := make(map[string]int64)
	for code, count := range stats.StatusCodes {
		statusCodes[fmt.Sprintf("%d", code)] = count
	}

	return &pb.StatsData{
		SlaveId:         sb.slaveID,
		TaskId:          stats.TaskID,
		Timestamp:       time.Now().Unix(),
		TotalRequests:   stats.TotalRequests,
		SuccessRequests: stats.SuccessRequests,
		FailedRequests:  stats.FailedRequests,
		AvgLatency:      stats.AvgLatency,
		P95Latency:      stats.P95Latency,
		P99Latency:      stats.P99Latency,
		Qps:             stats.QPS,
		StatusCodes:     statusCodes,
	}
}

// sendToMaster 发送统计数据到 Master
func (sb *StatsBuffer) sendToMaster(data *pb.StatsData) error {
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

	// 发送数据
	if err := sb.reportStream.Send(data); err != nil {
		// 发送失败通常意味着流已经失效（Master 重启/主备切换），
		// 丢弃旧流让下次尝试重新建流到（可能已切换的）新 Master
		if sb.reportStream != nil {
			// 关闭旧流，忽略二次错误（连接本就断了）
			_, _ = sb.reportStream.CloseAndRecv()
			sb.reportStream = nil
		}
		return err
	}
	return nil
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
func (sb *StatsBuffer) aggregate(results []*types.RequestResult, taskID string) *common.SlaveStats {
	stats := &common.SlaveStats{
		SlaveID:     sb.slaveID,
		TaskID:      taskID,
		StatusCodes: make(map[int]int64),
		ErrorTypes:  make(map[string]int64),
	}

	if len(results) == 0 {
		return stats
	}

	latencies := make([]float64, 0, len(results))

	for _, r := range results {
		stats.TotalRequests++
		if r.Success {
			stats.SuccessRequests++
		} else {
			stats.FailedRequests++
			// 收集错误类型
			if r.Error != nil {
				stats.ErrorTypes[r.Error.Error()]++
			}
		}
		latencies = append(latencies, float64(r.Duration.Milliseconds()))
		stats.StatusCodes[r.StatusCode]++
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

	// 计算成功率和 QPS
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100
	}
	stats.QPS = float64(stats.TotalRequests)
	stats.TotalQPS = stats.QPS

	return stats
}
