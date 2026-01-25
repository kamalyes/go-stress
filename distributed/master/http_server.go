/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 20:00:00
 * @FilePath: \go-stress\distributed\master\http_server.go
 * @Description: Master HTTP API 服务器（分布式管理）
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/distributed/common"
	pb "github.com/kamalyes/go-stress/distributed/proto"
	"github.com/kamalyes/go-stress/statistics"
)

// HTTPServer 分布式管理 HTTP 服务器
type HTTPServer struct {
	master         *Master
	server         *http.Server
	realtimeServer *statistics.RealtimeServer // 复用现有的实时报告服务器
	logger         logger.ILogger
}

// NewHTTPServer 创建 HTTP 服务器
func NewHTTPServer(master *Master, port int, log logger.ILogger) *HTTPServer {
	hs := &HTTPServer{
		master: master,
		logger: log,
	}

	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/v1/slaves", hs.handleSlaves)
	mux.HandleFunc("/api/v1/slaves/", hs.handleSlaveDetail)
	mux.HandleFunc("/api/v1/tasks", hs.handleTasks)
	mux.HandleFunc("/api/v1/tasks/", hs.handleTaskDetail)
	// 注意：必须在 /api/v1/tasks/ 之后注册，避免路由冲突

	// 分布式管理页面
	mux.HandleFunc("/", hs.handleIndex)
	mux.HandleFunc("/distributed", hs.handleDistributed)
	mux.HandleFunc("/distributed/tasks/", hs.handleTaskDetailPage)
	mux.HandleFunc("/distributed/slaves/", hs.handleSlaveDetailPage)

	// 实时报告页面（展示聚合数据）
	mux.HandleFunc("/realtime", hs.handleRealtime)
	mux.HandleFunc("/api/realtime/stats", hs.handleRealtimeStats)
	mux.HandleFunc("/api/details", hs.handleDetails)

	// 静态资源路由（用于实时报告页面）
	mux.HandleFunc("/report.css", hs.handleReportCSS)
	mux.HandleFunc("/report.js", hs.handleReportJS)
	mux.HandleFunc("/report_actions.js", hs.handleReportActionsJS)
	mux.HandleFunc("/http-client.js", hs.handleHTTPClientJS)
	mux.HandleFunc("/distributed.css", hs.handleDistributedCSS)
	mux.HandleFunc("/distributed.js", hs.handleDistributedJS)
	mux.HandleFunc("/task_detail.css", hs.handleTaskDetailCSS)
	mux.HandleFunc("/task_detail.js", hs.handleTaskDetailJS)
	mux.HandleFunc("/slave_detail.js", hs.handleSlaveDetailJS)

	hs.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: hs.corsMiddleware(hs.logMiddleware(mux)),
	}

	// 启动实时报告服务器（复用 statistics.RealtimeServer）
	// 注意：这里需要创建一个适配器，将 Master Collector 数据提供给 RealtimeServer
	// 暂时传nil,后续实现适配器
	hs.realtimeServer = nil // TODO: 实现 Collector 适配器

	return hs
}

// Start 启动 HTTP 服务器
func (hs *HTTPServer) Start() error {
	hs.logger.InfoKV("Starting HTTP server", "addr", hs.server.Addr)

	// 启动实时报告服务器
	if hs.realtimeServer != nil {
		if err := hs.realtimeServer.Start(); err != nil {
			return fmt.Errorf("failed to start realtime server: %w", err)
		}
	}

	go func() {
		if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			hs.logger.ErrorKV("HTTP server error", "error", err)
		}
	}()
	return nil
}

