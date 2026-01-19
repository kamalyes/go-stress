/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 17:11:06
 * @FilePath: \go-stress\types\statistics.go
 * @Description: 统计相关类型定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package types

import (
	"time"
)

// RequestResult 请求结果（用于统计）
type RequestResult struct {
	Success    bool          // 是否成功
	StatusCode int           // HTTP 状态码
	Duration   time.Duration // 请求耗时
	Size       float64       // 响应大小
	Error      error         // 错误信息
	Timestamp  time.Time     // 时间戳
	Skipped    bool          // 是否被跳过（因依赖失败）
	SkipReason string        // 跳过原因（记录具体哪个依赖API失败）
	GroupID    uint64        // 分组ID（同一个worker的依赖链共享同一个GroupID）
	APIName    string        // API名称（如 create_ticket, send_message）

	// 请求详情
	URL     string            // 请求URL
	Method  string            // 请求方法
	Query   string            // Query参数
	Headers map[string]string // 请求头
	Body    string            // 请求体

	// 响应详情
	ResponseBody    string            // 响应体
	ResponseHeaders map[string]string // 响应头

	// 验证信息
	Verifications []VerificationResult // 验证结果列表

	// 提取变量
	ExtractedVars map[string]string // 提取的变量
}

// VerificationResult 验证结果
type VerificationResult struct {
	Type    VerifyType `json:"type"`    // 验证类型：STATUS_CODE, JSONPATH, CONTAINS等
	Success bool       `json:"success"` // 验证是否成功
	Skipped bool       `json:"skipped"` // 是否被跳过（未执行）
	Message string     `json:"message"` // 验证消息（成功或失败原因）
	Expect  string     `json:"expect"`  // 期望值
	Actual  string     `json:"actual"`  // 实际值
}

// Statistics 统计数据
type Statistics struct {
	TotalRequests   uint64        // 总请求数
	SuccessRequests uint64        // 成功请求数
	FailedRequests  uint64        // 失败请求数
	TotalDuration   time.Duration // 总耗时
	MinDuration     time.Duration // 最小耗时
	MaxDuration     time.Duration // 最大耗时
	AvgDuration     time.Duration // 平均耗时
}

// ReportMode 报告模式
type ReportMode string

const (
	ReportModeStatic   ReportMode = "static"   // 静态HTML报告
	ReportModeRealtime ReportMode = "realtime" // 实时报告
)

// ReportData 统一的报告数据结构（所有字段都是原始类型）
type ReportData struct {
	Mode         ReportMode `json:"mode"`          // 报告模式
	GenerateTime time.Time  `json:"generate_time"` // 生成时间

	// 基础统计
	TotalRequests   uint64  `json:"total_requests"`   // 总请求数
	SuccessRequests uint64  `json:"success_requests"` // 成功请求数
	FailedRequests  uint64  `json:"failed_requests"`  // 失败请求数
	SuccessRate     float64 `json:"success_rate"`     // 成功率（0-100）

	// 性能指标
	QPS         float64       `json:"qps"`          // 每秒请求数
	MinDuration time.Duration `json:"min_duration"` // 最小耗时
	MaxDuration time.Duration `json:"max_duration"` // 最大耗时
	AvgDuration time.Duration `json:"avg_duration"` // 平均耗时
	P50Duration time.Duration `json:"p50_duration"` // P50耗时
	P90Duration time.Duration `json:"p90_duration"` // P90耗时
	P95Duration time.Duration `json:"p95_duration"` // P95耗时
	P99Duration time.Duration `json:"p99_duration"` // P99耗时

	// 数据量
	TotalSize float64 `json:"total_size"` // 总数据量（字节）

	// 统计详情
	ErrorStats      []ErrorStat      `json:"error_stats"`       // 错误统计
	StatusCodeStats []StatusCodeStat `json:"status_code_stats"` // 状态码统计
	RequestDetails  []RequestDetail  `json:"request_details"`   // 请求明细

	// 测试配置（可选）
	Config *TestConfig `json:"config,omitempty"` // 测试配置
}

// ErrorStat 错误统计
type ErrorStat struct {
	Error      string  `json:"error"`      // 错误信息
	Count      uint64  `json:"count"`      // 出现次数
	Percentage float64 `json:"percentage"` // 占比（0-100）
}

// StatusCodeStat 状态码统计
type StatusCodeStat struct {
	StatusCode int     `json:"status_code"` // 状态码
	Count      uint64  `json:"count"`       // 出现次数
	Percentage float64 `json:"percentage"`  // 占比（0-100）
}

// RequestDetail 请求明细
type RequestDetail struct {
	Timestamp  time.Time     `json:"timestamp"`   // 时间戳
	Method     string        `json:"method"`      // 请求方法
	URL        string        `json:"url"`         // 请求URL
	StatusCode int           `json:"status_code"` // 状态码
	Duration   time.Duration `json:"duration"`    // 耗时
	Size       float64       `json:"size"`        // 响应大小
	Success    bool          `json:"success"`     // 是否成功
	Error      string        `json:"error"`       // 错误信息
}

// TestConfig 测试配置
type TestConfig struct {
	Concurrency uint64        `json:"concurrency"` // 并发数
	Requests    uint64        `json:"requests"`    // 总请求数
	Duration    time.Duration `json:"duration"`    // 测试时长
	URL         string        `json:"url"`         // 测试URL
}
