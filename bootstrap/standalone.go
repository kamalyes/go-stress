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
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/executor"
)

// StandaloneOptions Standalone 模式选项
type StandaloneOptions struct {
	ConfigFile   string
	CurlFile     string
	Concurrency  uint64
	Requests     uint64
	Timeout      time.Duration
	StorageMode  StorageMode
	ReportPrefix string
	MaxMemory    string
	Logger       logger.ILogger
	ConfigFunc   func() *config.Config
}

// RunStandalone 运行独立模式
func RunStandalone(opts StandaloneOptions) error {
	result := executor.RunTask(executor.RunOptions{
		ConfigFile:    opts.ConfigFile,
		CurlFile:      opts.CurlFile,
		Concurrency:   opts.Concurrency,
		Requests:      opts.Requests,
		Timeout:       opts.Timeout,
		StorageMode:   executor.StorageMode(opts.StorageMode),
		ReportPrefix:  opts.ReportPrefix,
		MaxMemory:     opts.MaxMemory,
		Logger:        opts.Logger,
		ConfigFunc:    opts.ConfigFunc,
		IsDistributed: false,
	})

	return result.Error
}
