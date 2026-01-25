/*
* @Author: kamalyes 501893067@qq.com
* @Date: 2025-12-30 00:00:00
* @LastEditors: kamalyes 501893067@qq.com
* @LastEditTime: 2025-12-30 13:30:52
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

// RequestDetail 请求明细
type RequestDetail struct {
	ID         string        `json:"id"`
	Timestamp  time.Time     `json:"timestamp"`
	Duration   time.Duration `json:"duration"`
	StatusCode int           `json:"status_code"`
	Success    bool          `json:"success"`
	Skipped    bool          `json:"skipped"`
	SkipReason string        `json:"skip_reason,omitempty"` // 跳过原因
	GroupID    uint64        `json:"group_id"`
	APIName    string        `json:"api_name,omitempty"`
	Error      string        `json:"error,omitempty"`
	Size       float64       `json:"size"`

	// 请求信息
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Query   string            `json:"query,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`

	// 响应信息
	ResponseBody    string            `json:"response_body,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`

	// 验证信息
	Verifications []VerificationResult `json:"verifications,omitempty"`

	// 提取的变量
	ExtractedVars map[string]string `json:"extracted_vars,omitempty"`
}

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
	storage DetailStorageInterface

	// ID 生成器（使用 Snowflake 算法生成全局唯一ID）
	idGenerator *idgen.SnowflakeGenerator

	// 外部上报器（用于分布式模式）
	externalReporter func(*RequestResult)
	reporterMu       *syncx.RWLock

	// 运行模式
	runMode string // "cli" 或 "config"
}

// NewCollector 创建收集器（默认内存模式）
func NewCollector() *Collector {
	return NewCollectorWithMemoryStorage("local")
}

// NewCollectorWithMemoryStorage 创建内存存储收集器
func NewCollectorWithMemoryStorage(nodeID string) *Collector {
	storage := NewMemoryStorage(nodeID, logger.Default)

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
		storage:         storage,
		idGenerator:     idgen.NewSnowflakeGenerator(1, 1),
		minDuration:     time.Hour,
	}
}

// NewCollectorWithStorage 创建带 SQLite 存储的收集器
func NewCollectorWithStorage(dbPath, nodeID string) *Collector {
	storage, err := NewDetailStorage(dbPath, nodeID, logger.Default)
	if err != nil {
		logger.Default.Errorf("❌ 创建 SQLite 存储失败: %v，降级为内存模式", err)
		return NewCollectorWithMemoryStorage(nodeID)
	}

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
		storage:         storage,
		idGenerator:     idgen.NewSnowflakeGenerator(1, 1),
		minDuration:     time.Hour,
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

	// 异步写入SQLite存储（如果启用）
	if c.storage != nil {
		detail := &RequestDetail{
			ID:              c.idGenerator.GenerateRequestID(), // 使用 Snowflake 生成全局唯一ID
			Timestamp:       time.Now(),
			Duration:        result.Duration,
			StatusCode:      result.StatusCode,
			Success:         result.Success,
			Skipped:         result.Skipped,
			SkipReason:      result.SkipReason,
			GroupID:         result.GroupID,
			APIName:         result.APIName,
			Size:            result.Size,
			URL:             result.URL,
			Method:          result.Method,
			Query:           result.Query,
			Headers:         result.Headers,
			Body:            result.Body,
			ResponseBody:    result.ResponseBody,
			ResponseHeaders: result.ResponseHeaders,
			Verifications:   result.Verifications,
			ExtractedVars:   result.ExtractedVars,
			Error:           mathx.IfDo(result.Error != nil, func() string { return result.Error.Error() }, ""),
		}
		c.storage.Write(detail)
	}
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
			RequestDetails:  nil, // 详情数据从 SQLite 按需加载
		}

		if totalReqs > 0 {
			// 使用 mathx.Percentage 计算成功率
			report.SuccessRate = mathx.Percentage(successReqs, totalReqs)
			report.AvgDuration = c.totalDuration / time.Duration(totalReqs)
			report.QPS = float64(totalReqs) / totalTime.Seconds()
		}

		report.MinDuration = c.minDuration
		report.MaxDuration = c.maxDuration

		// 使用 mathx 计算的百分位
		if len(c.durations) > 0 {
			report.P50 = time.Duration(percentiles[50] * float64(time.Second))
			report.P90 = time.Duration(percentiles[90] * float64(time.Second))
			report.P95 = time.Duration(percentiles[95] * float64(time.Second))
			report.P99 = time.Duration(percentiles[99] * float64(time.Second))
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
		MinDuration:     c.minDuration,
		MaxDuration:     c.maxDuration,
		TotalSize:       c.totalSize,
	}

	if totalReqs > 0 {
		snapshot.AvgDuration = c.totalDuration / time.Duration(totalReqs)
	}

	return snapshot
}

// GetStatusCodes 获取状态码统计
func (c *Collector) GetStatusCodes() map[int]uint64 {
	return c.statusCodes.ToMap()
}

// GetRequestDetails 获取请求明细（支持分页和筛选）
func (c *Collector) GetRequestDetails(offset, limit int, statusFilter StatusFilter) []*RequestDetail {
	// 优先从 SQLite 存储读取
	if c.storage != nil {
		details, err := c.storage.Query(offset, limit, statusFilter)
		if err == nil {
			return details
		}
		logger.Default.Errorf("从存储读取失败: %v", err)
	}

	// 降级：返回空切片
	return []*RequestDetail{}
}

// GetRequestDetailsCount 获取请求明细总数
func (c *Collector) GetRequestDetailsCount(statusFilter StatusFilter) int {
	// 优先从 SQLite 存储读取
	if c.storage != nil {
		count, err := c.storage.Count(statusFilter)
		if err == nil {
			return count
		}
		logger.Default.Errorf("统计总数失败: %v", err)
	}

	// 降级：返回0
	return 0
}

// SetExternalReporter 设置外部上报器
func (c *Collector) SetExternalReporter(reporter func(*RequestResult)) {
	c.reporterMu.Lock()
	defer c.reporterMu.Unlock()
	c.externalReporter = reporter
}

// SetRunMode 设置运行模式
func (c *Collector) SetRunMode(mode string) {
	c.runMode = mode
}

// ClearExternalReporter 清除外部上报器
func (c *Collector) ClearExternalReporter() {
	c.reporterMu.Lock()
	defer c.reporterMu.Unlock()
	c.externalReporter = nil
}

// Close 关闭收集器，释放资源
func (c *Collector) Close() error {
	if c.storage != nil {
		return c.storage.Close()
	}
	return nil
}

// Snapshot 统计快照（用于实时显示）
type Snapshot struct {
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
	MinDuration     time.Duration
	MaxDuration     time.Duration
	AvgDuration     time.Duration
	TotalSize       float64
}

// Metrics 实时指标
type Metrics struct {
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
}
