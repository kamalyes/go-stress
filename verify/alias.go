/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 11:25:25
 * @FilePath: \go-stress\verify\alias.go
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
	Response   = types.Response
	VerifyType = types.VerifyType
)

// 常量别名
const (
	VerifyStatusCode = types.VerifyTypeStatusCode
	VerifyJSON       = types.VerifyTypeJSONPath
	VerifyContains   = types.VerifyTypeContains
	VerifyRegex      = types.VerifyTypeRegex
)
