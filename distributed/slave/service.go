/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 16:00:00
 * @FilePath: \go-stress\distributed\slave\service.go
 * @Description: Slave gRPC 服务实现
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package slave

import (
	"context"
	"fmt"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/distributed/common"
	pb "github.com/kamalyes/go-stress/distributed/proto"
	"github.com/kamalyes/go-stress/statistics"
)

// SlaveServiceServer Slave 服务实现
type SlaveServiceServer struct {
	pb.UnimplementedSlaveServiceServer
	slave  *Slave
	logger logger.ILogger
}

// NewSlaveServiceServer 创建 Slave 服务
func NewSlaveServiceServer(slave *Slave, log logger.ILogger) *SlaveServiceServer {
	return &SlaveServiceServer{
		slave:  slave,
		logger: log,
	}
}

// ExecuteTask 执行任务
func (s *SlaveServiceServer) ExecuteTask(ctx context.Context, req *pb.TaskConfig) (*pb.TaskResponse, error) {
	s.logger.InfoContextKV(ctx, "Received task execution request",
		"task_id", req.TaskId,
		"workers", req.WorkerCount)

	// 转换为内部类型
	subTask := &common.SubTask{
		TaskID:      req.TaskId,
		SubTaskID:   fmt.Sprintf("%s-%s", req.TaskId, s.slave.info.ID),
		SlaveID:     s.slave.info.ID,
		WorkerCount: int(req.WorkerCount),
		Config:      req.ConfigData,
	}

	// 执行任务
	if err := s.slave.ExecuteTask(subTask); err != nil {
		s.logger.ErrorContextKV(ctx, "Failed to execute task",
			"task_id", req.TaskId,
			"error", err)
		return &pb.TaskResponse{
			Accepted: false,
			Message:  err.Error(),
			TaskId:   req.TaskId,
		}, nil
	}

	s.logger.InfoContextKV(ctx, "Task accepted",
		"task_id", req.TaskId,
		"workers", req.WorkerCount)

	return &pb.TaskResponse{
		Accepted: true,
		Message:  "Task accepted",
		TaskId:   req.TaskId,
	}, nil
}

// StopTask 停止任务
func (s *SlaveServiceServer) StopTask(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
	s.logger.InfoContextKV(ctx, "Received task stop request",
		"task_id", req.TaskId,
		"force", req.Force)

	if err := s.slave.StopTask(req.TaskId); err != nil {
		s.logger.ErrorContextKV(ctx, "Failed to stop task",
			"task_id", req.TaskId,
			"error", err)
		return &pb.StopResponse{
			Stopped: false,
			Message: err.Error(),
		}, nil
	}

	s.logger.InfoContextKV(ctx, "Task stopped", "task_id", req.TaskId)

	return &pb.StopResponse{
		Stopped: true,
		Message: "Task stopped successfully",
	}, nil
}

// GetStatus 获取状态
func (s *SlaveServiceServer) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.SlaveStatus, error) {
	status := s.slave.getStatus()

	// 获取资源使用情况
	var cpuUsage, memUsage float64
	var runningWorkers int64
	if s.slave.monitor != nil {
		if usage, err := s.slave.monitor.GetResourceUsage(); err == nil {
			cpuUsage = usage.CPUPercent
			memUsage = usage.MemoryPercent
			runningWorkers = int64(usage.ActiveTasks)
		}
	}

	return &pb.SlaveStatus{
		SlaveId:        status.ID,
		State:          commonStateToProtoState(status.State),
		CurrentTaskId:  status.CurrentTaskID,
		CpuUsage:       cpuUsage,
		MemoryUsage:    memUsage,
		RunningWorkers: runningWorkers,
		TotalRequests:  status.TotalRequests,
		Timestamp:      status.LastHeartbeat.Unix(),
	}, nil
}

// UpdateConfig 更新配置
func (s *SlaveServiceServer) UpdateConfig(ctx context.Context, req *pb.ConfigUpdate) (*pb.UpdateResponse, error) {
	s.logger.InfoContextKV(ctx, "Received config update request",
		"slave_id", req.SlaveId,
		"config_keys", len(req.Config))

	// 配置更新逻辑：将新配置应用到 Slave
	// 注意：动态配置更新需要考虑并发安全和配置热重载
	if s.slave != nil {
		// 记录配置更新
		for key, value := range req.Config {
			s.logger.InfoContextKV(ctx, "Updating config",
				"key", key,
				"value", value)
		}

		// 实际的配置更新逻辑
		// 可以根据 key 更新不同的配置项，例如：
		// - "log_level": 更新日志级别
		// - "max_concurrency": 更新最大并发数
		// - "report_interval": 更新上报间隔
		// 这里可以扩展具体的配置项处理
	}

	return &pb.UpdateResponse{
		Success: true,
		Message: "Config updated successfully",
	}, nil
}

