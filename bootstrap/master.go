/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 11:57:20
 * @FilePath: \go-stress\bootstrap\master.go
 * @Description: Master 模式启动器
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-stress/distributed/master"
)

// MasterOptions Master 启动选项
type MasterOptions struct {
	GRPCPort    int
	HTTPPort    int
	Secret      string
	Logger      logger.ILogger
	ConfigFile  string // 压测任务配置文件
	CurlFile    string
	Concurrency uint64
	Requests    uint64
	URL         string
	ConfigFunc  func() *common.TaskConfig // 从命令行构建任务配置的函数
	AutoSubmit  bool                      // 是否自动提交任务（有配置时）
	WaitSlaves  int                       // 等待的最小 Slave 数量
	WaitTimeout time.Duration             // 等待 Slave 的超时时间

	// Slave 数量计算配置
	WorkersPerSlave int // 每个 Slave 承担的 Worker 数量,默认 100
	MinSlaveCount   int // 最小需要的 Slave 数量,默认 1

	// Master 配置
	HeartbeatInterval time.Duration // 心跳间隔,默认 5s
	HeartbeatTimeout  time.Duration // 心跳超时,默认 15s
	MaxFailures       int           // 最大失败次数,默认 3
	TokenExpiration   time.Duration // Token 过期时间,默认 24h
	TokenIssuer       string        // Token 签发者,默认 "go-stress-master"
}

// RunMaster 运行 Master 节点
func RunMaster(opts MasterOptions) error {
	opts.Logger.Info("🎯 启动 Master 节点...")

	masterCfg := &common.MasterConfig{
		GRPCPort:          opts.GRPCPort,
		HTTPPort:          opts.HTTPPort,
		HeartbeatInterval: opts.HeartbeatInterval, // 由 master.go 中 mathx.IfZero 兜底为 5s
		HeartbeatTimeout:  opts.HeartbeatTimeout,  // 由 master.go 中 mathx.IfZero 兜底为 15s
		MaxFailures:       opts.MaxFailures,       // 由 master.go 中 mathx.IfNotZero 兜底为 3
		Secret:            opts.Secret,            // 由 master.go 中 mathx.IfEmpty 兜底为默认密钥
		TokenExpiration:   opts.TokenExpiration,   // 由 master.go 中 mathx.IfZero 兜底为 24h
		TokenIssuer:       opts.TokenIssuer,       // 由 master.go 中 mathx.IfEmpty 兜底为 "go-stress-master"
		WorkersPerSlave:   opts.WorkersPerSlave,   // 由 master.go 中 mathx.IfNotZero 兜底为 100
		MinSlaveCount:     opts.MinSlaveCount,     // 由 master.go 中 mathx.IfNotZero 兜底为 1
	}

	m, err := master.NewMaster(masterCfg, opts.Logger)
	if err != nil {
		return fmt.Errorf("创建 Master 失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		opts.Logger.Warn("\n\n⚠️  收到中断信号，正在停止...")
		cancel()
		m.Stop()
	}()

	if err := m.Start(ctx); err != nil {
		return fmt.Errorf("启动 Master 失败: %w", err)
	}

	opts.Logger.Info("✅ Master 节点运行中...")
	opts.Logger.Info("   gRPC 端口: %d", opts.GRPCPort)
	opts.Logger.Info("   HTTP 端口: %d", opts.HTTPPort)

	// 如果有任务配置，自动提交任务
	if opts.AutoSubmit && (opts.ConfigFile != "" || opts.CurlFile != "" || opts.URL != "") {
		if err := autoSubmitTask(ctx, m, opts); err != nil {
			opts.Logger.Errorf("❌ 自动提交任务失败: %v", err)
			return err
		}
	} else {
		opts.Logger.Info("\n💡 使用以下命令提交任务:")
		opts.Logger.Info("   curl -X POST http://localhost:%d/api/v1/tasks \\", opts.HTTPPort)
		opts.Logger.Info("     -H 'Content-Type: application/json' \\")
		opts.Logger.Info("     -d @task_config.json")
	}

	// 等待退出
	<-ctx.Done()
	opts.Logger.Info("👋 Master 节点已停止")
	return nil
}

// autoSubmitTask 自动提交任务
func autoSubmitTask(ctx context.Context, m *master.Master, opts MasterOptions) error {
	opts.Logger.Info("\n🚀 准备自动提交分布式任务...")

	// 等待 Slave 就绪
	if opts.WaitSlaves > 0 {
		opts.Logger.Info("⏳ 等待至少 %d 个 Slave 节点就绪...", opts.WaitSlaves)
		if err := waitForSlaves(ctx, m, opts.WaitSlaves, opts.WaitTimeout, opts.Logger); err != nil {
			return err
		}
	}

	// 构建任务配置
	var taskConfig *common.Task
	var err error

	if opts.ConfigFile != "" {
		taskConfig, err = loadTaskFromConfigFile(opts.ConfigFile, opts.Logger)
	} else if opts.CurlFile != "" {
		taskConfig, err = loadTaskFromCurlFile(opts.CurlFile, opts)
	} else if opts.ConfigFunc != nil {
		taskCfg := opts.ConfigFunc()
		taskConfig = convertToTask(taskCfg)
	} else if opts.URL != "" {
		taskConfig = buildTaskFromFlags(opts)
	} else {
		return fmt.Errorf("没有提供任务配置")
	}

	if err != nil {
		return fmt.Errorf("加载任务配置失败: %w", err)
	}

	// 提交任务
	opts.Logger.Info("📤 提交任务到 Master...")
	if err := m.SubmitTask(taskConfig); err != nil {
		return fmt.Errorf("提交任务失败: %w", err)
	}

	opts.Logger.Info("✅ 任务已提交: %s", taskConfig.ID)
	opts.Logger.Info("   目标: %s", taskConfig.Target)
	opts.Logger.Info("   总并发: %d", taskConfig.TotalWorkers)
	opts.Logger.Info("   持续时间: %ds", taskConfig.Duration)

	return nil
}

// waitForSlaves 等待指定数量的 Slave 就绪
func waitForSlaves(ctx context.Context, m *master.Master, minCount int, timeout time.Duration, log logger.ILogger) error {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			count := m.GetSlavePool().Count()
			if count >= minCount {
				log.Info("✅ %d 个 Slave 节点已就绪", count)
				return nil
			}

			remaining := time.Until(deadline)
			if remaining <= 0 {
				return fmt.Errorf("等待 Slave 超时，当前: %d, 需要: %d", count, minCount)
			}

			log.Debug("等待 Slave 就绪... 当前: %d/%d (剩余: %.0fs)", count, minCount, remaining.Seconds())
		}
	}
}

