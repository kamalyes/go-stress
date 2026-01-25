/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 12:15:19
 * @FilePath: \go-stress\bootstrap\standalone.go
 * @Description: Standalone 模式启动器
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/executor"
	"github.com/kamalyes/go-stress/statistics"
	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-toolbox/pkg/osx"
	"github.com/kamalyes/go-toolbox/pkg/units"
)

// StandaloneOptions Standalone 模式选项
type StandaloneOptions struct {
	ConfigFile   string
	CurlFile     string
	Concurrency  uint64
	Requests     uint64
	Timeout      time.Duration
	StorageMode  types.StorageMode
	ReportPrefix string
	MaxMemory    string
	Logger       logger.ILogger
	ConfigFunc   func() *config.Config // 从命令行构建配置的函数
}

// RunStandalone 运行独立模式
func RunStandalone(opts StandaloneOptions) error {
	var cfg *config.Config
	var err error

	// 从curl文件加载
	if opts.CurlFile != "" {
		opts.Logger.Info("📄 解析curl文件: %s", opts.CurlFile)
		cfg, err = config.ParseCurlFile(opts.CurlFile)
		if err != nil {
			return fmt.Errorf("解析curl文件失败: %w", err)
		}
		// 如果命令行指定了并发数和请求数，覆盖curl配置
		if opts.Concurrency > 0 {
			cfg.Concurrency = opts.Concurrency
		}
		if opts.Requests > 0 {
			cfg.Requests = opts.Requests
		}
		if opts.Timeout > 0 {
			cfg.Timeout = opts.Timeout
		}
	} else if opts.ConfigFile != "" {
		// 从配置文件加载
		opts.Logger.Info("📄 加载配置文件: %s", opts.ConfigFile)
		loader := config.NewLoader()
		cfg, err = loader.LoadFromFile(opts.ConfigFile)
		if err != nil {
			return fmt.Errorf("加载配置文件失败: %w", err)
		}
	} else if opts.ConfigFunc != nil {
		// 使用命令行参数
		cfg = opts.ConfigFunc()
		cfg.RunMode = types.RunModeStandaloneCLI
	} else {
		return fmt.Errorf("必须提供配置文件、curl文件或命令行参数")
	}

	// 验证配置
	if err := validateStandaloneConfig(cfg); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建执行器（根据存储模式选择）
	var exec *executor.Executor

	switch opts.StorageMode {
	case types.StorageModeMemory:
		opts.Logger.Info("💾 存储模式: 内存 (高速、无限制、不持久化)")
		exec, err = executor.NewExecutorWithMemoryStorage(cfg)

	case types.StorageModeSQLite:
		reportDir := filepath.Join(opts.ReportPrefix, fmt.Sprintf("%d", time.Now().Unix()))
		if err := os.MkdirAll(reportDir, os.ModePerm); err != nil {
			return fmt.Errorf("创建报告目录失败: %w", err)
		}
		dbPath := filepath.Join(reportDir, "details.db")
		opts.Logger.Info("💾 存储模式: SQLite (持久化、无限制、可查询)")
		opts.Logger.Info("💾 数据库路径: %s", dbPath)
		exec, err = executor.NewExecutorWithSQLiteStorage(cfg, dbPath)

	default:
		return fmt.Errorf("未知的存储模式: %s (支持: %s, %s)",
			opts.StorageMode, types.StorageModeMemory, types.StorageModeSQLite)
	}

	if err != nil {
		return fmt.Errorf("创建执行器失败: %w", err)
	}

	// 创建context，支持Ctrl+C中断
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 确保程序退出前关闭实时报告服务器
	defer func() {
		if exec.GetRealtimeServer() != nil {
			opts.Logger.Debug("🔒 正在关闭实时报告服务器...")
			if err := exec.GetRealtimeServer().Stop(); err != nil {
				opts.Logger.Warnf("⚠️  关闭实时报告服务器失败: %v", err)
			}
		}
	}()

	// 监听信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		opts.Logger.Warn("\n\n⚠️  收到中断信号，正在停止...")
		cancel()
	}()

	// 启动内存监控（如果配置了阈值）
	if opts.MaxMemory != "" {
		if err := startMemoryMonitor(ctx, opts.MaxMemory, cancel, opts.Logger); err != nil {
			opts.Logger.Warnf("⚠️  %v", err)
		}
	}

	// 执行压测
	report, err := exec.Run(ctx)
	if err != nil {
		// 如果是用户中断（context canceled），不视为错误
		if err.Error() == "执行压测失败: context canceled" ||
			strings.Contains(err.Error(), "context canceled") {
			opts.Logger.Warn("⚠️  用户已中断压测")
		} else {
			return fmt.Errorf("压测执行失败: %w", err)
		}
	}

	// 打印报告
	if report != nil {
		report.Print()
	}

	// 生成并保存报告
	if err := saveReports(exec, report, opts.ReportPrefix, opts.Logger); err != nil {
		opts.Logger.Warnf("⚠️  保存报告失败: %v", err)
	}

	// 等待用户查看报告
	waitForExit(exec, sigCh, ctx, opts.Logger)

	return nil
}

