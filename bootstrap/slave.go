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
	"github.com/kamalyes/go-toolbox/pkg/random"
)

// SlaveOptions Slave 启动选项
type SlaveOptions struct {
	SlaveID        string
	MasterAddr     string
	GRPCPort       int
	RealtimePort   int // 实时报告服务器端口（0表示禁用，默认自动分配）
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
		opts.Logger.InfoKV("📝 自动生成 Slave ID", "slave_id", opts.SlaveID)
	}

	// 设置默认值
	if opts.MaxConcurrency <= 0 {
		opts.MaxConcurrency = 5
	}

	// 如果未指定实时报告端口，则自动分配一个可用端口
	if opts.RealtimePort == 0 {
		// 构建端口候选列表（8088-8187，支持100个slave）
		ports := make([]int, 100)
		for i := 0; i < 100; i++ {
			ports[i] = 8088 + i
		}
		if port, err := random.GenerateAvailablePort(ports...); err == nil {
			opts.RealtimePort = port
			opts.Logger.InfoKV("自动分配实时报告端口", "port", port)
		} else {
			opts.Logger.WarnKV("无法分配实时报告端口，将禁用实时报告功能", "error", err)
			opts.RealtimePort = 0 // 禁用实时报告
		}
	}

	slaveCfg := &common.SlaveConfig{
		SlaveID:         opts.SlaveID,
		MasterAddr:      opts.MasterAddr,
		GRPCPort:        int32(opts.GRPCPort),
		RealtimePort:    opts.RealtimePort,
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
	opts.Logger.InfoKV("   Slave ID", "id", opts.SlaveID)
	opts.Logger.InfoKV("   Master 地址", "addr", opts.MasterAddr)
	opts.Logger.InfoKV("   gRPC 端口", "port", opts.GRPCPort)
	if opts.RealtimePort > 0 {
		opts.Logger.InfoKV("   实时报告端口", "realtime_port", opts.RealtimePort)
	}
	opts.Logger.InfoKV("   区域", "region", opts.Region)
	opts.Logger.InfoKV("   最大并发", "max_concurrency", opts.MaxConcurrency)

	// 等待退出
	<-ctx.Done()
	opts.Logger.Info("👋 Slave 节点已停止")
	return nil
}
