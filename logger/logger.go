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
	"io"
	"time"

	"github.com/kamalyes/go-logger"
)

// 类型别名
type (
	ILogger   = logger.ILogger
	LogConfig = logger.LogConfig
	LogLevel  = logger.LogLevel
)

// 常量别名 - 日志级别
const (
	DEBUG = logger.DEBUG
	INFO  = logger.INFO
	WARN  = logger.WARN
	ERROR = logger.ERROR
	FATAL = logger.FATAL
)

// 函数别名
var (
	NewLogger       = logger.NewLogger
	NewRotateWriter = logger.NewRotateWriter
)

// Default 全局默认 logger 实例
var Default logger.ILogger

func init() {
	Default = New()
}

func DefaultConfig() *logger.LogConfig {
	config := logger.DefaultConfig().
		WithPrefix("[STRESS] ").
		WithShowCaller(false).
		WithColorful(true).
		WithTimeFormat(time.DateTime)
	return config
}

// New 获取默认配置（带 STRESS 前缀）
func New() *logger.Logger {
	return logger.NewLogger(DefaultConfig())
}

// SetDefault 设置全局默认 logger
func SetDefault(l logger.ILogger) {
	Default = l
}

// NewLoggerWithWriter 创建新日志器（便捷函数）
func NewLoggerWithWriter(prefix string, writer io.Writer) *logger.Logger {
	config := logger.DefaultConfig().
		WithPrefix(prefix).
		WithOutput(writer)
	return logger.NewLogger(config)
}

// LogLevelFlag 日志级别标志（实现 flag.Value 接口）
type LogLevelFlag struct {
	Level logger.LogLevel
}

// String 返回日志级别的字符串表示（实现 flag.Value 接口）
func (f *LogLevelFlag) String() string {
	return f.Level.String()
}

// Set 从字符串设置日志级别（实现 flag.Value 接口）
func (f *LogLevelFlag) Set(value string) error {
	level, err := logger.ParseLevel(value)
	if err != nil {
		return err
	}
	f.Level = level
	return nil
}
