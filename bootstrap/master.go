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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/distributed/common"
	"github.com/kamalyes/go-stress/distributed/master"
)

// MasterOptions Master 启动选项
type MasterOptions struct {
	GRPCPort int
	HTTPPort int
	Secret   string
	Logger   logger.ILogger

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
	opts.Logger.Info("\n💡 使用以下命令提交任务:")
	opts.Logger.Info("   curl -X POST http://localhost:%d/api/v1/tasks \\", opts.HTTPPort)
	opts.Logger.Info("     -H 'Content-Type: application/json' \\")
	opts.Logger.Info("     -d @task_config.json")

	// 等待退出
	<-ctx.Done()
	opts.Logger.Info("👋 Master 节点已停止")
	return nil
}
