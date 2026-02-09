/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-27 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-27 09:15:28
 * @FilePath: \go-stress\distributed\master\queue_test.go
 * @Description: 任务队列测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-toolbox/pkg/idgen"
	"github.com/stretchr/testify/assert"
)

// MockTaskSplitter mock 任务分片器
type MockTaskSplitter struct{}

func (m *MockTaskSplitter) Split(task *common.Task, slaves []*common.SlaveInfo) ([]*common.SubTask, error) {
	return []*common.SubTask{}, nil
}

// 全局 ID 生成器
var idGenerator = idgen.NewSnowflakeGenerator(1, 1)

// generateTaskID 生成测试任务 ID
func generateTaskID() string {
	return idGenerator.GenerateRequestID()
}

// TestSubmitTask 测试提交任务
func TestSubmitTask(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	err := queue.Submit(task)
	assert.NoError(t, err)

	stats := queue.Stats()
	assert.Equal(t, 1, stats.Pending)

	pending := queue.GetPending()
	assert.Equal(t, 1, len(pending))
	assert.Equal(t, taskID, pending[0].ID)
}

// TestPopPending 测试弹出待处理任务
func TestPopPending(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)

	popped := queue.PopPending()
	assert.NotNil(t, popped)
	assert.Equal(t, taskID, popped.ID)

	stats := queue.Stats()
	assert.Equal(t, 0, stats.Pending)
}

// TestMoveToRunning 测试移动任务到运行中
func TestMoveToRunning(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)

	err := queue.MoveToRunning(taskID)
	assert.NoError(t, err)

	stats := queue.Stats()
	assert.Equal(t, 0, stats.Pending)
	assert.Equal(t, 1, stats.Running)

	running := queue.GetRunning()
	assert.Equal(t, 1, len(running))
	assert.Equal(t, taskID, running[0].ID)
}

// TestMoveToComplete 测试移动任务到完成
func TestMoveToComplete(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)
	queue.MoveToRunning(taskID)

	err := queue.MoveToComplete(taskID)
	assert.NoError(t, err)

	stats := queue.Stats()
	assert.Equal(t, 0, stats.Running)
	assert.Equal(t, 1, stats.Complete)

	retrieved, ok := queue.Get(taskID)
	assert.True(t, ok)
	assert.Equal(t, common.TaskStateCompleted, retrieved.State)
}

// TestMoveToFailed 测试移动任务到失败
func TestMoveToFailed(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)
	queue.MoveToRunning(taskID)

	err := queue.MoveToFailed(taskID, "network timeout")
	assert.NoError(t, err)

	stats := queue.Stats()
	assert.Equal(t, 0, stats.Running)
	assert.Equal(t, 1, stats.Failed)

	retrieved, ok := queue.Get(taskID)
	assert.True(t, ok)
	assert.Equal(t, common.TaskStateFailed, retrieved.State)
	assert.Equal(t, "network timeout", retrieved.Metadata["failure_reason"])
}

// TestCancel 测试取消任务
func TestCancel(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)

	err := queue.Cancel(taskID)
	assert.NoError(t, err)

	stats := queue.Stats()
	assert.Equal(t, 0, stats.Pending)
	assert.Equal(t, 1, stats.Stopped)

	retrieved, ok := queue.Get(taskID)
	assert.True(t, ok)
	assert.Equal(t, common.TaskStateStopped, retrieved.State)
}

// TestCancelRunningTask 测试取消运行中的任务
func TestCancelRunningTask(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)
	queue.MoveToRunning(taskID)

	err := queue.Cancel(taskID)
	assert.NoError(t, err)

	stats := queue.Stats()
	assert.Equal(t, 0, stats.Running)
	assert.Equal(t, 1, stats.Stopped)
}

// TestGetAllTasks 测试获取所有任务
func TestGetAllTasks(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	for i := 0; i < 5; i++ {
		task := &common.Task{
			ID:           generateTaskID(),
			TotalWorkers: 10,
		}
		queue.Submit(task)
	}

	allTasks := queue.GetAllTasks()
	assert.Equal(t, 5, len(allTasks))
}

// TestGetTask 测试获取单个任务
func TestGetTask(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)

	retrieved, ok := queue.Get(taskID)
	assert.True(t, ok)
	assert.Equal(t, taskID, retrieved.ID)
}

