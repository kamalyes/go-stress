/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 20:00:00
 * @FilePath: \go-stress\executor\extractor_test.go
 * @Description: 数据提取器测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"testing"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
)

// 测试 JSONPath 提取器 - 从响应提取
func TestJSONPathExtractor_FromResponse(t *testing.T) {
	extractor := NewJSONPathExtractor("$.data.user_id", config.ExtractorSourceResponse)

	ctx := &ExtractorContext{
		Response: &types.Response{
			Body: []byte(`{"data": {"user_id": "12345", "name": "test"}}`),
		},
	}

	value, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "12345", value)
}

// 测试 JSONPath 提取器 - 从请求提取
func TestJSONPathExtractor_FromRequest(t *testing.T) {
	extractor := NewJSONPathExtractor("$.session_id", config.ExtractorSourceRequest)

	ctx := &ExtractorContext{
		Request: &Request{
			Body: `{"session_id": "abc-123", "user": "admin"}`,
		},
	}

	value, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "abc-123", value)
}

// 测试 JSONPath 提取器 - 空响应
func TestJSONPathExtractor_EmptyResponse(t *testing.T) {
	extractor := NewJSONPathExtractor("$.data.id", config.ExtractorSourceResponse)

	ctx := &ExtractorContext{
		Response: &types.Response{
			Body: []byte{},
		},
	}

	_, err := extractor.Extract(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "响应体为空")
}

// 测试 Regex 提取器 - 从响应提取
func TestRegexExtractor_FromResponse(t *testing.T) {
	extractor, err := NewRegexExtractor(`token=(\w+)`, config.ExtractorSourceResponse)
	assert.NoError(t, err)

	ctx := &ExtractorContext{
		Response: &types.Response{
			Body: []byte("session token=abcd1234 expires=3600"),
		},
	}

	value, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "abcd1234", value)
}

// 测试 Regex 提取器 - 从请求提取
func TestRegexExtractor_FromRequest(t *testing.T) {
	extractor, err := NewRegexExtractor(`user_id=(\d+)`, config.ExtractorSourceRequest)
	assert.NoError(t, err)

	ctx := &ExtractorContext{
		Request: &Request{
			Body: "action=login&user_id=12345&timestamp=1234567890",
		},
	}

	value, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "12345", value)
}

// 测试 Regex 提取器 - 未匹配
func TestRegexExtractor_NoMatch(t *testing.T) {
	extractor, err := NewRegexExtractor(`id=(\d+)`, config.ExtractorSourceResponse)
	assert.NoError(t, err)

	ctx := &ExtractorContext{
		Response: &types.Response{
			Body: []byte("no match here"),
		},
	}

	_, err = extractor.Extract(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "正则表达式未匹配到数据")
}

// 测试 Header 提取器 - 从响应提取
func TestHeaderExtractor_FromResponse(t *testing.T) {
	extractor := NewHeaderExtractor("X-Request-ID", config.ExtractorSourceResponse)

	ctx := &ExtractorContext{
		Response: &types.Response{
			Headers: map[string]string{
				"X-Request-ID": "req-123-456",
				"Content-Type": "application/json",
			},
		},
	}

	value, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "req-123-456", value)
}

// 测试 Header 提取器 - 从请求提取
func TestHeaderExtractor_FromRequest(t *testing.T) {
	extractor := NewHeaderExtractor("Authorization", config.ExtractorSourceRequest)

	ctx := &ExtractorContext{
		Request: &Request{
			Headers: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "test-client",
			},
		},
	}

	value, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Bearer token123", value)
}

