/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\slave\slave.go
 * @Description: Slave 节点实现
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package slave

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/distributed/common"
	pb "github.com/kamalyes/go-stress/distributed/proto"
	"github.com/kamalyes/go-stress/executor"
	"github.com/kamalyes/go-toolbox/pkg/netx"
	"github.com/kamalyes/go-toolbox/pkg/osx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Slave Slave 节点 - 使用 syncx 模块重构
type Slave struct {
	config        *common.SlaveConfig
	info          *common.SlaveInfo
	executor      *executor.Executor
	statsBuffer   *StatsBuffer
	monitor       *ResourceMonitor
	grpcServer    *grpc.Server
	masterClient  pb.MasterServiceClient
	masterConn    *grpc.ClientConn
	logger        logger.ILogger
	running       *syncx.Bool // 使用 syncx.Bool
	currentTaskID string
	cancelFunc    context.CancelFunc
	heartbeatTask *syncx.PeriodicTaskManager // 使用 syncx.PeriodicTask 管理心跳
}

// NewSlave 创建 Slave 实例
func NewSlave(config *common.SlaveConfig, log logger.ILogger) (*Slave, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// 获取主机信息
	hostname := osx.SafeGetHostName()
	localIP, err := netx.GetPrivateIP()
	if err != nil {
		localIP = "127.0.0.1"
	}

	info := &common.SlaveInfo{
		ID:       config.SlaveID,
		Hostname: hostname,
		IP:       localIP,
		GRPCPort: config.GRPCPort,
		CPUCores: runtime.NumCPU(),
		Memory:   getMemorySize(),
		Version:  "1.0.0",
		Region:   config.Region,
		Labels:   config.Labels,
		State:    common.SlaveStateIdle,
	}

	slave := &Slave{
		config:        config,
		info:          info,
		statsBuffer:   NewStatsBuffer(config.SlaveID, config.ReportBuffer, log),
		monitor:       NewResourceMonitor(log, 5*time.Second),
		logger:        log,
		running:       syncx.NewBool(false),
		heartbeatTask: syncx.NewPeriodicTaskManager(),
	}

	return slave, nil
}

// Start 启动 Slave 服务
func (s *Slave) Start(ctx context.Context) error {
	if !s.running.CAS(false, true) {
		return fmt.Errorf("slave is already running")
	}

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	// 连接到 Master
	if err := s.connectToMaster(); err != nil {
		return fmt.Errorf("failed to connect to master: %w", err)
	}

	// 注册到 Master
	if err := s.register(); err != nil {
		return fmt.Errorf("failed to register to master: %w", err)
	}

	// 启动心跳 - 使用 syncx.PeriodicTask
	s.startHeartbeat(ctx)

	// 启动统计上报
	s.statsBuffer.SetMasterClient(s.masterClient)
	go s.statsBuffer.Start(ctx)

	// 启动资源监控
	if s.config.ResourceMonitor && s.monitor != nil {
		go s.monitor.Start(ctx)
	}

	// 启动 gRPC 服务器
	if err := s.startGRPCServer(); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	s.logger.InfoContextKV(ctx, "Slave started successfully",
		"slave_id", s.info.ID,
		"master_addr", s.config.MasterAddr)

	return nil
}

// Stop 停止 Slave 服务
func (s *Slave) Stop() error {
	if !s.running.CAS(true, false) {
		return fmt.Errorf("slave is not running")
	}

	s.logger.InfoMsg("Stopping slave...")

	// 注销
	s.unregister()

	// 停止组件
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	// 停止 gRPC 服务器
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	// 停止 Executor (无法直接停止，需要等待其自然完成或通过 context 取消)
	// executor 通过 context 取消机制控制，cancelFunc 已调用
	if s.executor != nil {
		// executor 会在 context 取消后自动停止
		s.logger.InfoMsg("Executor will stop via context cancellation")
	}

	s.logger.InfoMsg("Slave stopped")
	return nil
}

// connectToMaster 连接到 Master
func (s *Slave) connectToMaster() error {
	s.logger.InfoKV("Connecting to master", "addr", s.config.MasterAddr)

	// 创建 gRPC 连接
	conn, err := grpc.NewClient(
		s.config.MasterAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to create master client: %w", err)
	}

	s.masterConn = conn
	s.masterClient = pb.NewMasterServiceClient(conn)

	s.logger.InfoKV("Connected to master", "addr", s.config.MasterAddr)
	return nil
}

