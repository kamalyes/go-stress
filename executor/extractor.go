/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 14:00:00
 * @FilePath: \go-stress\executor\extractor.go
 * @Description: 响应数据提取器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-stress/types"
	"github.com/oliveagle/jsonpath"
)

// Extractor 数据提取器接口
type Extractor interface {
	// Extract 从响应中提取数据
	Extract(resp *types.Response) (string, error)
}

// JSONPathExtractor JSONPath提取器
type JSONPathExtractor struct {
	path string
}

// NewJSONPathExtractor 创建JSONPath提取器
func NewJSONPathExtractor(path string) *JSONPathExtractor {
	return &JSONPathExtractor{path: path}
}

// Extract 使用JSONPath提取数据
func (e *JSONPathExtractor) Extract(resp *types.Response) (string, error) {
	if resp == nil || len(resp.Body) == 0 {
		return "", fmt.Errorf("响应体为空")
	}

	// 解析JSON
	var data interface{}
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return "", fmt.Errorf("解析JSON失败: %w", err)
	}

	// 使用JSONPath提取
	result, err := jsonpath.JsonPathLookup(data, e.path)
	if err != nil {
		return "", fmt.Errorf("JSONPath提取失败 [%s]: %w", e.path, err)
	}

	// 转换为字符串
	switch v := result.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	default:
		// 复杂对象转JSON字符串
		bytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("转换结果失败: %w", err)
		}
		return string(bytes), nil
	}
}

// RegexExtractor 正则表达式提取器
type RegexExtractor struct {
	pattern *regexp.Regexp
}

// NewRegexExtractor 创建正则表达式提取器
func NewRegexExtractor(pattern string) (*RegexExtractor, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("编译正则表达式失败: %w", err)
	}
	return &RegexExtractor{pattern: re}, nil
}

// Extract 使用正则表达式提取数据
func (e *RegexExtractor) Extract(resp *types.Response) (string, error) {
	if resp == nil || len(resp.Body) == 0 {
		return "", fmt.Errorf("响应体为空")
	}

	matches := e.pattern.FindStringSubmatch(string(resp.Body))
	if len(matches) < 2 {
		return "", fmt.Errorf("正则表达式未匹配到数据")
	}

	// 返回第一个捕获组
	return matches[1], nil
}

// HeaderExtractor 响应头提取器
type HeaderExtractor struct {
	headerName string
}

// NewHeaderExtractor 创建响应头提取器
func NewHeaderExtractor(headerName string) *HeaderExtractor {
	return &HeaderExtractor{headerName: headerName}
}

// Extract 从响应头提取数据
func (e *HeaderExtractor) Extract(resp *types.Response) (string, error) {
	if resp == nil || resp.Headers == nil {
		return "", fmt.Errorf("响应头为空")
	}

	value, exists := resp.Headers[e.headerName]
	if !exists {
		return "", fmt.Errorf("响应头 [%s] 不存在", e.headerName)
	}

	return value, nil
}

// ExtractorManager 提取器管理器
type ExtractorManager struct {
	extractors map[string]Extractor
}

// NewExtractorManager 创建提取器管理器
func NewExtractorManager(configs []config.ExtractorConfig) (*ExtractorManager, error) {
	manager := &ExtractorManager{
		extractors: make(map[string]Extractor),
	}

	for _, cfg := range configs {
		extractor, err := createExtractor(cfg)
		if err != nil {
			return nil, fmt.Errorf("创建提取器 [%s] 失败: %w", cfg.Name, err)
		}
		manager.extractors[cfg.Name] = extractor
	}

	return manager, nil
}

// createExtractor 根据配置创建提取器
func createExtractor(cfg config.ExtractorConfig) (Extractor, error) {
	// 默认类型为jsonpath
	extractorType := cfg.Type
	if extractorType == "" {
		extractorType = ExtractorTypeJSONPath
	}

	switch extractorType {
	case ExtractorTypeJSONPath:
		if cfg.JSONPath == "" {
			return nil, fmt.Errorf("JSONPath不能为空")
		}
		return NewJSONPathExtractor(cfg.JSONPath), nil

	case ExtractorTypeRegex:
		if cfg.Regex == "" {
			return nil, fmt.Errorf("正则表达式不能为空")
		}
		return NewRegexExtractor(cfg.Regex)

	case ExtractorTypeHeader:
		if cfg.Header == "" {
			return nil, fmt.Errorf("响应头名称不能为空")
		}
		return NewHeaderExtractor(cfg.Header), nil

	default:
		return nil, fmt.Errorf("不支持的提取器类型: %s", extractorType)
	}
}

// ExtractAll 从响应中提取所有变量
func (m *ExtractorManager) ExtractAll(resp *types.Response, defaultValues map[string]string) map[string]string {
	results := make(map[string]string)

	for name, extractor := range m.extractors {
		value, err := extractor.Extract(resp)
		if err != nil {
			// 提取失败，使用默认值
			if defaultVal, exists := defaultValues[name]; exists {
				logger.Default.Warn("提取变量 [%s] 失败，使用默认值: %s, 错误: %v", name, defaultVal, err)
				results[name] = defaultVal
			} else {
				logger.Default.Warn("提取变量 [%s] 失败且无默认值: %v", name, err)
			}
			continue
		}
		results[name] = value
		logger.Default.Debug("成功提取变量 [%s] = %s", name, value)
	}

	return results
}
