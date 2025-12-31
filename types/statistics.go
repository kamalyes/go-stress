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
