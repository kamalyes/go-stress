/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 09:59:13
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 17:18:08
 * @FilePath: \go-stress\protocol\http_verify.go
 * @Description: HTTP验证器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package protocol

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-toolbox/pkg/stringx"
	"github.com/oliveagle/jsonpath"
)

// HTTPVerifier HTTP验证器
type HTTPVerifier struct {
	config *config.VerifyConfig
}

// NewHTTPVerifier 创建HTTP验证器
func NewHTTPVerifier(cfg *config.VerifyConfig) *HTTPVerifier {
	if cfg == nil {
		cfg = &config.VerifyConfig{
			Type:   VerifyTypeStatusCode,
			Expect: 200,
		}
	}
	return &HTTPVerifier{config: cfg}
}

// Verify 验证HTTP响应
func (v *HTTPVerifier) Verify(resp *Response) (bool, error) {
	if resp.Error != nil {
		return false, resp.Error
	}

	// 初始化验证结果列表
	if resp.Verifications == nil {
		resp.Verifications = make([]VerificationResult, 0)
	}

	switch v.config.Type {
	case VerifyTypeStatusCode:
		return v.verifyStatusCode(resp)
	case VerifyTypeJSONPath:
		return v.verifyJSONPath(resp)
	case VerifyTypeContains:
		return v.verifyContains(resp)
	default:
		return true, nil
	}
}

// verifyStatusCode 验证状态码
func (v *HTTPVerifier) verifyStatusCode(resp *Response) (bool, error) {
	expectedCode := 200 // 默认期望200

	// 处理 Expect 字段,支持int和string类型
	switch exp := v.config.Expect.(type) {
	case int:
		expectedCode = exp
	case float64: // JSON解析时数字会变成float64
		expectedCode = int(exp)
	case string:
		// 如果是字符串,尝试解析为整数
		// 注意:这里不应该出现模板变量,因为状态码验证只接受整数
		// 如果包含{{}}说明配置有误，跳过解析
		if len(exp) > 0 && exp[0] != '{' {
			// 使用 strconv.Atoi 确保整个字符串都是有效数字
			if parsed, err := strconv.Atoi(strings.TrimSpace(exp)); err == nil {
				expectedCode = parsed
			}
			// 如果解析失败，保持默认值 200
		}
	case nil:
		// 使用默认值200
	}

	success := resp.StatusCode == expectedCode
	result := VerificationResult{
		Type:    v.config.Type,
		Success: success,
		Expect:  fmt.Sprintf("%d", expectedCode),
		Actual:  fmt.Sprintf("%d", resp.StatusCode),
	}

	if !success {
		result.Message = fmt.Sprintf("状态码不匹配: 期望 %d, 实际 %d", expectedCode, resp.StatusCode)
	} else {
		result.Message = "状态码验证通过"
	}

	resp.Verifications = append(resp.Verifications, result)

	if !success {
		return false, fmt.Errorf("%s", result.Message)
	}

	return true, nil
}

// verifyJSONPath 验证JSON路径
func (v *HTTPVerifier) verifyJSONPath(resp *Response) (bool, error) {
	// 检查状态码是否为成功状态（不添加验证结果）
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result := VerificationResult{
			Type:    v.config.Type,
			Success: false,
			Message: fmt.Sprintf("HTTP请求失败，状态码: %d", resp.StatusCode),
			Expect:  "2xx",
			Actual:  fmt.Sprintf("%d", resp.StatusCode),
		}
		resp.Verifications = append(resp.Verifications, result)
		return false, fmt.Errorf("HTTP请求失败: %s", result.Message)
	}

	var data interface{}
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		result := VerificationResult{
			Type:    v.config.Type,
			Success: false,
			Message: fmt.Sprintf("解析JSON失败: %v", err),
			Expect:  v.config.JSONPath,
			Actual:  "解析失败",
		}
		resp.Verifications = append(resp.Verifications, result)
		return false, fmt.Errorf("解析JSON失败: %s", result.Message)
	}

	// 使用 jsonpath 库查询
	value, err := jsonpath.JsonPathLookup(data, v.config.JSONPath)
	if err != nil {
		result := VerificationResult{
			Type:    v.config.Type,
			Success: false,
			Message: fmt.Sprintf("JSON路径查询失败: %v", err),
			Expect:  v.config.JSONPath,
			Actual:  "查询失败",
		}
		resp.Verifications = append(resp.Verifications, result)
		return false, fmt.Errorf("JSON路径查询失败: %s", result.Message)
	}

	// 如果配置了期望值,进行比较验证
	success := true
	message := "JSON路径验证通过"
	expectStr := fmt.Sprintf("%v", v.config.JSONPath)
	actualStr := fmt.Sprintf("%v", value)

	if v.config.Expect != nil {
		expectStr = fmt.Sprintf("%v", v.config.Expect)
		// 如果配置了 regex: true，自动设置 operator 为 regex
		operator := v.config.Operator
		if v.config.Regex {
			operator = "regex"
		}
		// 根据操作符进行不同的比较
		success, message = v.compareValues(value, v.config.Expect, operator)
		if !success && message == "" {
			message = fmt.Sprintf("JSON路径值不匹配: 期望 %v, 实际 %v", v.config.Expect, value)
		}
		// 如果有描述信息，加到验证结果中
		if v.config.Description != "" {
			if success {
				message = v.config.Description + ": 验证通过"
			} else {
				message = v.config.Description + ": " + message
			}
		}
	}

	result := VerificationResult{
		Type:    v.config.Type,
		Success: success,
		Message: message,
		Expect:  expectStr,
		Actual:  actualStr,
	}
	resp.Verifications = append(resp.Verifications, result)

	if !success {
		return false, fmt.Errorf("%s", message)
	}

	return true, nil
}

