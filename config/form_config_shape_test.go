/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-11 20:31:52
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-11 20:31:52
 * @FilePath: \go-stress\config\form_config_shape_test.go
 * @Description: 分布式管理页表单生成配置的形态契约测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package config

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

// formConfigShapeFixture 模拟分布式管理页表单（statistics/distributed.js 的
// generateConfigFromForm）修复后生成的配置：path 拼进完整 URL、校验规则与
// 提取器挂在唯一的 API 上、duration/timeout 用 "300s" 形态
func formConfigShapeFixture() map[string]any {
	return map[string]any{
		"protocol":    "http",
		"method":      "GET",
		"url":         "http://localhost:3000/api/health",
		"concurrency": 50,
		"requests":    1000,
		"duration":    "300s",
		"timeout":     "10s",
		"headers":     map[string]string{"Accept": "application/json"},
		"apis": []map[string]any{
			{
				"name": "main",
				"verify": []map[string]any{
					{"type": "STATUS_CODE", "expect": 200},
					{"type": "CONTAINS", "expect": `"status":"ok"`},
				},
				"extractors": []map[string]any{
					{"name": "rid", "type": "jsonpath", "jsonpath": "$.id"},
				},
			},
		},
	}
}

// TestFormConfigShapeAccepted 表单生成的配置必须能被 Loader 原样接受：
// master 创建任务时先按 YAML 再按 JSON 两段式解析（见 http_server.parseTaskConfig），
// 两条路径都得过，否则界面上建任务直接失败
func TestFormConfigShapeAccepted(t *testing.T) {
	t.Run("YAML模式", func(t *testing.T) {
		yamlBytes, err := yaml.Marshal(formConfigShapeFixture())
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := NewLoader().LoadFromBytes(yamlBytes, "yaml")
		if err != nil {
			t.Fatalf("YAML 解析失败: %v\n%s", err, yamlBytes)
		}
		assertFormConfigShape(t, cfg)
	})

	t.Run("JSON模式", func(t *testing.T) {
		jsonBytes, err := json.Marshal(formConfigShapeFixture())
		if err != nil {
			t.Fatal(err)
		}
		// master 的两段式：先按 YAML 试（JSON 是 YAML 的子集），
		// "300s" 这类 duration 字符串靠 yaml.v3 对 time.Duration 的特殊处理存活
		cfg, err := NewLoader().LoadFromBytes(jsonBytes, "yaml")
		if err != nil {
			cfg, err = NewLoader().LoadFromBytes(jsonBytes, "json")
			if err != nil {
				t.Fatalf("YAML/JSON 双路解析均失败: %v\n%s", err, jsonBytes)
			}
		}
		assertFormConfigShape(t, cfg)
	})
}

// TestFormConfigLegacyShapeRejected 旧表单形态必须被拒绝：
// 顶层 path/extractors 不存在、顶层 verify 只支持单条（*VerifyConfig）、
// 校验类型必须用后端常量（STATUS_CODE 而非 status）。
// Loader 开了「拒绝未知字段」，这些写法以前会让任务创建直接报错
func TestFormConfigLegacyShapeRejected(t *testing.T) {
	legacy := map[string]any{
		"protocol": "http",
		"url":      "http://localhost:3000",
		"path":     "/api/health",
		"verify": []map[string]any{
			{"type": "status", "status_code": 200},
		},
		"extractors": []map[string]any{
			{"name": "rid", "type": "jsonpath", "jsonpath": "$.id"},
		},
	}
	jsonBytes, _ := json.Marshal(legacy)
	if _, err := NewLoader().LoadFromBytes(jsonBytes, "json"); err == nil {
		t.Fatal("旧表单形态不该被接受，接受即意味着字段对齐又被破坏")
	}
}

func assertFormConfigShape(t *testing.T, cfg *Config) {
	t.Helper()
	if len(cfg.APIs) != 1 {
		t.Fatalf("APIs 数量 = %d, want 1", len(cfg.APIs))
	}
	api := cfg.APIs[0]
	if len(api.Verify) != 2 {
		t.Fatalf("校验规则数量 = %d, want 2", len(api.Verify))
	}
	if api.Verify[0].Type != "STATUS_CODE" || len(api.Extractors) != 1 {
		t.Fatalf("校验/提取器形态不对: verify=%+v extractors=%+v", api.Verify, api.Extractors)
	}
	if cfg.URL != "http://localhost:3000/api/health" {
		t.Fatalf("URL = %q, want 拼接后的完整 URL", cfg.URL)
	}
	if cfg.Timeout != 10_000_000_000 || cfg.Duration != 300_000_000_000 {
		t.Fatalf("duration/timeout 解析不对: duration=%v timeout=%v", cfg.Duration, cfg.Timeout)
	}
}
