/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:00:00
 * @FilePath: \go-stress\executor\pool.go
 * @Description: 客户端连接池
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"fmt"
	"sync"
)

// ClientPool 客户端连接池
type ClientPool struct {
	factory ClientFactory
	pool    chan Client
	maxSize int
	created int
	mu      sync.Mutex
}

// NewClientPool 创建客户端连接池
func NewClientPool(factory ClientFactory, maxSize int) *ClientPool {
	return &ClientPool{
		factory: factory,
		pool:    make(chan Client, maxSize),
		maxSize: maxSize,
		created: 0,
	}
}

// Get 从池中获取客户端
func (cp *ClientPool) Get() (Client, error) {
	select {
	case client := <-cp.pool:
		// 从池中获取
		return client, nil
	default:
		// 池中没有，创建新的
		cp.mu.Lock()
		defer cp.mu.Unlock()

		if cp.created < cp.maxSize {
			client, err := cp.factory()
			if err != nil {
				return nil, fmt.Errorf("创建客户端失败: %w", err)
			}
			cp.created++
			return client, nil
		}

		// 等待可用的客户端
		return <-cp.pool, nil
	}
}

// Put 将客户端放回池中
func (cp *ClientPool) Put(client Client) {
	select {
	case cp.pool <- client:
		// 成功放回池中
	default:
		// 池已满，关闭客户端
		client.Close()
		cp.mu.Lock()
		cp.created--
		cp.mu.Unlock()
	}
}

// Close 关闭连接池
func (cp *ClientPool) Close() {
	close(cp.pool)
	for client := range cp.pool {
		client.Close()
	}
}
