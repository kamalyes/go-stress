/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:00:00
 * @FilePath: \go-stress\types\protocol.go
 * @Description: 协议相关类型定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package types

import (
	"context"
	"time"
)

// ProtocolType 协议类型
type ProtocolType string

const (
	ProtocolHTTP      ProtocolType = "http"
	ProtocolGRPC      ProtocolType = "grpc"
	ProtocolWebSocket ProtocolType = "websocket"
)

// String 返回协议类型的字符串表示
func (p ProtocolType) String() string {
	return string(p)
}

// Request 通用请求结构
type Request struct {
	URL      string            `json:"url" yaml:"url"`
	Method   string            `json:"method" yaml:"method"`
	Headers  map[string]string `json:"headers" yaml:"headers"`
	Body     string            `json:"body" yaml:"body"`
	Metadata map[string]any    `json:"metadata" yaml:"metadata"` // 协议特定数据
}

// Response 通用响应结构
type Response struct {
	StatusCode     int                  `json:"status_code"`
	Headers        map[string]string    `json:"headers"`
	Body           []byte               `json:"body"`
	RequestURL     string               `json:"request_url"`
	RequestMethod  string               `json:"request_method"`
	RequestHeaders map[string]string    `json:"request_headers"`
	RequestBody    string               `json:"request_body"`
	RequestQuery   string               `json:"request_query"`
	Duration       time.Duration        `json:"duration"`
	Error          error                `json:"error,omitempty"`
	Verifications  []VerificationResult `json:"verifications,omitempty"`
}

// Client 协议客户端接口
type Client interface {
	// Connect 建立连接
	Connect(ctx context.Context) error

	// Send 发送请求
	Send(ctx context.Context, req *Request) (*Response, error)

	// Close 关闭连接
	Close() error

	// Type 返回协议类型
	Type() ProtocolType
}
