/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 09:59:13
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 11:15:15
 * @FilePath: \go-stress\config\variable.go
 * @Description: 变量解析器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package config

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/kamalyes/go-toolbox/pkg/convert"
	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/netx"
	"github.com/kamalyes/go-toolbox/pkg/osx"
	"github.com/kamalyes/go-toolbox/pkg/random"
	"github.com/kamalyes/go-toolbox/pkg/stringx"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
)

// VariableResolver 变量解析器
type VariableResolver struct {
	variables map[string]any
	sequence  *syncx.Uint64 // 使用 syncx.Uint64
	funcMap   template.FuncMap
}

// NewVariableResolver 创建变量解析器
func NewVariableResolver() *VariableResolver {
	v := &VariableResolver{
		variables: make(map[string]any),
		sequence:  syncx.NewUint64(0), // 使用 syncx
	}

	v.funcMap = template.FuncMap{
		// 环境变量
		"env": func(key string) string {
			return os.Getenv(key)
		},

		// 序列号
		"seq": func() uint64 {
			return v.sequence.Add(1) // 使用 syncx
		},

		// 时间函数
		"now": func() time.Time {
			return time.Now()
		},
		"unix": func() int64 {
			return time.Now().Unix()
		},
		"unixTime": func() int64 { // unix 的别名
			return time.Now().Unix()
		},
		"unixNano": func() int64 {
			return time.Now().UnixNano()
		},
		"timestamp": func() int64 {
			return time.Now().UnixMilli()
		},
		"date": func(format string) string {
			return time.Now().Format(format)
		},
		"dateAdd": func(duration string) time.Time {
			d, _ := time.ParseDuration(duration)
			return time.Now().Add(d)
		},
		"dateFormat": func(t time.Time, format string) string {
			return t.Format(format)
		},

		// 随机函数 - 基础
		"randomInt": func(min, max int) int {
			return random.RandInt(min, max)
		},
		"randomFloat": func(min, max float64) float64 {
			return random.RandFloat(min, max)
		},
		"randomString": func(length int) string {
			return random.RandString(length, random.LOWERCASE|random.CAPITAL|random.NUMBER)
		},
		"randomAlpha": func(length int) string {
			return random.RandString(length, random.LOWERCASE|random.CAPITAL)
		},
		"randomNumber": func(length int) string {
			return random.RandString(length, random.NUMBER)
		},
		"randomUUID": func() string {
			return random.UUID()
		},
		"uuid": func() string { // randomUUID 的别名
			return random.UUID()
		},
		"randomBool": func() bool {
			return random.FRandBool()
		},

		// 随机函数 - 特殊
		"randomEmail": func() string {
			return fmt.Sprintf("user_%s@example.com", random.RandString(8, random.LOWERCASE|random.NUMBER))
		},
		"randomPhone": func() string {
			return fmt.Sprintf("1%s", random.RandString(10, random.NUMBER))
		},
		"randomIP": func() string {
			return fmt.Sprintf("%d.%d.%d.%d",
				random.RandInt(1, 255),
				random.RandInt(0, 255),
				random.RandInt(0, 255),
				random.RandInt(1, 255))
		},
		"randomMAC": func() string {
			mac := make([]byte, 6)
			for i := range mac {
				mac[i] = byte(random.RandInt(0, 255))
			}
			return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
				mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
		},
		"randomUserAgent": func() string {
			agents := []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
			}
			return agents[random.RandInt(0, len(agents)-1)]
		},

		// 增强随机函数 - 业务场景
		"randomName": func() string {
			firstNames := []string{"Zhang", "Wang", "Li", "Liu", "Chen", "Yang", "Huang", "Zhao", "Wu", "Zhou"}
			lastNames := []string{"Wei", "Fang", "Lei", "Na", "Ming", "Qiang", "Jing", "Hua", "Ping", "Tao"}
			return firstNames[random.RandInt(0, len(firstNames)-1)] + lastNames[random.RandInt(0, len(lastNames)-1)]
		},
		"randomCity": func() string {
			cities := []string{"Beijing", "Shanghai", "Guangzhou", "Shenzhen", "Hangzhou", "Chengdu", "Wuhan", "Nanjing", "Xi'an", "Chongqing"}
			return cities[random.RandInt(0, len(cities)-1)]
		},
		"randomCountry": func() string {
			countries := []string{"China", "USA", "Japan", "UK", "Germany", "France", "Canada", "Australia", "India", "Brazil"}
			return countries[random.RandInt(0, len(countries)-1)]
		},
		"randomColor": func() string {
			colors := []string{"red", "blue", "green", "yellow", "purple", "orange", "pink", "black", "white", "gray"}
			return colors[random.RandInt(0, len(colors)-1)]
		},
		"randomHexColor": func() string {
			return fmt.Sprintf("#%02x%02x%02x", random.RandInt(0, 255), random.RandInt(0, 255), random.RandInt(0, 255))
		},
		"randomDate": func() string {
			// 随机生成最近一年内的日期
			days := random.RandInt(0, 365)
			return time.Now().AddDate(0, 0, -days).Format("2006-01-02")
		},
		"randomTime": func() string {
			hour := random.RandInt(0, 23)
			minute := random.RandInt(0, 59)
			second := random.RandInt(0, 59)
			return fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)
		},
		"randomDateTime": func() string {
			days := random.RandInt(0, 365)
			hour := random.RandInt(0, 23)
			minute := random.RandInt(0, 59)
			second := random.RandInt(0, 59)
			t := time.Now().AddDate(0, 0, -days)
			return time.Date(t.Year(), t.Month(), t.Day(), hour, minute, second, 0, time.Local).Format("2006-01-02 15:04:05")
		},
		"randomPrice": func(min, max int) string {
			price := random.RandInt(min*100, max*100)
			return fmt.Sprintf("%.2f", float64(price)/100)
		},
		"randomIDCard": func() string {
			// 简化的18位身份证号生成
			area := fmt.Sprintf("%06d", random.RandInt(110000, 659000))
			birth := time.Now().AddDate(-random.RandInt(18, 60), 0, -random.RandInt(0, 365)).Format("20060102")
			seq := fmt.Sprintf("%03d", random.RandInt(0, 999))
			return area + birth + seq + "X"
		},

		// 字符串函数 - 基础
		"upper": func(s string) string {
			return stringx.ToUpper(s)
		},
		"lower": func(s string) string {
			return stringx.ToLower(s)
		},
		"title": func(s string) string {
			return stringx.ToTitle(s)
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
		},
		"trimPrefix": func(s, prefix string) string {
			return strings.TrimPrefix(s, prefix)
		},
		"trimSuffix": func(s, suffix string) string {
			return strings.TrimSuffix(s, suffix)
		},
		"substr": func(s string, start, length int) string {
			return stringx.SubString(s, start, length)
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"join": func(arr []string, sep string) string {
			return strings.Join(arr, sep)
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},
		"repeat": func(s string, count int) string {
			return strings.Repeat(s, count)
		},
		"reverse": func(s string) string {
			return stringx.Reverse(s)
		},

		// 加密/哈希函数
		"md5": func(s string) string {
			h := md5.Sum([]byte(s))
			return hex.EncodeToString(h[:])
		},
		"sha1": func(s string) string {
			h := sha1.Sum([]byte(s))
			return hex.EncodeToString(h[:])
		},
		"sha256": func(s string) string {
			h := sha256.Sum256([]byte(s))
			return hex.EncodeToString(h[:])
		},

		// 编码/解码函数
		"base64": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		},
		"base64Decode": func(s string) string {
			b, _ := base64.StdEncoding.DecodeString(s)
			return string(b)
		},
		"urlEncode": func(s string) string {
			return url.QueryEscape(s)
		},
		"urlDecode": func(s string) string {
			decoded, _ := url.QueryUnescape(s)
			return decoded
		},
		"hexEncode": func(s string) string {
			return hex.EncodeToString([]byte(s))
		},
		"hexDecode": func(s string) string {
			b, _ := hex.DecodeString(s)
			return string(b)
		},

		// 数学函数 - 使用 mathx 模块
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"max": func(a, b int) int {
			return mathx.Max(a, b) // 使用 mathx
		},
		"min": func(a, b int) int {
			return mathx.Min(a, b) // 使用 mathx
		},
		"abs": func(n int) int {
			return mathx.Abs(n) // 使用 mathx
		},
		"pow": func(x, y float64) float64 {
			return math.Pow(x, y)
		},
		"sqrt": func(x float64) float64 {
			return math.Sqrt(x)
		},
		"ceil": func(x float64) float64 {
			return math.Ceil(x)
		},
		"floor": func(x float64) float64 {
			return math.Floor(x)
		},
		"round": func(x float64) float64 {
			return math.Round(x)
		},

		// 网络函数
		"localIP": func() string {
			ip, err := netx.GetPrivateIP()
			if err != nil {
				return "127.0.0.1"
			}
			return ip
		},
		"hostname": func() string {
			return osx.SafeGetHostName()
		},

		// 条件函数
		"ternary": func(condition bool, trueVal, falseVal any) any {
			if condition {
				return trueVal
			}
			return falseVal
		},
		"default": func(value, defaultValue any) any {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},

		// 类型转换 - 使用 convert 模块
		"toString": func(v any) string {
			return fmt.Sprintf("%v", v)
		},
		"toInt": func(s string) int {
			result, _ := convert.MustIntT[int](s, nil)
			return result
		},
		"toFloat": func(s string) float64 {
			result, _ := convert.MustIntT[float64](s, nil)
			return result
		},

		// 变量引用
		"var": func(key string) any {
			if val, ok := v.variables[key]; ok {
				return val
			}
			return ""
		},
	}

	return v
}