// 测试 Header 提取器 - Header 不存在
func TestHeaderExtractor_NotFound(t *testing.T) {
	extractor := NewHeaderExtractor("X-Missing-Header", config.ExtractorSourceResponse)

	ctx := &ExtractorContext{
		Response: &types.Response{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	_, err := extractor.Extract(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

// 测试 Expression 提取器
func TestExpressionExtractor(t *testing.T) {
	extractor, err := NewExpressionExtractor("{{.first_name}} {{.last_name}}")
	assert.NoError(t, err)

	ctx := &ExtractorContext{
		Variables: map[string]string{
			"first_name": "John",
			"last_name":  "Doe",
		},
	}

	value, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "John Doe", value)
}

// 测试 Expression 提取器 - 变量不存在
func TestExpressionExtractor_MissingVariable(t *testing.T) {
	extractor, err := NewExpressionExtractor("{{.first_name}} {{.last_name}}")
	assert.NoError(t, err)

	ctx := &ExtractorContext{
		Variables: map[string]string{
			"first_name": "John",
			// last_name 缺失
		},
	}

	value, err := extractor.Extract(ctx)
	// template 会将缺失变量替换为 <no value>
	assert.NoError(t, err)
	assert.Contains(t, value, "John")
}

// 测试 TransformPipeline - 模板方式
func TestTransformPipeline_Template(t *testing.T) {
	pipeline := NewTransformPipeline(config.NewVariableResolver())

	transforms := []config.TransformConfig{
		{
			Template: "prefix_{{.value}}_suffix",
		},
	}

	result, err := pipeline.Apply("test", transforms)
	assert.NoError(t, err)
	assert.Equal(t, "prefix_test_suffix", result)
}

// 测试 TransformPipeline - 函数方式（使用 VariableResolver 函数）
func TestTransformPipeline_Function(t *testing.T) {
	resolver := config.NewVariableResolver()
	pipeline := NewTransformPipeline(resolver)

	// 先测试 resolver 本身是否正常
	result, err := resolver.Resolve(`{{upper "hello"}}`)
	assert.NoError(t, err, "VariableResolver 应该能解析 upper 函数")
	assert.Equal(t, "HELLO", result)

	// 测试转换管道
	transforms := []config.TransformConfig{
		{
			Function: "upper",
		},
	}

	result, err = pipeline.Apply("hello", transforms)
	assert.NoError(t, err)
	assert.Equal(t, "HELLO", result)
}

// 测试 TransformPipeline - 链式转换
func TestTransformPipeline_Chain(t *testing.T) {
	resolver := config.NewVariableResolver()
	pipeline := NewTransformPipeline(resolver)

	// 先测试单个函数
	result, err := resolver.Resolve(`{{trim "  hello  "}}`)
	assert.NoError(t, err, "VariableResolver 应该能解析 trim 函数")
	assert.Equal(t, "hello", result)

	result, err = resolver.Resolve(`{{upper "hello"}}`)
	assert.NoError(t, err, "VariableResolver 应该能解析 upper 函数")
	assert.Equal(t, "HELLO", result)

	// 测试链式转换
	transforms := []config.TransformConfig{
		{
			Function: "trim",
		},
		{
			Function: "upper",
		},
	}

	result, err = pipeline.Apply("  hello  ", transforms)
	assert.NoError(t, err)
	assert.Equal(t, "HELLO", result)
}

// 测试 ExtractorManager - 集成测试
func TestExtractorManager_Integration(t *testing.T) {
	configs := []config.ExtractorConfig{
		{
			Name:     "user_id",
			Type:     types.ExtractorTypeJSONPath,
			JSONPath: "$.data.user_id",
			Source:   config.ExtractorSourceResponse,
		},
		{
			Name:   "session_id",
			Type:   types.ExtractorTypeRegex,
			Regex:  `session_id=(\w+)`,
			Source: config.ExtractorSourceRequest,
		},
		{
			Name:   "request_id",
			Type:   types.ExtractorTypeHeader,
			Header: "X-Request-ID",
			Source: config.ExtractorSourceResponse,
		},
	}

	manager, err := NewExtractorManager(configs)
	assert.NoError(t, err)

	ctx := &ExtractorContext{
		Request: &Request{
			Body: "action=login&session_id=abc123&timestamp=1234567890",
		},
		Response: &types.Response{
			Body: []byte(`{"data": {"user_id": "12345", "name": "test"}}`),
			Headers: map[string]string{
				"X-Request-ID": "req-999",
			},
		},
	}

	results := manager.ExtractAll(ctx, map[string]string{})

	assert.Equal(t, "12345", results["user_id"])
	assert.Equal(t, "abc123", results["session_id"])
	assert.Equal(t, "req-999", results["request_id"])
}

// 测试 ExtractorManager - 带转换
func TestExtractorManager_WithTransforms(t *testing.T) {
	configs := []config.ExtractorConfig{
		{
			Name:     "username",
			Type:     types.ExtractorTypeJSONPath,
			JSONPath: "$.username",
			Source:   config.ExtractorSourceResponse,
			Transforms: []config.TransformConfig{
				{
					Function: "trim",
				},
				{
					Function: "upper",
				},
			},
		},
	}

	manager, err := NewExtractorManager(configs)
	assert.NoError(t, err)

	ctx := &ExtractorContext{
		Response: &types.Response{
			Body: []byte(`{"username": "  john  "}`),
		},
	}

	results := manager.ExtractAll(ctx, map[string]string{})

	assert.Equal(t, "JOHN", results["username"])
}

// 测试 ExtractorManager - 默认值
func TestExtractorManager_DefaultValue(t *testing.T) {
	configs := []config.ExtractorConfig{
		{
			Name:     "missing_field",
			Type:     types.ExtractorTypeJSONPath,
			JSONPath: "$.not.exist",
			Source:   config.ExtractorSourceResponse,
			Default:  "default_value",
		},
	}

	manager, err := NewExtractorManager(configs)
	assert.NoError(t, err)

	ctx := &ExtractorContext{
		Response: &types.Response{
			Body: []byte(`{"data": {"id": "123"}}`),
		},
	}

	defaultValues := map[string]string{
		"missing_field": "default_value",
	}

	results := manager.ExtractAll(ctx, defaultValues)

	assert.Equal(t, "default_value", results["missing_field"])
}

// 测试创建无效的提取器
func TestCreateExtractor_Invalid(t *testing.T) {
	// JSONPath 为空
	_, err := createExtractor(config.ExtractorConfig{
		Name:     "test",
		Type:     types.ExtractorTypeJSONPath,
		JSONPath: "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JSONPath不能为空")

	// Regex 为空
	_, err = createExtractor(config.ExtractorConfig{
		Name:  "test",
		Type:  types.ExtractorTypeRegex,
		Regex: "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "正则表达式不能为空")

	// 不支持的类型
	_, err = createExtractor(config.ExtractorConfig{
		Name: "test",
		Type: "UNSUPPORTED",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的提取器类型")
}

// ======================== 参数化和转换高级测试 ========================

// 测试转换 - 字符串处理
func TestTransform_StringOperations(t *testing.T) {
	resolver := config.NewVariableResolver()
	pipeline := NewTransformPipeline(resolver)

	tests := []struct {
		name       string
		input      string
		transforms []config.TransformConfig
		expected   string
	}{
		{
			name:  "upper",
			input: "hello",
			transforms: []config.TransformConfig{
				{Function: "upper"},
			},
			expected: "HELLO",
		},
		{
			name:  "lower",
			input: "HELLO",
			transforms: []config.TransformConfig{
				{Function: "lower"},
			},
			expected: "hello",
		},
		{
			name:  "trim",
			input: "  hello  ",
			transforms: []config.TransformConfig{
				{Function: "trim"},
			},
			expected: "hello",
		},
		{
			name:  "title",
			input: "hello world",
			transforms: []config.TransformConfig{
				{Function: "title"},
			},
			expected: "Hello World",
		},
		{
			name:  "链式: trim + upper",
			input: "  hello  ",
			transforms: []config.TransformConfig{
				{Function: "trim"},
				{Function: "upper"},
			},
			expected: "HELLO",
		},
		{
			name:  "链式: trim + title",
			input: "  hello world  ",
			transforms: []config.TransformConfig{
				{Function: "trim"},
				{Function: "title"},
			},
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pipeline.Apply(tt.input, tt.transforms)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试转换 - 加密和编码
func TestTransform_EncryptionAndEncoding(t *testing.T) {
	resolver := config.NewVariableResolver()
	pipeline := NewTransformPipeline(resolver)

	tests := []struct {
		name       string
		input      string
		transforms []config.TransformConfig
		check      func(t *testing.T, result string)
	}{
		{
			name:  "md5",
			input: "hello",
			transforms: []config.TransformConfig{
				{Function: "md5"},
			},
			check: func(t *testing.T, result string) {
				assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", result)
			},
		},
		{
			name:  "sha1",
			input: "hello",
			transforms: []config.TransformConfig{
				{Function: "sha1"},
			},
			check: func(t *testing.T, result string) {
				assert.Equal(t, "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d", result)
			},
		},
		{
			name:  "sha256",
			input: "hello",
			transforms: []config.TransformConfig{
				{Function: "sha256"},
			},
			check: func(t *testing.T, result string) {
				assert.Len(t, result, 64)
			},
		},
		{
			name:  "base64",
			input: "hello",
			transforms: []config.TransformConfig{
				{Function: "base64"},
			},
			check: func(t *testing.T, result string) {
				assert.Equal(t, "aGVsbG8=", result)
			},
		},
		{
			name:  "链式: upper + md5",
			input: "hello",
			transforms: []config.TransformConfig{
				{Function: "upper"},
				{Function: "md5"},
			},
			check: func(t *testing.T, result string) {
				// MD5("HELLO")
				assert.Len(t, result, 32)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pipeline.Apply(tt.input, tt.transforms)
			assert.NoError(t, err)
			tt.check(t, result)
		})
	}
}

// 测试转换 - 模板方式
func TestTransform_TemplateMode(t *testing.T) {
	pipeline := NewTransformPipeline(config.NewVariableResolver())

	tests := []struct {
		name       string
		input      string
		transforms []config.TransformConfig
		expected   string
	}{
		{
			name:  "简单模板",
			input: "test",
			transforms: []config.TransformConfig{
				{Template: "prefix_{{.value}}_suffix"},
			},
			expected: "prefix_test_suffix",
		},
		{
			name:  "带参数的模板",
			input: "test",
			transforms: []config.TransformConfig{
				{
					Template: "{{.value}}_{{.arg0}}_{{.arg1}}",
					Args:     []interface{}{"param1", "param2"},
				},
			},
			expected: "test_param1_param2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pipeline.Apply(tt.input, tt.transforms)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试 VariableResolver 内置函数
func TestVariableResolver_Functions(t *testing.T) {
	resolver := config.NewVariableResolver()

	tests := []struct {
		name     string
		template string
		check    func(t *testing.T, result string)
	}{
		{
			name:     "randomString",
			template: `{{randomString 10}}`,
			check: func(t *testing.T, result string) {
				assert.Len(t, result, 10)
			},
		},
		{
			name:     "uuid",
			template: `{{uuid}}`,
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "-")
				assert.Len(t, result, 36)
			},
		},
		{
			name:     "unix",
			template: `{{unix}}`,
			check: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
			},
		},
		{
			name:     "unixTime",
			template: `{{unixTime}}`,
			check: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
			},
		},
		{
			name:     "unixNano",
			template: `{{unixNano}}`,
			check: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
			},
		},
		{
			name:     "date",
			template: `{{date "2006-01-02"}}`,
			check: func(t *testing.T, result string) {
				assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, result)
			},
		},
		{
			name:     "randomEmail",
			template: `{{randomEmail}}`,
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "@")
				assert.Contains(t, result, ".com")
			},
		},
		{
			name:     "randomPhone",
			template: `{{randomPhone}}`,
			check: func(t *testing.T, result string) {
				assert.Len(t, result, 11)
				assert.True(t, result[0] == '1')
			},
		},
		{
			name:     "randomIP",
			template: `{{randomIP}}`,
			check: func(t *testing.T, result string) {
				assert.Regexp(t, `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`, result)
			},
		},
		{
			name:     "组合: upper + md5",
			template: `{{md5 (upper "hello")}}`,
			check: func(t *testing.T, result string) {
				assert.Len(t, result, 32)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.template)
			assert.NoError(t, err)
			tt.check(t, result)
		})
	}
}
