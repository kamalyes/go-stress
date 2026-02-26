/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-02-26 15:08:57
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-02-26 15:08:57
 * @FilePath: \go-stress\distributed\master\failover.go
 * @Description: Slave 故障时的任务处理（失败闭环/降级运行）
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"fmt"

	"github.com/kamalyes/go-stress/distributed/common"
)

// handleSlaveFailure Slave 故障处理入口
//
// 由 SlavePool 在 Slave 进入故障态（心跳超时/主动下线）时回调，
// 对该 Slave 参与的运行中任务做闭环处理：
//   - 任务全部分配到的 Slave 都已故障 → 任务标记失败（否则将永远卡在 Running）
//   - 仍有存活 Slave → 任务降级继续（剩余 Slave 的完成/失败上报仍会驱动状态机）
//
// 不在此处做"任务重调度到其他 Slave"：压测任务的 Worker 已按分片跑在
// 各 Slave 上，任务级重放会造成请求重复发送，语义上应由上层按需重新
// 提交新任务（保留完整审计链路）
func (m *Master) handleSlaveFailure(slaveID string) {
	running := m.taskQueue.GetRunning()
	if len(running) == 0 {
		return
	}

	for _, task := range running {
		if !containsSlave(task.AssignedSlaves, slaveID) {
			continue
		}

		alive := m.countAliveAssignedSlaves(task, slaveID)
		if alive == 0 {
			reason := fmt.Sprintf("slave %s failed and no other assigned slave is alive", slaveID)
			m.logger.WarnKV("Task failed due to slave failure",
				"task_id", task.ID,
				"failed_slave", slaveID)
			if err := m.taskQueue.MoveToFailed(task.ID, reason); err != nil {
				// 任务可能恰好被并发的完成/停止路径先行归档，状态机会拒绝非法迁移
				m.logger.WarnKV("Failed to mark task as failed after slave failure",
					"task_id", task.ID, "error", err)
			}
		} else {
			m.logger.WarnKV("Slave failed, task continues on remaining slaves",
				"task_id", task.ID,
				"failed_slave", slaveID,
				"alive_slaves", alive)
		}

		// 无论任务归档与否，故障 Slave 上的任务记录都要清理，
		// 避免 Slave 恢复后残留脏的 RunningTasks 列表
		if err := m.slavePool.RemoveTask(slaveID, task.ID); err != nil {
			m.logger.DebugKV("Failed to remove task from failed slave",
				"slave_id", slaveID, "task_id", task.ID, "error", err)
		}
	}
}

// countAliveAssignedSlaves 统计任务分配的 Slave 中（除故障者外）还有几个存活
func (m *Master) countAliveAssignedSlaves(task *common.Task, failedSlaveID string) int {
	alive := 0
	for _, id := range task.AssignedSlaves {
		if id == failedSlaveID {
			continue
		}
		info, ok := m.slavePool.Get(id)
		if !ok {
			continue
		}
		if info.State != common.SlaveStateError && info.State != common.SlaveStateOffline {
			alive++
		}
	}
	return alive
}

// containsSlave 判断 slaveID 是否在列表中
func containsSlave(slaveIDs []string, slaveID string) bool {
	for _, id := range slaveIDs {
		if id == slaveID {
			return true
		}
	}
	return false
}
