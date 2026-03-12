/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 09:19:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 10:19:00
 * @FilePath: \go-stress\executor\boundary_extractor_test.go
 * @Description: 边界提取、第 N 次匹配与 Header 大小写兼容测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"encoding/json"
	"testing"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoundaryExtractor(t *testing.T) {
	for _, tc := range []struct {
		name, left, right, body string
		match                   int
		want                    string
		fail                    bool
	}{
		{"first match by default", "[", "]", "[first] [second]", 0, "first", false},
		{"second match", "[", "]", "[first] [second]", 2, "second", false},
		{"literal regex characters", "a.*[", "]+$", "a.*[token]+$", 1, "token", false},
		{"multiline and unicode", "开始", "结束", "开始\r\n 中文 \n结束", 1, "\r\n 中文 \n", false},
		{"empty capture", "[", "]", "[]", 1, "", false},
		{"empty left", "", "]", "prefix]rest", 1, "prefix", false},
		{"empty right", "[", "", "prefix[value", 1, "value", false},
		{"empty boundaries", "", "", "whole body", 1, "whole body", false},
		{"nested left boundaries follow JMeter", "[", "]", "[[value]]", 2, "value", false},
		{"missing left", "[", "]", "value]", 1, "", true},
		{"missing right", "[", "]", "[value", 1, "", true},
		{"not enough matches", "[", "]", "[value]", 2, "", true},
		{"empty boundary has one match", "", "]", "a]b]", 2, "", true},
		{"blank body", "", "", " \t\r\n", 1, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex, err := NewBoundaryExtractor(tc.left, tc.right, "", tc.match)
			require.NoError(t, err)
			value, err := ex.Extract(&ExtractorContext{Response: &types.Response{Body: []byte(tc.body)}})
			if tc.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, value)
			}
		})
	}
}

func TestExtractorMatchNumberAndConfigSerialization(t *testing.T) {
	for _, kind := range []types.ExtractorType{types.ExtractorTypeRegex, types.ExtractorTypeBoundary} {
		t.Run(string(kind), func(t *testing.T) {
			cfg := config.ExtractorConfig{Name: "token", Type: kind, Regex: `\[([^]]*)\]`,
				LeftBoundary: "[", RightBoundary: "]", MatchNumber: 2}
			data, err := json.Marshal(cfg)
			require.NoError(t, err)
			var restored config.ExtractorConfig
			require.NoError(t, json.Unmarshal(data, &restored))
			require.Equal(t, cfg, restored, "分布式配置经 JSON 传输后必须保留新增字段")
			ex, err := createExtractor(restored)
			require.NoError(t, err)
			value, err := ex.Extract(&ExtractorContext{Response: &types.Response{Body: []byte("[first] [second]")}})
			require.NoError(t, err)
			assert.Equal(t, "second", value)
			_, err = ex.Extract(&ExtractorContext{Response: &types.Response{Body: []byte("[only]")}})
			assert.Error(t, err)
			cfg.MatchNumber = -1
			_, err = createExtractor(cfg)
			assert.Error(t, err)
		})
	}
}

func TestBoundaryManagerDefaultsAndTransforms(t *testing.T) {
	manager, err := NewExtractorManager([]config.ExtractorConfig{{
		Name: "token", Type: types.ExtractorTypeBoundary, LeftBoundary: "[", RightBoundary: "]",
		Transforms: []config.TransformConfig{{Function: "upper"}},
	}}, logger.NewEmptyLogger())
	require.NoError(t, err)
	got := manager.ExtractAll(&ExtractorContext{Response: &types.Response{Body: []byte("[token]")}}, nil)
	assert.Equal(t, "TOKEN", got["token"])
	got = manager.ExtractAll(&ExtractorContext{Response: &types.Response{Body: []byte("missing")}}, map[string]string{"token": "fallback"})
	assert.Equal(t, "fallback", got["token"])
}

func TestExtractorRequestSourceIsolated(t *testing.T) {
	for _, kind := range []types.ExtractorType{types.ExtractorTypeRegex, types.ExtractorTypeBoundary, types.ExtractorTypeHeader} {
		t.Run(string(kind), func(t *testing.T) {
			ex, err := createExtractor(config.ExtractorConfig{Name: "id", Type: kind,
				Source: config.ExtractorSourceRequest, Regex: `\[([^]]*)\]`,
				LeftBoundary: "[", RightBoundary: "]", Header: "x-token"})
			require.NoError(t, err)
			ctx := &ExtractorContext{Response: &types.Response{Body: []byte("[response]"), Headers: map[string]string{"X-Token": "response"}}}
			_, err = ex.Extract(ctx)
			assert.Error(t, err, "缺少请求时不能误取响应中的同名值")
			ctx.Request = &types.Request{Body: "[request]", Headers: map[string]string{"X-Token": "request"}}
			value, err := ex.Extract(ctx)
			require.NoError(t, err)
			assert.Equal(t, "request", value)
		})
	}
	_, err := NewBoundaryExtractor("[", "]", "unsupported", 1)
	assert.Error(t, err)
}

func TestHeaderExtractorCaseInsensitive(t *testing.T) {
	for _, source := range []config.ExtractorSource{config.ExtractorSourceRequest, config.ExtractorSourceResponse} {
		for _, header := range []string{"x-request-id", "X-Request-Id", "X-REQUEST-ID"} {
			ex := NewHeaderExtractor(header, source)
			ctx := &ExtractorContext{
				Request:  &types.Request{Headers: map[string]string{"X-Request-ID": "request"}},
				Response: &types.Response{Headers: map[string]string{"X-Request-ID": "response"}},
			}
			value, err := ex.Extract(ctx)
			require.NoError(t, err)
			assert.Equal(t, string(source), value)
		}
	}
}