// validateStandaloneConfig 验证配置
func validateStandaloneConfig(cfg *config.Config) error {
	// 多API模式下，URL已经在config.Loader中验证过了
	if len(cfg.APIs) == 0 {
		if cfg.URL == "" {
			return fmt.Errorf("URL不能为空")
		}
	}

	if cfg.Concurrency == 0 {
		return fmt.Errorf("并发数不能为0")
	}

	if cfg.Requests == 0 {
		return fmt.Errorf("请求数不能为0")
	}

	// gRPC特定验证
	if cfg.Protocol == types.ProtocolGRPC {
		if cfg.GRPC == nil {
			return fmt.Errorf("gRPC配置不能为空")
		}
		if cfg.GRPC.UseReflection {
			if cfg.GRPC.Service == "" {
				return fmt.Errorf("gRPC服务名不能为空")
			}
			if cfg.GRPC.Method == "" {
				return fmt.Errorf("gRPC方法名不能为空")
			}
		}
	}

	return nil
}

// startMemoryMonitor 启动内存监控
func startMemoryMonitor(ctx context.Context, maxMemory string, cancel context.CancelFunc, log logger.ILogger) error {
	threshold, err := units.ParseBytes(maxMemory)
	if err != nil {
		return fmt.Errorf("内存阈值格式错误: %w,将忽略内存监控", err)
	}

	log.Infof("🔍 启动内存监控，阈值: %s (%d MB)", maxMemory, threshold/(1024*1024))

	monitor := osx.NewAdvancedMonitor().
		AddThreshold(osx.LevelWarning, threshold*80/100).
		AddThreshold(osx.LevelCritical, threshold).
		SetMetricType(osx.MetricAlloc).
		SetCheckOnce(false).
		SetMaxHistory(200).
		EnableGrowthCheck(20.0, 30*time.Second).
		OnWarning(func(snapshot osx.Snapshot) {
			log.Warnf("[⚠️  警告] 内存使用: %s / %s (%.1f%%), Goroutines: %d",
				units.FormatBytes(snapshot.Alloc),
				maxMemory,
				float64(snapshot.Alloc)/float64(threshold)*100,
				snapshot.Goroutines)
		}).
		OnCritical(func(snapshot osx.Snapshot) {
			log.Warnf("\n[🚨 严重] 内存使用超过阈值: %s / %s (%.1f%%)",
				units.FormatBytes(snapshot.Alloc),
				maxMemory,
				float64(snapshot.Alloc)/float64(threshold)*100)
			log.Warnf("  GC次数: %d, Goroutines: %d", snapshot.NumGC, snapshot.Goroutines)
			log.Warn("🛑 自动停止测试任务...")
			cancel()
		}).
		OnGrowthAlert(func(rate osx.GrowthRate, snapshot osx.Snapshot) {
			log.Warnf("[📈 增长告警] 增长率: %.2f%%, 绝对增长: %s, 时间窗口: %v",
				rate.Percentage,
				units.FormatBytes(uint64(rate.Absolute)),
				rate.Duration)
		}).
		OnCheck(func(snapshot osx.Snapshot) {
			log.Debugf("📊 内存监控 - Alloc: %s, Sys: %s, Goroutines: %d, GC: %d",
				units.FormatBytes(snapshot.Alloc),
				units.FormatBytes(snapshot.Sys),
				snapshot.Goroutines,
				snapshot.NumGC)
		})

	go monitor.Start(ctx, 5*time.Second)
	return nil
}

// saveReports 保存报告
func saveReports(exec *executor.Executor, report *statistics.Report, reportPrefix string, log logger.ILogger) error {
	reportDir := filepath.Join(reportPrefix, fmt.Sprintf("%d", time.Now().Unix()))

	if err := os.MkdirAll(reportDir, os.ModePerm); err != nil {
		if err := exec.GetCollector().Close(); err != nil {
			log.Warnf("⚠️  关闭存储失败: %v", err)
		}
		return fmt.Errorf("创建报告目录失败: %w", err)
	}

	// 生成并保存HTML报告
	htmlReportFile := filepath.Join(reportDir, "index.html")
	totalDuration := time.Duration(0)
	if report != nil {
		totalDuration = report.TotalTime
	}

	if err := exec.GetCollector().GenerateHTMLReport(totalDuration, htmlReportFile); err != nil {
		return fmt.Errorf("生成HTML报告失败: %w", err)
	}

	log.Info("🌐 在浏览器中打开查看详细图表: file:///%s", htmlReportFile)

	// 确保所有数据都写入存储
	if err := exec.GetCollector().Close(); err != nil {
		return fmt.Errorf("关闭存储失败: %w", err)
	}

	return nil
}

// waitForExit 等待退出
func waitForExit(exec *executor.Executor, sigCh chan os.Signal, ctx context.Context, log logger.ILogger) {
	realtimePort := 8088
	if realtimeServer := exec.GetRealtimeServer(); realtimeServer != nil {
		realtimePort = realtimeServer.GetPort()
	}

	log.Info("\n💡 提示: 实时报告服务器仍在运行")
	log.Info("   访问 http://localhost:%d 查看实时报告", realtimePort)
	log.Info("   按 Ctrl+C 退出程序")

	select {
	case <-sigCh:
		log.Info("\n👋 程序已退出")
	case <-ctx.Done():
		log.Info("\n👋 程序已退出")
	}
}