// Stop 停止 HTTP 服务器
func (hs *HTTPServer) Stop() error {
	// 停止实时报告服务器
	if hs.realtimeServer != nil {
		if err := hs.realtimeServer.Stop(); err != nil {
			hs.logger.ErrorKV("Failed to stop realtime server", "error", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return hs.server.Shutdown(ctx)
}

// ===== API Handlers =====

// handleSlaves 处理 Slave 列表请求
func (hs *HTTPServer) handleSlaves(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// 获取所有 Slave
		slaves := hs.master.GetSlavePool().GetAllSlaves()

		// 统计各状态数量
		var idleCount, runningCount, offlineCount, errorCount int
		for _, slave := range slaves {
			switch slave.State {
			case common.SlaveStateIdle:
				idleCount++
			case common.SlaveStateRunning, common.SlaveStateBusy:
				runningCount++
			case common.SlaveStateOffline, common.SlaveStateUnreachable:
				offlineCount++
			case common.SlaveStateError:
				errorCount++
			default:
				// 其他状态归为在线
				idleCount++
			}
		}

		hs.writeJSON(w, http.StatusOK, map[string]interface{}{
			"slaves": slaves,
			"total":  len(slaves),
			"stats": map[string]int{
				"idle":    idleCount,
				"running": runningCount,
				"offline": offlineCount,
				"error":   errorCount,
				"online":  idleCount + runningCount, // 在线总数 = 空闲 + 运行中
			},
		})
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSlaveDetail 处理单个 Slave 详情请求
func (hs *HTTPServer) handleSlaveDetail(w http.ResponseWriter, r *http.Request) {
	slaveID := strings.TrimPrefix(r.URL.Path, "/api/v1/slaves/")
	if slaveID == "" {
		http.Error(w, "Slave ID required", http.StatusBadRequest)
		return
	}

	// 从 SlavePool 获取 Slave 信息
	slave := hs.master.GetSlavePool().GetSlave(slaveID)
	if slave == nil {
		http.Error(w, "Slave not found", http.StatusNotFound)
		return
	}

	hs.writeJSON(w, http.StatusOK, slave)
}

// handleTasks 处理任务列表和创建请求
func (hs *HTTPServer) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 获取所有任务（pending + running + complete + failed）
		tasks := hs.master.GetTaskQueue().GetAllTasks()
		hs.writeJSON(w, http.StatusOK, map[string]interface{}{
			"tasks": tasks,
			"total": len(tasks),
		})

	case http.MethodPost:
		// 创建任务（不分发，等待手动启动）
		var req struct {
			ConfigFile string `json:"config_file"` // 配置文件路径或内容
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// 解析配置
		task, err := hs.parseTaskConfig(req.ConfigFile)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse config: %v", err), http.StatusBadRequest)
			return
		}

		// 只创建任务，不立即分发（等待用户手动启动）
		if err := hs.master.GetTaskQueue().Submit(task); err != nil {
			http.Error(w, fmt.Sprintf("Failed to create task: %v", err), http.StatusInternalServerError)
			return
		}

		hs.logger.InfoKV("Task created", "task_id", task.ID, "protocol", task.Protocol)

		hs.writeJSON(w, http.StatusCreated, map[string]interface{}{
			"task_id": task.ID,
			"message": "Task created successfully, use /api/v1/tasks/{id}/start to start execution",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTaskDetail 处理任务详情和操作
func (hs *HTTPServer) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}

	taskID := parts[0]

	switch r.Method {
	case http.MethodGet:
		// 获取任务详情
		if len(parts) > 1 && parts[1] == "report" {
			// TODO: 获取聚合报告（从 Master Collector 获取汇聚数据）
			http.Error(w, "Task report not implemented yet", http.StatusNotImplemented)
			return
		} else if len(parts) > 2 && parts[1] == "slaves" {
			// 获取某个 Slave 的统计数据
			slaveID := parts[2]
			stats, exists := hs.master.GetCollector().GetSlaveStats(slaveID)
			if !exists {
				http.Error(w, "Slave stats not found", http.StatusNotFound)
				return
			}
			hs.writeJSON(w, http.StatusOK, stats)
		} else {
			// 获取任务基本信息
			task, exists := hs.master.GetTaskQueue().Get(taskID)
			if !exists {
				http.Error(w, "Task not found", http.StatusNotFound)
				return
			}
			hs.writeJSON(w, http.StatusOK, task)
		}

	case http.MethodPost:
		// 启动任务（支持 Slave 选择）
		if len(parts) > 1 && parts[1] == "start" {
			hs.handleTaskStart(w, r, taskID)
			return
		}
		// 重试任务
		if len(parts) > 1 && parts[1] == "retry" {
			hs.handleTaskRetry(w, r, taskID)
			return
		}
		http.Error(w, "Invalid action", http.StatusBadRequest)

	case http.MethodDelete:
		// 停止任务（通过 gRPC 通知所有 Slave 停止）
		if err := hs.master.StopTask(taskID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to stop task: %v", err), http.StatusInternalServerError)
			return
		}
		hs.writeJSON(w, http.StatusOK, map[string]string{
			"message": "Task stopped",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ===== Page Handlers =====

// handleIndex 处理首页
func (hs *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	// 重定向到分布式管理页面
	http.Redirect(w, r, "/distributed", http.StatusFound)
}

// handleDistributed 处理分布式管理主页
func (hs *HTTPServer) handleDistributed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(statistics.GetDistributedHTML()))
}

// handleTaskDetailPage 处理任务详情页面
func (hs *HTTPServer) handleTaskDetailPage(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimPrefix(r.URL.Path, "/distributed/tasks/")
	if taskID == "" {
		http.Redirect(w, r, "/distributed", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(statistics.GetTaskDetailHTML()))
}

// handleSlaveDetailPage 处理Slave详情页面
func (hs *HTTPServer) handleSlaveDetailPage(w http.ResponseWriter, r *http.Request) {
	slaveID := strings.TrimPrefix(r.URL.Path, "/distributed/slaves/")
	if slaveID == "" {
		http.Redirect(w, r, "/distributed", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(statistics.GetSlaveDetailHTML()))
}

// handleTaskStart 处理任务启动请求
func (hs *HTTPServer) handleTaskStart(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体（可选参数）
	var req struct {
		SlaveIDs    []string `json:"slave_ids"`    // 指定 Slave ID（可选）
		SlaveRegion string   `json:"slave_region"` // 指定区域（可选）
	}

	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			hs.logger.WarnKV("Failed to parse request body", "error", err)
		}
	}

	// 获取任务
	task, exists := hs.master.GetTaskQueue().Get(taskID)
	if !exists || task == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// 检查任务状态
	if task.State != "pending" {
		http.Error(w, fmt.Sprintf("Task is already %s, cannot start", task.State), http.StatusBadRequest)
		return
	}

	hs.logger.InfoKV("Starting task",
		"task_id", taskID,
		"slave_ids", req.SlaveIDs,
		"slave_ids_count", len(req.SlaveIDs),
		"slave_region", req.SlaveRegion)

	// 构建启动选项
	var options *TaskStartOptions
	if len(req.SlaveIDs) > 0 || req.SlaveRegion != "" {
		options = &TaskStartOptions{
			SlaveIDs:    req.SlaveIDs,
			SlaveRegion: req.SlaveRegion,
		}
		hs.logger.InfoKV("Task start options created",
			"slave_ids", options.SlaveIDs,
			"slave_region", options.SlaveRegion)
	} else {
		hs.logger.InfoKV("No slave filter specified, will use default selection", "task_id", taskID)
	}

	// 启动任务（使用指定的 Slave 过滤条件）
	if err := hs.master.StartTaskWithOptions(taskID, options); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start task: %v", err), http.StatusInternalServerError)
		return
	}

	hs.writeJSON(w, http.StatusOK, map[string]interface{}{
		"task_id": taskID,
		"message": "Task started successfully",
		"state":   "running",
	})
}

// handleTaskRetry 处理任务重试请求
func (hs *HTTPServer) handleTaskRetry(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取原任务
	oldTask, exists := hs.master.GetTaskQueue().Get(taskID)
	if !exists || oldTask == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// 检查任务状态：只允许重试已完成、失败或停止的任务
	if oldTask.State != "completed" && oldTask.State != "failed" && oldTask.State != "stopped" {
		http.Error(w, fmt.Sprintf("Task is %s, can only retry completed/failed/stopped tasks", oldTask.State), http.StatusBadRequest)
		return
	}

	hs.logger.InfoKV("Retrying task",
		"original_task_id", taskID,
		"original_state", oldTask.State)

	// 创建新任务（复制原任务配置）
	newTask := &common.Task{
		ID:           hs.master.idGenerator.GenerateRequestID(),
		Protocol:     oldTask.Protocol,
		Target:       oldTask.Target,
		TotalWorkers: oldTask.TotalWorkers,
		ConfigData:   oldTask.ConfigData,
		Metadata:     make(map[string]string),
	}

	// 添加重试元数据
	if newTask.Metadata == nil {
		newTask.Metadata = make(map[string]string)
	}
	newTask.Metadata["retry_from"] = taskID
	newTask.Metadata["retry_reason"] = string(oldTask.State)

	// 提交新任务
	if err := hs.master.GetTaskQueue().Submit(newTask); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit retry task: %v", err), http.StatusInternalServerError)
		return
	}

	hs.logger.InfoKV("Retry task created",
		"original_task_id", taskID,
		"new_task_id", newTask.ID)

	hs.writeJSON(w, http.StatusOK, map[string]interface{}{
		"original_task_id": taskID,
		"new_task_id":      newTask.ID,
		"message":          "Task retry submitted successfully",
		"state":            "pending",
	})
}

// handleRealtime 处理实时报告页面
func (hs *HTTPServer) handleRealtime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 创建空的 Report 用于生成 HTML 模板
	// 实际数据通过 /api/realtime/stats 动态加载
	emptyReport := &statistics.Report{}

	formatter := &statistics.HTMLFormatter{
		IsRealtime:   true,
		JSONFilename: "",
	}

	htmlBytes, err := formatter.Format(emptyReport)
	if err != nil {
		hs.logger.ErrorKV("Failed to generate realtime HTML", "error", err)
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	w.Write(htmlBytes)
}

// handleRealtimeStats 处理实时统计数据API
func (hs *HTTPServer) handleRealtimeStats(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	query := r.URL.Query()
	slaveID := query.Get("slave_id")
	taskID := query.Get("task_id")

	// 从 Master Collector 获取聚合统计数据
	stats := hs.master.GetCollector().GetAggregatedStats()

	// 如果指定了 slave_id，只返回该 Slave 的数据
	if slaveID != "" {
		if slaveStats, ok := stats.BySlave[slaveID]; ok {
			response := map[string]interface{}{
				"total_requests":   slaveStats.TotalRequests,
				"success_requests": slaveStats.SuccessRequests,
				"failed_requests":  slaveStats.FailedRequests,
				"success_rate":     slaveStats.SuccessRate,
				"qps":              slaveStats.QPS,
				"avg_latency":      slaveStats.AvgLatency,
				"min_latency":      slaveStats.MinLatency,
				"max_latency":      slaveStats.MaxLatency,
				"p50_latency":      slaveStats.P50Latency,
				"p90_latency":      slaveStats.P90Latency,
				"p95_latency":      slaveStats.P95Latency,
				"p99_latency":      slaveStats.P99Latency,
				"status_codes":     slaveStats.StatusCodes,
				"errors":           slaveStats.ErrorTypes,
				"total_agents":     1,
				"slave_id":         slaveID,
				"task_id":          taskID,
			}
			hs.writeJSON(w, http.StatusOK, response)
			return
		} else {
			hs.writeJSON(w, http.StatusNotFound, map[string]interface{}{
				"error": fmt.Sprintf("Slave %s not found", slaveID),
			})
			return
		}
	}

	// 返回聚合数据
	response := map[string]interface{}{
		"total_requests":   stats.TotalRequests,
		"success_requests": stats.SuccessRequests,
		"failed_requests":  stats.FailedRequests,
		"success_rate":     stats.SuccessRate,
		"qps":              stats.TotalQPS,
		"avg_latency":      stats.AvgLatency,
		"min_latency":      stats.MinLatency,
		"max_latency":      stats.MaxLatency,
		"p50_latency":      stats.P50Latency,
		"p90_latency":      stats.P90Latency,
		"p95_latency":      stats.P95Latency,
		"p99_latency":      stats.P99Latency,
		"status_codes":     stats.StatusCodes,
		"errors":           stats.ErrorTypes,
		"total_agents":     stats.TotalAgents,
		"by_slave":         stats.BySlave,
		"task_id":          taskID,
	}

	hs.writeJSON(w, http.StatusOK, response)
}

// handleDetails 处理请求详情查询（从 Slave 节点获取）
func (hs *HTTPServer) handleDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 解析查询参数
	query := r.URL.Query()
	slaveID := query.Get("slave_id") // 必需参数
	offset := 0
	limit := 100
	status := query.Get("status") // all | success | failed | skipped

	if o := query.Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}
	if l := query.Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	// 限制每次最多返回1000条
	if limit > 1000 {
		limit = 1000
	}

	// 如果没有指定 slave_id，返回错误
	if slaveID == "" {
		http.Error(w, "slave_id parameter is required", http.StatusBadRequest)
		return
	}

	// 获取 Slave 的 gRPC 客户端
	client, err := hs.master.GetSlaveClient(slaveID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Slave %s not found", slaveID), http.StatusNotFound)
		return
	}

	// 调用 Slave 的 GetRequestDetails RPC
	hs.logger.InfoKV("Calling slave GetRequestDetails",
		"slave_id", slaveID,
		"offset", offset,
		"limit", limit,
		"status", status)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := client.GetRequestDetails(ctx, &pb.DetailsRequest{
		SlaveId: slaveID,
		Offset:  int32(offset),
		Limit:   int32(limit),
		Status:  status,
	})

	if err != nil {
		hs.logger.ErrorKV("Failed to get details from slave", "slave_id", slaveID, "error", err)
		http.Error(w, fmt.Sprintf("Failed to get details: %v", err), http.StatusInternalServerError)
		return
	}

	hs.logger.InfoKV("Received details from slave",
		"slave_id", slaveID,
		"total", resp.Total,
		"details_count", len(resp.Details))

	// 转换为前端需要的格式
	response := map[string]interface{}{
		"total":          resp.Total,
		"offset":         resp.Offset,
		"limit":          resp.Limit,
		"details":        resp.Details,
		"total_requests": resp.TotalRequests,
		"success_count":  resp.SuccessCount,
		"failed_count":   resp.FailedCount,
		"skipped_count":  resp.SkippedCount,
	}

	json.NewEncoder(w).Encode(response)
}

// handleReportCSS 提供实时报告的 CSS 文件
func (hs *HTTPServer) handleReportCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write([]byte(statistics.GetReportCSS()))
}

// handleReportJS 提供实时报告的 JS 文件
func (hs *HTTPServer) handleReportJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte(statistics.GetReportJS()))
}

