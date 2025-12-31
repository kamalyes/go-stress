/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:00:00
 * @FilePath: \go-stress\executor\alias.go
 * @Description: executor 模块类型别名
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"github.com/kamalyes/go-stress/types"
)

// 类型别名 - 从 types 包导入
type (
	// 协议相关
	Client       = types.Client
	Request      = types.Request
	Response     = types.Response
	ProtocolType = types.ProtocolType

	// 执行器相关
	Result         = types.Result
	ClientFactory  = types.ClientFactory
	RequestHandler = types.RequestHandler
	Middleware     = types.Middleware

	// 统计相关
	RequestResult = types.RequestResult
	ExtractorType = types.ExtractorType
)

// 常量别名
const (
	ProtocolHTTP      = types.ProtocolHTTP
	ProtocolGRPC      = types.ProtocolGRPC
	ProtocolWebSocket = types.ProtocolWebSocket

	ExtractorTypeJSONPath = types.ExtractorTypeJSONPath
	ExtractorTypeRegex    = types.ExtractorTypeRegex
	ExtractorTypeHeader   = types.ExtractorTypeHeader
)
