/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 22:00:00
 * @FilePath: \go-stress\executor\request_source.go
 * @Description: 请求源接口 - 统一请求构建（消除单API和多API的冗余区分）
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"fmt"

	"github.com/kamalyes/go-logger"
)

// RequestSource 请求源接口（统一所有模式）
type RequestSource interface {
	// NextRequest 获取下一个请求配置
	// 返回: apiConfig, groupID, isFromDependency
	NextRequest() (*APIConfig, uint64, bool)

	// Name 获取当前请求的名称（用于日志）
	Name() string
}

// ===== 统一的请求源实现（不再区分单API和多API） =====

// APISource 通用API请求源（支持单个或多个API）
type APISource struct {
	selector APISelector
	logger   logger.ILogger
}

// NewAPISource 创建API请求源（统一入口）
func NewAPISource(selector APISelector, log logger.ILogger) *APISource {
	return &APISource{selector: selector, logger: log}
}

// NextRequest 获取下一个请求配置
func (s *APISource) NextRequest() (*APIConfig, uint64, bool) {
	if s.selector == nil {
		return nil, 0, false
	}

	apiCfg := s.selector.Next()
	if apiCfg == nil {
		s.logger.Error("API选择器返回空配置")
		return nil, 0, false
	}

	return apiCfg, 0, false
}

// Name 获取当前请求的名称
func (s *APISource) Name() string {
	return "api-source"
}

// ===== 依赖链专用（这个确实有特殊性：需要指定API名称和GroupID） =====
type DependencyAPISource struct {
	apiName  string
	resolver *DependencyResolver
	groupID  uint64
	logger   logger.ILogger
}

// NewDependencyAPISource 创建依赖链API请求源
func NewDependencyAPISource(apiName string, resolver *DependencyResolver, groupID uint64, log logger.ILogger) *DependencyAPISource {
	return &DependencyAPISource{
		apiName:  apiName,
		resolver: resolver,
		groupID:  groupID,
		logger:   log,
	}
}

// NextRequest 获取下一个请求配置
func (d *DependencyAPISource) NextRequest() (*APIConfig, uint64, bool) {
	if d.resolver == nil {
		return nil, d.groupID, true
	}

	api := d.resolver.GetAPI(d.apiName)
	if api == nil {
		d.logger.Errorf("找不到 API [%s]", d.apiName)
		return nil, d.groupID, true
	}

	return api, d.groupID, true
}

// Name 获取当前请求的名称
func (d *DependencyAPISource) Name() string {
	return d.apiName
}

// RequestContext 请求执行上下文（统一的执行信息）
type RequestContext struct {
	Source       RequestSource
	APIConfig    *APIConfig
	GroupID      uint64
	IsDependency bool
	WorkerID     uint64
	WorkerDepCtx *WorkerDependencyContext
	ShouldSkip   bool
	SkipReason   string
	FailedDeps   []string
}

// NewRequestContext 创建请求执行上下文
func NewRequestContext(source RequestSource, workerID uint64, depCtx *WorkerDependencyContext) (*RequestContext, error) {
	apiCfg, groupID, isDep := source.NextRequest()
	if apiCfg == nil {
		return nil, fmt.Errorf("无法获取请求配置")
	}

	return &RequestContext{
		Source:       source,
		APIConfig:    apiCfg,
		GroupID:      groupID,
		IsDependency: isDep,
		WorkerID:     workerID,
		WorkerDepCtx: depCtx,
		ShouldSkip:   false,
	}, nil
}
