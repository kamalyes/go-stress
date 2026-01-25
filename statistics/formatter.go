/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-24 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 10:00:00
 * @FilePath: \go-stress\statistics\formatter.go
 * @Description: 报告格式化器 - 统一的格式化接口层
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/kamalyes/go-toolbox/pkg/mathx"
	"github.com/kamalyes/go-toolbox/pkg/units"
)

// ReportFormatter 报告格式化器接口
type ReportFormatter interface {
	// Format 格式化报告为特定格式的字节数据
	Format(report *Report) ([]byte, error)

	// ContentType 返回内容类型（如 "text/html", "application/json"）
	ContentType() string
}

// ===== JSON 格式化器 =====

// JSONFormatter JSON格式化器
type JSONFormatter struct {
	Indent bool // 是否格式化输出
}

// Format 格式化为JSON
func (f *JSONFormatter) Format(report *Report) ([]byte, error) {
	if f.Indent {
		return json.MarshalIndent(report, "", "  ")
	}
	return json.Marshal(report)
}

// ContentType 返回JSON内容类型
func (f *JSONFormatter) ContentType() string {
	return "application/json"
}

// ===== HTML 格式化器 =====

// HTMLFormatter HTML格式化器
type HTMLFormatter struct {
	IsRealtime   bool   // 是否实时模式
	JSONFilename string // JSON数据文件名（用于实时报告加载数据）
}

// TemplateData 模板渲染数据 - 格式化后的展示数据
type TemplateData struct {
	IsRealtime   bool
	GenerateTime string
	JSONFilename string  // JSON数据文件名(用于实时报告加载数据)
	Report       *Report // 直接传递 Report,在模板中格式化

	// 格式化的辅助方法(在模板中调用)
	FormatDuration  func(time.Duration) string
	FormatPercent   func(float64) string
	FormatSize      func(float64) string
	FormatErrorMap  func(map[string]uint64) []ErrorStat
	FormatStatusMap func(map[int]uint64) []StatusCodeStat
}

// ErrorStat 错误统计项（用于模板展示）
type ErrorStat struct {
	Error      string
	Count      uint64
	Percentage string
}

// StatusCodeStat 状态码统计项（用于模板展示）
type StatusCodeStat struct {
	StatusCode int
	Count      uint64
	Percentage string
}

// Format 格式化为HTML - 直接传Report给模板，避免中间层转换
func (f *HTMLFormatter) Format(report *Report) ([]byte, error) {
	// 创建模板数据 - 包含Report和格式化函数
	data := &TemplateData{
		IsRealtime:   f.IsRealtime,
		GenerateTime: time.Now().Format(time.DateTime),
		JSONFilename: f.JSONFilename,
		Report:       report,
		// 格式化函数（模板中可以调用）
		FormatDuration: func(d time.Duration) string { return d.String() },
		FormatPercent:  func(v float64) string { return fmt.Sprintf("%.2f%%", v) },
		FormatSize:     func(v float64) string { return units.BytesSize(v) },
		FormatErrorMap: func(errors map[string]uint64) []ErrorStat {
			return f.convertErrors(errors, report.TotalRequests)
		},
		FormatStatusMap: func(codes map[int]uint64) []StatusCodeStat {
			return f.convertStatusCodes(codes, report.TotalRequests)
		},
	}

	// 解析并渲染模板
	tmpl, err := template.New("report").Parse(reportHTML)
	if err != nil {
		return nil, fmt.Errorf("parse template failed: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template failed: %w", err)
	}

	return buf.Bytes(), nil
}

// ContentType 返回HTML内容类型
func (f *HTMLFormatter) ContentType() string {
	return "text/html; charset=utf-8"
}

// convertErrors 转换错误统计为展示格式
func (f *HTMLFormatter) convertErrors(errors map[string]uint64, total uint64) []ErrorStat {
	result := make([]ErrorStat, 0, len(errors))
	for err, count := range errors {
		percentage := mathx.Percentage(count, total)
		result = append(result, ErrorStat{
			Error:      err,
			Count:      count,
			Percentage: fmt.Sprintf("%.2f%%", percentage),
		})
	}
	// 按数量降序排序
	mathx.SortByCount(result, func(e ErrorStat) uint64 { return e.Count })
	return result
}

// convertStatusCodes 转换状态码统计为展示格式
func (f *HTMLFormatter) convertStatusCodes(codes map[int]uint64, total uint64) []StatusCodeStat {
	result := make([]StatusCodeStat, 0, len(codes))
	for code, count := range codes {
		percentage := mathx.Percentage(count, total)
		result = append(result, StatusCodeStat{
			StatusCode: code,
			Count:      count,
			Percentage: fmt.Sprintf("%.2f%%", percentage),
		})
	}
	// 按状态码升序排序
	mathx.SortByKey(result, func(s StatusCodeStat) int { return s.StatusCode })
	return result
}

// ===== 文本格式化器 =====

// TextFormatter 纯文本格式化器（控制台输出）
type TextFormatter struct{}

// Format 格式化为纯文本
func (f *TextFormatter) Format(report *Report) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("\n📊 压测统计报告\n\n")
	buf.WriteString(fmt.Sprintf("总请求数: %d\n", report.TotalRequests))
	buf.WriteString(fmt.Sprintf("成功请求: %d\n", report.SuccessRequests))
	buf.WriteString(fmt.Sprintf("失败请求: %d\n", report.FailedRequests))
	buf.WriteString(fmt.Sprintf("成功率: %.2f%%\n", report.SuccessRate))
	buf.WriteString(fmt.Sprintf("QPS: %.2f\n", report.QPS))
	buf.WriteString(fmt.Sprintf("总耗时: %s\n", report.TotalTime))
	buf.WriteString(fmt.Sprintf("最小耗时: %s\n", report.MinLatency))
	buf.WriteString(fmt.Sprintf("最大耗时: %s\n", report.MaxLatency))
	buf.WriteString(fmt.Sprintf("平均耗时: %s\n", report.AvgLatency))
	buf.WriteString(fmt.Sprintf("P50: %s\n", report.P50Latency))
	buf.WriteString(fmt.Sprintf("P90: %s\n", report.P90Latency))
	buf.WriteString(fmt.Sprintf("P95: %s\n", report.P95Latency))
	buf.WriteString(fmt.Sprintf("P99: %s\n", report.P99Latency))
	buf.WriteString(fmt.Sprintf("总数据量: %s\n", units.BytesSize(report.TotalSize)))

	// 错误统计
	if len(report.Errors) > 0 {
		buf.WriteString("\n错误统计:\n")
		for err, count := range report.Errors {
			percentage := mathx.Percentage(count, report.TotalRequests)
			buf.WriteString(fmt.Sprintf("  %s: %d (%.2f%%)\n", err, count, percentage))
		}
	}

	// 状态码统计
	if len(report.StatusCodes) > 0 {
		buf.WriteString("\n状态码统计:\n")
		for code, count := range report.StatusCodes {
			percentage := mathx.Percentage(count, report.TotalRequests)
			buf.WriteString(fmt.Sprintf("  %d: %d (%.2f%%)\n", code, count, percentage))
		}
	}

	return buf.Bytes(), nil
}

// ContentType 返回文本内容类型
func (f *TextFormatter) ContentType() string {
	return "text/plain; charset=utf-8"
}
