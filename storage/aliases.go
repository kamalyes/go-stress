/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-stress\storage\aliases.go
 * @Description: storage 模块类型别名
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package storage

import (
	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/types"
)

// 类型别名 - 从 types 包导入
type (
	RequestResult      = types.RequestResult
	Statistics         = types.Statistics
	VerificationResult = types.VerificationResult
	StorageMode        = types.StorageMode
)

// 常量别名
const (
	// 存储模式
	StorageModeMemory = types.StorageModeMemory
	StorageModeSQLite = types.StorageModeSQLite
	StorageModeBadger = types.StorageModeBadger
)

// 外部类型别名
type ILogger = logger.ILogger
