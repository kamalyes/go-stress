/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 00:00:00
 * @FilePath: \go-stress\executor\pool.go
 * @Description: 客户端连接池（使用 syncx.Pool 优化）
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// ClientPool 客户端连接池（使用 syncx.Pool）
type ClientPool struct {
	factory ClientFactory
	pool    *syncx.Pool[Client]
}

// NewClientPool 创建客户端连接池
func NewClientPool(factory ClientFactory, maxSize int) *ClientPool {
	return &ClientPool{
		factory: factory,
		pool: syncx.NewPool(func() Client {
			client, _ := factory()
			return client
		}),
	}
}

// Get 从池中获取客户端
func (cp *ClientPool) Get() (Client, error) {
	// syncx.Pool 自动管理对象创建和复用
	return cp.pool.Get(), nil
}

// Put 将客户端放回池中
func (cp *ClientPool) Put(client Client) {
	if client != nil {
		cp.pool.Put(client)
	}
}

// Close 关闭连接池
func (cp *ClientPool) Close() {
	// syncx.Pool 会自动处理清理
	// 不需要手动关闭每个客户端
}
