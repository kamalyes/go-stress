/*
* @Author: kamalyes 501893067@qq.com
* @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 11:15:22
* @FilePath: \go-stress\statistics\collector.go
* @Description: 统计数据收集器
*
* Copyright (c) 2025 by kamalyes, All Rights Reserved.
*/
package statistics

import (
	"time"

	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-toolbox/pkg/idgen"
	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// Collector 统计收集器
type Collector struct {
	// 使用 syncx 原子类型
	totalRequests   *syncx.Uint64
	successRequests *syncx.Uint64
	failedRequests  *syncx.Uint64
	skippedRequests *syncx.Uint64 // 跳过请求计数器

	// 时长统计（需要加锁）
	mu            *syncx.RWLock
	totalDuration time.Duration
	minDuration   time.Duration
	maxDuration   time.Duration
	durations     []float64 // 用于计算百分位（转为秒）

	totalSize float64

	// 使用 syncx.Map 替换 map + mutex
	errors      *syncx.Map[string, uint64]
	statusCodes *syncx.Map[int, uint64]

	// 统一的存储接口（支持 SQLite 和 Memory 两种实现）
	storage StorageInterface

	// ID 生成器（使用 Snowflake 算法生成全局唯一ID）
	idGenerator *idgen.SnowflakeGenerator

	// 外部上报器（用于分布式模式）
	externalReporter func(*RequestResult)
	reporterMu       *syncx.RWLock

	// 运行模式
	runMode RunMode

	// 配置信息（用于报告显示）
	protocol    string
	concurrency uint64
	totalReqs   uint64

	// 关闭标志
	closed *syncx.Bool
}

// NewCollectorWithStorageInterface 使用已创建的存储接口创建收集器（工厂模式）
func NewCollectorWithStorageInterface(strg StorageInterface) *Collector {
	return &Collector{
		totalRequests:   syncx.NewUint64(0),
		successRequests: syncx.NewUint64(0),
		failedRequests:  syncx.NewUint64(0),
		skippedRequests: syncx.NewUint64(0),
		mu:              syncx.NewRWLock(),
		reporterMu:      syncx.NewRWLock(),
		durations:       make([]float64, 0, 10000),
		errors:          syncx.NewMap[string, uint64](),
		statusCodes:     syncx.NewMap[int, uint64](),
		storage:         strg,
		idGenerator:     idgen.NewSnowflakeGenerator(1, 1),
		minDuration:     time.Hour,
		closed:          syncx.NewBool(false),
	}
}

// Collect 收集单次请求结果
func (c *Collector) Collect(result *RequestResult) {
	if result == nil {
		logger.Default.Warn("⚠️  收到空的请求结果，跳过收集")
		return
	}

	// 调用外部上报器（如果设置了）
	c.reporterMu.RLock()
	if c.externalReporter != nil {
		c.externalReporter(result)
	}
	c.reporterMu.RUnlock()

	// 原子操作，无需加锁
	c.totalRequests.Add(1)

	if result.Skipped {
		// 跳过的请求单独计数，不计入成功或失败
		c.skippedRequests.Add(1)
	} else if result.Success {
		// 只有非跳过的请求才计入成功
		c.successRequests.Add(1)
	} else {
		// 只有非跳过的请求才计入失败
		c.failedRequests.Add(1)

		// 记录错误 - 使用 syncx.Map 线程安全
		if result.Error != nil {
			errMsg := result.Error.Error()
			old, _ := c.errors.LoadOrStore(errMsg, 0)
			c.errors.Store(errMsg, old+1)
		}
	}

	// 统计状态码 - 使用 syncx.Map
	if result.StatusCode > 0 {
		old, _ := c.statusCodes.LoadOrStore(result.StatusCode, 0)
		c.statusCodes.Store(result.StatusCode, old+1)
	}

	// 统计耗时 - 使用 syncx.WithLock 包装
	syncx.WithLock(c.mu, func() {
		c.totalDuration += result.Duration
		c.durations = append(c.durations, result.Duration.Seconds())

		c.minDuration = mathx.Min(c.minDuration, result.Duration)
		c.maxDuration = mathx.Max(c.maxDuration, result.Duration)

		c.totalSize += result.Size
	})

	// 生成唯一ID和错误消息
	result.ID = c.idGenerator.GenerateRequestID()
	if result.Error != nil {
		result.ErrorMsg = result.Error.Error()
	}

	// 直接写入存储
	c.storage.Write(result)
}

// GenerateReport 生成统计报告
func (c *Collector) GenerateReport(totalTime time.Duration) *Report {
	return syncx.WithRLockReturnValue(c.mu, func() *Report {
		// 使用 mathx 批量计算百分位
		percentiles := mathx.Percentiles(c.durations, 50, 90, 95, 99)

		// 使用 ToMap() 高级方法获取统计数据
		errorsMap := c.errors.ToMap()
		statusCodesMap := c.statusCodes.ToMap()

		totalReqs := c.totalRequests.Load()
		successReqs := c.successRequests.Load()

		report := &Report{
			TotalRequests:   totalReqs,
			SuccessRequests: successReqs,
			FailedRequests:  c.failedRequests.Load(),
			TotalTime:       totalTime,
			TotalSize:       c.totalSize,
			Errors:          errorsMap,
			StatusCodes:     statusCodesMap,
			RequestDetails:  nil, // 详情数据按需加载（通过 QueryDetails/QueryAll 从存储层获取）
		}

		if totalReqs > 0 {
			// 使用 mathx.Percentage 计算成功率
			report.SuccessRate = mathx.Percentage(successReqs, totalReqs)
			report.AvgLatency = c.totalDuration / time.Duration(totalReqs)
			report.QPS = float64(totalReqs) / totalTime.Seconds()
		}

		report.MinLatency = c.minDuration
		report.MaxLatency = c.maxDuration

		// 使用 mathx 计算的百分位
		if len(percentiles) > 0 {
			report.P50Latency = time.Duration(percentiles[50] * float64(time.Second))
			report.P90Latency = time.Duration(percentiles[90] * float64(time.Second))
			report.P95Latency = time.Duration(percentiles[95] * float64(time.Second))
			report.P99Latency = time.Duration(percentiles[99] * float64(time.Second))
		}

		return report
	})
}

// GetMetrics 获取实时指标
func (c *Collector) GetMetrics() *Metrics {
	return &Metrics{
		TotalRequests:   c.totalRequests.Load(),
		SuccessRequests: c.successRequests.Load(),
		FailedRequests:  c.failedRequests.Load(),
	}
}

// GetSnapshot 获取统计快照
func (c *Collector) GetSnapshot() *Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalReqs := c.totalRequests.Load()

	snapshot := &Snapshot{
		TotalRequests:   totalReqs,
		SuccessRequests: c.successRequests.Load(),
		FailedRequests:  c.failedRequests.Load(),
		MinLatency:      c.minDuration,
		MaxLatency:      c.maxDuration,
		TotalSize:       c.totalSize,
	}

	if totalReqs > 0 {
		snapshot.AvgLatency = c.totalDuration / time.Duration(totalReqs)
	}

	return snapshot
}