// TestClean 测试清理已完成任务
func TestClean(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)
	queue.MoveToRunning(taskID)
	queue.MoveToComplete(taskID)

	// 清理超过 0 秒前的任务（即立即清理所有已完成任务）
	queue.Clean(0 * time.Second)

	stats := queue.Stats()
	assert.Equal(t, 0, stats.Complete)

	_, ok := queue.Get(taskID)
	assert.False(t, ok)
}

// TestConcurrentOperations 测试并发操作
func TestConcurrentOperations(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	// 创建多个任务
	taskCount := 100
	for i := 1; i <= taskCount; i++ {
		task := &common.Task{
			ID:           fmt.Sprintf("task-%d", i),
			TotalWorkers: 10,
		}
		queue.Submit(task)
	}

	// 并发处理任务
	done := make(chan bool, taskCount)

	for i := 1; i <= taskCount; i++ {
		go func(taskID string) {
			queue.MoveToRunning(taskID)
			time.Sleep(10 * time.Millisecond)
			queue.MoveToComplete(taskID)
			done <- true
		}(fmt.Sprintf("task-%d", i))
	}

	// 等待所有任务完成
	for i := 0; i < taskCount; i++ {
		<-done
	}

	stats := queue.Stats()
	assert.Equal(t, taskCount, stats.Complete)
	assert.Equal(t, 0, stats.Running)
}

// TestStartCleaner 测试启动清理器
func TestStartCleaner(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)
	queue.MoveToRunning(taskID)
	queue.MoveToComplete(taskID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 启动清理器，每 100ms 清理一次，清理 0 秒前的任务（即立即清理）
	go queue.StartCleaner(ctx, 100*time.Millisecond, 0*time.Second)

	// 等待清理器运行
	time.Sleep(150 * time.Millisecond)

	stats := queue.Stats()
	assert.Equal(t, 0, stats.Complete)
}

// TestTaskStateTransitions 测试任务状态转换
func TestTaskStateTransitions(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	// Submit -> Pending
	queue.Submit(task)
	retrieved, _ := queue.Get(taskID)
	assert.Equal(t, common.TaskStatePending, retrieved.State)

	// Pending -> Running
	queue.MoveToRunning(taskID)
	retrieved, _ = queue.Get(taskID)
	assert.Equal(t, common.TaskStateRunning, retrieved.State)

	// Running -> Completed
	queue.MoveToComplete(taskID)
	retrieved, _ = queue.Get(taskID)
	assert.Equal(t, common.TaskStateCompleted, retrieved.State)
}

// TestMultipleTasks 测试多个任务的操作
func TestMultipleTasks(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	// 提交三个任务
	taskID1 := generateTaskID()
	taskID2 := generateTaskID()
	taskID3 := generateTaskID()

	for _, taskID := range []string{taskID1, taskID2, taskID3} {
		task := &common.Task{
			ID:           taskID,
			TotalWorkers: 10,
		}
		queue.Submit(task)
	}

	// 启动第一个任务
	queue.MoveToRunning(taskID1)
	queue.MoveToComplete(taskID1)

	// 启动第二个任务
	queue.MoveToRunning(taskID2)
	queue.MoveToFailed(taskID2, "error")

	// 取消第三个任务
	queue.Cancel(taskID3)

	stats := queue.Stats()
	assert.Equal(t, 1, stats.Complete)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 1, stats.Stopped)
}

// TestEmptyQueue 测试空队列操作
func TestEmptyQueue(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	// 从空队列弹出
	popped := queue.PopPending()
	assert.Nil(t, popped)

	// 从空队列获取
	pending := queue.GetPending()
	assert.Equal(t, 0, len(pending))

	// 获取不存在的任务
	_, ok := queue.Get("non-existent")
	assert.False(t, ok)
}

// TestErrorCases 测试错误情况
func TestErrorCases(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)
	queue.MoveToRunning(taskID)
	queue.MoveToComplete(taskID)

	// 尝试将已完成的任务移动到运行中
	err := queue.MoveToRunning(taskID)
	assert.Error(t, err)

	// 尝试取消已完成的任务
	err = queue.Cancel(taskID)
	assert.Error(t, err)
}

