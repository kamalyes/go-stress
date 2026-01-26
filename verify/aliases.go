/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 11:25:25
 * @FilePath: \go-stress\verify\aliases.go
 * @Description: verify 模块类型别名
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package verify

import (
	"github.com/kamalyes/go-stress/types"
)

// 类型别名 - 从 types 包导入
type (
	Response           = types.Response
	VerifyType         = types.VerifyType
	VerificationResult = types.VerificationResult
)

// 常量别名
const (
	VerifyTypeStatusCode   = types.VerifyTypeStatusCode
	VerifyTypeJSONPath     = types.VerifyTypeJSONPath
	VerifyTypeContains     = types.VerifyTypeContains
	VerifyTypeRegex        = types.VerifyTypeRegex
	VerifyTypeJSONSchema   = types.VerifyTypeJSONSchema
	VerifyTypeJSONValid    = types.VerifyTypeJSONValid
	VerifyTypeHeader       = types.VerifyTypeHeader
	VerifyTypeResponseTime = types.VerifyTypeResponseTime
	VerifyTypeResponseSize = types.VerifyTypeResponseSize
	VerifyTypeEmail        = types.VerifyTypeEmail
	VerifyTypeIP           = types.VerifyTypeIP
	VerifyTypeURL          = types.VerifyTypeURL
	VerifyTypeUUID         = types.VerifyTypeUUID
	VerifyTypeBase64       = types.VerifyTypeBase64
	VerifyTypeLength       = types.VerifyTypeLength
	VerifyTypePrefix       = types.VerifyTypePrefix
	VerifyTypeSuffix       = types.VerifyTypeSuffix
	VerifyTypeEmpty        = types.VerifyTypeEmpty
	VerifyTypeNotEmpty     = types.VerifyTypeNotEmpty
)

// 函数别名
var NewVerificationResultFromCompare = types.NewVerificationResultFromCompare
