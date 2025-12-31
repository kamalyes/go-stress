/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-09 11:20:08
 * @FilePath: \go-stress\statistics\html_report.go
 * @Description: HTML报告生成器（类似JMeter报告）
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-toolbox/pkg/units"
)

// HTMLReportData HTML报告数据
type HTMLReportData struct {
	// 模式标识
	IsRealtime bool // true=实时模式, false=静态模式

	// 基础信息
	GenerateTime string
	TestDuration string

	// 统计数据
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
	SuccessRate     string

	// 性能指标
	QPS         string
	TotalSize   string
	AvgDuration string
	MinDuration string
	MaxDuration string

	// 百分位数据
	P50 string
	P90 string
	P95 string
	P99 string

	// 错误统计
	ErrorStats []ErrorStat

	// 状态码统计
	StatusCodeStats []StatusCodeStat

	// 请求明细（静态模式使用）
	RequestDetails []RequestDetailDisplay

	// 图表数据（JSON格式）
	DurationChartData string
	ErrorChartData    string
	StatusChartData   string

	// JSON文件路径（仅供参考）
	JSONFilename string
}

// RequestDetailDisplay 请求明细显示数据
type RequestDetailDisplay struct {
	ID              uint64
	Timestamp       string
	URL             string
	Method          string
	Query           string
	Headers         map[string]string
	Body            string
	Duration        string
	StatusCode      int
	Success         bool
	Size            string
	ResponseBody    string
	ResponseHeaders map[string]string
	Error           string
	Verifications   []VerificationResult
}

// ErrorStat 错误统计
type ErrorStat struct {
	Error      string
	Count      uint64
	Percentage string
}

// StatusCodeStat 状态码统计
type StatusCodeStat struct {
	StatusCode int
	Count      uint64
	Percentage string
}

