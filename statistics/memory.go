/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-24 15:30:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 16:00:00
 * @FilePath: \go-stress\statistics\memory.go
 * @Description: 内存存储层 - 高速无限制存储（实现 DetailStorageInterface）
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// MemoryStorage 内存存储（按状态分类存储，高性能版本）
type MemoryStorage struct {
	// 按状态分类存储，提升查询性能
	allDetails     []*RequestDetail // 全部记录（按时间倒序）
	successDetails []*RequestDetail // 成功记录
	failedDetails  []*RequestDetail // 失败记录
	skippedDetails []*RequestDetail // 跳过记录

	mu     *syncx.RWLock
	nodeID string // 节点ID
	logger logger.ILogger
	closed bool

	// 实时计数器（O(1) 查询）
	totalCount   *syncx.Uint64
	successCount *syncx.Uint64
	failedCount  *syncx.Uint64
	skippedCount *syncx.Uint64
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage(nodeID string, log logger.ILogger) *MemoryStorage {
	log.Infof("💾 内存存储已启用 (节点: %s, 按状态分类存储)", nodeID)

	return &MemoryStorage{
		allDetails:     make([]*RequestDetail, 0, 10000),
		successDetails: make([]*RequestDetail, 0, 8000),
		failedDetails:  make([]*RequestDetail, 0, 1000),
		skippedDetails: make([]*RequestDetail, 0, 1000),
		mu:             syncx.NewRWLock(),
		nodeID:         nodeID,
		logger:         log,
		closed:         false,
		totalCount:     syncx.NewUint64(0),
		successCount:   syncx.NewUint64(0),
		failedCount:    syncx.NewUint64(0),
		skippedCount:   syncx.NewUint64(0),
	}
}

// Write 写入详情（按状态分类存储，实现 DetailStorageInterface）
func (m *MemoryStorage) Write(detail *RequestDetail) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	// 写入全部记录（插入到头部，保持倒序）
	m.allDetails = append([]*RequestDetail{detail}, m.allDetails...)
	m.totalCount.Add(1)

	// 根据状态分类存储
	if detail.Skipped {
		m.skippedDetails = append([]*RequestDetail{detail}, m.skippedDetails...)
		m.skippedCount.Add(1)
	} else if detail.Success {
		m.successDetails = append([]*RequestDetail{detail}, m.successDetails...)
		m.successCount.Add(1)
	} else {
		m.failedDetails = append([]*RequestDetail{detail}, m.failedDetails...)
		m.failedCount.Add(1)
	}

	// 每写入10000条输出一次统计
	count := m.totalCount.Load()
	if count%10000 == 0 {
		m.logger.Debugf("📊 内存已存储 %d 条记录 (成功:%d, 失败:%d, 跳过:%d)",
			count, m.successCount.Load(), m.failedCount.Load(), m.skippedCount.Load())
	}
}

// Query 查询详情（O(1) 定位 + O(limit) 复制，高性能）
func (m *MemoryStorage) Query(offset, limit int, statusFilter StatusFilter) ([]*RequestDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 根据状态选择对应的切片（O(1)）
	var source []*RequestDetail
	switch statusFilter {
	case StatusFilterSuccess:
		source = m.successDetails
	case StatusFilterFailed:
		source = m.failedDetails
	case StatusFilterSkipped:
		source = m.skippedDetails
	case StatusFilterAll:
		source = m.allDetails
	default:
		source = m.allDetails
	}

	// 分页（O(1) 切片操作）
	if offset >= len(source) {
		return []*RequestDetail{}, nil
	}

	end := offset + limit
	if end > len(source) {
		end = len(source)
	}

	return source[offset:end], nil
}

// Count 统计总数（O(1) 原子读取，极高性能）
func (m *MemoryStorage) Count(statusFilter StatusFilter) (int, error) {
	// 直接从原子计数器读取，无需加锁遍历（O(1)）
	switch statusFilter {
	case StatusFilterSuccess:
		return int(m.successCount.Load()), nil
	case StatusFilterFailed:
		return int(m.failedCount.Load()), nil
	case StatusFilterSkipped:
		return int(m.skippedCount.Load()), nil
	case StatusFilterAll:
		return int(m.totalCount.Load()), nil
	default:
		return int(m.totalCount.Load()), nil
	}
}

// Close 关闭存储（实现 DetailStorageInterface）
func (m *MemoryStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	// 输出最终统计
	total := m.totalCount.Load()
	success := m.successCount.Load()
	failed := m.failedCount.Load()
	skipped := m.skippedCount.Load()

	m.logger.Infof("✅ 内存存储已关闭")
	m.logger.Infof("   📝 总记录: %d 条 (成功:%d, 失败:%d, 跳过:%d)", total, success, failed, skipped)
	m.logger.Infof("   💾 内存占用: 约 %.2f MB", float64(total*500)/1024/1024) // 粗略估算

	return nil
}

// GetNodeID 获取节点ID（实现 DetailStorageInterface）
func (m *MemoryStorage) GetNodeID() string {
	return m.nodeID
}

// GetStats 获取存储统计信息
func (m *MemoryStorage) GetStats() (total, success, failed, skipped uint64) {
	return m.totalCount.Load(),
		m.successCount.Load(),
		m.failedCount.Load(),
		m.skippedCount.Load()
}