// loadTaskFromConfigFile 从配置文件加载任务
func loadTaskFromConfigFile(configFile string, log logger.ILogger) (*common.Task, error) {
	log.Info("📄 加载任务配置文件: %s", configFile)

	cfg, err := config.NewLoader().LoadFromFile(configFile)
	if err != nil {
		return nil, err
	}

	// 将 config.Config 转换为 common.Task
	configData, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}

	task := &common.Task{
		Protocol:     string(cfg.Protocol),
		Target:       cfg.URL,
		TotalWorkers: int(cfg.Concurrency),
		Duration:     int(cfg.Requests),
		ConfigData:   configData,
	}

	return task, nil
}

// loadTaskFromCurlFile 从 curl 文件加载任务
func loadTaskFromCurlFile(curlFile string, opts MasterOptions) (*common.Task, error) {
	opts.Logger.Info("📄 解析 curl 文件: %s", curlFile)

	cfg, err := config.ParseCurlFile(curlFile)
	if err != nil {
		return nil, err
	}

	// 覆盖参数
	if opts.Concurrency > 0 {
		cfg.Concurrency = opts.Concurrency
	}
	if opts.Requests > 0 {
		cfg.Requests = opts.Requests
	}

	configData, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}

	task := &common.Task{
		Protocol:     string(cfg.Protocol),
		Target:       cfg.URL,
		TotalWorkers: int(cfg.Concurrency),
		Duration:     int(cfg.Requests),
		ConfigData:   configData,
	}

	return task, nil
}

// buildTaskFromFlags 从命令行参数构建任务
func buildTaskFromFlags(opts MasterOptions) *common.Task {
	cfg := &config.Config{
		Protocol:    ProtocolHTTP,
		URL:         opts.URL,
		Concurrency: opts.Concurrency,
		Requests:    opts.Requests,
	}

	configData, _ := json.Marshal(cfg)

	return &common.Task{
		Protocol:     string(cfg.Protocol),
		Target:       cfg.URL,
		TotalWorkers: int(cfg.Concurrency),
		Duration:     int(cfg.Requests),
		ConfigData:   configData,
	}
}

// convertToTask 转换 TaskConfig 为 Task
func convertToTask(cfg *common.TaskConfig) *common.Task {
	if cfg == nil {
		return nil
	}

	return &common.Task{
		Protocol:     cfg.Protocol,
		Target:       cfg.Target,
		TotalWorkers: int(cfg.WorkerCount),
		Duration:     cfg.Duration,
		RampUp:       cfg.RampUp,
		ConfigData:   cfg.ConfigData,
	}
}
