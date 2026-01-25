/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 23:21:31
 * @FilePath: \go-stress\verify\registry.go
 * @Description: 验证器注册中心
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package verify

import (
	"fmt"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// Verifier 验证器接口
type Verifier interface {
	// Verify 验证响应
	Verify(resp *types.Response) (bool, error)
}

// VerifierFactory 验证器工厂函数
type VerifierFactory func(cfg *config.VerifyConfig) Verifier

// Registry 验证器注册中心
type Registry struct {
	mu        *syncx.RWLock
	factories map[types.VerifyType]VerifierFactory
}

var globalRegistry = &Registry{
	mu:        syncx.NewRWLock(),
	factories: make(map[types.VerifyType]VerifierFactory),
}

// Register 注册验证器工厂
func Register(vType types.VerifyType, factory VerifierFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.factories[vType] = factory
}

// Get 获取验证器（通过工厂创建）
func Get(vType types.VerifyType, cfg *config.VerifyConfig) (Verifier, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	factory, ok := globalRegistry.factories[vType]
	if !ok {
		return nil, fmt.Errorf("验证器不存在: %s", vType)
	}
	return factory(cfg), nil
}

// init 自动注册所有支持的验证器类型
func init() {
	// 注册所有验证类型的工厂函数
	verifyTypes := []types.VerifyType{
		types.VerifyTypeStatusCode,
		types.VerifyTypeJSONPath,
		types.VerifyTypeContains,
		types.VerifyTypeRegex,
		types.VerifyTypeJSONSchema,
		types.VerifyTypeJSONValid,
		types.VerifyTypeHeader,
		types.VerifyTypeResponseTime,
		types.VerifyTypeResponseSize,
		types.VerifyTypeEmail,
		types.VerifyTypeIP,
		types.VerifyTypeURL,
		types.VerifyTypeUUID,
		types.VerifyTypeBase64,
		types.VerifyTypeLength,
		types.VerifyTypePrefix,
		types.VerifyTypeSuffix,
		types.VerifyTypeEmpty,
		types.VerifyTypeNotEmpty,
	}

	// 所有类型都使用 HTTPVerifier
	factory := func(cfg *config.VerifyConfig) Verifier {
		return NewHTTPVerifier(cfg)
	}

	for _, vType := range verifyTypes {
		Register(vType, factory)
	}
}
