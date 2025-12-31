//go:build !windows
// +build !windows

/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-31 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-31 16:00:00
 * @FilePath: \go-stress\logger\logger_unix.go
 * @Description: Unix/Linux/macOS 平台特定的日志功能
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package logger

// enableWindowsANSI Unix/Linux/macOS 平台不需要特殊处理，原生支持 ANSI 颜色
func enableWindowsANSI() {
	// Unix/Linux/macOS 平台原生支持 ANSI 转义序列，无需额外处理
}
