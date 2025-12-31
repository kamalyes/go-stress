/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 09:40:00
 * @FilePath: \go-stress\executor\controller.go
 * @Description: 压测控制器接口
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import "time"

// Controller 压测控制器接口
type Controller interface {
	// IsPaused 检查是否暂停
	IsPaused() bool
	// IsStopped 检查是否停止
	IsStopped() bool
}

// NoOpController 空操作控制器（默认实现）
type NoOpController struct{}

func (n *NoOpController) IsPaused() bool  { return false }
func (n *NoOpController) IsStopped() bool { return false }

// WaitWhilePaused 等待直到不再暂停或停止
func WaitWhilePaused(ctrl Controller) bool {
	if ctrl == nil {
		return false
	}

	for ctrl.IsPaused() {
		if ctrl.IsStopped() {
			return true // 已停止
		}
		time.Sleep(100 * time.Millisecond)
	}

	return ctrl.IsStopped()
}
