/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\master\splitter.go
 * @Description: 任务分片策略实现
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"fmt"

	"github.com/kamalyes/go-stress/distributed/common"
)

// TaskSplitter 任务分片器接口
type TaskSplitter interface {
	Split(task *common.Task, slaves []*common.SlaveInfo) ([]*common.SubTask, error)
}

// EqualSplitter 平均分片器
type EqualSplitter struct{}

// NewEqualSplitter 创建平均分片器
func NewEqualSplitter() *EqualSplitter {
	return &EqualSplitter{}
}

// Split 平均分配任务
func (s *EqualSplitter) Split(task *common.Task, slaves []*common.SlaveInfo) ([]*common.SubTask, error) {
	if len(slaves) == 0 {
		return nil, fmt.Errorf("no slaves available")
	}

	slaveCount := len(slaves)
	workersPerSlave := task.TotalWorkers / slaveCount
	remainder := task.TotalWorkers % slaveCount

	subTasks := make([]*common.SubTask, 0, slaveCount)

	for i, slave := range slaves {
		workerCount := workersPerSlave
		// 将余数分配给前面的 Slave
		if i < remainder {
			workerCount++
		}

		if workerCount == 0 {
			continue
		}

		subTask := &common.SubTask{
			TaskID:      task.ID,
			SubTaskID:   fmt.Sprintf("%s-part-%d", task.ID, i),
			SlaveID:     slave.ID,
			WorkerCount: workerCount,
			Config:      task.ConfigData,
		}
		subTasks = append(subTasks, subTask)
	}

	return subTasks, nil
}

// WeightedSplitter 权重分片器
type WeightedSplitter struct {
	weights map[string]float64 // slave_id -> weight
}

// NewWeightedSplitter 创建权重分片器
func NewWeightedSplitter() *WeightedSplitter {
	return &WeightedSplitter{
		weights: make(map[string]float64),
	}
}

// SetWeight 设置 Slave 权重
func (s *WeightedSplitter) SetWeight(slaveID string, weight float64) {
	s.weights[slaveID] = weight
}

// Split 按权重分配任务
func (s *WeightedSplitter) Split(task *common.Task, slaves []*common.SlaveInfo) ([]*common.SubTask, error) {
	if len(slaves) == 0 {
		return nil, fmt.Errorf("no slaves available")
	}

	// 计算总权重
	totalWeight := 0.0
	for _, slave := range slaves {
		weight := s.getWeight(slave)
		totalWeight += weight
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("total weight is zero")
	}

	subTasks := make([]*common.SubTask, 0, len(slaves))
	allocatedWorkers := 0

	for i, slave := range slaves {
		weight := s.getWeight(slave)
		var workerCount int

		// 最后一个 Slave 分配剩余的所有 Worker
		if i == len(slaves)-1 {
			workerCount = task.TotalWorkers - allocatedWorkers
		} else {
			workerCount = int(float64(task.TotalWorkers) * weight / totalWeight)
		}

		if workerCount == 0 {
			continue
		}

		subTask := &common.SubTask{
			TaskID:      task.ID,
			SubTaskID:   fmt.Sprintf("%s-%s", task.ID, slave.ID),
			SlaveID:     slave.ID,
			WorkerCount: workerCount,
			Config:      task.ConfigData,
		}
		subTasks = append(subTasks, subTask)
		allocatedWorkers += workerCount
	}

	return subTasks, nil
}

// getWeight 获取 Slave 的权重
func (s *WeightedSplitter) getWeight(slave *common.SlaveInfo) float64 {
	if w, ok := s.weights[slave.ID]; ok {
		return w
	}
	// 默认按 CPU 核心数作为权重
	return float64(slave.CPUCores)
}

// CustomSplitter 自定义分片器
type CustomSplitter struct {
	splitFunc func(*common.Task, []*common.SlaveInfo) ([]*common.SubTask, error)
}

// NewCustomSplitter 创建自定义分片器
func NewCustomSplitter(fn func(*common.Task, []*common.SlaveInfo) ([]*common.SubTask, error)) *CustomSplitter {
	return &CustomSplitter{
		splitFunc: fn,
	}
}

// Split 使用自定义函数分配任务
func (s *CustomSplitter) Split(task *common.Task, slaves []*common.SlaveInfo) ([]*common.SubTask, error) {
	if s.splitFunc == nil {
		return nil, fmt.Errorf("split function not set")
	}
	return s.splitFunc(task, slaves)
}

// GetSplitter 根据策略获取分片器
func GetSplitter(strategy common.SplitStrategy) TaskSplitter {
	switch strategy {
	case common.SplitStrategyEqual:
		return NewEqualSplitter()
	case common.SplitStrategyWeighted:
		return NewWeightedSplitter()
	default:
		return NewEqualSplitter()
	}
}
