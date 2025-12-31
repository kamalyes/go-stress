/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:00:00
 * @FilePath: \go-stress\types\executor.go
 * @Description: 执行器相关类型定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package types

import (
	"context"
	"time"
)

// Result 单次请求结果
type Result struct {
	Success   bool          // 是否成功
	Duration  time.Duration // 请求耗时
	ErrorCode int           // 错误码
	Error     error         // 错误信息
	SeqID     uint64        // 序列号
}

// ClientFactory 客户端工厂函数
type ClientFactory func() (Client, error)

// RequestHandler 请求处理函数
type RequestHandler func(ctx context.Context, req *Request) (*Response, error)

// Middleware 中间件函数
type Middleware func(next RequestHandler) RequestHandler
