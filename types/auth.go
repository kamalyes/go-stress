/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 10:59:05
 * @FilePath: \go-stress\types\auth.go
 * @Description: 认证相关类型定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package types

// AuthType 认证类型
type AuthType string

const (
	AuthTypeNone   AuthType = "NONE"   // 无认证
	AuthTypeBasic  AuthType = "BASIC"  // Basic认证
	AuthTypeBearer AuthType = "BEARER" // Bearer Token认证
	AuthTypeSign   AuthType = "SIGN"   // 签名认证
)
