/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-11-20 12:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-22 17:15:11
 * @FilePath: \go-stress\config\loader.go
 * @Description: 配置加载器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kamalyes/go-stress/types"
	"gopkg.in/yaml.v3"
)

// Loader 配置加载器
type Loader struct {
	varResolver *VariableResolver
}

// NewLoader 创建配置加载器
func NewLoader() *Loader {
	return &Loader{
		varResolver: NewVariableResolver(),
	}
}

// LoadFromFile 从文件加载配置
func (l *Loader) LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	config := DefaultConfig()

	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("解析YAML配置失败: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("解析JSON配置失败: %w", err)
		}
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	// 设置上下文
	l.varResolver.SetVariables(config.Variables)
	// 注入变量解析器到 config
	config.VarResolver = l.varResolver

	// 处理多API配置继承
	if err := l.mergeAPIsWithCommon(config); err != nil {
		return nil, fmt.Errorf("合并API配置失败: %w", err)
	}

	// 调试输出：查看合并后的API配置
	if len(config.APIs) > 0 {
		fmt.Printf("📋 配置了 %d 个API:\n", len(config.APIs))
		for i, api := range config.APIs {
			fmt.Printf("  [%d] %s: %s %s\n", i+1, api.Name, api.Method, api.URL)
		}
	}

	// 验证配置
	if err := l.validate(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// mergeAPIsWithCommon 将公共配置合并到各个API配置中
func (l *Loader) mergeAPIsWithCommon(config *Config) error {
	// 如果没有定义APIs，则使用单个配置模式（向后兼容）
	if len(config.APIs) == 0 {
		return nil
	}

	// 遍历每个API配置，合并公共配置
	for i := range config.APIs {
		api := &config.APIs[i]

		// 构建完整URL
		// 优先级：api.URL > api.Host+api.Path > config.Host+api.Path > config.URL
		if api.URL == "" {
			// 继承Host
			host := api.Host
			if host == "" && config.Host != "" {
				host = config.Host
			}

			// 如果有Host和Path，组合成完整URL
			if host != "" && api.Path != "" {
				api.URL = host + api.Path
			} else if host != "" {
				// 只有Host没有Path，使用Host作为URL
				api.URL = host
			} else if api.Path != "" {
				// 只有Path没有Host，Path就是完整URL（向后兼容）
				api.URL = api.Path
			} else if config.URL != "" {
				// 使用公共URL（向后兼容）
				api.URL = config.URL
			}
		}

		// 如果还是没有URL，报错
		if api.URL == "" {
			return fmt.Errorf("第%d个API [%s] 的URL不能为空（需要URL或Host+Path）", i+1, api.Name)
		}

		// 继承Method
		if api.Method == "" && config.Method != "" {
			api.Method = config.Method
		}
		if api.Method == "" {
			api.Method = "GET" // 默认值
		}

		// 合并Headers（公共headers + API特定headers，API的优先）
		if api.Headers == nil {
			api.Headers = make(map[string]string)
		}
		// 先复制公共headers
		for k, v := range config.Headers {
			if _, exists := api.Headers[k]; !exists {
				api.Headers[k] = v
			}
		}

		// 继承Body
		if api.Body == "" && config.Body != "" {
			api.Body = config.Body
		}

		// 继承Verify配置
		if len(api.Verify) == 0 {
			api.Verify = []VerifyConfig{*config.Verify}
		}

		// 设置默认权重
		if api.Weight <= 0 {
			api.Weight = 1
		}

	}

	return nil
}

// validate 验证配置
func (l *Loader) validate(config *Config) error {
	fmt.Printf("🔍 验证配置: APIs数量=%d, config.URL=%s\n", len(config.APIs), config.URL)

	// 如果定义了APIs，已经在mergeAPIsWithCommon中验证过了
	if len(config.APIs) > 0 {
		fmt.Printf("✅ 使用多API模式，跳过单URL验证\n")
		// APIs配置已经通过merge验证
		// 这里只需要验证基础配置
	} else {
		fmt.Printf("⚠️ 单API模式，检查URL\n")
		// 单API模式，验证URL
		if config.URL == "" {
			return fmt.Errorf("URL不能为空")
		}
	}

	if config.Concurrency == 0 {
		return fmt.Errorf("并发数必须大于0")
	}

	if config.Requests == 0 && config.Duration == 0 {
		return fmt.Errorf("请求数和持续时间至少要设置一个")
	}

	// 协议特定验证
	switch config.Protocol {
	case types.ProtocolGRPC:
		if config.GRPC == nil {
			return fmt.Errorf("gRPC配置不能为空")
		}
		if !config.GRPC.UseReflection && config.GRPC.ProtoFile == "" {
			return fmt.Errorf("未启用反射时必须指定proto文件")
		}
		if config.GRPC.Service == "" || config.GRPC.Method == "" {
			return fmt.Errorf("gRPC服务名和方法名不能为空")
		}
	}

	return nil
}

// GetVariableResolver 获取变量解析器
func (l *Loader) GetVariableResolver() *VariableResolver {
	return l.varResolver
}
