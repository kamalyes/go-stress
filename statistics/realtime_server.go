/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:10:00
 * @FilePath: \go-stress\statistics\realtime_server.go
 * @Description: 实时报告服务器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kamalyes/go-stress/logger"
)

// RealtimeServer 实时报告服务器
type RealtimeServer struct {
	collector   *Collector
	server      *http.Server
	clients     map[chan []byte]bool
	mu          sync.RWMutex
	startTime   time.Time
	endTime     time.Time
	isCompleted bool
	isPaused    bool
	isStopped   bool
	port        int
	ctx         context.Context
	cancel      context.CancelFunc
	pauseCtx    context.Context
	pauseCancel context.CancelFunc
}

// RealtimeData 实时数据
type RealtimeData struct {
	Timestamp       int64   `json:"timestamp"`
	TotalRequests   uint64  `json:"total_requests"`
	SuccessRequests uint64  `json:"success_requests"`
	FailedRequests  uint64  `json:"failed_requests"`
	SuccessRate     float64 `json:"success_rate"`
	QPS             float64 `json:"qps"`
	AvgDuration     int64   `json:"avg_duration_ms"`
	MinDuration     int64   `json:"min_duration_ms"`
	MaxDuration     int64   `json:"max_duration_ms"`
	Elapsed         int64   `json:"elapsed_seconds"`
	IsCompleted     bool    `json:"is_completed"` // 是否已完成
	IsPaused        bool    `json:"is_paused"`    // 是否已暂停
	IsStopped       bool    `json:"is_stopped"`   // 是否已停止

	// 错误统计
	Errors map[string]uint64 `json:"errors,omitempty"`

	// 状态码统计
	StatusCodes map[int]uint64 `json:"status_codes,omitempty"`

	// 最近的响应时间点（用于实时图表）
	RecentDurations []int64 `json:"recent_durations,omitempty"`
}

// NewRealtimeServer 创建实时报告服务器
func NewRealtimeServer(collector *Collector, port int) *RealtimeServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &RealtimeServer{
		collector: collector,
		clients:   make(map[chan []byte]bool),
		startTime: time.Now(),
		port:      port,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 启动服务器
func (s *RealtimeServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/report.css", s.handleCSS)
	mux.HandleFunc("/report.js", s.handleJS)
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/api/data", s.handleData)
	mux.HandleFunc("/api/details", s.handleDetails)
	mux.HandleFunc("/api/pause", s.handlePause)
	mux.HandleFunc("/api/resume", s.handleResume)
	mux.HandleFunc("/api/stop", s.handleStop)
	mux.HandleFunc("/api/status", s.handleStatus)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		logger.Default.Info("🌐 实时报告服务器启动: http://localhost:%d", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Default.Errorf("实时报告服务器错误: %v", err)
		}
	}()

	// 启动数据广播
	go s.broadcastLoop()

	return nil
}

// MarkCompleted 标记测试完成（固定结束时间，避免 QPS 继续变化）
func (s *RealtimeServer) MarkCompleted() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isCompleted {
		s.endTime = time.Now()
		s.isCompleted = true
		logger.Default.Debug("实时服务器已标记为完成状态")
	}
}

// Stop 停止服务器
func (s *RealtimeServer) Stop() error {
	// 取消context，停止broadcastLoop
	if s.cancel != nil {
		s.cancel()
	}

	// 不直接关闭 channel，让 defer 来处理
	// 只清空 clients map，各个 goroutine 会通过 context.Done() 退出
	s.mu.Lock()
	s.clients = make(map[chan []byte]bool)
	s.mu.Unlock()

	// 关闭 HTTP 服务器
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleIndex 处理首页
func (s *RealtimeServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 使用简化HTML模板，设置为实时模式
	data := &HTMLReportData{
		IsRealtime: true,
	}

	tmpl, err := template.New("realtime").Parse(reportHTML)
	if err != nil {
		http.Error(w, "模板解析失败", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "模板执行失败", http.StatusInternalServerError)
	}
}

// handleCSS 提供CSS样式文件
func (s *RealtimeServer) handleCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write([]byte(reportCSS))
}

// handleJS 提供JavaScript脚本文件
func (s *RealtimeServer) handleJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	// 替换占位符为实时模式
	jsContent := strings.ReplaceAll(reportJS, "IS_REALTIME_PLACEHOLDER", "true")
	jsContent = strings.ReplaceAll(jsContent, "JSON_FILENAME_PLACEHOLDER", "")
	w.Write([]byte(jsContent))
}