// handleReportActionsJS 提供实时报告的操作 JS 文件
func (hs *HTTPServer) handleReportActionsJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte(statistics.GetReportActionsJS()))
}

// handleHTTPClientJS 提供 HTTP 客户端 JS 文件
func (hs *HTTPServer) handleHTTPClientJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte(statistics.GetHTTPClientJS()))
}

// handleTaskDetailCSS 提供任务详情页面的 CSS 文件
func (hs *HTTPServer) handleTaskDetailCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write([]byte(statistics.GetTaskDetailCSS()))
}

// handleDistributedCSS 提供分布式管理页面的 CSS 文件
func (hs *HTTPServer) handleDistributedCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write([]byte(statistics.GetDistributedCSS()))
}

// handleDistributedJS 提供分布式管理页面的 JS 文件
func (hs *HTTPServer) handleDistributedJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte(statistics.GetDistributedJS()))
}

// handleTaskDetailJS 提供任务详情页面的 JS 文件
func (hs *HTTPServer) handleTaskDetailJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte(statistics.GetTaskDetailJS()))
}

// handleSlaveDetailJS 提供 Slave 详情页面的 JS 文件
func (hs *HTTPServer) handleSlaveDetailJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte(statistics.GetSlaveDetailJS()))
}

// ===== Middleware =====

