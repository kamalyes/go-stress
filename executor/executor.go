/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:25:08
 * @FilePath: \go-stress\executor\executor.go
 * @Description: 压测执行器 - 核心编排器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-stress/protocol"
	"github.com/kamalyes/go-stress/statistics"
	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-stress/verify"
	"github.com/kamalyes/go-toolbox/pkg/breaker"
	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// StatsReporter 统计上报接口（用于分布式模式）
type StatsReporter interface {
	Add(result *types.RequestResult)
	SetTaskID(taskID string)
}

// Executor 压测执行器（核心编排器）
// 职责：
// 1. 组装各个组件（连接池、中间件、调度器）
// 2. 编排整体压测流程
// 3. 生成最终报告
type Executor struct {
	config         *config.Config
	collector      *statistics.Collector
	scheduler      *Scheduler
	pool           *ClientPool
	realtimeServer *statistics.RealtimeServer
	// 分布式相关
	statsReporter StatsReporter // 用于分布式模式下的统计上报
	isDistributed bool          // 是否为分布式模式
}

// NewExecutor 创建执行器（默认内存模式）
func NewExecutor(cfg *config.Config) (*Executor, error) {
	return NewExecutorWithMemoryStorage(cfg)
}

// NewExecutorWithMemoryStorage 创建内存存储模式的执行器
func NewExecutorWithMemoryStorage(cfg *config.Config) (*Executor, error) {
	collector := statistics.NewCollectorWithMemoryStorage("local")
	return newExecutor(cfg, collector)
}

// NewExecutorWithSQLiteStorage 创建 SQLite 存储模式的执行器
func NewExecutorWithSQLiteStorage(cfg *config.Config, dbPath string) (*Executor, error) {
	collector := statistics.NewCollectorWithStorage(dbPath, "local")
	return newExecutor(cfg, collector)
}

// newExecutor 内部构造函数（通用逻辑）
func newExecutor(cfg *config.Config, collector *statistics.Collector) (*Executor, error) {

	// 1. 创建客户端工厂
	clientFactory := createClientFactory(cfg)

	// 2. 创建连接池
	pool := NewClientPool(clientFactory, int(cfg.Concurrency))

	// 3. 构建中间件链
	handler, err := buildMiddlewareChain(cfg, clientFactory)
	if err != nil {
		return nil, fmt.Errorf("构建中间件链失败: %w", err)
	}

	// 4. 创建API选择器或请求构建器
	var reqBuilder *RequestBuilder
	var apiSelector APISelector

	if len(cfg.APIs) > 0 {
		// 多API模式：创建API选择器
		apiSelector = CreateAPISelector(cfg)
		logger.Default.Info("📋 多API模式: 共%d个API配置", len(cfg.APIs))
	} else {
		// 单API模式：创建请求构建器（向后兼容）
		reqBuilder = NewRequestBuilder(cfg.URL, cfg.Method, cfg.Headers, cfg.Body)
		logger.Default.Info("📋 单API模式")
	}

	// 5. 创建调度器
	var rampUp time.Duration
	if cfg.Advanced != nil {
		rampUp = cfg.Advanced.RampUp
	}

	// 直接从 config 取变量解析器
	scheduler := NewScheduler(SchedulerConfig{
		WorkerCount:      cfg.Concurrency,
		RequestPerWorker: cfg.Requests,
		RampUpDuration:   rampUp,
		ClientPool:       pool,
		Handler:          handler,
		Collector:        collector,
		ReqBuilder:       reqBuilder,
		APISelector:      apiSelector,
		VarResolver:      cfg.VarResolver,
		Controller:       nil, // 稍后设置
	})

	exec := &Executor{
		config:        cfg,
		collector:     collector,
		scheduler:     scheduler,
		pool:          pool,
		isDistributed: false, // 默认非分布式模式
	}
	return exec, nil
}

// createClientFactory 创建客户端工厂
func createClientFactory(cfg *config.Config) ClientFactory {
	return func() (types.Client, error) {
		switch cfg.Protocol {
		case types.ProtocolHTTP:
			return protocol.NewHTTPClient(cfg)
		case types.ProtocolGRPC:
			return protocol.NewGRPCClient(cfg)
		default:
			return nil, fmt.Errorf("不支持的协议: %s", cfg.Protocol)
		}
	}
}

