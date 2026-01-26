/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-stress\types\extractor.go
 * @Description: 提取器相关类型定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package types

// ExtractorType 提取器类型
type ExtractorType string

const (
	ExtractorTypeJSONPath   ExtractorType = "JSONPATH"   // JSONPath提取
	ExtractorTypeRegex      ExtractorType = "REGEX"      // 正则表达式提取
	ExtractorTypeHeader     ExtractorType = "HEADER"     // 响应头提取
	ExtractorTypeExpression ExtractorType = "EXPRESSION" // 表达式提取
)
