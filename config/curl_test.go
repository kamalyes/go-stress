/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 12:52:19
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-31 15:58:30
 * @FilePath: \go-stress\config\curl_test.go
 * @Description:
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package config

import (
	"os"
	"testing"

	"github.com/kamalyes/go-stress/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewCurlParser(t *testing.T) {
	curlCmd := `curl 'http://localhost:8081/v1/messages/send' \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer token123' \
  --data-raw '{"content":"test-message","priority":1}' \
  --insecure`

	cfg, err := NewCurlParser(curlCmd, logger.New())
	assert.NoError(t, err, "解析curl命令失败")

	// 验证URL
	assert.Equal(t, "http://localhost:8081/v1/messages/send", cfg.URL, "URL解析错误")

	// 验证方法
	assert.Equal(t, "POST", cfg.Method, "方法解析错误")

	// 验证协议
	assert.Equal(t, "http", cfg.Protocol.String(), "协议解析错误")

	// 验证请求头
	assert.NotEmpty(t, cfg.Headers, "请求头解析失败")
	assert.Equal(t, "application/json", cfg.Headers["Content-Type"], "Content-Type解析错误")
	assert.Equal(t, "Bearer token123", cfg.Headers["Authorization"], "Authorization解析错误")

	// 验证请求体
	assert.NotEmpty(t, cfg.Body, "请求体解析失败")

	t.Logf("解析成功！")
	t.Logf("URL: %s", cfg.URL)
	t.Logf("方法: %s", cfg.Method)
	t.Logf("请求头数量: %d", len(cfg.Headers))
	t.Logf("请求体: %s", cfg.Body)
}

func TestParseCurlFile(t *testing.T) {
	// 测试解析curl文件
	data, err := os.ReadFile("../testserver/example.curl.txt")
	assert.NoError(t, err, "读取curl文件失败")

	cfg, err := NewCurlParser(string(data), logger.New())
	assert.NoError(t, err, "解析curl文件失败")
	assert.NotEmpty(t, cfg.URL, "URL解析失败")
	assert.NotEmpty(t, cfg.Method, "方法解析失败")

	t.Logf("文件解析成功！")
	t.Logf("URL: %s", cfg.URL)
	t.Logf("方法: %s", cfg.Method)
	t.Logf("请求头数量: %d", len(cfg.Headers))
}

func TestParseCurlWithDoubleQuotes(t *testing.T) {
	curlCmd := `curl "http://example.com/api" -H "Content-Type: application/json" -X POST --data "{\"key\":\"value\"}"`

	cfg, err := NewCurlParser(curlCmd, logger.New())
	assert.NoError(t, err, "解析curl命令失败")
	assert.Equal(t, "http://example.com/api", cfg.URL, "URL解析错误")
	assert.Equal(t, "POST", cfg.Method, "方法解析错误")

	t.Logf("双引号格式解析成功！")
}

func TestParseCurlGET(t *testing.T) {
	curlCmd := `curl 'https://api.example.com/users/123' -H 'Accept: application/json'`

	cfg, err := NewCurlParser(curlCmd, logger.New())
	assert.NoError(t, err, "解析curl命令失败")
	assert.Equal(t, "GET", cfg.Method, "GET方法解析错误")
	assert.Empty(t, cfg.Body, "GET请求不应该有body")

	t.Logf("GET请求解析成功！")
}

// TestParseCurlUnixStyle 测试 Unix/Bash 风格的 curl（使用 \ 续行符和单引号）
func TestParseCurlUnixStyle(t *testing.T) {
	curlCmd := `curl 'http://localhost:8081/v1/messages/send' \
  -H 'Accept: application/json' \
  -H 'Accept-Language: zh-CN,zh;q=0.9' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImFjY2Vzc19jb250cm9sIiwidHlwIjoiSldUIn0' \
  -H 'X-Nonce: hm089tg6v3inewqv1klurr' \
  -H 'X-Session-ID: 08884d6d8d9fffa5456a359f67b48843' \
  -H 'X-Signature: L/CzPEa4a9BV8Sme9Jj8qVpYXb7496xiWeRFTnhfkqk=' \
  -H 'X-Timestamp: 1767160549' \
  --data-raw '{"session_id":"08884d6d8d9fffa5456a359f67b48843","sender_id":"1991706697093091328","content":"👤 测试内容","priority":2}' \
  --insecure`

	cfg, err := NewCurlParser(curlCmd, logger.New())
	assert.NoError(t, err, "解析Unix风格curl命令失败")

	// 验证URL
	assert.Equal(t, "http://localhost:8081/v1/messages/send", cfg.URL, "URL解析错误")

	// 验证方法
	assert.Equal(t, "POST", cfg.Method, "方法解析错误")

	// 验证请求头
	assert.Equal(t, "application/json", cfg.Headers["Content-Type"], "Content-Type解析错误")
	assert.Equal(t, "08884d6d8d9fffa5456a359f67b48843", cfg.Headers["X-Session-ID"], "X-Session-ID解析错误")

	// 验证Body包含emoji
	assert.NotEmpty(t, cfg.Body, "Body解析失败")
	assert.Contains(t, cfg.Body, "👤", "Body中应该包含emoji字符")

	t.Logf("Unix风格curl解析成功！")
	t.Logf("Body: %s", cfg.Body)
}

// TestParseCurlWindowsStyle 测试 Windows cmd 风格的 curl（使用 ^ 转义符）
func TestParseCurlWindowsStyle(t *testing.T) {
	curlCmd := `curl ^"http://localhost:8081/v1/messages/send^" ^
  -H ^"Accept: application/json^" ^
  -H ^"Content-Type: application/json^" ^
  -H ^"Authorization: Bearer token123^" ^
  -H ^"X-Session-ID: 08884d6d8d9fffa5456a359f67b48843^" ^
  --data-raw ^"^{^\^"session_id^\^":^\^"08884d6d8d9fffa5456a359f67b48843^\^",^\^"sender_id^\^":^\^"1991706697093091328^\^",^\^"content^\^":^\^"测试内容^\^",^\^"priority^\^":2^}^" ^
  --insecure`

	cfg, err := NewCurlParser(curlCmd, logger.New())
	assert.NoError(t, err, "解析Windows风格curl命令失败")

	// 验证URL
	assert.Equal(t, "http://localhost:8081/v1/messages/send", cfg.URL, "URL解析错误")

	// 验证方法
	assert.Equal(t, "POST", cfg.Method, "方法解析错误")

	// 验证请求头
	assert.Equal(t, "application/json", cfg.Headers["Content-Type"], "Content-Type解析错误")
	assert.Equal(t, "Bearer token123", cfg.Headers["Authorization"], "Authorization解析错误")

	// 验证Body
	assert.NotEmpty(t, cfg.Body, "Body解析失败")
	assert.Contains(t, cfg.Body, "session_id", "Body应该包含session_id字段")

	t.Logf("Windows风格curl解析成功！")
	t.Logf("Body: %s", cfg.Body)
}
