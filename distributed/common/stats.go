/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\common\stats.go
 * @Description: 统计相关类型定义
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package common

import (
	"time"
)

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUPercent     float64   `json:"cpu_percent"`      // CPU 使用率 0-100
	MemoryPercent  float64   `json:"memory_percent"`   // 内存使用率 0-100
	MemoryUsed     int64     `json:"memory_used"`      // 已使用内存(字节)
	MemoryTotal    int64     `json:"memory_total"`     // 总内存(字节)
	ActiveTasks    int       `json:"active_tasks"`     // 活跃任务数
	QueuedTasks    int       `json:"queued_tasks"`     // 排队任务数
	LoadAverage    float64   `json:"load_average"`     // 系统负载平均值
	NetworkInMbps  float64   `json:"network_in_mbps"`  // 网络入流量(Mbps)
	NetworkOutMbps float64   `json:"network_out_mbps"` // 网络出流量(Mbps)
	DiskIOUtil     float64   `json:"disk_io_util"`     // 磁盘 IO 使用率 0-100
	Timestamp      time.Time `json:"timestamp"`        // 采集时间
}

// AggregatedStats 聚合统计数据
type AggregatedStats struct {
	TaskID          string                 `json:"task_id"`
	TimeRange       TimeRange              `json:"time_range"`
	TotalAgents     int                    `json:"total_agents"`
	TotalRequests   int64                  `json:"total_requests"`
	SuccessRequests int64                  `json:"success_requests"`
	FailedRequests  int64                  `json:"failed_requests"`
	SuccessRate     float64                `json:"success_rate"`
	AvgLatency      float64                `json:"avg_latency"`
	MinLatency      float64                `json:"min_latency"`
	MaxLatency      float64                `json:"max_latency"`
	P50Latency      float64                `json:"p50_latency"`
	P90Latency      float64                `json:"p90_latency"`
	P95Latency      float64                `json:"p95_latency"`
	P99Latency      float64                `json:"p99_latency"`
	TotalQPS        float64                `json:"total_qps"`
	BySlave         map[string]*SlaveStats `json:"by_slave"`
	StatusCodes     map[int]int64          `json:"status_codes"`
	ErrorTypes      map[string]int64       `json:"error_types"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// SlaveStats Slave 统计数据
type SlaveStats struct {
	TaskID          string           `json:"task_id"` // 任务ID
	SlaveID         string           `json:"slave_id"`
	TotalRequests   int64            `json:"total_requests"`
	SuccessRequests int64            `json:"success_requests"`
	FailedRequests  int64            `json:"failed_requests"`
	SuccessRate     float64          `json:"success_rate"`
	AvgLatency      float64          `json:"avg_latency"`
	MinLatency      float64          `json:"min_latency"`
	MaxLatency      float64          `json:"max_latency"`
	P50Latency      float64          `json:"p50_latency"`
	P95Latency      float64          `json:"p95_latency"`
	P90Latency      float64          `json:"p90_latency"`
	P99Latency      float64          `json:"p99_latency"`
	QPS             float64          `json:"qps"`
	TotalQPS        float64          `json:"total_qps"`
	StatusCodes     map[int]int64    `json:"status_codes"`
	ErrorTypes      map[string]int64 `json:"error_types"`
}
