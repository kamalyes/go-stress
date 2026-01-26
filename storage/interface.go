/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-stress\storage\interface.go
 * @Description: 存储接口定义
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package storage

import "github.com/kamalyes/go-stress/types"

// StatusFilter 状态过滤器枚举
type StatusFilter int

const (
	StatusFilterAll     StatusFilter = iota // 全部
	StatusFilterSuccess                     // 成功
	StatusFilterFailed                      // 失败
	StatusFilterSkipped                     // 跳过
)

// String 返回状态过滤器的字符串表示
func (s StatusFilter) String() string {
	switch s {
	case StatusFilterSuccess:
		return "success"
	case StatusFilterFailed:
		return "failed"
	case StatusFilterSkipped:
		return "skipped"
	default:
		return "all"
	}
}

// ParseStatusFilter 从字符串解析状态过滤器
func ParseStatusFilter(s string) StatusFilter {
	switch s {
	case "success":
		return StatusFilterSuccess
	case "failed":
		return StatusFilterFailed
	case "skipped":
		return StatusFilterSkipped
	default:
		return StatusFilterAll
	}
}

// Interface 存储接口（统一所有存储实现）
type Interface interface {
	// Write 写入请求详情
	Write(detail *types.RequestResult)

	// Query 分页查询请求详情（支持 nodeID 和 taskID 过滤）
	Query(offset, limit int, statusFilter StatusFilter, nodeID, taskID string) ([]*types.RequestResult, error)

	// Count 统计总数（支持 nodeID 和 taskID 过滤）
	Count(statusFilter StatusFilter, nodeID, taskID string) (int, error)

	// Close 关闭存储并释放资源
	Close() error

	// GetNodeID 获取节点ID（用于分布式场景）
	GetNodeID() string
}