// register 注册到 Master
func (s *Slave) register() error {
	s.logger.InfoKV("Registering to master", "slave_id", s.info.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建注册请求
	req := &pb.SlaveInfo{
		SlaveId:  s.info.ID,
		Hostname: s.info.Hostname,
		Ip:       s.info.IP,
		GrpcPort: int32(s.info.GRPCPort),
		CpuCores: int32(s.info.CPUCores),
		Memory:   s.info.Memory,
		Version:  s.info.Version,
		Region:   s.info.Region,
		Labels:   s.info.Labels,
	}

	// 调用 Master 的 RegisterSlave RPC
	resp, err := s.masterClient.RegisterSlave(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration rejected: %s", resp.Message)
	}

	s.logger.InfoKV("Registered successfully",
		"slave_id", s.info.ID,
		"token", resp.Token,
		"heartbeat_interval", resp.HeartbeatInterval)

	return nil
}

// unregister 从 Master 注销
func (s *Slave) unregister() error {
	s.logger.InfoKV("Unregistering from master", "slave_id", s.info.ID)

	if s.masterClient == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.UnregisterRequest{
		SlaveId: s.info.ID,
		Reason:  "shutdown",
	}

	resp, err := s.masterClient.UnregisterSlave(ctx, req)
	if err != nil {
		s.logger.WarnKV("Failed to unregister", "error", err)
		return err
	}

	if resp.Success {
		s.logger.InfoKV("Unregistered successfully", "slave_id", s.info.ID)
	}

	// 关闭连接
	if s.masterConn != nil {
		s.masterConn.Close()
	}

	return nil
}

// startHeartbeat 启动心跳 - 使用 syncx.PeriodicTask
func (s *Slave) startHeartbeat(ctx context.Context) {
	task := syncx.NewPeriodicTask("heartbeat", 5*time.Second, func(taskCtx context.Context) error {
		return s.sendHeartbeat()
	}).SetOnError(func(name string, err error) {
		s.logger.WarnContextKV(ctx, "Heartbeat error", "error", err)
	})

	s.heartbeatTask.AddTask(task)
	s.heartbeatTask.Start()
}

// sendHeartbeat 发送心跳
func (s *Slave) sendHeartbeat() error {
	if s.masterClient == nil {
		return fmt.Errorf("master client is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	status := s.getStatus()

	// 获取资源使用情况
	var cpuUsage, memUsage float64
	var runningWorkers int64
	if s.monitor != nil {
		if usage, err := s.monitor.GetResourceUsage(); err == nil {
			cpuUsage = usage.CPUPercent
			memUsage = usage.MemoryPercent
			runningWorkers = int64(usage.ActiveTasks)
		}
	}

	// 构建心跳请求
	req := &pb.HeartbeatRequest{
		SlaveId:   s.info.ID,
		Timestamp: time.Now().Unix(),
		Status: &pb.SlaveStatus{
			SlaveId:        s.info.ID,
			State:          commonStateToProtoState(status.State),
			CurrentTaskId:  status.CurrentTaskID,
			CpuUsage:       cpuUsage,
			MemoryUsage:    memUsage,
			RunningWorkers: runningWorkers,
			TotalRequests:  status.TotalRequests,
			Timestamp:      time.Now().Unix(),
		},
	}

	// 调用 Master 的 Heartbeat RPC
	resp, err := s.masterClient.Heartbeat(ctx, req)
	if err != nil {
		s.logger.WarnKV("Heartbeat failed", "error", err)
		return err
	}

	if !resp.Ok {
		s.logger.WarnKV("Heartbeat rejected", "message", resp.Message)
		return fmt.Errorf("heartbeat rejected: %s", resp.Message)
	}

	s.logger.DebugKV("Heartbeat sent",
		"slave_id", s.info.ID,
		"state", status.State)
	return nil
}

// startGRPCServer 启动 gRPC 服务器
func (s *Slave) startGRPCServer() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcServer = grpc.NewServer()

	// 注册服务
	slaveService := NewSlaveServiceServer(s, s.logger)
	pb.RegisterSlaveServiceServer(s.grpcServer, slaveService)

	go func() {
		s.logger.InfoKV("Slave gRPC server listening", "port", s.config.GRPCPort)
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.ErrorKV("gRPC server error", "error", err)
		}
	}()

	return nil
}

// ExecuteTask 执行任务 - 使用 syncx.GoExecutor
func (s *Slave) ExecuteTask(taskConfig *common.SubTask) error {
	if s.currentTaskID != "" {
		return fmt.Errorf("slave is already executing task: %s", s.currentTaskID)
	}

	s.currentTaskID = taskConfig.TaskID
	s.info.State = common.SlaveStateRunning

	// 设置 StatsBuffer 的任务 ID
	s.statsBuffer.SetTaskID(taskConfig.TaskID)

	s.logger.InfoKV("Executing task",
		"task_id", taskConfig.TaskID,
		"workers", taskConfig.WorkerCount)

	// 解析任务配置
	var cfg config.Config
	if err := json.Unmarshal(taskConfig.Config, &cfg); err != nil {
		return fmt.Errorf("failed to parse task config: %w", err)
	}

	// 设置 worker 数量
	cfg.Concurrency = uint64(taskConfig.WorkerCount)

	// 创建 Executor
	executor, err := executor.NewExecutor(&cfg)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// 设置外部上报器 - 传递 Add 方法作为回调
	executor.GetCollector().SetExternalReporter(s.statsBuffer.Add)
	s.executor = executor

	// 使用 syncx.GoExecutor 在后台运行 Executor
	syncx.Go().
		OnError(func(err error) {
			s.logger.ErrorKV("Executor failed", "error", err)
			s.info.State = common.SlaveStateError
		}).
		OnPanic(func(r interface{}) {
			s.logger.ErrorKV("Executor panicked", "panic", r)
			s.info.State = common.SlaveStateError
		}).
		ExecWithContext(func(ctx context.Context) error {
			_, err := s.executor.Run(ctx)
			return err
		})

	return nil
}

// StopTask 停止任务
func (s *Slave) StopTask(taskID string) error {
	if s.currentTaskID != taskID {
		return fmt.Errorf("task %s is not running", taskID)
	}

	s.logger.InfoKV("Stopping task", "task_id", taskID)

	// 停止 Executor
	if s.executor != nil {
		// Executor 没有显式的 Stop 方法，但它会在 Run 完成后自动清理
		// 这里我们只需要等待任务自然完成或者通过 context 取消
		s.executor = nil
	}

	// 清理任务状态
	s.currentTaskID = ""
	s.info.State = common.SlaveStateIdle
	s.statsBuffer.SetTaskID("")

	// 关闭统计流
	if err := s.statsBuffer.CloseStream(); err != nil {
		s.logger.WarnKV("Failed to close stats stream", "error", err)
	}

	return nil
}

// getStatus 获取 Slave 状态
func (s *Slave) getStatus() *common.SlaveInfo {
	status := *s.info
	status.CurrentTaskID = s.currentTaskID

	// 获取实时的 CPU 使用情况
	if cpuPercents, err := cpu.Percent(0, false); err == nil && len(cpuPercents) > 0 {
		status.ResourceUsage = &common.ResourceUsage{
			CPUPercent: cpuPercents[0],
		}
	}

	// 获取实时的内存使用情况
	if v, err := mem.VirtualMemory(); err == nil {
		if status.ResourceUsage == nil {
			status.ResourceUsage = &common.ResourceUsage{}
		}
		status.ResourceUsage.MemoryPercent = v.UsedPercent
		status.ResourceUsage.MemoryUsed = int64(v.Used)
	}

	return &status
}

// getMemorySize 获取系统总内存大小（字节）
func getMemorySize() int64 {
	v, err := mem.VirtualMemory()
	if err != nil {
		// 如果获取失败，返回默认值 8GB
		return 8 * 1024 * 1024 * 1024
	}
	return int64(v.Total)
}

// commonStateToProtoState 转换状态为 proto 类型
func commonStateToProtoState(state common.SlaveState) pb.AgentState {
	switch state {
	case common.SlaveStateIdle:
		return pb.AgentState_AGENT_STATE_IDLE
	case common.SlaveStateRunning:
		return pb.AgentState_AGENT_STATE_RUNNING
	case common.SlaveStateStopping:
		return pb.AgentState_AGENT_STATE_STOPPING
	case common.SlaveStateError:
		return pb.AgentState_AGENT_STATE_ERROR
	case common.SlaveStateOffline:
		return pb.AgentState_AGENT_STATE_OFFLINE
	default:
		return pb.AgentState_AGENT_STATE_UNSPECIFIED
	}
}
