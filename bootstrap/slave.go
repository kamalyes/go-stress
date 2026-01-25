/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 16:30:00
 * @FilePath: \go-stress\bootstrap\slave.go
 * @Description: Slave 模式启动器
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
	"github.com/kamalyes/go-stress/distributed/slave"
)

// SlaveOptions Slave 启动选项
type SlaveOptions struct {
	SlaveID        string
	MasterAddr     string
	GRPCPort       int
	Region         string
	MaxConcurrency int
	CanReuse       bool
	Logger         logger.ILogger
}

// RunSlave 运行 Slave 节点
func RunSlave(opts SlaveOptions) error {
	opts.Logger.Info("🤖 启动 Slave 节点...")

	if opts.MasterAddr == "" {
		return fmt.Errorf("Slave 模式必须指定 Master 地址")
	}

	// 自动生成 Slave ID
	if opts.SlaveID == "" {
		opts.SlaveID = fmt.Sprintf("slave-%s-%d", opts.Region, time.Now().Unix())
		opts.Logger.Info("📝 自动生成 Slave ID: %s", opts.SlaveID)
	}

	// 设置默认值
	if opts.MaxConcurrency <= 0 {
		opts.MaxConcurrency = 5
	}

	slaveCfg := &common.SlaveConfig{
		SlaveID:         opts.SlaveID,
		MasterAddr:      opts.MasterAddr,
		GRPCPort:        int32(opts.GRPCPort),
		Region:          opts.Region,
		Labels:          map[string]string{"region": opts.Region},
		MaxConcurrency:  opts.MaxConcurrency,
		CanReuse:        opts.CanReuse,
		ReportBuffer:    1000,
		ReportInterval:  5 * time.Second,
		ResourceMonitor: true,
	}

	s, err := slave.NewSlave(slaveCfg, opts.Logger)
	if err != nil {
		return fmt.Errorf("创建 Slave 失败: %w", err)
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
		s.Stop()
	}()

	if err := s.Start(ctx); err != nil {
		return fmt.Errorf("启动 Slave 失败: %w", err)
	}

	opts.Logger.Info("✅ Slave 节点运行中...")
	opts.Logger.Info("   Slave ID: %s", opts.SlaveID)
	opts.Logger.Info("   Master 地址: %s", opts.MasterAddr)
	opts.Logger.Info("   gRPC 端口: %d", opts.GRPCPort)
	opts.Logger.Info("   区域: %s", opts.Region)
	opts.Logger.Info("   最大并发: %d", opts.MaxConcurrency)

	// 等待退出
	<-ctx.Done()
	opts.Logger.Info("👋 Slave 节点已停止")
	return nil
}
