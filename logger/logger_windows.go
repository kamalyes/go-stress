//go:build windows
// +build windows

/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-31 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-31 16:00:00
 * @FilePath: \go-stress\logger\logger_windows.go
 * @Description: Windows 平台特定的日志功能
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package logger

import (
	"syscall"
	"unsafe"
)

const (
	// Windows Virtual Terminal Processing 标志
	enableVirtualTerminalProcessing = 0x0004
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode         = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode         = kernel32.NewProc("SetConsoleMode")
	procGetStdHandle           = kernel32.NewProc("GetStdHandle")
	stdOutputHandle    uintptr = ^uintptr(10) + 1 // STD_OUTPUT_HANDLE = -11
)

// enableWindowsANSI 启用 Windows 控制台 ANSI 转义序列支持
func enableWindowsANSI() {
	handle, _, _ := procGetStdHandle.Call(stdOutputHandle)
	if handle == 0 {
		return
	}

	var mode uint32
	ret, _, _ := procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
	if ret == 0 {
		return
	}

	// 启用 Virtual Terminal Processing
	mode |= enableVirtualTerminalProcessing
	procSetConsoleMode.Call(handle, uintptr(mode))
}
