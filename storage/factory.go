/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 10:37:54
 * @FilePath: \go-stress\storage\storage_factory.go
 * @Description: 存储工厂 - 统一创建不同类型的存储适配器
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package storage

import (
	"fmt"

	"github.com/kamalyes/go-logger"
)

// StorageConfig 存储配置
type StorageConfig struct {
	Type   StorageMode            // 存储类型
	Path   string                 // 存储路径（sqlite/badger）
	NodeID string                 // 节点ID
	Params map[string]interface{} // 额外参数
}

// StorageFactory 存储工厂
type StorageFactory struct {
	logger logger.ILogger
}

// NewStorageFactory 创建存储工厂
func NewStorageFactory(log logger.ILogger) *StorageFactory {
	return &StorageFactory{
		logger: log,
	}
}

// CreateStorage 创建存储实例
func (f *StorageFactory) CreateStorage(config *StorageConfig) (Interface, error) {
	if config == nil {
		return nil, fmt.Errorf("存储配置不能为空")
	}

	f.logger.Infof("📦 创建存储实例: type=%s, nodeID=%s, path=%s",
		config.Type, config.NodeID, config.Path)

	switch config.Type {
	case StorageModeMemory:
		return f.createMemoryStorage(config)

	case StorageModeSQLite:
		return f.createSQLiteStorage(config)

	case StorageModeBadger:
		return f.createBadgerStorage(config)

	default:
		return nil, fmt.Errorf("不支持的存储类型: %s (支持: memory, sqlite, badger)", config.Type)
	}
}

// createMemoryStorage 创建内存存储
func (f *StorageFactory) createMemoryStorage(config *StorageConfig) (Interface, error) {
	f.logger.Info("💾 创建内存存储...")

	storage := NewMemoryStorage(config.NodeID, f.logger)

	f.logger.Infof("✅ 内存存储创建成功 (节点: %s)", config.NodeID)
	return storage, nil
}

// createSQLiteStorage 创建 SQLite 存储
func (f *StorageFactory) createSQLiteStorage(config *StorageConfig) (Interface, error) {
	if config.Path == "" {
		return nil, fmt.Errorf("SQLite 存储需要指定 path 参数")
	}

	f.logger.Infof("🗄️  创建 SQLite 存储: %s", config.Path)

	storage, err := NewDetailStorage(config.Path, config.NodeID, f.logger)
	if err != nil {
		return nil, fmt.Errorf("创建 SQLite 存储失败: %w", err)
	}

	f.logger.Infof("✅ SQLite 存储创建成功 (节点: %s, 路径: %s)", config.NodeID, config.Path)
	return storage, nil
}

// createBadgerStorage 创建 BadgerDB 存储
func (f *StorageFactory) createBadgerStorage(config *StorageConfig) (Interface, error) {
	if config.Path == "" {
		return nil, fmt.Errorf("BadgerDB 存储需要指定 path 参数")
	}

	f.logger.Infof("🗄️  创建 BadgerDB 存储: %s", config.Path)

	storage, err := NewBadgerStorage(config.Path, config.NodeID, f.logger)
	if err != nil {
		return nil, fmt.Errorf("创建 BadgerDB 存储失败: %w", err)
	}

	f.logger.Infof("✅ BadgerDB 存储创建成功 (节点: %s, 路径: %s)", config.NodeID, config.Path)
	return storage, nil
}

// ValidateConfig 验证存储配置
func (f *StorageFactory) ValidateConfig(config *StorageConfig) error {
	if config == nil {
		return fmt.Errorf("存储配置不能为空")
	}

	if config.NodeID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	switch config.Type {
	case StorageModeMemory:
		// 内存存储不需要额外验证
		return nil

	case StorageModeSQLite, StorageModeBadger:
		if config.Path == "" {
			return fmt.Errorf("%s 存储需要指定 path 参数", config.Type)
		}
		return nil

	default:
		return fmt.Errorf("不支持的存储类型: %s", config.Type)
	}
}

// GetSupportedTypes 返回支持的存储类型列表
func (f *StorageFactory) GetSupportedTypes() []StorageMode {
	return []StorageMode{
		StorageModeMemory,
		StorageModeSQLite,
		StorageModeBadger,
	}
}

// GetStorageInfo 获取存储类型信息
func (f *StorageFactory) GetStorageInfo(storageType StorageMode) map[string]interface{} {
	switch storageType {
	case StorageModeMemory:
		return map[string]interface{}{
			"type":        "memory",
			"name":        "内存存储",
			"description": "高速内存存储，适合实时压测，数据不持久化",
			"pros":        []string{"性能最高", "零配置", "实时统计"},
			"cons":        []string{"不持久化", "内存占用高", "进程重启数据丢失"},
			"use_case":    []string{"实时压测", "短期测试", "性能优先场景"},
		}

	case StorageModeSQLite:
		return map[string]interface{}{
			"type":        "sqlite",
			"name":        "SQLite 数据库",
			"description": "轻量级文件数据库，支持持久化和复杂查询",
			"pros":        []string{"持久化", "SQL 查询", "事务支持", "零配置"},
			"cons":        []string{"单机限制", "写入性能一般", "并发受限"},
			"use_case":    []string{"单机压测", "需要持久化", "复杂查询场景"},
		}

	case StorageModeBadger:
		return map[string]interface{}{
			"type":        "badger",
			"name":        "BadgerDB",
			"description": "高性能 LSM-Tree 存储，纯 Go 实现，写入性能极佳",
			"pros":        []string{"高性能写入", "纯 Go 实现", "压缩存储", "事务支持"},
			"cons":        []string{"查询灵活性低", "LSM 特性需理解", "空间放大"},
			"use_case":    []string{"高并发压测", "海量数据", "写多读少场景"},
		}

	default:
		return map[string]interface{}{
			"type":  string(storageType),
			"error": "未知的存储类型",
		}
	}
}