// buildMiddlewareChain 构建中间件链
// 执行顺序：熔断器 -> 重试器 -> 验证器 -> 客户端
func buildMiddlewareChain(cfg *config.Config, factory ClientFactory) (RequestHandler, error) {
	// 创建临时客户端用于中间件
	client, err := factory()
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %w", err)
	}

	chain := NewMiddlewareChain()

	// 1. 熔断器中间件（最外层）
	if cfg.Advanced != nil && cfg.Advanced.EnableBreaker {
		circuit := breaker.New("stress-test", breaker.Config{
			MaxFailures:       cfg.Advanced.MaxFailures,
			ResetTimeout:      cfg.Advanced.ResetTimeout,
			HalfOpenSuccesses: 2,
		})
		chain.Use(BreakerMiddleware(circuit))
	}

	// 2. 重试中间件
	if cfg.Advanced != nil && cfg.Advanced.EnableRetry {
		retrier := retry.NewRunner[error]()
		chain.Use(RetryMiddleware(retrier))
	}

	// 3. 验证中间件
	if cfg.Verify != nil && cfg.Verify.Type != "" {
		verifier, err := verify.Get(types.VerifyType(cfg.Verify.Type))
		if err != nil {
			return nil, fmt.Errorf("获取验证器失败: %w", err)
		}
		chain.Use(VerifyMiddleware(verifier))
	}

	// 4. 构建处理器（客户端是最底层）
	handler := chain.Build(ClientMiddleware(client))

	return handler, nil
}

// Run 执行压测
func (e *Executor) Run(ctx context.Context) (*statistics.Report, error) {
	// 打印启动信息
	e.printStartInfo()

	// 启动实时报告服务器
	port := 8088 // 默认端口
	if e.config.Advanced != nil && e.config.Advanced.RealtimePort > 0 {
		port = e.config.Advanced.RealtimePort
	}
	e.realtimeServer = statistics.NewRealtimeServer(e.collector, port)
	if err := e.realtimeServer.Start(); err != nil {
		logger.Default.Warnf("⚠️  启动实时报告服务器失败: %v", err)
	} else {
		// 将RealtimeServer设置为控制器
		e.scheduler.controller = e.realtimeServer
		// 自动打开浏览器
		realtimeURL := fmt.Sprintf("http://localhost:%d", port)
		logger.Default.Info("🌐 实时监控地址: %s", realtimeURL)
		go openBrowser(realtimeURL)
	}

	startTime := time.Now()

	// 运行调度器
	err := e.scheduler.Run(ctx)

	totalDuration := time.Since(startTime)

	// 标记测试完成（固定 QPS 计算时间）
	if e.realtimeServer != nil {
		e.realtimeServer.MarkCompleted()
	}

	// 清理资源
	e.pool.Close()

	// 生成报告（即使出错也要生成）
	report := e.collector.GenerateReport(totalDuration)

	// 检查是否因为context取消而中断
	if err != nil {
		// 如果是用户主动取消，不关闭实时服务器，返回当前报告
		if strings.Contains(err.Error(), "context canceled") {
			logger.Default.Warn("\n⚠️  压测已被用户中断")
			logger.Default.Info("📊 正在保存当前统计数据...")
			return report, fmt.Errorf("执行压测失败: %w", err)
		}
		// 其他错误，关闭服务器
		e.realtimeServer.Stop()
		return nil, fmt.Errorf("执行压测失败: %w", err)
	}

	logger.Default.Info("\n✅ 压测完成!")
	logger.Default.Info("📊 实时报告服务器继续运行，按 Ctrl+C 可停止并退出")
	return report, nil
}

// printStartInfo 打印启动信息
func (e *Executor) printStartInfo() {
	logger.Default.Info("\n🚀 开始压测...")
	logger.Default.Info("📊 协议: %s", e.config.Protocol)
	logger.Default.Info("🔢 并发数: %d", e.config.Concurrency)
	logger.Default.Info("📈 每并发请求数: %d", e.config.Requests)
	logger.Default.Info("⏱️  超时时间: %v", e.config.Timeout)
	if e.config.Advanced != nil && e.config.Advanced.RampUp > 0 {
		logger.Default.Info("⏲️  渐进启动: %v", e.config.Advanced.RampUp)
	}
	logger.Default.Info("")
}

// GetCollector 获取统计收集器
func (e *Executor) GetCollector() *statistics.Collector {
	return e.collector
}

// SetStatsReporter 设置统计上报器（用于分布式模式）
func (e *Executor) SetStatsReporter(reporter StatsReporter) {
	e.statsReporter = reporter
	e.isDistributed = true
	// 在分布式模式下，同时将结果发送到本地收集器和远程上报器
	if reporter != nil {
		e.collector.SetExternalReporter(func(result *types.RequestResult) {
			reporter.Add(result)
		})
	}
}

// IsDistributed 是否为分布式模式
func (e *Executor) IsDistributed() bool {
	return e.isDistributed
}

// openBrowser 在默认浏览器中打开URL
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
	if err != nil {
		logger.Default.Debugf("自动打开浏览器失败: %v", err)
	}
}
