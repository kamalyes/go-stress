/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 11:17:55
 * @FilePath: \go-stress\executor\api_selector.go
 * @Description: API选择器 - 支持多API轮询和权重分配
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"math/rand"
	"sync/atomic"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/logger"
)

// APISelector API选择器接口
type APISelector interface {
	// Next 获取下一个API请求配置
	Next() *APIRequestConfig

	// HasDependencies 是否有依赖关系
	HasDependencies() bool

	// GetDependencyResolver 获取依赖解析器（如果有依赖）
	GetDependencyResolver() *DependencyResolver
}

// APIRequestConfig API请求配置
type APIRequestConfig struct {
	Name       string
	URL        string
	Method     string
	Headers    map[string]string
	Body       string
	Repeat     int                      // 重复执行次数
	Verify     []config.VerifyConfig
	Extractors []config.ExtractorConfig // 数据提取器配置
}

// roundRobinSelector 轮询选择器
type roundRobinSelector struct {
	apis    []config.APIConfig
	counter uint64
}

// NewRoundRobinSelector 创建轮询选择器
func NewRoundRobinSelector(apis []config.APIConfig) APISelector {
	return &roundRobinSelector{
		apis:    apis,
		counter: 0,
	}
}

// Next 轮询获取下一个API
func (s *roundRobinSelector) Next() *APIRequestConfig {
	if len(s.apis) == 0 {
		return nil
	}

	// 原子递增计数器
	idx := atomic.AddUint64(&s.counter, 1) - 1
	api := s.apis[idx%uint64(len(s.apis))]

	// 深拷贝 Headers，避免并发修改
	headers := make(map[string]string, len(api.Headers))
	for k, v := range api.Headers {
		headers[k] = v
	}

	return &APIRequestConfig{
		Name:       api.Name,
		URL:        api.URL,
		Method:     api.Method,
		Headers:    headers,
		Body:       api.Body,
		Verify:     api.Verify,
		Extractors: api.Extractors,
	}
}

// HasDependencies 是否有依赖关系
func (s *roundRobinSelector) HasDependencies() bool {
	return false
}

// GetDependencyResolver 获取依赖解析器
func (s *roundRobinSelector) GetDependencyResolver() *DependencyResolver {
	return nil
}

// weightedSelector 加权选择器
type weightedSelector struct {
	apis    []config.APIConfig
	weights []int
	total   int
}

// NewWeightedSelector 创建加权选择器
func NewWeightedSelector(apis []config.APIConfig) APISelector {
	weights := make([]int, len(apis))
	total := 0
	for i, api := range apis {
		weight := api.Weight
		if weight <= 0 {
			weight = 1
		}
		weights[i] = weight
		total += weight
	}

	return &weightedSelector{
		apis:    apis,
		weights: weights,
		total:   total,
	}
}

// Next 根据权重随机选择API
func (s *weightedSelector) Next() *APIRequestConfig {
	if len(s.apis) == 0 {
		return nil
	}

	// 生成随机数
	r := rand.Intn(s.total)
	sum := 0

	// 根据权重选择
	for i, weight := range s.weights {
		sum += weight
		if r < sum {
			api := s.apis[i]
			// 深拷贝 Headers，避免并发修改
			headers := make(map[string]string, len(api.Headers))
			for k, v := range api.Headers {
				headers[k] = v
			}
			return &APIRequestConfig{
				Name:       api.Name,
				URL:        api.URL,
				Method:     api.Method,
				Headers:    headers,
				Body:       api.Body,
				Verify:     api.Verify,
				Extractors: api.Extractors,
			}
		}
	}

	// 默认返回第一个
	api := s.apis[0]
	// 深拷贝 Headers，避免并发修改
	headers := make(map[string]string, len(api.Headers))
	for k, v := range api.Headers {
		headers[k] = v
	}
	return &APIRequestConfig{
		Name:       api.Name,
		URL:        api.URL,
		Method:     api.Method,
		Headers:    headers,
		Body:       api.Body,
		Verify:     api.Verify,
		Extractors: api.Extractors,
	}
}

// HasDependencies 是否有依赖关系
func (s *weightedSelector) HasDependencies() bool {
	return false
}

// GetDependencyResolver 获取依赖解析器
func (s *weightedSelector) GetDependencyResolver() *DependencyResolver {
	return nil
}

// singleAPISelector 单API选择器（向后兼容）
type singleAPISelector struct {
	config *APIRequestConfig
}