// ============ 状态机测试 ============

// TestStateMachineBasicTransitions 测试状态机基本状态转换
func TestStateMachineBasicTransitions(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	// 提交任务（Pending 状态）
	err := queue.Submit(task)
	assert.NoError(t, err)

	// 检查初始状态机已创建
	sm := queue.getTaskStateMachine(taskID)
	assert.NotNil(t, sm)

	// 转换到 Running
	err = queue.MoveToRunning(taskID)
	assert.NoError(t, err)

	// 转换到 Completed
	err = queue.MoveToComplete(taskID)
	assert.NoError(t, err)

	// 验证最终状态
	task, exists := queue.Get(taskID)
	assert.True(t, exists)
	assert.Equal(t, common.TaskStateCompleted, task.State)
}

// TestStateMachineInvalidTransitions 测试非法的状态转换被拒绝
func TestStateMachineInvalidTransitions(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	// 提交任务
	err := queue.Submit(task)
	assert.NoError(t, err)

	// 转换到 Running（合法）
	err = queue.MoveToRunning(taskID)
	assert.NoError(t, err)

	// 尝试从 Running 转到 Running（非法）
	// 这会导致状态机拒绝转换
	// 但我们需要先完成或失败，再测试不能再转到其他状态

	// 先标记为完成
	err = queue.MoveToComplete(taskID)
	assert.NoError(t, err)

	// 尝试从 Completed 转到 Failed（非法 - 完成状态是终态）
	taskID2 := generateTaskID()
	task2 := &common.Task{
		ID:           taskID2,
		TotalWorkers: 10,
	}
	queue.Submit(task2)
	queue.MoveToRunning(taskID2)
	queue.MoveToComplete(taskID2)

	// 尝试将已完成的任务标记为失败应该失败
	err = queue.MoveToFailed(taskID2, "test failure")
	assert.Error(t, err)
}

// TestStateMachineMultipleTasksIndependent 测试多个任务的状态机独立性
func TestStateMachineMultipleTasksIndependent(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID1 := generateTaskID()
	taskID2 := generateTaskID()
	taskID3 := generateTaskID()

	// 创建三个任务
	task1 := &common.Task{ID: taskID1, TotalWorkers: 10}
	task2 := &common.Task{ID: taskID2, TotalWorkers: 10}
	task3 := &common.Task{ID: taskID3, TotalWorkers: 10}

	queue.Submit(task1)
	queue.Submit(task2)
	queue.Submit(task3)

	// 获取三个任务的状态机
	sm1 := queue.getTaskStateMachine(taskID1)
	sm2 := queue.getTaskStateMachine(taskID2)
	sm3 := queue.getTaskStateMachine(taskID3)

	// 验证状态机是独立的（不同的对象）
	assert.NotEqual(t, fmt.Sprintf("%p", sm1), fmt.Sprintf("%p", sm2))
	assert.NotEqual(t, fmt.Sprintf("%p", sm2), fmt.Sprintf("%p", sm3))

	// 对三个任务进行不同的转换
	queue.MoveToRunning(taskID1)
	queue.MoveToRunning(taskID2)
	// taskID3 保持 Pending

	queue.MoveToComplete(taskID1)
	queue.MoveToFailed(taskID2, "test error")
	queue.Cancel(taskID3) // Pending -> Stopped

	// 验证最终状态都符合预期
	t1, _ := queue.Get(taskID1)
	t2, _ := queue.Get(taskID2)
	t3, _ := queue.Get(taskID3)

	assert.Equal(t, common.TaskStateCompleted, t1.State)
	assert.Equal(t, common.TaskStateFailed, t2.State)
	assert.Equal(t, common.TaskStateStopped, t3.State)

	// 验证转换历史也是独立的
	history1 := sm1.GetHistory()
	history2 := sm2.GetHistory()
	history3 := sm3.GetHistory()

	// task1: Pending -> Running -> Completed (2 transitions)
	assert.Equal(t, 2, len(history1))

	// task2: Pending -> Running -> Failed (2 transitions)
	assert.Equal(t, 2, len(history2))

	// task3: Pending -> Stopped (1 transition)
	assert.Equal(t, 1, len(history3))
}

