/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:00:00
 * @FilePath: \go-stress\statistics\aliases.go
 * @Description: statistics 模块类型别名
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"github.com/kamalyes/go-stress/types"
)

// 类型别名 - 从 types 包导入
type (
	RequestResult      = types.RequestResult
	Statistics         = types.Statistics
	VerificationResult = types.VerificationResult
	RunMode            = types.RunMode

	// 存储相关
	StorageMode = types.StorageMode
)

// 常量别名
const (
	// 存储模式
	StorageModeMemory = types.StorageModeMemory
	StorageModeSQLite = types.StorageModeSQLite
	StorageModeBadger = types.StorageModeBadger
)
