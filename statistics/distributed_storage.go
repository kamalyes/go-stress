/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-24 01:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 01:10:00
 * @FilePath: \go-stress\statistics\distributed_storage.go
 * @Description: 分布式存储管理器 - Master汇总各节点数据
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"sync"

	"github.com/kamalyes/go-logger"
)

// DistributedStorage 分布式存储管理器（Master节点使用）
type DistributedStorage struct {
	localStorage *DetailStorage            // 本地存储
	remoteNodes  map[string]*DetailStorage // 远程节点存储映射 (nodeID -> storage)
	mu           sync.RWMutex
	logger       logger.ILogger
}

// NewDistributedStorage 创建分布式存储管理器
func NewDistributedStorage(localDBPath, nodeID string, log logger.ILogger) (*DistributedStorage, error) {
	localStorage, err := NewDetailStorage(localDBPath, nodeID, log)
	if err != nil {
		return nil, err
	}

	return &DistributedStorage{
		localStorage: localStorage,
		remoteNodes:  make(map[string]*DetailStorage),
		logger:       log,
	}, nil
}

// RegisterRemoteNode 注册远程节点（为分布式汇总预留接口）
func (ds *DistributedStorage) RegisterRemoteNode(nodeID, dbPath string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// 注意：实际分布式场景下，这里应该是通过gRPC获取远程数据
	// 当前实现是为了演示，假设可以访问远程节点的DB文件
	storage, err := NewDetailStorage(dbPath, nodeID, ds.logger)
	if err != nil {
		return err
	}

	ds.remoteNodes[nodeID] = storage
	return nil
}

// WriteLocal 写入本地存储
func (ds *DistributedStorage) WriteLocal(detail *RequestDetail) {
	ds.localStorage.Write(detail)
}

// QueryAll 查询所有节点的详情（汇总）
func (ds *DistributedStorage) QueryAll(offset, limit int, statusFilter StatusFilter) ([]*RequestDetail, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var allDetails []*RequestDetail

	// 查询本地（使用实际的offset和limit，避免一次性加载过多数据）
	localDetails, err := ds.localStorage.Query(offset, limit, statusFilter)
	if err == nil {
		allDetails = append(allDetails, localDetails...)
	}

	// 查询所有远程节点（每个节点查询相同的offset/limit范围）
	for _, remoteStorage := range ds.remoteNodes {
		remoteDetails, err := remoteStorage.Query(offset, limit, statusFilter)
		if err == nil {
			allDetails = append(allDetails, remoteDetails...)
		}
	}

	// 注意：这里简化了处理，实际应该按timestamp或ID全局排序后再分页
	// 当前实现是每个节点独立分页，然后合并结果
	return allDetails, nil
}

// CountAll 统计所有节点的总数
func (ds *DistributedStorage) CountAll(statusFilter StatusFilter) (int, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	total := 0

	// 统计本地
	localCount, err := ds.localStorage.Count(statusFilter)
	if err == nil {
		total += localCount
	}

	// 统计所有远程节点
	for _, remoteStorage := range ds.remoteNodes {
		remoteCount, err := remoteStorage.Count(statusFilter)
		if err == nil {
			total += remoteCount
		}
	}

	return total, nil
}

// Close 关闭所有存储
func (ds *DistributedStorage) Close() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.localStorage != nil {
		ds.localStorage.Close()
	}

	for _, remoteStorage := range ds.remoteNodes {
		remoteStorage.Close()
	}

	return nil
}