// handleStream 处理SSE流
func (s *RealtimeServer) handleStream(w http.ResponseWriter, r *http.Request) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建客户端通道
	clientChan := make(chan []byte, 10)
	s.mu.Lock()
	s.clients[clientChan] = true
	s.mu.Unlock()

	// 使用标志记录 channel 是否已关闭
	var closeOnce sync.Once
	closeChannel := func() {
		closeOnce.Do(func() {
			close(clientChan)
		})
	}

	// 客户端断开时清理
	defer func() {
		s.mu.Lock()
		delete(s.clients, clientChan)
		s.mu.Unlock()
		closeChannel()
	}()

	// 发送初始数据
	data := s.collectData()
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	w.(http.Flusher).Flush()

	// 持续推送数据
	for {
		select {
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleData 处理数据API请求
func (s *RealtimeServer) handleData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	data := s.collectData()
	json.NewEncoder(w).Encode(data)
}

// collectData 收集当前数据
func (s *RealtimeServer) collectData() *RealtimeData {
	snapshot := s.collector.GetSnapshot()

	// 如果已完成，使用固定的总时间；否则使用当前经过的时间
	var elapsed float64
	s.mu.RLock()
	isCompleted := s.isCompleted
	isPaused := s.isPaused
	isStopped := s.isStopped
	if s.isCompleted {
		elapsed = s.endTime.Sub(s.startTime).Seconds()
	} else {
		elapsed = time.Since(s.startTime).Seconds()
	}
	s.mu.RUnlock()

	data := &RealtimeData{
		Timestamp:       time.Now().Unix(),
		TotalRequests:   snapshot.TotalRequests,
		SuccessRequests: snapshot.SuccessRequests,
		FailedRequests:  snapshot.FailedRequests,
		AvgDuration:     snapshot.AvgDuration.Milliseconds(),
		MinDuration:     snapshot.MinDuration.Milliseconds(),
		MaxDuration:     snapshot.MaxDuration.Milliseconds(),
		Elapsed:         int64(elapsed),
		IsCompleted:     isCompleted,
		IsPaused:        isPaused,
		IsStopped:       isStopped,
	}

	if snapshot.TotalRequests > 0 && elapsed > 0 {
		data.SuccessRate = float64(snapshot.SuccessRequests) / float64(snapshot.TotalRequests) * 100
		data.QPS = float64(snapshot.TotalRequests) / elapsed
	}

	// 获取错误和状态码统计
	s.collector.mu.Lock()
	data.Errors = make(map[string]uint64)
	for k, v := range s.collector.errors {
		data.Errors[k] = v
	}
	data.StatusCodes = make(map[int]uint64)
	for k, v := range s.collector.statusCodes {
		data.StatusCodes[k] = v
	}

	// 获取最近20个响应时间用于实时图表
	durationsLen := len(s.collector.durations)
	if durationsLen > 0 {
		start := 0
		if durationsLen > 20 {
			start = durationsLen - 20
		}
		data.RecentDurations = make([]int64, 0, 20)
		for i := start; i < durationsLen; i++ {
			data.RecentDurations = append(data.RecentDurations, s.collector.durations[i].Milliseconds())
		}
	}
	s.collector.mu.Unlock()

	return data
}

// handleDetails 处理请求明细API
func (s *RealtimeServer) handleDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 解析查询参数
	query := r.URL.Query()
	offset := 0
	limit := 100
	onlyErrors := query.Get("errors") == "true"

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

	details := s.collector.GetRequestDetails(offset, limit, onlyErrors)
	total := s.collector.GetRequestDetailsCount(onlyErrors)

	response := map[string]interface{}{
		"total":   total,
		"offset":  offset,
		"limit":   limit,
		"details": details,
	}

	json.NewEncoder(w).Encode(response)
}

// handlePause 处理暂停请求
func (s *RealtimeServer) handlePause(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	s.mu.Lock()
	if !s.isPaused && !s.isStopped {
		s.isPaused = true
		logger.Default.Warn("⏸  压测已暂停")
	}
	s.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "已暂停",
		"status":  "paused",
	})
}

// handleResume 处理恢复请求
func (s *RealtimeServer) handleResume(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	s.mu.Lock()
	if s.isPaused && !s.isStopped {
		s.isPaused = false
		logger.Default.Info("▶️  压测已恢复")
	}
	s.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "已恢复",
		"status":  "running",
	})
}

// handleStop 处理停止请求
func (s *RealtimeServer) handleStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	s.mu.Lock()
	s.isStopped = true
	s.isPaused = false
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	logger.Default.Warn("⏹  压测已停止")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "已停止",
		"status":  "stopped",
	})
}

// handleStatus 处理状态查询请求
func (s *RealtimeServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	s.mu.RLock()
	defer s.mu.RUnlock()

	status := "running"
	if s.isStopped {
		status = "stopped"
	} else if s.isPaused {
		status = "paused"
	} else if s.isCompleted {
		status = "completed"
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       status,
		"is_paused":    s.isPaused,
		"is_stopped":   s.isStopped,
		"is_completed": s.isCompleted,
	})
}

// IsPaused 检查是否暂停
func (s *RealtimeServer) IsPaused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isPaused
}

// IsStopped 检查是否停止
func (s *RealtimeServer) IsStopped() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isStopped
}

// broadcastLoop 广播循环
func (s *RealtimeServer) broadcastLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			// 收到退出信号
			return
		case <-ticker.C:
			s.mu.RLock()
			if len(s.clients) == 0 {
				s.mu.RUnlock()
				continue
			}
			s.mu.RUnlock()

			data := s.collectData()
			jsonData, err := json.Marshal(data)
			if err != nil {
				continue
			}

			s.mu.RLock()
			for clientChan := range s.clients {
				select {
				case clientChan <- jsonData:
				default:
					// 通道已满，跳过
				}
			}
			s.mu.RUnlock()
		}
	}
}
