/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 23:59:25
 * @FilePath: \go-stress\distributed\common\states.go
 * @Description: 分布式压测状态枚举定义
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package common

// SlaveState Slave 状态 | EN Slave State
type SlaveState string

const (
	SlaveStateIdle        SlaveState = "idle"        // 空闲，可接受新任务 | EN Idle, can accept new tasks
	SlaveStateRunning     SlaveState = "running"     // 运行中 | EN Running
	SlaveStateStopping    SlaveState = "stopping"    // 停止中 | EN Stopping
	SlaveStateError       SlaveState = "error"       // 错误状态 | EN Error state
	SlaveStateOffline     SlaveState = "offline"     // 离线 | EN Offline
	SlaveStateBusy        SlaveState = "busy"        // 繁忙（可选择是否复用） | EN Busy (reusable optional)
	SlaveStateOverloaded  SlaveState = "overloaded"  // 过载 | EN Overloaded
	SlaveStateUnreachable SlaveState = "unreachable" // 不可达 | EN Unreachable
)

// NodeRole 节点角色 | EN Node Role
type NodeRole string

const (
	NodeRoleMaster NodeRole = "master" // Master 角色 | EN Master role
	NodeRoleSlave  NodeRole = "slave"  // Slave 角色 | EN Slave role
	NodeRoleBoth   NodeRole = "both"   // 双角色（既是 Master 也是 Slave） | EN Dual role (both Master and Slave)
)

// TaskState 任务状态 | EN Task State
type TaskState string

const (
	TaskStatePending  TaskState = "pending"  // 待执行 | EN Pending
	TaskStateRunning  TaskState = "running"  // 运行中 | EN Running
	TaskStateComplete TaskState = "complete" // 已完成 | EN Completed
	TaskStateFailed   TaskState = "failed"   // 执行失败 | EN Failed
	TaskStateStopped  TaskState = "stopped"  // 已停止 | EN Stopped
)