// TestStateMachineFailureTransitions 测试失败的状态转换
func TestStateMachineFailureTransitions(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)

	// Pending -> Running
	err := queue.MoveToRunning(taskID)
	assert.NoError(t, err)

	// Running -> Failed
	err = queue.MoveToFailed(taskID, "connection timeout")
	assert.NoError(t, err)

	// 验证状态
	task, exists := queue.Get(taskID)
	assert.True(t, exists)
	assert.Equal(t, common.TaskStateFailed, task.State)
	assert.Equal(t, "connection timeout", task.Metadata["failure_reason"])

	// 检查转换历史
	sm := queue.getTaskStateMachine(taskID)
	history := sm.GetHistory()
	assert.Equal(t, 2, len(history))
	assert.Equal(t, common.TaskStateFailed, history[1].To)
}

// TestStateMachineCancelTransitions 测试取消的状态转换
func TestStateMachineCancelTransitions(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	// 测试从 Pending 取消
	taskID1 := generateTaskID()
	task1 := &common.Task{ID: taskID1, TotalWorkers: 10}
	queue.Submit(task1)

	err := queue.Cancel(taskID1)
	assert.NoError(t, err)

	t1, _ := queue.Get(taskID1)
	assert.Equal(t, common.TaskStateStopped, t1.State)

	// 检查转换历史
	sm1 := queue.getTaskStateMachine(taskID1)
	history1 := sm1.GetHistory()
	assert.Equal(t, 1, len(history1)) // Pending -> Stopped

	// 测试从 Running 取消
	taskID2 := generateTaskID()
	task2 := &common.Task{ID: taskID2, TotalWorkers: 10}
	queue.Submit(task2)
	queue.MoveToRunning(taskID2)

	err = queue.Cancel(taskID2)
	assert.NoError(t, err)

	t2, _ := queue.Get(taskID2)
	assert.Equal(t, common.TaskStateStopped, t2.State)

	// 检查转换历史
	sm2 := queue.getTaskStateMachine(taskID2)
	history2 := sm2.GetHistory()
	assert.Equal(t, 2, len(history2)) // Pending -> Running -> Stopped
}

// TestStateMachineAllowedTransitionsDefinition 测试状态转换规则定义
func TestStateMachineAllowedTransitionsDefinition(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)
	sm := queue.getTaskStateMachine(taskID)

	// 检查允许的转换
	allowedFromPending := sm.GetAllowedTransitions()

	// 应该能从 Pending 转到 Running 或 Stopped
	assert.Contains(t, allowedFromPending, common.TaskStateRunning)
	assert.Contains(t, allowedFromPending, common.TaskStateStopped)

	// 转到 Running
	queue.MoveToRunning(taskID)

	// 现在检查从 Running 允许的转换
	allowedFromRunning := sm.GetAllowedTransitions()

	// 应该能从 Running 转到 Completed, Failed 或 Stopped
	assert.Contains(t, allowedFromRunning, common.TaskStateCompleted)
	assert.Contains(t, allowedFromRunning, common.TaskStateFailed)
	assert.Contains(t, allowedFromRunning, common.TaskStateStopped)
}

// TestStateMachineTransitionTiming 测试状态转换时间戳
func TestStateMachineTransitionTiming(t *testing.T) {
	log := logger.New()
	queue := NewTaskQueue(&MockTaskSplitter{}, log)

	taskID := generateTaskID()
	task := &common.Task{
		ID:           taskID,
		TotalWorkers: 10,
	}

	queue.Submit(task)
	sm := queue.getTaskStateMachine(taskID)

	beforeTime := time.Now()

	// 执行转换
	queue.MoveToRunning(taskID)
	time.Sleep(10 * time.Millisecond)
	queue.MoveToComplete(taskID)

	afterTime := time.Now()

	// 检查转换时间
	lastTransition, exists := sm.GetLastTransition()
	assert.True(t, exists)
	assert.True(t, lastTransition.Timestamp.After(beforeTime) || lastTransition.Timestamp.Equal(beforeTime))
	assert.True(t, !lastTransition.Timestamp.After(afterTime))

	// 检查转换持续时间
	duration := sm.GetLastTransitionDuration()
	assert.Greater(t, duration, time.Duration(0))
}
