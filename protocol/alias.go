/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-31 13:17:06
 * @FilePath: \go-stress\protocol\alias.go
 * @Description: protocol 模块类型别名
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package protocol

import (
	"github.com/kamalyes/go-stress/types"
)

// 类型别名 - 从 types 包导入
type (
	Client             = types.Client
	Request            = types.Request
	Response           = types.Response
	ProtocolType       = types.ProtocolType
	VerifyType         = types.VerifyType
	VerificationResult = types.VerificationResult
)

// 常量别名
const (
	ProtocolHTTP      = types.ProtocolHTTP
	ProtocolGRPC      = types.ProtocolGRPC
	ProtocolWebSocket = types.ProtocolWebSocket

	// 基础验证类型
	VerifyTypeStatusCode = types.VerifyTypeStatusCode
	VerifyTypeJSONPath   = types.VerifyTypeJSONPath
	VerifyTypeContains   = types.VerifyTypeContains
	VerifyTypeRegex      = types.VerifyTypeRegex
	VerifyTypeCustom     = types.VerifyTypeCustom
)

type ExpectOperator = types.ExpectOperator

const (
	// 操作符类型
	OpEQ          = types.OpEQ
	OpNE          = types.OpNE
	OpGT          = types.OpGT
	OpGTE         = types.OpGTE
	OpLT          = types.OpLT
	OpLTE         = types.OpLTE
	OpContains    = types.OpContains
	OpNotContains = types.OpNotContains
	OpHasPrefix   = types.OpHasPrefix
	OpHasSuffix   = types.OpHasSuffix
	OpEmpty       = types.OpEmpty
	OpNotEmpty    = types.OpNotEmpty
	OpRegex       = types.OpRegex

	// 符号形式的操作符
	OpEqual              = types.OpEqual
	OpDoubleEqual        = types.OpDoubleEqual
	OpNotEqual           = types.OpNotEqual
	OpGreaterThan        = types.OpGreaterThan
	OpGreaterThanOrEqual = types.OpGreaterThanOrEqual
	OpLessThan           = types.OpLessThan
	OpLessThanOrEqual    = types.OpLessThanOrEqual
	OpRegexp             = types.OpRegexp
)
