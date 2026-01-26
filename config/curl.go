/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 12:52:19
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-31 15:15:55
 * @FilePath: \go-stress\config\curl.go
 * @Description:
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/kamalyes/go-stress/logger"
)

// CurlStyle curl命令风格
type CurlStyle int

const (
	// StyleUnix Unix/Bash风格 (使用 \ 作为续行符和单引号)
	StyleUnix CurlStyle = iota
	// StyleWindowsCmd Windows cmd风格 (使用 ^ 作为转义符)
	StyleWindowsCmd
)

// 预编译的正则表达式
var (
	urlPatterns = []*regexp.Regexp{
		regexp.MustCompile(`curl\s+'([^']+)'`),
		regexp.MustCompile(`curl\s+"([^"]+)"`),
		regexp.MustCompile(`curl\s+([^\s-][^\s]+)`),
		regexp.MustCompile(`--url\s+'([^']+)'`),
		regexp.MustCompile(`--url\s+"([^"]+)"`),
		regexp.MustCompile(`--url\s+([^\s-][^\s]+)`),
	}

	headerPatterns = []*regexp.Regexp{
		regexp.MustCompile(`-H\s+'([^']+)'`),
		regexp.MustCompile(`-H\s+"([^"]+)"`),
		regexp.MustCompile(`--header\s+'([^']+)'`),
		regexp.MustCompile(`--header\s+"([^"]+)"`),
	}

	methodPatterns = []*regexp.Regexp{
		regexp.MustCompile(`-X\s+'([^']+)'`),
		regexp.MustCompile(`-X\s+"([^"]+)"`),
		regexp.MustCompile(`-X\s+([A-Z]+)`),
		regexp.MustCompile(`--request\s+'([^']+)'`),
		regexp.MustCompile(`--request\s+"([^"]+)"`),
		regexp.MustCompile(`--request\s+([A-Z]+)`),
	}
)

// Windows cmd 转义序列（顺序很重要！）
var windowsEscapes = []struct {
	from, to string
}{
	{`^\^"`, `\"`}, // 组合转义必须最先处理
	{`^"`, `"`},
	{`^^`, `^`},
	{`^{`, `{`},
	{`^}`, `}`},
	{`^[`, `[`},
	{`^]`, `]`},
	{`^:`, `:`},
	{`^,`, `,`},
	{`^ `, ` `},
}

// CurlParser curl命令解析器
type CurlParser struct {
	raw   string
	style CurlStyle
}

// ParseCurlFile 从文件解析curl命令
func ParseCurlFile(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("curl文件路径不能为空")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取curl文件失败: %w", err)
	}

	return ParseCurlCommand(string(data))
}

// ParseCurlCommand 解析curl命令字符串
func ParseCurlCommand(curlCmd string) (*Config, error) {
	style := detectCurlStyle(curlCmd)
	logger.Default.Debug("检测到curl风格: %v", style)

	parser := &CurlParser{
		raw:   curlCmd,
		style: style,
	}
	config, err := parser.parse()
	if err != nil {
		return nil, err
	}

	// 🔥 创建变量解析器但不提前解析，让变量在每次请求时动态生成
	config.VarResolver = NewVariableResolver()

	return config, nil
}

// detectCurlStyle 检测curl命令风格
// 策略：统计 ^ 和 \ 的出现频率，Windows cmd 风格会有大量 ^ 转义
func detectCurlStyle(cmd string) CurlStyle {
	backslashCount := strings.Count(cmd, "\\")
	caretCount := strings.Count(cmd, "^")

	logger.Default.Debug("检测风格 - 反斜杠: %d, 脱字符: %d", backslashCount, caretCount)

	// Windows cmd 风格特征：^ 数量 > \ 数量的 2 倍
	if caretCount > backslashCount*2 {
		return StyleWindowsCmd
	}

	return StyleUnix
}

