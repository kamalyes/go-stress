/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\master\queue.go
 * @Description: 任务队列管理
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"context"
	"fmt"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// TaskQueue 任务队列
type TaskQueue struct {
	mu           *syncx.RWLock           // 使用 syncx.RWLock 替代 sync.RWMutex
	pending      []*common.Task          // 待执行任务队列
	pendingIndex map[string]int          // taskID -> pending 数组索引，加速查找
	running      map[string]*common.Task // 运行中任务
	complete     map[string]*common.Task // 已完成任务
	failed       map[string]*common.Task // 失败任务
	stopped      map[string]*common.Task // 已停止任务
	allTasks     map[string]*common.Task // 所有任务的统一索引 (O1 查询)
	splitter     TaskSplitter
	logger       logger.ILogger
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(splitter TaskSplitter, log logger.ILogger) *TaskQueue {
	return &TaskQueue{
		mu:           syncx.NewRWLock(),
		pending:      make([]*common.Task, 0),
		pendingIndex: make(map[string]int),
		running:      make(map[string]*common.Task),
		complete:     make(map[string]*common.Task),
		failed:       make(map[string]*common.Task),
		stopped:      make(map[string]*common.Task),
		allTasks:     make(map[string]*common.Task),
		splitter:     splitter,
		logger:       log,
	}
}

// Submit 提交任务
func (tq *TaskQueue) Submit(task *common.Task) error {
	return syncx.WithLockReturnValue(tq.mu, func() error {
		task.State = common.TaskStatePending
		task.CreatedAt = time.Now()
		tq.pendingIndex[task.ID] = len(tq.pending)
		tq.pending = append(tq.pending, task)
		tq.allTasks[task.ID] = task

		tq.logger.InfoKV("Task submitted", "task_id", task.ID, "workers", task.TotalWorkers)
		return nil
	})
}

// GetPending 获取待处理任务
func (tq *TaskQueue) GetPending() []*common.Task {
	return syncx.WithRLockReturnValue(tq.mu, func() []*common.Task {
		tasks := make([]*common.Task, len(tq.pending))
		copy(tasks, tq.pending)
		return tasks
	})
}

// PopPending 弹出第一个待处理任务
func (tq *TaskQueue) PopPending() *common.Task {
	return syncx.WithLockReturnValue(tq.mu, func() *common.Task {
		if len(tq.pending) == 0 {
			return nil
		}

		task := tq.pending[0]
		tq.pending = tq.pending[1:]
		delete(tq.pendingIndex, task.ID)

		// 更新索引：所有剩余任务的索引 -1
		for i, t := range tq.pending {
			tq.pendingIndex[t.ID] = i
		}
		return task
	})
}

// MoveToRunning 将任务移动到运行中
func (tq *TaskQueue) MoveToRunning(taskID string) error {
	return syncx.WithLockReturnValue(tq.mu, func() error {
		// 使用索引快速查找
		i, exists := tq.pendingIndex[taskID]
		if !exists {
			return fmt.Errorf("task %s not found in pending queue", taskID)
		}

		task := tq.pending[i]
		task.State = common.TaskStateRunning
		task.StartedAt = time.Now()
		tq.running[taskID] = task

		// 从 pending 中移除
		tq.pending = append(tq.pending[:i], tq.pending[i+1:]...)
		delete(tq.pendingIndex, taskID)

		// 更新后续任务的索引
		for j := i; j < len(tq.pending); j++ {
			tq.pendingIndex[tq.pending[j].ID] = j
		}

		tq.logger.InfoKV("Task started", "task_id", taskID)
		return nil
	})
}

// MoveToComplete 将任务移动到完成
func (tq *TaskQueue) MoveToComplete(taskID string) error {
	return syncx.WithLockReturnValue(tq.mu, func() error {
		// 检查任务是否已经处于终态
		if _, exists := tq.complete[taskID]; exists {
			tq.logger.WarnKV("Task is already completed", "task_id", taskID)
			return nil // 幂等操作，不报错
		}
		if _, exists := tq.failed[taskID]; exists {
			return fmt.Errorf("task %s is already failed, cannot mark as completed", taskID)
		}
		if _, exists := tq.stopped[taskID]; exists {
			return fmt.Errorf("task %s is already stopped, cannot mark as complete", taskID)
		}

		task, exists := tq.running[taskID]
		if !exists {
			return fmt.Errorf("task %s not found in running queue", taskID)
		}

		task.State = common.TaskStateCompleted
		task.CompletedAt = time.Now()
		tq.complete[taskID] = task
		delete(tq.running, taskID)

		tq.logger.InfoKV("Task completed",
			"task_id", taskID,
			"duration", task.CompletedAt.Sub(task.StartedAt))
		return nil
	})
}

// MoveToFailed 将任务移动到失败
func (tq *TaskQueue) MoveToFailed(taskID string, reason string) error {
	return syncx.WithLockReturnValue(tq.mu, func() error {
		// 检查任务是否已经处于终态
		if _, exists := tq.complete[taskID]; exists {
			return fmt.Errorf("task %s is already completed, cannot mark as failed", taskID)
		}
		if _, exists := tq.failed[taskID]; exists {
			tq.logger.WarnKV("Task is already failed", "task_id", taskID, "reason", reason)
			return nil // 幂等操作，不报错
		}
		if _, exists := tq.stopped[taskID]; exists {
			return fmt.Errorf("task %s is already stopped, cannot mark as failed", taskID)
		}

		task, exists := tq.running[taskID]
		if !exists {
			return fmt.Errorf("task %s not found in running queue", taskID)
		}

		task.State = common.TaskStateFailed
		task.CompletedAt = time.Now()
		if task.Metadata == nil {
			task.Metadata = make(map[string]string)
		}
		task.Metadata["failure_reason"] = reason
		tq.failed[taskID] = task
		delete(tq.running, taskID)

		tq.logger.ErrorKV("Task failed", "task_id", taskID, "reason", reason)
		return nil
	})
}

// Get 获取任务 (O(1))
func (tq *TaskQueue) Get(taskID string) (*common.Task, bool) {
	return syncx.WithRLockReturnWithE(tq.mu, func() (*common.Task, bool) {
		task, exists := tq.allTasks[taskID]
		return task, exists
	})
}

// GetRunning 获取所有运行中的任务
func (tq *TaskQueue) GetRunning() []*common.Task {
	return syncx.WithRLockReturnValue(tq.mu, func() []*common.Task {
		tasks := make([]*common.Task, 0, len(tq.running))
		for _, task := range tq.running {
			tasks = append(tasks, task)
		}
		return tasks
	})
}

// GetAllTasks 获取所有任务（O(n) 复制但只遍历一次）
func (tq *TaskQueue) GetAllTasks() []*common.Task {
	return syncx.WithRLockReturnValue(tq.mu, func() []*common.Task {
		tasks := make([]*common.Task, 0, len(tq.allTasks))
		for _, task := range tq.allTasks {
			tasks = append(tasks, task)
		}
		return tasks
	})
}

// Split 分片任务
func (tq *TaskQueue) Split(task *common.Task, slaves []*common.SlaveInfo) ([]*common.SubTask, error) {
	if tq.splitter == nil {
		return nil, fmt.Errorf("task splitter not set")
	}
	return tq.splitter.Split(task, slaves)
}

// Cancel 取消任务
func (tq *TaskQueue) Cancel(taskID string) error {
	return syncx.WithLockReturnValue(tq.mu, func() error {
		// 检查任务是否已经处于终态（不允许再停止）
		if _, exists := tq.complete[taskID]; exists {
			return fmt.Errorf("task %s is already completed, cannot stop", taskID)
		}
		if _, exists := tq.failed[taskID]; exists {
			return fmt.Errorf("task %s is already failed, cannot stop", taskID)
		}
		if _, exists := tq.stopped[taskID]; exists {
			return fmt.Errorf("task %s is already stopped", taskID)
		}

		// 使用索引快速查找 pending
		if i, exists := tq.pendingIndex[taskID]; exists {
			task := tq.pending[i]
			task.State = common.TaskStateStopped
			task.CompletedAt = time.Now()

			// 移动到 stopped 队列
			tq.stopped[taskID] = task

			// 从 pending 中移除
			tq.pending = append(tq.pending[:i], tq.pending[i+1:]...)
			delete(tq.pendingIndex, taskID)

			// 更新后续任务的索引
			for j := i; j < len(tq.pending); j++ {
				tq.pendingIndex[tq.pending[j].ID] = j
			}

			tq.logger.InfoKV("Task cancelled from pending", "task_id", taskID)
			return nil
		}

		// 从 running 中移除
		if task, exists := tq.running[taskID]; exists {
			task.State = common.TaskStateStopped
			task.CompletedAt = time.Now()

			// 移动到 stopped 队列
			tq.stopped[taskID] = task
			delete(tq.running, taskID)

			tq.logger.InfoKV("Task stopped from running", "task_id", taskID)
			return nil
		}

		return fmt.Errorf("task %s not found in pending or running", taskID)
	})
}

// Stats 获取队列统计
func (tq *TaskQueue) Stats() map[string]int {
	return syncx.WithRLockReturnValue(tq.mu, func() map[string]int {
		return map[string]int{
			"pending":  len(tq.pending),
			"running":  len(tq.running),
			"complete": len(tq.complete),
			"failed":   len(tq.failed),
			"stopped":  len(tq.stopped),
		}
	})
}

// Clean 清理已完成的任务
func (tq *TaskQueue) Clean(maxAge time.Duration) {
	syncx.WithLock(tq.mu, func() {
		now := time.Now()

		// 清理完成的任务
		for id, task := range tq.complete {
			if now.Sub(task.CompletedAt) > maxAge {
				delete(tq.complete, id)
				delete(tq.allTasks, id)
			}
		}

		// 清理失败的任务
		for id, task := range tq.failed {
			if now.Sub(task.CompletedAt) > maxAge {
				delete(tq.failed, id)
				delete(tq.allTasks, id)
			}
		}
	})
}

// StartCleaner 启动清理器
func (tq *TaskQueue) StartCleaner(ctx context.Context, interval, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tq.Clean(maxAge)
		}
	}
}