func (hs *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (hs *HTTPServer) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		hs.logger.DebugKV("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start))
	})
}

// ===== Helpers =====

// parseTaskConfig 解析任务配置(支持 YAML 字符串、JSON 字符串或文件路径)
func (hs *HTTPServer) parseTaskConfig(configInput string) (*common.Task, error) {
	// 创建配置加载器（复用单机模式的完整逻辑）
	loader := config.NewLoader()

	var cfg *config.Config
	var err error

	// 优先尝试作为文件路径
	if _, statErr := os.Stat(configInput); statErr == nil {
		// 使用标准 LoadFromFile，包含完整的变量解析、API合并、验证流程
		cfg, err = loader.LoadFromFile(configInput)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
		return hs.buildTask(cfg)
	}

	// 不是文件，尝试作为 YAML 字符串解析
	cfg, err = loader.LoadFromBytes([]byte(configInput), "yaml")
	if err == nil {
		return hs.buildTask(cfg)
	}
	yamlErr := err

	// YAML 解析失败，尝试 JSON
	cfg, err = loader.LoadFromBytes([]byte(configInput), "json")
	if err == nil {
		return hs.buildTask(cfg)
	}
	jsonErr := err

	return nil, fmt.Errorf("invalid config: YAML error: %v, JSON error: %v", yamlErr, jsonErr)
}

