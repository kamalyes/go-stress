/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:00:00
 * @FilePath: \go-stress\logger\logger.go
 * @Description: go-stress 日志接口，直接复用 go-logger
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package logger

import (
	"time"

	"github.com/kamalyes/go-logger"
)

// 直接导出 go-logger 的类型和常量
type (
	StressLogger = logger.ILogger
	LogLevel     = logger.LogLevel
	LogConfig    = logger.LogConfig
)

const (
	DEBUG LogLevel = logger.DEBUG
	INFO  LogLevel = logger.INFO
	WARN  LogLevel = logger.WARN
	ERROR LogLevel = logger.ERROR
	FATAL LogLevel = logger.FATAL
)

// 全局默认日志器
var Default StressLogger

func init() {
	// Windows 环境下启用 ANSI 颜色支持
	enableWindowsANSI()

	// 初始化默认日志器
	Default = logger.NewLogger(DefaultConfig())
}

// SetDefault 设置默认日志器
func SetDefault(l StressLogger) {
	Default = l
}

// New 创建新的日志器
func New(config *LogConfig) StressLogger {
	return logger.NewLogger(config)
}

// DefaultConfig 获取默认配置（带 STRESS 前缀）
func DefaultConfig() *LogConfig {
	return logger.DefaultConfig().
		WithPrefix("[STRESS] ").
		WithShowCaller(false).
		WithColorful(true). // 始终启用颜色，已在 init 中启用 Windows 支持
		WithTimeFormat(time.DateTime)
}

// ParseLogLevel 解析日志级别字符串
func ParseLogLevel(level string) LogLevel {
	switch level {
	case "debug", "DEBUG":
		return DEBUG
	case "info", "INFO":
		return INFO
	case "warn", "WARN", "warning", "WARNING":
		return WARN
	case "error", "ERROR":
		return ERROR
	case "fatal", "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// 便捷函数 - 直接导出 go-logger 的工具函数
var (
	NewFileWriter    = logger.NewFileWriter
	NewRotateWriter  = logger.NewRotateWriter
	NewConsoleWriter = logger.NewConsoleWriter
)
