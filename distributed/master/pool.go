/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\master\pool.go
 * @Description: Slave 连接池管理
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

// SlavePool Slave 连接池 - 使用 syncx.Map 实现线程安全
type SlavePool struct {
	slaves      *syncx.Map[string, *common.SlaveInfo] // 使用 syncx.Map 替代 sync.RWMutex + map
	selector    SlaveSelector
	healthCheck *HealthChecker
	logger      logger.ILogger
}

// NewSlavePool 创建 Slave 池
func NewSlavePool(selector SlaveSelector, log logger.ILogger) *SlavePool {
	pool := &SlavePool{
		slaves:   syncx.NewMap[string, *common.SlaveInfo](),
		selector: selector,
		logger:   log,
	}
	pool.healthCheck = NewHealthChecker(pool, log)
	return pool
}

// Register 注册 Slave
func (sp *SlavePool) Register(slave *common.SlaveInfo) error {
	if _, loaded := sp.slaves.Load(slave.ID); loaded {
		return fmt.Errorf("slave %s already registered", slave.ID)
	}

	slave.RegisteredAt = time.Now()
	slave.LastHeartbeat = time.Now()
	slave.State = common.SlaveStateIdle
	sp.slaves.Store(slave.ID, slave)

	return nil
}

// Unregister 注销 Slave
func (sp *SlavePool) Unregister(slaveID string) error {
	if _, exists := sp.slaves.Load(slaveID); !exists {
		return fmt.Errorf("slave %s not found", slaveID)
	}

	sp.slaves.Delete(slaveID)
	return nil
}

// Get 获取指定 Slave
func (sp *SlavePool) Get(slaveID string) (*common.SlaveInfo, bool) {
	return sp.slaves.Load(slaveID)
}

// GetAll 获取所有 Slave
func (sp *SlavePool) GetAll() []*common.SlaveInfo {
	slaves := make([]*common.SlaveInfo, 0)
	sp.slaves.Range(func(_ string, slave *common.SlaveInfo) bool {
		slaves = append(slaves, slave)
		return true
	})
	return slaves
}

// GetHealthy 获取所有健康的 Slave
func (sp *SlavePool) GetHealthy() []*common.SlaveInfo {
	return sp.slaves.Filter(func(_ string, slave *common.SlaveInfo) bool {
		return slave.State != common.SlaveStateOffline && slave.State != common.SlaveStateError
	})
}

// GetIdle 获取所有空闲的 Slave
func (sp *SlavePool) GetIdle() []*common.SlaveInfo {
	return sp.slaves.Filter(func(_ string, slave *common.SlaveInfo) bool {
		return slave.State == common.SlaveStateIdle
	})
}

// Select 选择指定数量的 Slave
func (sp *SlavePool) Select(count int) []*common.SlaveInfo {
	healthy := sp.GetHealthy()
	if sp.selector != nil {
		return sp.selector.Select(healthy, count)
	}
	// 默认返回前 count 个
	if count >= len(healthy) {
		return healthy
	}
	return healthy[:count]
}

// SelectWithFilter 根据筛选条件选择 Slave
func (sp *SlavePool) SelectWithFilter(count int, filter *common.SlaveFilter) []*common.SlaveInfo {
	// 如果没有筛选条件，使用默认选择
	if filter == nil {
		return sp.Select(count)
	}

	// 先获取所有 Slave
	allSlaves := make([]*common.SlaveInfo, 0)
	sp.slaves.Range(func(_ string, slave *common.SlaveInfo) bool {
		allSlaves = append(allSlaves, slave)
		return true
	})

	// 应用筛选条件
	filtered := make([]*common.SlaveInfo, 0)
	for _, slave := range allSlaves {
		if filter.IsSlaveValid(slave) {
			filtered = append(filtered, slave)
		}
	}

	// 如果设置了优先空闲，先按状态排序
	if filter.PreferIdle {
		idle := make([]*common.SlaveInfo, 0)
		busy := make([]*common.SlaveInfo, 0)
		for _, slave := range filtered {
			if slave.State == common.SlaveStateIdle || len(slave.RunningTasks) == 0 {
				idle = append(idle, slave)
			} else {
				busy = append(busy, slave)
			}
		}
		// 优先返回空闲的
		filtered = append(idle, busy...)
	}

	// 使用选择器选择
	if sp.selector != nil {
		return sp.selector.Select(filtered, count)
	}

	// 默认返回前 count 个
	if count >= len(filtered) {
		return filtered
	}
	return filtered[:count]
}

// UpdateSlaveState 更新 Slave 状态
func (sp *SlavePool) UpdateSlaveState(slaveID string, state common.SlaveState) error {
	if !sp.slaves.Update(slaveID, func(slave *common.SlaveInfo) *common.SlaveInfo {
		slave.State = state
		return slave
	}) {
		return fmt.Errorf("slave %s not found", slaveID)
	}
	return nil
}