// buildTask 从配置构建任务
func (hs *HTTPServer) buildTask(cfg *config.Config) (*common.Task, error) {

	// 序列化为 ConfigData
	configData, err := json.Marshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// 获取持续时间（优先使用Duration，其次使用Requests计算）
	durationSeconds := int(cfg.Duration.Seconds())
	if durationSeconds == 0 && cfg.Requests > 0 {
		// 如果没有指定Duration，根据Requests估算（假设每秒1000请求）
		durationSeconds = int(cfg.Requests / cfg.Concurrency)
		if durationSeconds == 0 {
			durationSeconds = 60 // 默认60秒
		}
	}

	// 获取RampUp时间
	rampUpSeconds := 0
	if cfg.Advanced != nil {
		rampUpSeconds = int(cfg.Advanced.RampUp.Seconds())
	}

	// 构建任务
	task := &common.Task{
		ID:           hs.master.GenerateTaskID(),
		Protocol:     string(cfg.Protocol),
		Target:       cfg.URL, // URL 已经在 MergeAPIsWithCommon 中正确构建（Host+Path）
		TotalWorkers: int(cfg.Concurrency),
		Duration:     durationSeconds,
		RampUp:       rampUpSeconds,
		ConfigData:   configData,
	}

	return task, nil
}

func (hs *HTTPServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		hs.logger.ErrorKV("Failed to encode JSON", "error", err)
	}
}
