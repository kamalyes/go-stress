/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 10:00:00
 * @FilePath: \go-stress\statistics\html_report.go
 * @Description: HTML报告生成器（重构后：使用 Builder + Formatter 架构）
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kamalyes/go-stress/logger"
)

// GenerateHTMLReport 生成HTML报告 - 使用新架构（Builder + Formatter）
func (c *Collector) GenerateHTMLReport(totalTime time.Duration, filename string) error {
	return c.GenerateHTMLReportWithLimit(totalTime, filename, -1) // -1 表示导出全部详情
}

// GenerateHTMLReportWithLimit 生成HTML报告（指定详情导出数量）
func (c *Collector) GenerateHTMLReportWithLimit(totalTime time.Duration, filename string, detailsLimit int) error {
	// 第一步：使用 ReportBuilder 构建完整报告（包含明细）
	builder := NewReportBuilder(c)
	report := builder.BuildFullReportWithLimit(totalTime, detailsLimit)

	// 第二步：生成报告目录和文件名
	reportDir := filepath.Dir(filename)
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	jsonFilename := baseName + ".json"
	jsonFullPath := filepath.Join(reportDir, jsonFilename)

	// 第三步：生成并保存 JSON 数据文件
	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON failed: %w", err)
	}
	if err := os.WriteFile(jsonFullPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("write JSON file failed: %w", err)
	}
	logger.Default.Info("✅ JSON数据已生成: %s", jsonFullPath)

	// 第四步：生成静态资源文件（CSS、JS）
	if err := generateStaticFiles(reportDir, jsonFilename); err != nil {
		return fmt.Errorf("generate static files failed: %w", err)
	}

	// 第五步：使用 HTMLFormatter 格式化为 HTML
	formatter := &HTMLFormatter{
		IsRealtime:   false,
		JSONFilename: jsonFilename,
	}
	htmlBytes, err := formatter.Format(report)
	if err != nil {
		return fmt.Errorf("format HTML failed: %w", err)
	}

	// 第六步：写入 HTML 文件
	if err := os.WriteFile(filename, htmlBytes, 0644); err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	logger.Default.Info("✅ HTML报告已生成: %s", filename)
	return nil
}

// generateStaticFiles 生成静态资源文件（CSS、JS）
func generateStaticFiles(reportDir, jsonFilename string) error {
	// 生成 CSS 文件
	cssPath := filepath.Join(reportDir, "report.css")
	if err := os.WriteFile(cssPath, []byte(reportCSS), 0644); err != nil {
		return fmt.Errorf("write CSS file failed: %w", err)
	}
	logger.Default.Info("✅ CSS文件已生成: %s", cssPath)

	// 生成 JS 文件（静态模式：IS_REALTIME = false，替换JSON文件名占位符）
	jsContent := strings.ReplaceAll(reportJS, "IS_REALTIME_PLACEHOLDER", "false")
	jsContent = strings.ReplaceAll(jsContent, "JSON_FILENAME_PLACEHOLDER", jsonFilename)
	jsPath := filepath.Join(reportDir, "report.js")
	if err := os.WriteFile(jsPath, []byte(jsContent), 0644); err != nil {
		return fmt.Errorf("write JS file failed: %w", err)
	}
	logger.Default.Info("✅ JS文件已生成: %s", jsPath)

	return nil
}
