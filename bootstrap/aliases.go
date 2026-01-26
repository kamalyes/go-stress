/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 10:33:49
 * @FilePath: \go-stress\bootstrap\aliases.go
 * @Description: 类型别名统一管理
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bootstrap

import "github.com/kamalyes/go-stress/types"

// 运行模式别名
type (
	RunMode = types.RunMode
)

// 运行模式常量
const (
	RunModeStandaloneCLI  = types.RunModeStandaloneCLI
	RunModeStandaloneFile = types.RunModeStandaloneFile
	RunModeMaster         = types.RunModeMaster
	RunModeSlave          = types.RunModeSlave
)

// 存储模式别名
type (
	StorageMode = types.StorageMode
)

// 存储模式常量
const (
	StorageModeMemory = types.StorageModeMemory
	StorageModeSQLite = types.StorageModeSQLite
	StorageModeBadger = types.StorageModeBadger
)

// 协议类型别名
type (
	ProtocolType = types.ProtocolType
)

// 协议类型常量
const (
	ProtocolHTTP      = types.ProtocolHTTP
	ProtocolGRPC      = types.ProtocolGRPC
	ProtocolWebSocket = types.ProtocolWebSocket
)