// GenerateHTMLReport 生成HTML报告
func (c *Collector) GenerateHTMLReport(totalTime time.Duration, filename string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 准备报告数据
	data := &HTMLReportData{
		IsRealtime:      false, // 静态模式
		GenerateTime:    time.Now().Format(time.DateTime),
		TestDuration:    totalTime.String(),
		TotalRequests:   c.totalRequests,
		SuccessRequests: c.successRequests,
		FailedRequests:  c.failedRequests,
	}

	// 计算成功率
	if c.totalRequests > 0 {
		successRate := float64(c.successRequests) / float64(c.totalRequests) * 100
		data.SuccessRate = fmt.Sprintf("%.2f%%", successRate)

		// 计算QPS
		qps := float64(c.totalRequests) / totalTime.Seconds()
		data.QPS = fmt.Sprintf("%.2f", qps)

		// 平均耗时
		avgDuration := c.totalDuration / time.Duration(c.totalRequests)
		data.AvgDuration = avgDuration.String()
	}

	// 数据大小
	data.TotalSize = units.BytesSize(float64(c.totalSize))
	data.MinDuration = c.minDuration.String()
	data.MaxDuration = c.maxDuration.String()

	// 排序耗时数据
	sort.Slice(c.durations, func(i, j int) bool {
		return c.durations[i] < c.durations[j]
	})

	// 百分位数据
	if len(c.durations) > 0 {
		data.P50 = c.percentile(0.50).String()
		data.P90 = c.percentile(0.90).String()
		data.P95 = c.percentile(0.95).String()
		data.P99 = c.percentile(0.99).String()
	}

	// 错误统计
	data.ErrorStats = make([]ErrorStat, 0, len(c.errors))
	for err, count := range c.errors {
		percentage := float64(count) / float64(c.totalRequests) * 100
		data.ErrorStats = append(data.ErrorStats, ErrorStat{
			Error:      err,
			Count:      count,
			Percentage: fmt.Sprintf("%.2f%%", percentage),
		})
	}
	// 按错误次数排序
	sort.Slice(data.ErrorStats, func(i, j int) bool {
		return data.ErrorStats[i].Count > data.ErrorStats[j].Count
	})

	// 状态码统计
	data.StatusCodeStats = make([]StatusCodeStat, 0, len(c.statusCodes))
	for code, count := range c.statusCodes {
		percentage := float64(count) / float64(c.totalRequests) * 100
		data.StatusCodeStats = append(data.StatusCodeStats, StatusCodeStat{
			StatusCode: code,
			Count:      count,
			Percentage: fmt.Sprintf("%.2f%%", percentage),
		})
	}
	// 按状态码排序
	sort.Slice(data.StatusCodeStats, func(i, j int) bool {
		return data.StatusCodeStats[i].StatusCode < data.StatusCodeStats[j].StatusCode
	})

	// 准备图表数据 - 不再嵌入到HTML，改为让JS从JSON读取
	data.DurationChartData = "[]" // 占位
	data.ErrorChartData = "[]"
	data.StatusChartData = "[]"

	// 不再嵌入请求明细，改为从JSON加载
	data.RequestDetails = nil

	// 保存JSON文件路径信息（只保存文件名，不保存完整路径）
	jsonFilename := strings.TrimSuffix(filename, ".html") + ".json"
	// 提取文件名
	jsonBasename := jsonFilename
	if lastSlash := strings.LastIndexAny(jsonFilename, "/\\"); lastSlash != -1 {
		jsonBasename = jsonFilename[lastSlash+1:]
	}
	data.JSONFilename = jsonBasename

	// 复制 errors 和 statusCodes map（避免数据竞争）
	errorsCopy := make(map[string]uint64, len(c.errors))
	for k, v := range c.errors {
		errorsCopy[k] = v
	}

	statusCodesCopy := make(map[int]uint64, len(c.statusCodes))
	for k, v := range c.statusCodes {
		statusCodesCopy[k] = v
	}

	// 复制请求明细
	detailsCopy := make([]RequestDetail, len(c.requestDetails))
	copy(detailsCopy, c.requestDetails)

	// 生成完整的 Report 对象用于导出 JSON
	report := &Report{
		TotalRequests:   c.totalRequests,
		SuccessRequests: c.successRequests,
		FailedRequests:  c.failedRequests,
		TotalTime:       totalTime,
		TotalSize:       c.totalSize,
		MinDuration:     c.minDuration,
		MaxDuration:     c.maxDuration,
		Errors:          errorsCopy,
		StatusCodes:     statusCodesCopy,
		RequestDetails:  detailsCopy,
	}

	if c.totalRequests > 0 {
		report.SuccessRate = float64(c.successRequests) / float64(c.totalRequests) * 100
		report.AvgDuration = c.totalDuration / time.Duration(c.totalRequests)
		report.QPS = float64(c.totalRequests) / totalTime.Seconds()
	}

	// 计算百分位
	if len(c.durations) > 0 {
		report.P50 = c.percentile(0.50)
		report.P90 = c.percentile(0.90)
		report.P95 = c.percentile(0.95)
		report.P99 = c.percentile(0.99)
	}

	// 保存 JSON 数据文件
	if err := report.SaveToFile(jsonFilename); err != nil {
		return fmt.Errorf("保存JSON数据失败: %w", err)
	}
	logger.Default.Debug("已生成JSON数据文件: %s", jsonFilename)

	// 获取报告目录
	reportDir := filename[:strings.LastIndexAny(filename, "/\\")]
	if reportDir == "" {
		reportDir = "."
	}

	// 生成 CSS 文件
	cssFile := reportDir + "/report.css"
	if err := os.WriteFile(cssFile, []byte(reportCSS), 0644); err != nil {
		return fmt.Errorf("生成CSS文件失败: %w", err)
	}
	logger.Default.Debug("已生成CSS文件: %s", cssFile)

	// 生成 JS 文件（替换占位符）
	jsContent := strings.ReplaceAll(reportJS, "IS_REALTIME_PLACEHOLDER", "false")
	jsContent = strings.ReplaceAll(jsContent, "JSON_FILENAME_PLACEHOLDER", data.JSONFilename)
	jsFile := reportDir + "/report.js"
	if err := os.WriteFile(jsFile, []byte(jsContent), 0644); err != nil {
		return fmt.Errorf("生成JS文件失败: %w", err)
	}
	logger.Default.Debug("已生成JS文件: %s", jsFile)

	// 使用简化的HTML模板
	tmpl, err := template.New("report").Parse(reportHTML)
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("生成报告失败: %w", err)
	}

	logger.Default.Info("✅ HTML报告已生成: %s", filename)
	logger.Default.Info("📄 CSS样式文件: %s", cssFile)
	logger.Default.Info("📜 JS脚本文件: %s", jsFile)
	logger.Default.Info("📊 JSON数据文件: %s", jsonFilename)

	return nil
}
