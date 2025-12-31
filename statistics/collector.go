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
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kamalyes/go-stress/types"
)

// RequestDetail 请求明细
type RequestDetail struct {
	ID         uint64        `json:"id"`
	Timestamp  time.Time     `json:"timestamp"`
	Duration   time.Duration `json:"duration"`
	StatusCode int           `json:"status_code"`
	Success    bool          `json:"success"`
	Skipped    bool          `json:"skipped"`
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
	Verifications []types.VerificationResult `json:"verifications,omitempty"`
	
	// 提取的变量
	ExtractedVars map[string]string `json:"extracted_vars,omitempty"`
}

// Collector 统计收集器
type Collector struct {
	mu sync.Mutex

	totalRequests   uint64
	successRequests uint64
	failedRequests  uint64

	totalDuration time.Duration
	minDuration   time.Duration
	maxDuration   time.Duration

	totalSize float64

	durations   []time.Duration // 用于计算百分位
	errors      map[string]uint64
	statusCodes map[int]uint64 // 状态码统计

	// 请求明细记录（最多保留最近10000条）
	requestDetails []RequestDetail
	maxDetails     int
}

// NewCollector 创建收集器
func NewCollector() *Collector {
	return &Collector{
		durations:      make([]time.Duration, 0, 10000),
		errors:         make(map[string]uint64),
		statusCodes:    make(map[int]uint64),
		requestDetails: make([]RequestDetail, 0, 10000),
		maxDetails:     10000,
		minDuration:    time.Hour, // 初始化为一个大值
	}
}

// Collect 收集单次请求结果
func (c *Collector) Collect(result *types.RequestResult) {
	atomic.AddUint64(&c.totalRequests, 1)

	if result.Success {
		atomic.AddUint64(&c.successRequests, 1)
	} else {
		atomic.AddUint64(&c.failedRequests, 1)

		// 记录错误
		if result.Error != nil {
			c.mu.Lock()
			c.errors[result.Error.Error()]++
			c.mu.Unlock()
		}
	}

	// 统计耗时
	c.mu.Lock()
	c.totalDuration += result.Duration
	c.durations = append(c.durations, result.Duration)

	if result.Duration < c.minDuration {
		c.minDuration = result.Duration
	}
	if result.Duration > c.maxDuration {
		c.maxDuration = result.Duration
	}

	c.totalSize += result.Size

	// 统计状态码
	if result.StatusCode > 0 {
		c.statusCodes[result.StatusCode]++
	}

	// 记录请求明细
	detail := RequestDetail{
		ID:              c.totalRequests,
		Timestamp:       time.Now(),
		Duration:        result.Duration,
		StatusCode:      result.StatusCode,
		Success:         result.Success,
		Skipped:         result.Skipped,
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
	}
	if result.Error != nil {
		detail.Error = result.Error.Error()
	}

	// 保持最多maxDetails条记录
	if len(c.requestDetails) >= c.maxDetails {
		// 删除最早的1000条，保留最新的
		c.requestDetails = c.requestDetails[1000:]
	}
	c.requestDetails = append(c.requestDetails, detail)

	c.mu.Unlock()
}

// GenerateReport 生成统计报告
func (c *Collector) GenerateReport(totalTime time.Duration) *Report {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 排序耗时数据
	sort.Slice(c.durations, func(i, j int) bool {
		return c.durations[i] < c.durations[j]
	})

	report := &Report{
		TotalRequests:   c.totalRequests,
		SuccessRequests: c.successRequests,
		FailedRequests:  c.failedRequests,
		TotalTime:       totalTime,
		TotalSize:       c.totalSize,
		Errors:          c.errors,
		StatusCodes:     c.statusCodes,
		RequestDetails:  c.requestDetails,
	}

	if c.totalRequests > 0 {
		report.SuccessRate = float64(c.successRequests) / float64(c.totalRequests) * 100
		report.AvgDuration = c.totalDuration / time.Duration(c.totalRequests)
		report.QPS = float64(c.totalRequests) / totalTime.Seconds()
	}

	report.MinDuration = c.minDuration
	report.MaxDuration = c.maxDuration

	// 计算百分位
	if len(c.durations) > 0 {
		report.P50 = c.percentile(0.50)
		report.P90 = c.percentile(0.90)
		report.P95 = c.percentile(0.95)
		report.P99 = c.percentile(0.99)
	}

	return report
}

// percentile 计算百分位
func (c *Collector) percentile(p float64) time.Duration {
	if len(c.durations) == 0 {
		return 0
	}

	index := int(float64(len(c.durations)-1) * p)
	return c.durations[index]
}

// GetMetrics 获取实时指标
func (c *Collector) GetMetrics() *Metrics {
	return &Metrics{
		TotalRequests:   atomic.LoadUint64(&c.totalRequests),
		SuccessRequests: atomic.LoadUint64(&c.successRequests),
		FailedRequests:  atomic.LoadUint64(&c.failedRequests),
	}
}

// GetSnapshot 获取统计快照
func (c *Collector) GetSnapshot() *Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshot := &Snapshot{
		TotalRequests:   atomic.LoadUint64(&c.totalRequests),
		SuccessRequests: atomic.LoadUint64(&c.successRequests),
		FailedRequests:  atomic.LoadUint64(&c.failedRequests),
		MinDuration:     c.minDuration,
		MaxDuration:     c.maxDuration,
		TotalSize:       c.totalSize,
	}

	if snapshot.TotalRequests > 0 {
		snapshot.AvgDuration = c.totalDuration / time.Duration(snapshot.TotalRequests)
	}

	return snapshot
}

// GetStatusCodes 获取状态码统计
func (c *Collector) GetStatusCodes() map[int]uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	codes := make(map[int]uint64, len(c.statusCodes))
	for k, v := range c.statusCodes {
		codes[k] = v
	}
	return codes
}

// GetRequestDetails 获取请求明细（支持分页和筛选）
func (c *Collector) GetRequestDetails(offset, limit int, onlyErrors bool) []RequestDetail {
	c.mu.Lock()
	defer c.mu.Unlock()

	var filtered []RequestDetail
	if onlyErrors {
		// 只返回失败的请求
		for i := len(c.requestDetails) - 1; i >= 0; i-- {
			if !c.requestDetails[i].Success {
				filtered = append(filtered, c.requestDetails[i])
			}
		}
	} else {
		// 返回所有请求（倒序，最新的在前面）
		filtered = make([]RequestDetail, len(c.requestDetails))
		for i := 0; i < len(c.requestDetails); i++ {
			filtered[i] = c.requestDetails[len(c.requestDetails)-1-i]
		}
	}

	// 分页
	if offset >= len(filtered) {
		return []RequestDetail{}
	}

	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[offset:end]
}

// GetRequestDetailsCount 获取请求明细总数
func (c *Collector) GetRequestDetailsCount(onlyErrors bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !onlyErrors {
		return len(c.requestDetails)
	}

	count := 0
	for _, detail := range c.requestDetails {
		if !detail.Success {
			count++
		}
	}
	return count
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