// SetVariables 设置变量
func (v *VariableResolver) SetVariables(vars map[string]any) {
	for k, val := range vars {
		v.variables[k] = val
	}
}

// SetVariable 设置单个变量
func (v *VariableResolver) SetVariable(key string, value any) {
	v.variables[key] = value
}

// VariableCount 返回当前变量数量
func (v *VariableResolver) VariableCount() int {
	return len(v.variables)
}

const (
	templateOpen  = "{{"
	templateClose = "}}"
)

// Resolve 变量解析方法
// 支持特性：
// 1. {{.varname}} 直接访问用户定义的变量
// 2. {{randomString 8}} 调用模板函数
// 3. {{.Env.PATH}}, {{.Time.Unix}} 访问特殊命名空间
func (v *VariableResolver) Resolve(input string) (string, error) {
	// 快速路径：如果不包含模板语法，直接返回（性能优化）
	if len(input) == 0 || !contains(input, templateOpen) || !contains(input, templateClose) {
		return input, nil
	}

	tmpl, err := template.New("resolver").Funcs(v.funcMap).Parse(input)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	// 核心改进：构建完整上下文，将用户变量展开到根级别
	ctx := v.buildContext()
	// 将所有用户变量展开到根级别，这样 {{.receiver_id}} 可以直接访问
	for k, val := range v.variables {
		// 避免覆盖系统保留字段
		if k != "Env" && k != "Variables" && k != "Seq" && k != "Time" {
			ctx[k] = val
		}
	}

	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("执行模板失败: %w", err)
	}

	return buf.String(), nil
}

// contains 快速字符串包含检查（比 strings.Contains 更快）
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// buildContext 构建模板上下文
func (v *VariableResolver) buildContext() map[string]any {
	return map[string]any{
		"Env":       envMap(),
		"Variables": v.variables,
		"Seq":       v.sequence.Add(1),
		"Time": map[string]any{
			"Unix":      time.Now().Unix(),
			"Timestamp": time.Now().UnixMilli(),
			"Now":       time.Now(),
		},
	}
}

// envMap 获取环境变量映射
func envMap() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		key, val := splitEnv(e)
		env[key] = val
	}
	return env
}

// splitEnv 分割环境变量字符串
func splitEnv(e string) (string, string) {
	for i := 0; i < len(e); i++ {
		if e[i] == '=' {
			return e[:i], e[i+1:]
		}
	}
	return e, ""
}

// ResolveToInt 解析为整数
func (v *VariableResolver) ResolveToInt(input string) (int, error) {
	resolved, err := v.Resolve(input)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(resolved)
}

// ResolveToBool 解析为布尔值
func (v *VariableResolver) ResolveToBool(input string) (bool, error) {
	resolved, err := v.Resolve(input)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(resolved)
}