// NewSingleAPISelector 创建单API选择器
func NewSingleAPISelector(cfg *config.Config) APISelector {
	var verify []config.VerifyConfig
	if cfg.Verify != nil {
		verify = []config.VerifyConfig{*cfg.Verify}
	}

	return &singleAPISelector{
		config: &APIRequestConfig{
			Name:    "default",
			URL:     cfg.URL,
			Method:  cfg.Method,
			Headers: cfg.Headers,
			Body:    cfg.Body,
			Verify:  verify,
		},
	}
}

// Next 返回单个API配置（每次返回新的副本，避免并发修改）
func (s *singleAPISelector) Next() *APIRequestConfig {
	// 深拷贝 Headers，避免并发修改
	headers := make(map[string]string, len(s.config.Headers))
	for k, v := range s.config.Headers {
		headers[k] = v
	}

	return &APIRequestConfig{
		Name:       s.config.Name,
		URL:        s.config.URL,
		Method:     s.config.Method,
		Headers:    headers,
		Body:       s.config.Body,
		Verify:     s.config.Verify,
		Extractors: s.config.Extractors,
	}
}

// HasDependencies 是否有依赖关系
func (s *singleAPISelector) HasDependencies() bool {
	return false
}

// GetDependencyResolver 获取依赖解析器
func (s *singleAPISelector) GetDependencyResolver() *DependencyResolver {
	return nil
}

// dependencySelector 依赖执行选择器（按依赖顺序执行）
type dependencySelector struct {
	resolver *DependencyResolver
	order    []string
	current  uint64
}

// NewDependencySelector 创建依赖选择器
func NewDependencySelector(apis []config.APIConfig) (APISelector, error) {
	resolver, err := NewDependencyResolver(apis)
	if err != nil {
		return nil, err
	}

	return &dependencySelector{
		resolver: resolver,
		order:    resolver.GetExecutionOrder(),
		current:  0,
	}, nil
}

// Next 按依赖顺序返回下一个API
func (s *dependencySelector) Next() *APIRequestConfig {
	if len(s.order) == 0 {
		return nil
	}

	// 循环执行API序列
	idx := atomic.AddUint64(&s.current, 1) - 1
	apiName := s.order[idx%uint64(len(s.order))]

	// 检查该API是否应该被跳过（因为依赖失败）
	if s.resolver.ShouldSkipAPI(apiName) {
		// 返回一个标记为跳过的配置
		return &APIRequestConfig{
			Name:    apiName,
			URL:     "",
			Method:  "SKIP",
			Headers: map[string]string{},
			Body:    "",
		}
	}

	api := s.resolver.GetAPI(apiName)
	if api == nil {
		return nil
	}

	// 深拷贝 Headers，避免并发修改
	headers := make(map[string]string, len(api.Headers))
	for k, v := range api.Headers {
		headers[k] = v
	}

	return &APIRequestConfig{
		Name:       api.Name,
		URL:        api.URL,
		Method:     api.Method,
		Headers:    headers,
		Body:       api.Body,
		Verify:     api.Verify,
		Extractors: api.Extractors,
	}
}

// HasDependencies 是否有依赖关系
func (s *dependencySelector) HasDependencies() bool {
	return true
}

// GetDependencyResolver 获取依赖解析器
func (s *dependencySelector) GetDependencyResolver() *DependencyResolver {
	return s.resolver
}

// CreateAPISelector 创建API选择器
func CreateAPISelector(cfg *config.Config) APISelector {
	// 如果没有定义APIs，使用单API模式
	if len(cfg.APIs) == 0 {
		return NewSingleAPISelector(cfg)
	}

	// 检查是否有依赖关系
	hasDeps := false
	for _, api := range cfg.APIs {
		if len(api.DependsOn) > 0 || len(api.Extractors) > 0 {
			hasDeps = true
			break
		}
	}

	// 如果有依赖关系或提取器，使用依赖选择器
	if hasDeps {
		selector, err := NewDependencySelector(cfg.APIs)
		if err != nil {
			logger.Default.Error("创建依赖选择器失败: %v，回退到轮询模式", err)
			return NewRoundRobinSelector(cfg.APIs)
		}
		return selector
	}

	// 检查是否有权重配置
	hasWeight := false
	for _, api := range cfg.APIs {
		if api.Weight > 1 {
			hasWeight = true
			break
		}
	}

	// 如果有权重配置，使用加权选择器
	if hasWeight {
		return NewWeightedSelector(cfg.APIs)
	}

	// 否则使用轮询选择器
	return NewRoundRobinSelector(cfg.APIs)
}

// BuildRequest 从API配置构建请求
func BuildRequest(apiCfg *APIRequestConfig) *Request {
	return &Request{
		URL:     apiCfg.URL,
		Method:  apiCfg.Method,
		Headers: apiCfg.Headers,
		Body:    apiCfg.Body,
	}
}