// resolveConfigVariables 解析配置中的变量
func resolveConfigVariables(resolver *VariableResolver, config *Config) error {
	// 解析URL
	if config.URL != "" {
		resolved, err := resolver.Resolve(config.URL)
		if err != nil {
			return fmt.Errorf("解析URL变量失败: %w", err)
		}
		config.URL = resolved
	}

	// 解析Body
	if config.Body != "" {
		resolved, err := resolver.Resolve(config.Body)
		if err != nil {
			return fmt.Errorf("解析Body变量失败: %w", err)
		}
		config.Body = resolved
	}

	// 解析Headers
	for k, v := range config.Headers {
		resolved, err := resolver.Resolve(v)
		if err != nil {
			return fmt.Errorf("解析Header变量失败 %s: %w", k, err)
		}
		config.Headers[k] = resolved
	}

	return nil
}

// parse 解析curl命令
func (p *CurlParser) parse() (*Config, error) {
	// 根据风格选择不同的规范化策略
	var normalized string
	if p.style == StyleWindowsCmd {
		normalized = p.normalizeWindowsCommand()
	} else {
		normalized = p.normalizeUnixCommand()
	}

	config := &Config{
		Protocol: "http",
		Method:   "GET",
		Headers:  make(map[string]string),
	}

	// 解析URL
	if err := p.parseURL(normalized, config); err != nil {
		return nil, err
	}

	// 解析请求头
	p.parseHeaders(normalized, config)

	// 解析请求方法
	p.parseMethod(normalized, config)

	// 解析请求体
	p.parseBody(normalized, config)

	// 解析其他参数
	p.parseOtherOptions(normalized, config)

	return config, nil
}

// normalizeUnixCommand 规范化 Unix/Bash 风格命令（\ 续行符）
func (p *CurlParser) normalizeUnixCommand() string {
	return p.joinContinuationLines(p.raw, "\\")
}

// normalizeWindowsCommand 规范化 Windows cmd 风格命令（^ 转义符）
func (p *CurlParser) normalizeWindowsCommand() string {
	result := p.joinContinuationLines(p.raw, "^")

	// 按顺序应用转义映射（顺序很重要：先处理组合转义）
	for _, esc := range windowsEscapes {
		result = strings.ReplaceAll(result, esc.from, esc.to)
	}

	return result
}

// joinContinuationLines 合并多行续行符命令为单行
func (p *CurlParser) joinContinuationLines(raw, continuationChar string) string {
	lines := strings.Split(raw, "\n")
	var result strings.Builder

	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, continuationChar)
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString(" ")
		}
	}

	return result.String()
}

// parseURL 解析URL
func (p *CurlParser) parseURL(cmd string, config *Config) error {
	for _, pattern := range urlPatterns {
		if matches := pattern.FindStringSubmatch(cmd); len(matches) > 1 {
			config.URL = strings.TrimSpace(matches[1])
			return nil
		}
	}
	return fmt.Errorf("未找到URL")
}

// parseHeaders 解析请求头
func (p *CurlParser) parseHeaders(cmd string, config *Config) {
	for _, pattern := range headerPatterns {
		matches := pattern.FindAllStringSubmatch(cmd, -1)
		for _, match := range matches {
			if len(match) > 1 {
				if key, value, ok := parseHeaderKeyValue(match[1]); ok {
					config.Headers[key] = value
				}
			}
		}
	}
}

// parseHeaderKeyValue 解析 "Key: Value" 格式的请求头
func parseHeaderKeyValue(header string) (key, value string, ok bool) {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
	}
	return "", "", false
}

// parseMethod 解析请求方法
func (p *CurlParser) parseMethod(cmd string, config *Config) {
	for _, pattern := range methodPatterns {
		if matches := pattern.FindStringSubmatch(cmd); len(matches) > 1 {
			config.Method = strings.ToUpper(strings.TrimSpace(matches[1]))
			return
		}
	}

	// 如果有 --data 相关参数，默认为 POST
	if strings.Contains(cmd, "--data") || strings.Contains(cmd, "--data-raw") || strings.Contains(cmd, "--data-binary") {
		config.Method = "POST"
	}
}

