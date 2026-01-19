/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\master\master.go
 * @Description: Master 主控制器
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/distributed/common"
	pb "github.com/kamalyes/go-stress/distributed/proto"
	"github.com/kamalyes/go-toolbox/pkg/errorx"
	"github.com/kamalyes/go-toolbox/pkg/idgen"
	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Master Master 节点主控制器
type Master struct {
	config       *common.MasterConfig
	slavePool    *SlavePool
	taskQueue    *TaskQueue
	collector    *StatsCollector
	grpcServer   *grpc.Server
	slaveClients *syncx.Map[string, pb.SlaveServiceClient] // 使用 syncx.Map 管理 slave_id -> client
	idGenerator  *idgen.SnowflakeGenerator                 // ID 生成器
	logger       logger.ILogger
	running      *syncx.Bool // 使用 syncx.Bool
	cancelFunc   context.CancelFunc
}

// NewMaster 创建 Master 实例
func NewMaster(config *common.MasterConfig, log logger.ILogger) (*Master, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// 创建 Slave 选择器
	selector := GetSelector(common.SelectStrategyLeastLoaded, nil)

	// 创建任务分片器
	splitter := GetSplitter(common.SplitStrategyEqual)

	master := &Master{
		config:       config,
		slavePool:    NewSlavePool(selector, log),
		taskQueue:    NewTaskQueue(splitter, log),
		collector:    NewStatsCollector(1000, log),
		slaveClients: syncx.NewMap[string, pb.SlaveServiceClient](),
		idGenerator:  idgen.NewSnowflakeGenerator(1, 1), // workerID=1, datacenter=1
		running:      syncx.NewBool(false),
		logger:       log,
	}

	return master, nil
}

// Start 启动 Master 服务
func (m *Master) Start(ctx context.Context) error {
	if !m.running.CAS(false, true) {
		return fmt.Errorf("master is already running")
	}

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel

	// 启动各个组件
	go m.slavePool.StartHealthCheck(ctx)
	go m.collector.Start(ctx)
	go m.taskQueue.StartCleaner(ctx, 10*time.Minute, 1*time.Hour)

	// 启动 gRPC 服务器
	if err := m.startGRPCServer(); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	m.logger.Info("Master started successfully",
		"grpc_port", m.config.GRPCPort,
		"http_port", m.config.HTTPPort)

	return nil
}

// Stop 停止 Master 服务
func (m *Master) Stop() error {
	if !m.running.CAS(true, false) {
		return fmt.Errorf("master is not running")
	}

	m.logger.Info("Stopping master...")

	// 停止所有组件
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	// 停止 gRPC 服务器
	if m.grpcServer != nil {
		m.grpcServer.GracefulStop()
	}

	m.logger.Info("Master stopped")
	return nil
}

// startGRPCServer 启动 gRPC 服务器
func (m *Master) startGRPCServer() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// 创建 gRPC 服务器
	m.grpcServer = grpc.NewServer()

	// 注册服务
	masterService := NewMasterServiceServer(m, m.logger)
	pb.RegisterMasterServiceServer(m.grpcServer, masterService)

	// 启动服务器
	go func() {
		m.logger.Info("gRPC server listening", "port", m.config.GRPCPort)
		if err := m.grpcServer.Serve(lis); err != nil {
			m.logger.Error("gRPC server error", "error", err)
		}
	}()

	return nil
}

// SubmitTask 提交任务
func (m *Master) SubmitTask(task *common.Task) error {
	// 使用 mathx.IfEmpty 确保有 task.ID
	task.ID = mathx.IfEmpty(task.ID, m.idGenerator.GenerateRequestID())

	// 提交到任务队列
	if err := m.taskQueue.Submit(task); err != nil {
		return errorx.WrapError("failed to submit task", err)
	}

	// 异步处理任务
	go m.processTask(task)

	return nil
}

// GenerateTaskID 生成任务ID（对外暴露）
func (m *Master) GenerateTaskID() string {
	return m.idGenerator.GenerateRequestID()
}

// processTask 处理任务
func (m *Master) processTask(task *common.Task) {
	// 选择 Slave
	slaves := m.slavePool.Select(m.getRequiredSlaveCount(task))
	if len(slaves) == 0 {
		m.logger.Error("No available slaves", "task_id", task.ID)
		m.taskQueue.MoveToFailed(task.ID, "no available slaves")
		return
	}

	// 分片任务
	subTasks, err := m.taskQueue.Split(task, slaves)
	if err != nil {
		errMsg := errorx.WrapError("failed to split task", err).Error()
		m.logger.Error(errMsg, "task_id", task.ID)
		m.taskQueue.MoveToFailed(task.ID, errMsg)
		return
	}

	// 更新任务状态
	task.AssignedSlaves = make([]string, len(slaves))
	for i, slave := range slaves {
		task.AssignedSlaves[i] = slave.ID
	}
	m.taskQueue.MoveToRunning(task.ID)

	// 使用 syncx.Parallel 并发分发子任务到各个 Slave
	successCount := syncx.NewInt32(0)

	syncx.NewParallelSliceExecutor[*common.SubTask, bool](subTasks).
		OnSuccess(func(idx int, st *common.SubTask, result bool) {
			successCount.Add(1)
			m.slavePool.AddTask(st.SlaveID, st.TaskID)
		}).
		OnError(func(idx int, st *common.SubTask, err error) {
			m.logger.Error("Failed to dispatch subtask",
				"task_id", st.TaskID,
				"subtask_id", st.SubTaskID,
				"slave_id", st.SlaveID,
				"error", err)
		}).
		Execute(func(idx int, st *common.SubTask) (bool, error) {
			err := m.dispatchSubTask(st)
			return err == nil, err
		})

	success := int(successCount.Load())
	if success == 0 {
		m.logger.Error("All subtasks failed", "task_id", task.ID)
		m.taskQueue.MoveToFailed(task.ID, "all subtasks failed")
	} else if success < len(subTasks) {
		m.logger.Warn("Some subtasks failed",
			"task_id", task.ID,
			"success", success,
			"total", len(subTasks))
	}

	m.logger.Info("Task distributed",
		"task_id", task.ID,
		"slave_count", len(slaves),
		"subtask_count", len(subTasks),
		"success", success)
}

