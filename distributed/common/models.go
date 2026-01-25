/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\common\models.go
 * @Description: 核心数据模型定义
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package common

import (
	"time"
)

// SlaveInfo Slave 信息
type SlaveInfo struct {
	ID              string            `json:"id"`
	Hostname        string            `json:"hostname"`
	IP              string            `json:"ip"`
	GRPCPort        int32             `json:"grpc_port"` // gRPC 服务端口
	CPUCores        int               `json:"cpu_cores"`
	Memory          int64             `json:"memory"`
	Version         string            `json:"version"`
	Region          string            `json:"region"`
	Labels          map[string]string `json:"labels"`
	State           SlaveState        `json:"state"`
	CurrentTaskID   string            `json:"current_task_id"`
	LastHeartbeat   time.Time         `json:"last_heartbeat"`
	RegisteredAt    time.Time         `json:"registered_at"`
	TotalRequests   int64             `json:"total_requests"`
	CurrentLoad     float64           `json:"current_load"` // 当前负载 0-1
	HealthCheckFail int               `json:"health_check_fail"`
	ResourceUsage   *ResourceUsage    `json:"resource_usage"`  // 资源使用情况
	Role            NodeRole          `json:"role"`            // 节点角色
	CanReuse        bool              `json:"can_reuse"`       // 是否允许复用（在忙碌时）
	RunningTasks    []string          `json:"running_tasks"`   // 正在运行的任务列表
	MaxConcurrency  int               `json:"max_concurrency"` // 最大并发任务数
}

// Task 任务定义
type Task struct {
	ID             string            `json:"id"`
	Protocol       string            `json:"protocol"`
	Target         string            `json:"target"`
	TotalWorkers   int               `json:"total_workers"`
	Duration       int               `json:"duration"`
	RampUp         int               `json:"ramp_up"`
	ConfigData     []byte            `json:"config_data"`
	ReportInterval int               `json:"report_interval"`
	State          TaskState         `json:"state"`
	AssignedSlaves []string          `json:"assigned_slaves"`
	CreatedAt      time.Time         `json:"created_at"`
	StartedAt      time.Time         `json:"started_at"`
	CompletedAt    time.Time         `json:"completed_at"`
	Metadata       map[string]string `json:"metadata"`
}

// SubTask 子任务（分配给单个 Slave）
type SubTask struct {
	TaskID      string `json:"task_id"`
	SubTaskID   string `json:"sub_task_id"`
	SlaveID     string `json:"slave_id"`
	WorkerCount int    `json:"worker_count"`
	Config      []byte `json:"config"`
}

// TaskConfig 任务配置（用于外部提交）
type TaskConfig struct {
	Protocol    string            `json:"protocol"`
	Target      string            `json:"target"`
	WorkerCount int32             `json:"worker_count"`
	Duration    int               `json:"duration"`
	RampUp      int               `json:"ramp_up"`
	ConfigData  []byte            `json:"config_data"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
