/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 01:08:55
 * @FilePath: \go-stress\verify\builtin.go
 * @Description: 内置验证器实现 - 使用 go-toolbox/validator
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package verify

import (
	"fmt"

	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/validator"
)

// ===== 验证器实现 =====

// StatusCodeVerifier 状态码验证器 - 使用 validator.ValidateStatusCode
type StatusCodeVerifier struct {
	ExpectedCode int                       // 期望的状态码
	Operator     validator.CompareOperator // 比较操作符（可选，默认为相等）
}

func (v *StatusCodeVerifier) Verify(resp *types.Response) (bool, error) {
	// 使用 mathx 三元操作符设置默认值
	expectedCode := mathx.IfNotZero(v.ExpectedCode, 200)
	operator := mathx.IfEmpty(string(v.Operator), string(validator.OpEqual))

	// 使用 validator.ValidateStatusCode 进行验证
	result := validator.ValidateStatusCode(resp.StatusCode, expectedCode, validator.CompareOperator(operator))
	if !result.Success {
		return false, fmt.Errorf("状态码验证失败: %s", result.Message)
	}

	return true, nil
}

// JSONVerifier JSON验证器 - 使用 validator.ValidateJSONFields
type JSONVerifier struct {
	Rules map[string]any // JSON路径验证规则
}

func (v *JSONVerifier) Verify(resp *types.Response) (bool, error) {
	// 验证是否为有效JSON
	if err := validator.ValidateJSON(resp.Body); err != nil {
		return false, fmt.Errorf("响应不是有效的JSON: %v", err)
	}

	// 如果有规则，批量验证字段
	if len(v.Rules) > 0 {
		results := validator.ValidateJSONFields(resp.Body, v.Rules)
		for _, result := range results {
			if !result.Success {
				return false, fmt.Errorf("字段验证失败: %s", result.Message)
			}
		}
	}

	return true, nil
}

// ContainsVerifier 包含验证器 - 使用 validator.ValidateContains
type ContainsVerifier struct {
	Substring string
}

func (v *ContainsVerifier) Verify(resp *types.Response) (bool, error) {
	result := validator.ValidateContains(resp.Body, v.Substring)
	if !result.Success {
		return false, fmt.Errorf("包含验证失败: %s", result.Message)
	}
	return true, nil
}

// RegexVerifier 正则验证器 - 使用 validator.ValidateRegex
type RegexVerifier struct {
	Pattern string
}

func (v *RegexVerifier) Verify(resp *types.Response) (bool, error) {
	result := validator.ValidateRegex(resp.Body, v.Pattern)
	if !result.Success {
		return false, fmt.Errorf("正则验证失败: %s", result.Message)
	}
	return true, nil
}

// IPVerifier IP地址验证器 - 使用 validator.IPBase
type IPVerifier struct{}

func (v *IPVerifier) Verify(resp *types.Response) (bool, error) {
	ip := string(resp.Body)
	base := &validator.IPBase{}
	if err := base.ValidateIP(ip); err != nil {
		return false, fmt.Errorf("无效的IP地址: %w", err)
	}
	return true, nil
}
