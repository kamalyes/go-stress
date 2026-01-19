/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 00:02:15
 * @FilePath: \go-stress\distributed\common\configs.go
 * @Description: 配置结构体定义
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package common

import (
	"time"
)

// DistributedConfig 分布式配置
type DistributedConfig struct {
	MasterAddr     string         `json:"master_addr" yaml:"master_addr"`
	SlaveCount     int            `json:"slave_count" yaml:"slave_count"`
	SlaveSelector  SelectStrategy `json:"slave_selector" yaml:"slave_selector"`
	TaskSplitter   SplitStrategy  `json:"task_splitter" yaml:"task_splitter"`
	SlaveFilter    *SlaveFilter   `json:"slave_filter" yaml:"slave_filter"` // Slave 筛选条件
	EnableTLS      bool           `json:"enable_tls" yaml:"enable_tls"`
	CertFile       string         `json:"cert_file" yaml:"cert_file"`
	KeyFile        string         `json:"key_file" yaml:"key_file"`
	ReportInterval int            `json:"report_interval" yaml:"report_interval"`
	BufferSize     int            `json:"buffer_size" yaml:"buffer_size"`
}

// MasterConfig Master 配置
type MasterConfig struct {
	GRPCPort          int           `json:"grpc_port" yaml:"grpc_port"`
	HTTPPort          int           `json:"http_port" yaml:"http_port"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval"`
	HeartbeatTimeout  time.Duration `json:"heartbeat_timeout" yaml:"heartbeat_timeout"`
	MaxFailures       int           `json:"max_failures" yaml:"max_failures"`
	SlaveFilter       *SlaveFilter  `json:"slave_filter" yaml:"slave_filter"` // Slave 筛选条件
	EnableTLS         bool          `json:"enable_tls" yaml:"enable_tls"`
	CertFile          string        `json:"cert_file" yaml:"cert_file"`
	KeyFile           string        `json:"key_file" yaml:"key_file"`
	Secret            string        `json:"secret" yaml:"secret"`                     // Token 签名密钥
	TokenExpiration   time.Duration `json:"token_expiration" yaml:"token_expiration"` // Token 过期时间
	TokenIssuer       string        `json:"token_issuer" yaml:"token_issuer"`         // Token 签发人
}

// SlaveConfig Slave 配置
type SlaveConfig struct {
	SlaveID         string            `json:"slave_id" yaml:"slave_id"`
	MasterAddr      string            `json:"master_addr" yaml:"master_addr"`
	GRPCPort        int32             `json:"grpc_port" yaml:"grpc_port"`
	Region          string            `json:"region" yaml:"region"`
	Labels          map[string]string `json:"labels" yaml:"labels"`
	MaxConcurrency  int               `json:"max_concurrency" yaml:"max_concurrency"` // 最大并发任务数
	CanReuse        bool              `json:"can_reuse" yaml:"can_reuse"`             // 是否允许复用
	EnableTLS       bool              `json:"enable_tls" yaml:"enable_tls"`
	CertFile        string            `json:"cert_file" yaml:"cert_file"`
	ReportBuffer    int               `json:"report_buffer" yaml:"report_buffer"`
	ReportInterval  time.Duration     `json:"report_interval" yaml:"report_interval"`
	ResourceMonitor bool              `json:"resource_monitor" yaml:"resource_monitor"` // 是否启用资源监控
}