// getRequiredSlaveCount 计算所需的 Slave 数量
func (m *Master) getRequiredSlaveCount(task *common.Task) int {
	// 简单策略：每100个worker需要1个slave
	count := mathx.IfNotZero(task.TotalWorkers/100, 1)

	// 最多使用所有可用的 Slave
	available := m.slavePool.Count()
	if count > available {
		count = available
	}

	return count
}

// GetSlavePool 获取 Slave 池
func (m *Master) GetSlavePool() *SlavePool {
	return m.slavePool
}

// GetTaskQueue 获取任务队列
func (m *Master) GetTaskQueue() *TaskQueue {
	return m.taskQueue
}

// GetCollector 获取统计收集器
func (m *Master) GetCollector() *StatsCollector {
	return m.collector
}

// GetStats 获取系统状态
func (m *Master) GetStats() map[string]interface{} {
	queueStats := m.taskQueue.Stats()

	return map[string]interface{}{
		"running":       m.running.Load(),
		"slave_count":   m.slavePool.Count(),
		"task_pending":  queueStats["pending"],
		"task_running":  queueStats["running"],
		"task_complete": queueStats["complete"],
		"task_failed":   queueStats["failed"],
	}
}

// dispatchSubTask 分发子任务到指定 Slave
func (m *Master) dispatchSubTask(subTask *common.SubTask) error {
	// 获取 Slave 客户端
	client, err := m.getSlaveClient(subTask.SlaveID)
	if err != nil {
		return fmt.Errorf("failed to get slave client: %w", err)
	}

	// 构建任务配置
	taskConfig := &pb.TaskConfig{
		TaskId:         subTask.TaskID,
		WorkerCount:    int32(subTask.WorkerCount),
		ConfigData:     subTask.Config,
		ReportInterval: 1, // 默认 1 秒上报一次
	}

	// 调用 Slave 的 ExecuteTask 方法
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ExecuteTask(ctx, taskConfig)
	if err != nil {
		return fmt.Errorf("failed to execute task on slave: %w", err)
	}

	if !resp.Accepted {
		return fmt.Errorf("task rejected by slave: %s", resp.Message)
	}

	m.logger.Info("Subtask dispatched",
		"task_id", subTask.TaskID,
		"subtask_id", subTask.SubTaskID,
		"slave_id", subTask.SlaveID,
		"workers", subTask.WorkerCount)

	return nil
}

// getSlaveClient 获取或创建 Slave 客户端 - 使用 syncx.Map
func (m *Master) getSlaveClient(slaveID string) (pb.SlaveServiceClient, error) {
	// 从 Map 中加载
	client, exists := m.slaveClients.Load(slaveID)
	if exists {
		return client, nil
	}

	// 获取 Slave 信息
	slave, exists := m.slavePool.Get(slaveID)
	if !exists {
		return nil, fmt.Errorf("slave %s not found", slaveID)
	}

	// 创建 gRPC 连接
	addr := fmt.Sprintf("%s:%d", slave.IP, slave.GRPCPort)
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create slave client: %w", err)
	}

	newClient := pb.NewSlaveServiceClient(conn)

	// 使用 LoadOrStore 避免并发创建
	actualClient, _ := m.slaveClients.LoadOrStore(slaveID, newClient)

	return actualClient, nil
}

// StopTask 停止任务 - 使用 syncx.Parallel
func (m *Master) StopTask(taskID string) error {
	task, exists := m.taskQueue.Get(taskID)
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// 使用 syncx.Parallel 并发向所有分配的 Slave 发送停止指令
	syncx.NewParallelSliceExecutor[string, bool](task.AssignedSlaves).
		OnError(func(idx int, slaveID string, err error) {
			m.logger.Error("Failed to stop task on slave", "slave_id", slaveID, "error", err)
		}).
		Execute(func(idx int, slaveID string) (bool, error) {
			client, err := m.getSlaveClient(slaveID)
			if err != nil {
				return false, err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err = client.StopTask(ctx, &pb.StopRequest{
				SlaveId: slaveID,
				TaskId:  taskID,
				Force:   false,
			})
			return err == nil, err
		})

	// 更新任务状态
	m.taskQueue.Cancel(taskID)

	return nil
}
