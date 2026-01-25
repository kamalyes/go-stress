/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 00:02:35
 * @FilePath: \go-stress\distributed\slave\monitor.go
 * @Description: 资源监控器
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package slave

import (
	"context"
	"runtime"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// ResourceMonitor 资源监控器
type ResourceMonitor struct {
	mu             *syncx.RWLock // 使用 syncx.RWLock 替代 sync.RWMutex
	logger         logger.ILogger
	updateInterval time.Duration
	activeTasks    int
	queuedTasks    int
	lastNetIO      *net.IOCountersStat
	lastNetIOTime  time.Time
}

// NewResourceMonitor 创建资源监控器
func NewResourceMonitor(log logger.ILogger, interval time.Duration) *ResourceMonitor {
	return &ResourceMonitor{
		mu:             syncx.NewRWLock(),
		logger:         log,
		updateInterval: interval,
		lastNetIOTime:  time.Now(),
	}
}

// Start 启动资源监控
func (rm *ResourceMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(rm.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 定期更新资源使用情况
			usage, err := rm.GetResourceUsage()
			if err != nil {
				rm.logger.ErrorContextKV(ctx, "Failed to get resource usage", map[string]interface{}{
					"error": err.Error(),
				})
			} else {
				rm.logger.DebugContextKV(ctx, "Resource usage updated", map[string]interface{}{
					"cpu_percent":    usage.CPUPercent,
					"memory_percent": usage.MemoryPercent,
					"active_tasks":   usage.ActiveTasks,
					"load_average":   usage.LoadAverage,
				})
			}
		}
	}
}

// GetResourceUsage 获取当前资源使用情况
func (rm *ResourceMonitor) GetResourceUsage() (*common.ResourceUsage, error) {
	usage := &common.ResourceUsage{
		Timestamp: time.Now(),
	}

	// CPU 使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		usage.CPUPercent = cpuPercent[0]
	}

	// 内存使用情况
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		usage.MemoryPercent = vmStat.UsedPercent
		usage.MemoryUsed = int64(vmStat.Used)
		usage.MemoryTotal = int64(vmStat.Total)
	}

	// 系统负载
	loadAvg, err := load.Avg()
	if err == nil {
		usage.LoadAverage = loadAvg.Load1
	}

	// 任务数量
	syncx.WithRLock(rm.mu, func() {
		usage.ActiveTasks = rm.activeTasks
		usage.QueuedTasks = rm.queuedTasks
	})

	// 网络 IO
	netIO, err := net.IOCounters(false)
	if err == nil && len(netIO) > 0 {
		currentIO := &netIO[0]
		currentTime := time.Now()

		if rm.lastNetIO != nil {
			duration := currentTime.Sub(rm.lastNetIOTime).Seconds()
			if duration > 0 {
				// 计算速率 (Mbps)
				bytesInDiff := float64(currentIO.BytesRecv - rm.lastNetIO.BytesRecv)
				bytesOutDiff := float64(currentIO.BytesSent - rm.lastNetIO.BytesSent)
				usage.NetworkInMbps = (bytesInDiff * 8) / (1024 * 1024 * duration)
				usage.NetworkOutMbps = (bytesOutDiff * 8) / (1024 * 1024 * duration)
			}
		}

		rm.lastNetIO = currentIO
		rm.lastNetIOTime = currentTime
	}

	// 磁盘 IO 使用率 (简化版本，可根据需要扩展)
	// 这里暂时不实现，可以使用 gopsutil/disk 包实现
	usage.DiskIOUtil = 0

	return usage, nil
}

// SetActiveTasks 设置活跃任务数
func (rm *ResourceMonitor) SetActiveTasks(count int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.activeTasks = count
}

// IncrementActiveTasks 增加活跃任务数
func (rm *ResourceMonitor) IncrementActiveTasks() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.activeTasks++
}

// DecrementActiveTasks 减少活跃任务数
func (rm *ResourceMonitor) DecrementActiveTasks() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.activeTasks > 0 {
		rm.activeTasks--
	}
}

// SetQueuedTasks 设置排队任务数
func (rm *ResourceMonitor) SetQueuedTasks(count int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.queuedTasks = count
}

// IncrementQueuedTasks 增加排队任务数
func (rm *ResourceMonitor) IncrementQueuedTasks() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.queuedTasks++
}

// DecrementQueuedTasks 减少排队任务数
func (rm *ResourceMonitor) DecrementQueuedTasks() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.queuedTasks > 0 {
		rm.queuedTasks--
	}
}

// GetGoRuntimeStats 获取 Go 运行时统计信息
func (rm *ResourceMonitor) GetGoRuntimeStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"goroutines":    runtime.NumGoroutine(),
		"heap_alloc_mb": float64(m.Alloc) / 1024 / 1024,
		"heap_sys_mb":   float64(m.HeapSys) / 1024 / 1024,
		"heap_inuse_mb": float64(m.HeapInuse) / 1024 / 1024,
		"heap_idle_mb":  float64(m.HeapIdle) / 1024 / 1024,
		"gc_num":        m.NumGC,
		"last_gc_time":  time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
		"next_gc_mb":    float64(m.NextGC) / 1024 / 1024,
	}
}

// IsHealthy 检查资源是否健康
func (rm *ResourceMonitor) IsHealthy() (bool, string) {
	usage, err := rm.GetResourceUsage()
	if err != nil {
		return false, "Failed to get resource usage: " + err.Error()
	}

	// CPU 使用率过高
	if usage.CPUPercent > 95 {
		return false, "CPU usage too high"
	}

	// 内存使用率过高
	if usage.MemoryPercent > 95 {
		return false, "Memory usage too high"
	}

	// 系统负载过高 (简化判断：负载 > CPU 核心数 * 2)
	numCPU := runtime.NumCPU()
	if usage.LoadAverage > float64(numCPU*2) {
		return false, "System load too high"
	}

	return true, ""
}