// parseBody 解析请求体
func (p *CurlParser) parseBody(cmd string, config *Config) {
	// 查找 --data-raw, --data, -d 参数位置
	dataKeywords := []string{"--data-raw", "--data", "-d"}
	var dataIdx int = -1

	for _, keyword := range dataKeywords {
		if idx := strings.Index(cmd, keyword); idx != -1 {
			dataIdx = idx
			break
		}
	}

	if dataIdx == -1 {
		return
	}

	// 从data参数位置提取引号内容
	remaining := cmd[dataIdx:]
	if body, ok := p.extractQuotedContent(remaining); ok {
		logger.Default.Debug("提取body成功 (前200字符): %s", truncateString(body, 200))
		config.Body = formatJSONIfPossible(body)
	}
}

// extractQuotedContent 提取引号内的内容（自动检测单引号或双引号）
func (p *CurlParser) extractQuotedContent(s string) (string, bool) {
	singleIdx := strings.Index(s, "'")
	doubleIdx := strings.Index(s, "\"")

	// 根据最先出现的引号类型决定提取方式
	if singleIdx != -1 && (doubleIdx == -1 || singleIdx < doubleIdx) {
		// Unix 风格：单引号，内容不转义
		return extractUntil(s[singleIdx+1:], '\''), true
	} else if doubleIdx != -1 {
		// Windows 风格：双引号，需要处理转义
		body := extractEscapedUntil(s[doubleIdx+1:], '"')
		return unescapeJSON(body), true
	}

	return "", false
}

// extractUntil 提取字符串直到遇到指定字符（不处理转义）
func extractUntil(s string, delimiter rune) string {
	var result strings.Builder
	for _, ch := range s {
		if ch == delimiter {
			return result.String()
		}
		result.WriteRune(ch)
	}
	return result.String()
}

// extractEscapedUntil 提取字符串直到遇到未转义的指定字符（处理转义）
func extractEscapedUntil(s string, delimiter rune) string {
	var result strings.Builder
	escaped := false

	for _, ch := range s {
		if escaped {
			result.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			result.WriteRune(ch)
			escaped = true
			continue
		}

		if ch == delimiter {
			return result.String()
		}

		result.WriteRune(ch)
	}
	return result.String()
}

// unescapeJSON 处理 JSON 字符串中的转义序列
func unescapeJSON(s string) string {
	var result strings.Builder
	escaped := false

	for _, ch := range s {
		if escaped {
			switch ch {
			case 'n':
				result.WriteRune('\n')
			case 't':
				result.WriteRune('\t')
			case 'r':
				result.WriteRune('\r')
			case '\\':
				result.WriteRune('\\')
			case '"':
				result.WriteRune('"')
			case '/':
				result.WriteRune('/')
			case 'b':
				result.WriteRune('\b')
			case 'f':
				result.WriteRune('\f')
			default:
				result.WriteRune('\\')
				result.WriteRune(ch)
			}
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		result.WriteRune(ch)
	}
	return result.String()
}

// formatJSONIfPossible 尝试格式化为美化的 JSON，失败则返回原字符串
func formatJSONIfPossible(body string) string {
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(body), &jsonObj); err == nil {
		if formatted, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
			return string(formatted)
		}
	}
	return body
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// parseOtherOptions 解析其他选项
func (p *CurlParser) parseOtherOptions(cmd string, config *Config) {
	// 解析 --insecure 或 -k (跳过SSL验证)
	if strings.Contains(cmd, "--insecure") || strings.Contains(cmd, " -k ") || strings.Contains(cmd, " -k$") {
		// 可以在这里设置SSL相关配置
	}

	// 解析 --compressed (接受压缩)
	if strings.Contains(cmd, "--compressed") {
		// 可以在这里设置压缩相关配置
	}

	// 解析URL中的协议类型
	if config.URL != "" {
		if strings.HasPrefix(config.URL, "https://") {
			// HTTPS配置
		} else if strings.HasPrefix(config.URL, "ws://") || strings.HasPrefix(config.URL, "wss://") {
			config.Protocol = "websocket"
		}
	}
}

// GetURLHost 从URL中提取主机名
func GetURLHost(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Host
}

// GetURLPath 从URL中提取路径
func GetURLPath(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Path
}
