/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:30:52
 * @FilePath: \go-stress\executor\dependency.go
 * @Description: API依赖解析和执行顺序管理
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"fmt"
	"sync"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
)

// DependencyResolver API依赖解析器
type DependencyResolver struct {
	apiConfigs     []config.APIConfig
	apiMap         map[string]*config.APIConfig
	executionOrder []string          // API执行顺序
	extractedVars  map[string]string // 提取的变量 (API name -> 变量集合)
	failedAPIs     map[string]bool   // 验证失败的API
	mu             sync.RWMutex
	logger         logger.ILogger
}

// NewDependencyResolver 创建依赖解析器
func NewDependencyResolver(apis []config.APIConfig, log logger.ILogger) (*DependencyResolver, error) {
	resolver := &DependencyResolver{
		apiConfigs:    apis,
		apiMap:        make(map[string]*config.APIConfig),
		extractedVars: make(map[string]string),
		failedAPIs:    make(map[string]bool),
		logger:        log,
	}

	// 构建API映射
	for i := range apis {
		api := &apis[i]
		if api.Name == "" {
			api.Name = fmt.Sprintf("api_%d", i+1)
		}
		resolver.apiMap[api.Name] = api
	}

	// 解析依赖关系并确定执行顺序
	if err := resolver.resolveDependencies(); err != nil {
		return nil, err
	}

	return resolver, nil
}

// resolveDependencies 解析依赖关系（拓扑排序）
func (r *DependencyResolver) resolveDependencies() error {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	order := []string{}

	var visit func(name string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("检测到循环依赖: %s", name)
		}

		visiting[name] = true

		api, exists := r.apiMap[name]
		if !exists {
			return fmt.Errorf("API [%s] 不存在", name)
		}

		// 先访问所有依赖
		for _, dep := range api.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[name] = false
		visited[name] = true
		order = append(order, name)
		return nil
	}

	// 遍历所有API
	for name := range r.apiMap {
		if err := visit(name); err != nil {
			return err
		}
	}

	r.executionOrder = order
	if r.logger != nil {
		r.logger.Info("📋 API执行顺序: %v", r.executionOrder)
	}
	return nil
}

// GetExecutionOrder 获取API执行顺序
func (r *DependencyResolver) GetExecutionOrder() []string {
	return r.executionOrder
}

// GetAPI 获取API配置
func (r *DependencyResolver) GetAPI(name string) *config.APIConfig {
	return r.apiMap[name]
}

// StoreExtractedVars 存储提取的变量
func (r *DependencyResolver) StoreExtractedVars(apiName string, vars map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for k, v := range vars {
		// 使用 apiName.varName 作为key，避免冲突
		key := fmt.Sprintf("%s.%s", apiName, k)
		r.extractedVars[key] = v
	}
}

// GetExtractedVar 获取提取的变量
func (r *DependencyResolver) GetExtractedVar(apiName, varName string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s.%s", apiName, varName)
	val, exists := r.extractedVars[key]
	return val, exists
}

// GetAllExtractedVars 获取所有提取的变量（用于变量替换）
func (r *DependencyResolver) GetAllExtractedVars() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vars := make(map[string]string)
	for k, v := range r.extractedVars {
		vars[k] = v
	}
	return vars
}

// HasDependencies 判断是否有API依赖关系
func (r *DependencyResolver) HasDependencies() bool {
	for _, api := range r.apiConfigs {
		if len(api.DependsOn) > 0 {
			return true
		}
	}
	return false
}

// MarkAPIFailed 标记API验证失败
func (r *DependencyResolver) MarkAPIFailed(apiName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failedAPIs[apiName] = true
	r.logger.Warnf("⚠️  API [%s] 验证失败，依赖它的后续 API 将被跳过", apiName)
}

// IsAPIFailed 检查API是否失败
func (r *DependencyResolver) IsAPIFailed(apiName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.failedAPIs[apiName]
}

// ShouldSkipAPI 检查是否应该跳过该API（因为依赖的API失败）
func (r *DependencyResolver) ShouldSkipAPI(apiName string) bool {
	api := r.GetAPI(apiName)
	if api == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// 检查所有依赖的API是否有失败的
	for _, dep := range api.DependsOn {
		if r.failedAPIs[dep] {
			r.logger.Warnf("⏭️  跳过 API [%s], 因为它依赖的 API [%s] 已失败", apiName, dep)
			return true
		}
	}

	return false
}