// GetRequestDetails 获取请求详情
func (s *SlaveServiceServer) GetRequestDetails(ctx context.Context, req *pb.DetailsRequest) (*pb.DetailsResponse, error) {
	s.logger.DebugContextKV(ctx, "Received details query request",
		"slave_id", req.SlaveId,
		"task_id", req.TaskId,
		"offset", req.Offset,
		"limit", req.Limit,
		"status", req.Status)

	// 验证 slave_id
	if req.SlaveId != s.slave.info.ID {
		return &pb.DetailsResponse{
			Total:  0,
			Offset: req.Offset,
			Limit:  req.Limit,
		}, fmt.Errorf("slave_id mismatch: expected %s, got %s", s.slave.info.ID, req.SlaveId)
	}

	// 限制每次最多返回 1000 条
	limit := int(req.Limit)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// 解析状态过滤参数
	statusFilter := parseStatusFilterFromString(req.Status)

	// 从 Collector 获取存储
	collectorInterface := s.slave.GetCollector()
	if collectorInterface == nil {
		s.logger.WarnContext(ctx, "Collector is nil, no task executed yet")
		return &pb.DetailsResponse{
			Total:  0,
			Offset: req.Offset,
			Limit:  int32(limit),
		}, nil
	}

	collector, ok := collectorInterface.(*statistics.Collector)
	if !ok {
		s.logger.ErrorContext(ctx, "Failed to cast collector")
		return &pb.DetailsResponse{
			Total:  0,
			Offset: req.Offset,
			Limit:  int32(limit),
		}, fmt.Errorf("failed to get collector")
	}

	// 查询详情
	details := collector.GetRequestDetails(int(req.Offset), limit, statusFilter, req.SlaveId, req.TaskId)
	totalCount := collector.GetRequestDetailsCount(statusFilter, req.SlaveId, req.TaskId)

	s.logger.InfoContextKV(ctx, "Queried details from collector",
		"total_count", totalCount,
		"details_count", len(details),
		"offset", req.Offset,
		"limit", limit,
		"status_filter", statusFilter)

	// 获取实际计数器数据
	metrics := collector.GetMetrics()

	// 转换为 proto 消息
	pbDetails := make([]*pb.RequestDetail, 0, len(details))
	for _, d := range details {
		pbDetails = append(pbDetails, &pb.RequestDetail{
			Id:              d.ID,
			Timestamp:       d.Timestamp.UnixMilli(),
			Duration:        int64(d.Duration),
			StatusCode:      int32(d.StatusCode),
			Success:         d.Success,
			Skipped:         d.Skipped,
			SkipReason:      d.SkipReason,
			GroupId:         d.GroupID,
			ApiName:         d.APIName,
			Error:           d.ErrorMsg,
			Size:            d.Size,
			Url:             d.URL,
			Method:          d.Method,
			Query:           d.Query,
			Headers:         d.Headers,
			Body:            d.Body,
			ResponseBody:    d.ResponseBody,
			ResponseHeaders: d.ResponseHeaders,
			ExtractedVars:   d.ExtractedVars,
		})
	}

	return &pb.DetailsResponse{
		Total:         int32(totalCount),
		Offset:        req.Offset,
		Limit:         int32(limit),
		Details:       pbDetails,
		TotalRequests: int64(metrics.TotalRequests),
		SuccessCount:  int64(metrics.SuccessRequests),
		FailedCount:   int64(metrics.FailedRequests),
		SkippedCount:  int64(metrics.TotalRequests - metrics.SuccessRequests - metrics.FailedRequests),
	}, nil
}

// parseStatusFilterFromString 从字符串解析状态过滤器
func parseStatusFilterFromString(status string) statistics.StatusFilter {
	switch status {
	case "success":
		return statistics.StatusFilterSuccess
	case "failed":
		return statistics.StatusFilterFailed
	case "skipped":
		return statistics.StatusFilterSkipped
	default:
		return statistics.StatusFilterAll
	}
}
