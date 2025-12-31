/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 15:00:59
 * @FilePath: \go-stress\verify\registry.go
 * @Description: 验证器注册中心
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package verify

import (
	"fmt"
	"sync"

	"github.com/kamalyes/go-stress/types"
)

// Verifier 验证器接口
type Verifier interface {
	// Verify 验证响应
	Verify(resp *types.Response) (bool, error)
}

// Registry 验证器注册中心
type Registry struct {
	mu        sync.RWMutex
	verifiers map[types.VerifyType]Verifier
}

var globalRegistry = &Registry{
	verifiers: make(map[types.VerifyType]Verifier),
}

// Register 注册验证器
func Register(vType types.VerifyType, verifier Verifier) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.verifiers[vType] = verifier
}

// Get 获取验证器
func Get(vType types.VerifyType) (Verifier, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	verifier, ok := globalRegistry.verifiers[vType]
	if !ok {
		return nil, fmt.Errorf("验证器不存在: %s", vType)
	}
	return verifier, nil
}

// 初始化内置验证器
func init() {
	Register(VerifyStatusCode, &StatusCodeVerifier{})
	Register(VerifyJSON, &JSONVerifier{})
	Register(VerifyContains, &ContainsVerifier{})
	Register(VerifyRegex, &RegexVerifier{})
}