// GetStatusCodes 获取状态码统计
func (c *Collector) GetStatusCodes() map[int]uint64 {
	return c.statusCodes.ToMap()
}

// GetRequestDetails 获取请求明细（支持分页和筛选）
func (c *Collector) GetRequestDetails(offset, limit int, statusFilter StatusFilter, nodeID, taskID string) []*RequestResult {
	// 即使 Collector 已关闭，依然允许读取已存储的数据
	if c.storage == nil {
		logger.Default.Warn("⚠️  存储未初始化")
		return []*RequestResult{}
	}

	details, err := c.storage.Query(offset, limit, statusFilter, nodeID, taskID)
	if err == nil {
		return details
	}

	// 记录错误（除非已关闭）
	if !c.closed.Load() {
		logger.Default.Warnf("⚠️  从存储读取失败: %v", err)
	}

	// 降级：返回空切片
	return []*RequestResult{}
}

// GetRequestDetailsCount 获取请求明细总数
func (c *Collector) GetRequestDetailsCount(statusFilter StatusFilter, nodeID, taskID string) int {
	// 即使 Collector 已关闭，依然允许读取已存储的数据计数
	if c.storage == nil {
		logger.Default.Warn("⚠️  存储未初始化")
		return 0
	}

	count, err := c.storage.Count(statusFilter, nodeID, taskID)
	if err == nil {
		return count
	}

	// 记录错误（除非已关闭）
	if !c.closed.Load() {
		logger.Default.Warnf("⚠️  统计总数失败: %v", err)
	}

	// 降级：返回0
	return 0
}

// GetRequestDetailsWithFilter 获取请求明细（支持指定 nodeID 和 taskID 过滤，用于分布式模式）
func (c *Collector) GetRequestDetailsWithFilter(offset, limit int, statusFilter StatusFilter, nodeID, taskID string) []*RequestResult {
	if c.storage == nil {
		logger.Default.Warn("⚠️  存储未初始化")
		return []*RequestResult{}
	}

	details, err := c.storage.Query(offset, limit, statusFilter, nodeID, taskID)
	if err == nil {
		return details
	}

	if !c.closed.Load() {
		logger.Default.Warnf("⚠️  从存储读取失败: %v", err)
	}

	return []*RequestResult{}
}

// GetRequestDetailsCountWithFilter 获取请求明细总数（支持指定 nodeID 和 taskID 过滤，用于分布式模式）
func (c *Collector) GetRequestDetailsCountWithFilter(statusFilter StatusFilter, nodeID, taskID string) int {
	if c.storage == nil {
		logger.Default.Warn("⚠️  存储未初始化")
		return 0
	}

	count, err := c.storage.Count(statusFilter, nodeID, taskID)
	if err == nil {
		return count
	}

	if !c.closed.Load() {
		logger.Default.Warnf("⚠️  统计总数失败: %v", err)
	}

	return 0
}

// SetExternalReporter 设置外部上报器
func (c *Collector) SetExternalReporter(reporter func(*RequestResult)) {
	c.reporterMu.Lock()
	defer c.reporterMu.Unlock()
	c.externalReporter = reporter
}

// SetRunMode 设置运行模式
func (c *Collector) SetRunMode(mode RunMode) {
	c.runMode = mode
}

// SetConfig 设置配置信息（用于报告显示）
func (c *Collector) SetConfig(protocol string, concurrency, totalReqs uint64) {
	c.protocol = protocol
	c.concurrency = concurrency
	c.totalReqs = totalReqs
}

// ClearExternalReporter 清除外部上报器
func (c *Collector) ClearExternalReporter() {
	c.reporterMu.Lock()
	defer c.reporterMu.Unlock()
	c.externalReporter = nil
}

// Close 关闭收集器，释放资源
func (c *Collector) Close() error {
	// 设置关闭标志
	c.closed.Store(true)
	logger.Default.Debug("📌 Collector 已标记为关闭状态")

	if c.storage != nil {
		logger.Default.Debug("📌 正在关闭存储...")
		return c.storage.Close()
	}
	return nil
}

// Snapshot 统计快照（用于实时显示）
type Snapshot struct {
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AvgLatency      time.Duration
	TotalSize       float64
}

// Metrics 实时指标
type Metrics struct {
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
}
