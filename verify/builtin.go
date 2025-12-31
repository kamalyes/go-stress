/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 00:00:00
 * @FilePath: \go-stress\verify\builtin.go
 * @Description: 内置验证器实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package verify

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-toolbox/pkg/validator"
)

// StatusCodeVerifier 状态码验证器
type StatusCodeVerifier struct {
	ExpectedCode int
}

func (v *StatusCodeVerifier) Verify(resp *types.Response) (bool, error) {
	if v.ExpectedCode == 0 {
		v.ExpectedCode = 200 // 默认200
	}
	return resp.StatusCode == v.ExpectedCode, nil
}

// JSONVerifier JSON验证器
type JSONVerifier struct {
	Rules map[string]any // JSON路径验证规则
}

func (v *JSONVerifier) Verify(resp *types.Response) (bool, error) {
	// 验证是否为有效JSON
	var data any
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return false, fmt.Errorf("响应不是有效的JSON: %w", err)
	}

	// 如果有规则，进行字段验证
	if len(v.Rules) > 0 {
		dataMap, ok := data.(map[string]any)
		if !ok {
			return false, fmt.Errorf("JSON根节点不是对象")
		}

		// 使用 go-toolbox 的 validator 进行验证
		for key, expected := range v.Rules {
			actual, ok := dataMap[key]
			if !ok {
				return false, fmt.Errorf("字段不存在: %s", key)
			}
			if actual != expected {
				return false, fmt.Errorf("字段值不匹配: %s, 期望: %v, 实际: %v", key, expected, actual)
			}
		}
	}

	return true, nil
}

// ContainsVerifier 包含验证器
type ContainsVerifier struct {
	Substring string
}

func (v *ContainsVerifier) Verify(resp *types.Response) (bool, error) {
	if v.Substring == "" {
		return true, nil
	}
	contains := strings.Contains(string(resp.Body), v.Substring)
	if !contains {
		return false, fmt.Errorf("响应不包含: %s", v.Substring)
	}
	return true, nil
}

// RegexVerifier 正则验证器
type RegexVerifier struct {
	Pattern string
	regex   *regexp.Regexp
}

func (v *RegexVerifier) Verify(resp *types.Response) (bool, error) {
	if v.regex == nil && v.Pattern != "" {
		regex, err := regexp.Compile(v.Pattern)
		if err != nil {
			return false, fmt.Errorf("正则表达式编译失败: %w", err)
		}
		v.regex = regex
	}

	if v.regex == nil {
		return true, nil
	}

	matched := v.regex.Match(resp.Body)
	if !matched {
		return false, fmt.Errorf("响应不匹配正则: %s", v.Pattern)
	}
	return true, nil
}

// IPVerifier IP地址验证器（使用go-toolbox）
type IPVerifier struct{}

func (v *IPVerifier) Verify(resp *types.Response) (bool, error) {
	ip := string(resp.Body)
	base := &validator.IPBase{}
	if err := base.ValidateIP(ip); err != nil {
		return false, fmt.Errorf("无效的IP地址: %w", err)
	}
	return true, nil
}
