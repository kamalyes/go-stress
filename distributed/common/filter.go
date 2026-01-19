/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\common\filter.go
 * @Description: Slave 筛选条件及校验方法
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package common

// SlaveFilter Slave 筛选条件
type SlaveFilter struct {
	IncludeIDs       []string          `json:"include_ids" yaml:"include_ids"`               // 包含的 Slave ID 列表
	ExcludeIDs       []string          `json:"exclude_ids" yaml:"exclude_ids"`               // 排除的 Slave ID 列表
	IncludeRegions   []string          `json:"include_regions" yaml:"include_regions"`       // 包含的地域列表
	ExcludeRegions   []string          `json:"exclude_regions" yaml:"exclude_regions"`       // 排除的地域列表
	IncludeLabels    map[string]string `json:"include_labels" yaml:"include_labels"`         // 必须包含的标签
	ExcludeLabels    map[string]string `json:"exclude_labels" yaml:"exclude_labels"`         // 必须不包含的标签
	RequiredStates   []SlaveState      `json:"required_states" yaml:"required_states"`       // 允许的状态列表
	ExcludedStates   []SlaveState      `json:"excluded_states" yaml:"excluded_states"`       // 排除的状态列表
	MinCPUCores      int               `json:"min_cpu_cores" yaml:"min_cpu_cores"`           // 最小 CPU 核心数
	MinMemory        int64             `json:"min_memory" yaml:"min_memory"`                 // 最小内存(字节)
	MaxCPUPercent    float64           `json:"max_cpu_percent" yaml:"max_cpu_percent"`       // 最大 CPU 使用率 0-100
	MaxMemoryPercent float64           `json:"max_memory_percent" yaml:"max_memory_percent"` // 最大内存使用率 0-100
	MaxLoad          float64           `json:"max_load" yaml:"max_load"`                     // 最大负载
	MaxActiveTasks   int               `json:"max_active_tasks" yaml:"max_active_tasks"`     // 最大活跃任务数
	AllowReuse       bool              `json:"allow_reuse" yaml:"allow_reuse"`               // 是否允许使用正在运行任务的 Slave
	PreferIdle       bool              `json:"prefer_idle" yaml:"prefer_idle"`               // 优先选择空闲 Slave
}

// IsSlaveValid 检查 Slave 是否符合筛选条件
func (f *SlaveFilter) IsSlaveValid(slave *SlaveInfo) bool {
	// 检查 ID 包含/排除
	if len(f.IncludeIDs) > 0 {
		found := false
		for _, id := range f.IncludeIDs {
			if slave.ID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.ExcludeIDs) > 0 {
		for _, id := range f.ExcludeIDs {
			if slave.ID == id {
				return false
			}
		}
	}

	// 检查地域包含/排除
	if len(f.IncludeRegions) > 0 {
		found := false
		for _, region := range f.IncludeRegions {
			if slave.Region == region {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.ExcludeRegions) > 0 {
		for _, region := range f.ExcludeRegions {
			if slave.Region == region {
				return false
			}
		}
	}

	// 检查标签包含/排除
	for k, v := range f.IncludeLabels {
		if slaveVal, ok := slave.Labels[k]; !ok || slaveVal != v {
			return false
		}
	}

	for k, v := range f.ExcludeLabels {
		if slaveVal, ok := slave.Labels[k]; ok && slaveVal == v {
			return false
		}
	}

	// 检查状态
	if len(f.RequiredStates) > 0 {
		found := false
		for _, state := range f.RequiredStates {
			if slave.State == state {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.ExcludedStates) > 0 {
		for _, state := range f.ExcludedStates {
			if slave.State == state {
				return false
			}
		}
	}

	// 检查资源限制
	if slave.CPUCores < f.MinCPUCores {
		return false
	}

	if slave.Memory < f.MinMemory {
		return false
	}

	// 检查资源使用情况
	if slave.ResourceUsage != nil {
		if f.MaxCPUPercent > 0 && slave.ResourceUsage.CPUPercent > f.MaxCPUPercent {
			return false
		}

		if f.MaxMemoryPercent > 0 && slave.ResourceUsage.MemoryPercent > f.MaxMemoryPercent {
			return false
		}

		if f.MaxLoad > 0 && slave.ResourceUsage.LoadAverage > f.MaxLoad {
			return false
		}

		if f.MaxActiveTasks > 0 && slave.ResourceUsage.ActiveTasks > f.MaxActiveTasks {
			return false
		}
	}

	// 检查是否允许复用
	if !f.AllowReuse && len(slave.RunningTasks) > 0 {
		return false
	}

	// 检查并发限制
	if slave.MaxConcurrency > 0 && len(slave.RunningTasks) >= slave.MaxConcurrency {
		return false
	}

	return true
}