// compareValues 根据操作符比较值
func (v *HTTPVerifier) compareValues(actual, expect interface{}, operator ExpectOperator) (bool, string) {
	actualStr := fmt.Sprintf("%v", actual)
	expectStr := fmt.Sprintf("%v", expect)

	// 如果没有指定操作符，默认使用等于比较
	if operator == "" {
		operator = OpEQ
	}

	switch operator {
	case OpEQ, OpEqual, OpDoubleEqual:
		return actualStr == expectStr, ""

	case OpNE, OpNotEqual:
		return actualStr != expectStr, ""

	case OpGT, OpGreaterThan:
		return v.compareNumeric(actualStr, expectStr, OpGT)

	case OpGTE, OpGreaterThanOrEqual:
		return v.compareNumeric(actualStr, expectStr, OpGTE)

	case OpLT, OpLessThan:
		return v.compareNumeric(actualStr, expectStr, OpLT)

	case OpLTE, OpLessThanOrEqual:
		return v.compareNumeric(actualStr, expectStr, OpLTE)

	case OpContains:
		return strings.Contains(actualStr, expectStr), ""

	case OpNotContains:
		return !strings.Contains(actualStr, expectStr), ""

	case OpHasPrefix:
		return strings.HasPrefix(actualStr, expectStr), ""

	case OpHasSuffix:
		return strings.HasSuffix(actualStr, expectStr), ""

	case OpEmpty:
		return actualStr == "", ""

	case OpNotEmpty:
		return actualStr != "", ""

	case OpRegex, OpRegexp:
		// 正则表达式匹配
		matched, err := regexp.MatchString(expectStr, actualStr)
		if err != nil {
			return false, fmt.Sprintf("正则表达式错误: %v", err)
		}
		if !matched {
			return false, fmt.Sprintf("正则不匹配: 期望 %s, 实际 %s", expectStr, actualStr)
		}
		return true, ""

	default:
		// 默认使用等于比较
		return actualStr == expectStr, ""
	}
}

// compareNumeric 比较数值
func (v *HTTPVerifier) compareNumeric(actualStr, expectStr string, op ExpectOperator) (bool, string) {
	actualNum, err1 := strconv.ParseFloat(actualStr, 64)
	expectNum, err2 := strconv.ParseFloat(expectStr, 64)

	if err1 != nil || err2 != nil {
		return false, "数值比较失败: 无法解析为数字"
	}

	switch op {
	case OpGT:
		return actualNum > expectNum, ""
	case OpGTE:
		return actualNum >= expectNum, ""
	case OpLT:
		return actualNum < expectNum, ""
	case OpLTE:
		return actualNum <= expectNum, ""
	default:
		return false, "未知的比较操作符"
	}
}

// verifyContains 验证包含字符串
func (v *HTTPVerifier) verifyContains(resp *Response) (bool, error) {
	// 检查状态码是否为成功状态（不添加验证结果）
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result := VerificationResult{
			Type:    v.config.Type,
			Success: false,
			Message: fmt.Sprintf("HTTP请求失败，状态码: %d", resp.StatusCode),
			Expect:  "2xx",
			Actual:  fmt.Sprintf("%d", resp.StatusCode),
		}
		resp.Verifications = append(resp.Verifications, result)
		return false, fmt.Errorf("HTTP请求失败: %s", result.Message)
	}

	bodyStr := string(resp.Body)
	containsStr, ok := v.config.Expect.(string)
	if !ok {
		result := VerificationResult{
			Type:    v.config.Type,
			Success: false,
			Message: "contains 验证需要字符串类型的 expect 值",
			Expect:  fmt.Sprintf("%v", v.config.Expect),
			Actual:  "类型错误",
		}
		resp.Verifications = append(resp.Verifications, result)
		return false, fmt.Errorf("类型错误: %s", result.Message)
	}

	success := stringx.Contains(bodyStr, containsStr)
	result := VerificationResult{
		Type:    v.config.Type,
		Success: success,
		Expect:  containsStr,
		Actual:  bodyStr,
	}

	if !success {
		result.Message = fmt.Sprintf("响应不包含期望的字符串: %s", containsStr)
	} else {
		result.Message = "包含字符串验证通过"
	}

	resp.Verifications = append(resp.Verifications, result)

	if !success {
		return false, fmt.Errorf("%s", result.Message)
	}

	return true, nil
}
