/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-stress\types\runtime.go
 * @Description: 运行时相关类型定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package types

// RunMode 运行模式
type RunMode string

const (
	RunModeMaster         RunMode = "master"
	RunModeSlave          RunMode = "slave"
	RunModeStandaloneCLI  RunMode = "cli"  // 独立模式
	RunModeStandaloneFile RunMode = "file" // 独立文件模式
)

// RunMode 实现 flag.Value 接口
func (s *RunMode) String() string {
	if s == nil {
		return string(RunModeStandaloneCLI)
	}
	return string(*s)
}

func (s *RunMode) Set(value string) error {
	*s = RunMode(value)
	return nil
}

// StorageMode 存储模式
type StorageMode string

const (
	// StorageModeMemory 内存模式 - 数据存储在内存中，速度快但程序退出后丢失
	StorageModeMemory StorageMode = "memory"

	// StorageModeSQLite 文件模式 - 数据持久化到SQLite，支持海量数据
	StorageModeSQLite StorageMode = "sqlite"

	// StorageModeBadger BadgerDB 模式 - 高性能 LSM-Tree 存储，纯 Go 实现
	StorageModeBadger StorageMode = "badger"
)

// StorageMode 实现 flag.Value 接口
func (s *StorageMode) String() string {
	if s == nil {
		return string(StorageModeMemory)
	}
	return string(*s)
}

func (s *StorageMode) Set(value string) error {
	*s = StorageMode(value)
	return nil
}
