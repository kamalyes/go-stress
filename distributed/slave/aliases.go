/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-stress\distributed\slave\aliases.go
 * @Description: 类型别名统一管理
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package slave

import "github.com/kamalyes/go-stress/types"

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
