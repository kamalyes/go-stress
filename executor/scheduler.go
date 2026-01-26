/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 12:00:00
 * @FilePath: \go-stress\executor\scheduler.go
 * @Description: Worker调度器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"context"
	"sync"
	"time"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-stress/statistics"
)

// Scheduler Worker调度器
type Scheduler struct {
	workerCount      uint64
	requestPerWorker uint64
	rampUpDuration   time.Duration
	clientPool       *ClientPool
	handler          RequestHandler
	collector        *statistics.Collector
	reqBuilder       *RequestBuilder // 单API模式使用
	apiSelector      APISelector     // 多API模式使用
	progress         *ProgressTracker
	varResolver      *config.VariableResolver // 新增
	controller       Controller               // 控制器
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	WorkerCount      uint64
	RequestPerWorker uint64
	RampUpDuration   time.Duration
	ClientPool       *ClientPool
	Handler          RequestHandler
	Collector        *statistics.Collector
	ReqBuilder       *RequestBuilder          // 单API模式使用（可选）
	APISelector      APISelector              // 多API模式使用（可选）
	VarResolver      *config.VariableResolver // 新增
	Controller       Controller               // 控制器（可选）
}

// NewScheduler 创建调度器
func NewScheduler(cfg SchedulerConfig) *Scheduler {
	totalRequests := cfg.WorkerCount * cfg.RequestPerWorker
	ctrl := cfg.Controller
	if ctrl == nil {
		ctrl = &NoOpController{}
	}
	return &Scheduler{
		workerCount:      cfg.WorkerCount,
		requestPerWorker: cfg.RequestPerWorker,
		rampUpDuration:   cfg.RampUpDuration,
		clientPool:       cfg.ClientPool,
		handler:          cfg.Handler,
		collector:        cfg.Collector,
		reqBuilder:       cfg.ReqBuilder,
		apiSelector:      cfg.APISelector,
		progress:         NewProgressTrackerWithCollector(totalRequests, cfg.Collector, cfg.WorkerCount),
		varResolver:      cfg.VarResolver,
		controller:       ctrl,
	}
}

// Run 运行调度器
func (s *Scheduler) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, s.workerCount)

	// 启动进度跟踪
	progressCtx, cancelProgress := context.WithCancel(ctx)
	defer cancelProgress()
	go s.progress.Start(progressCtx)

	// 启动workers
	for i := uint64(0); i < s.workerCount; i++ {
		// 渐进式启动
		if s.rampUpDuration > 0 {
			delay := time.Duration(float64(s.rampUpDuration) / float64(s.workerCount) * float64(i))
			time.Sleep(delay)
		}

		wg.Add(1)
		go func(workerID uint64) {
			defer wg.Done()

			if err := s.runWorker(ctx, workerID); err != nil {
				select {
				case errChan <- err:
				default:
				}
			}
		}(i)
	}

	// 等待所有worker完成
	wg.Wait()
	close(errChan)

	// 停止进度跟踪并关闭表格
	cancelProgress()
	s.progress.Complete()

	// 检查是否有错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// runWorker 运行单个worker
func (s *Scheduler) runWorker(ctx context.Context, workerID uint64) error {
	// 从连接池获取客户端
	client, err := s.clientPool.Get()
	if err != nil {
		logger.Default.Errorf("❌ Worker %d: 获取客户端失败: %v", workerID, err)
		return err
	}
	defer s.clientPool.Put(client)

	// 创建worker，传递变量解析器和控制器
	worker := NewWorker(WorkerConfig{
		ID:          workerID,
		Client:      client,
		Handler:     s.wrapHandlerWithProgress(s.handler),
		Collector:   s.collector,
		ReqCount:    s.requestPerWorker,
		ReqBuilder:  s.reqBuilder,
		APISelector: s.apiSelector,
		Controller:  s.controller,
	}, s.varResolver)

	// 运行worker
	return worker.Run(ctx)
}

// wrapHandlerWithProgress 包装处理器以跟踪进度
func (s *Scheduler) wrapHandlerWithProgress(handler RequestHandler) RequestHandler {
	return func(ctx context.Context, req *Request) (*Response, error) {
		resp, err := handler(ctx, req)
		s.progress.Increment()
		return resp, err
	}
}