// UpdateResourceUsage 更新 Slave 资源使用情况
func (sp *SlavePool) UpdateResourceUsage(slaveID string, usage *common.ResourceUsage) error {
	if !sp.slaves.Update(slaveID, func(slave *common.SlaveInfo) *common.SlaveInfo {
		slave.ResourceUsage = usage

		// 根据资源使用情况自动更新状态
		if usage != nil {
			if usage.ActiveTasks == 0 {
				slave.State = common.SlaveStateIdle
			} else if slave.MaxConcurrency > 0 && usage.ActiveTasks >= slave.MaxConcurrency {
				slave.State = common.SlaveStateOverloaded
			} else if usage.CPUPercent > 90 || usage.MemoryPercent > 90 {
				slave.State = common.SlaveStateOverloaded
			} else if usage.ActiveTasks > 0 {
				slave.State = common.SlaveStateBusy
			}
		}
		return slave
	}) {
		return fmt.Errorf("slave %s not found", slaveID)
	}
	return nil
}

// AddTask 为 Slave 添加任务
func (sp *SlavePool) AddTask(slaveID string, taskID string) error {
	if !sp.slaves.Update(slaveID, func(slave *common.SlaveInfo) *common.SlaveInfo {
		// 检查是否已存在
		for _, id := range slave.RunningTasks {
			if id == taskID {
				return slave // 已存在,不重复添加
			}
		}

		slave.RunningTasks = append(slave.RunningTasks, taskID)
		slave.CurrentTaskID = taskID

		// 更新状态
		if len(slave.RunningTasks) > 0 {
			if slave.MaxConcurrency > 0 && len(slave.RunningTasks) >= slave.MaxConcurrency {
				slave.State = common.SlaveStateOverloaded
			} else {
				slave.State = common.SlaveStateBusy
			}
		}
		return slave
	}) {
		return fmt.Errorf("slave %s not found", slaveID)
	}
	return nil
}

// RemoveTask 从 Slave 移除任务
func (sp *SlavePool) RemoveTask(slaveID string, taskID string) error {
	if !sp.slaves.Update(slaveID, func(slave *common.SlaveInfo) *common.SlaveInfo {
		// 移除任务
		tasks := make([]string, 0)
		for _, id := range slave.RunningTasks {
			if id != taskID {
				tasks = append(tasks, id)
			}
		}
		slave.RunningTasks = tasks

		// 更新当前任务 ID
		if slave.CurrentTaskID == taskID {
			if len(slave.RunningTasks) > 0 {
				slave.CurrentTaskID = slave.RunningTasks[0]
			} else {
				slave.CurrentTaskID = ""
			}
		}

		// 更新状态
		if len(slave.RunningTasks) == 0 {
			slave.State = common.SlaveStateIdle
		} else if slave.MaxConcurrency > 0 && len(slave.RunningTasks) < slave.MaxConcurrency {
			slave.State = common.SlaveStateBusy
		}
		return slave
	}) {
		return fmt.Errorf("slave %s not found", slaveID)
	}
	return nil
}

// UpdateHeartbeat 更新心跳时间
func (sp *SlavePool) UpdateHeartbeat(slaveID string) error {
	if !sp.slaves.Update(slaveID, func(slave *common.SlaveInfo) *common.SlaveInfo {
		slave.LastHeartbeat = time.Now()
		slave.HealthCheckFail = 0
		return slave
	}) {
		return fmt.Errorf("slave %s not found", slaveID)
	}
	return nil
}

// UpdateState 更新 Slave 状态（已废弃，请使用 UpdateSlaveState）
func (sp *SlavePool) UpdateState(slaveID string, state common.SlaveState) error {
	return sp.UpdateSlaveState(slaveID, state)
}

// MarkHealthy 标记 Slave 为健康
func (sp *SlavePool) MarkHealthy(slaveID string) error {
	if !sp.slaves.Update(slaveID, func(slave *common.SlaveInfo) *common.SlaveInfo {
		if slave.State == common.SlaveStateOffline || slave.State == common.SlaveStateError {
			slave.State = common.SlaveStateIdle
		}
		slave.HealthCheckFail = 0
		return slave
	}) {
		return fmt.Errorf("slave %s not found", slaveID)
	}
	return nil
}

// MarkUnhealthy 标记 Slave 为不健康
func (sp *SlavePool) MarkUnhealthy(slaveID string) error {
	if !sp.slaves.Update(slaveID, func(slave *common.SlaveInfo) *common.SlaveInfo {
		slave.State = common.SlaveStateError
		return slave
	}) {
		return fmt.Errorf("slave %s not found", slaveID)
	}
	return nil
}

// Count 获取 Slave 总数
func (sp *SlavePool) Count() int {
	return sp.slaves.Size()
}

// StartHealthCheck 启动健康检查
func (sp *SlavePool) StartHealthCheck(ctx context.Context) {
	sp.healthCheck.Start(ctx)
}

// SlaveSelector Slave 选择器接口
type SlaveSelector interface {
	Select(slaves []*common.SlaveInfo, count int) []*common.SlaveInfo
}
