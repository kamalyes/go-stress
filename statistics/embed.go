/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-09 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-09 11:15:00
 * @FilePath: \go-stress\statistics\report_js.go
 * @Description: 报告JavaScript脚本模板 - 使用embed嵌入JS文件
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package statistics

import _ "embed"

// FaviconSVG 通用 Favicon SVG 图标
const FaviconSVG = `data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Cdefs%3E%3ClinearGradient id='grad' x1='0%25' y1='0%25' x2='100%25' y2='100%25'%3E%3Cstop offset='0%25' style='stop-color:%23667eea;stop-opacity:1' /%3E%3Cstop offset='100%25' style='stop-color:%23764ba2;stop-opacity:1' /%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='100' height='100' rx='20' fill='url(%23grad)'/%3E%3Cpath d='M30 55 L45 70 L70 30' stroke='white' stroke-width='8' stroke-linecap='round' stroke-linejoin='round' fill='none'/%3E%3Cpath d='M25 30 L35 30 M65 70 L75 70 M30 25 L30 35 M70 65 L70 75' stroke='%2338ef7d' stroke-width='4' stroke-linecap='round'/%3E%3C/svg%3E`

//go:embed  report.js
var reportJS string

//go:embed report_actions.js
var reportActionsJS string

//go:embed http-client.js
var httpClientJS string

//go:embed distributed.html
var distributedHTML string

//go:embed distributed.css
var distributedCSS string

//go:embed distributed.js
var distributedJS string

//go:embed task_detail.html
var taskDetailHTML string

//go:embed task_detail.js
var taskDetailJS string

//go:embed task_detail.css
var taskDetailCSS string

//go:embed slave_detail.html
var slaveDetailHTML string

//go:embed slave_detail.js
var slaveDetailJS string

// GetRealtimeHTML 返回实时报告 HTML（用于分布式模式）
func GetRealtimeHTML(report *Report) (string, error) {
	formatter := &HTMLFormatter{
		IsRealtime:   true,
		JSONFilename: "", // 实时模式不需要 JSON 文件
	}
	htmlBytes, err := formatter.Format(report)
	if err != nil {
		return "", err
	}

	html := string(htmlBytes)

	// 注入 Favicon
	html = injectFavicon(html)

	return html, nil
}

// GetDistributedHTML 返回分布式管理页面完整 HTML（包含内嵌 CSS 和 JS）
func GetDistributedHTML() string {
	// 将 CSS 和 JS 注入到 HTML 中
	html := distributedHTML

	// 替换 CSS 引用为内嵌样式
	html = injectStyle(html, distributedCSS)

	// 替换 JS 引用为内嵌脚本
	html = injectScript(html, distributedJS)

	// 注入 Favicon
	html = injectFavicon(html)

	return html
}

// GetTaskDetailHTML 返回任务详情页面完整 HTML
func GetTaskDetailHTML() string {
	html := taskDetailHTML
	html = injectStyle(html, distributedCSS)                         // 复用 distributed.css
	html = injectStyleByFile(html, "task_detail.css", taskDetailCSS) // 注入 task_detail.css
	html = injectScriptByFile(html, "task_detail.js", taskDetailJS)
	html = injectFavicon(html)
	return html
}

// GetSlaveDetailHTML 返回 Slave 详情页面完整 HTML
func GetSlaveDetailHTML() string {
	html := slaveDetailHTML
	html = injectStyle(html, distributedCSS) // 复用 distributed.css
	html = injectScriptByFile(html, "slave_detail.js", slaveDetailJS)
	html = injectFavicon(html)
	return html
}

// ===== 静态资源访问函数 =====

// GetReportCSS 返回实时报告的 CSS 内容
func GetReportCSS() string {
	return reportCSS
}

// GetReportJS 返回实时报告的 JS 内容
func GetReportJS() string {
	return reportJS
}

// GetReportActionsJS 返回实时报告的操作 JS 内容
func GetReportActionsJS() string {
	return reportActionsJS
}

// GetHTTPClientJS 返回 HTTP 客户端 JS 内容
func GetHTTPClientJS() string {
	return httpClientJS
}

// GetTaskDetailCSS 返回任务详情页面的 CSS 内容
func GetTaskDetailCSS() string {
	return taskDetailCSS
}

// GetDistributedCSS 返回分布式管理页面的 CSS 内容
func GetDistributedCSS() string {
	return distributedCSS
}

// GetDistributedJS 返回分布式管理页面的 JS 内容
func GetDistributedJS() string {
	return distributedJS
}

// GetTaskDetailJS 返回任务详情页面的 JS 内容
func GetTaskDetailJS() string {
	return taskDetailJS
}

// GetSlaveDetailJS 返回 Slave 详情页面的 JS 内容
func GetSlaveDetailJS() string {
	return slaveDetailJS
}

// ===== 内嵌辅助函数 =====

// injectStyle 将外部 CSS 替换为内嵌样式
func injectStyle(html, css string) string {
	return injectStyleByFile(html, "distributed.css", css)
}

// injectStyleByFile 将指定的外部 CSS 文件替换为内嵌样式
func injectStyleByFile(html, filename, css string) string {
	oldLink := `<link rel="stylesheet" href="` + filename + `">`
	newStyle := "<style>\n" + css + "\n</style>"
	return replaceFirst(html, oldLink, newStyle)
}

// injectScript 将外部 JS 替换为内嵌脚本
func injectScript(html, js string) string {
	return injectScriptByFile(html, "distributed.js", js)
}

// injectScriptByFile 将指定的外部 JS 文件替换为内嵌脚本
func injectScriptByFile(html, filename, js string) string {
	oldScript := `<script src="` + filename + `"></script>`
	newScript := "<script>\n" + js + "\n</script>"
	return replaceFirst(html, oldScript, newScript)
}

// injectFavicon 注入 Favicon 到 HTML <head> 中
func injectFavicon(html string) string {
	faviconTag := `<link rel="icon" type="image/svg+xml" href="` + FaviconSVG + `">`
	// 在 </head> 前插入 favicon
	return replaceFirst(html, "</head>", "    "+faviconTag+"\n</head>")
}

// replaceFirst 替换第一个匹配的字符串
func replaceFirst(s, old, new string) string {
	i := 0
	for {
		j := i
		for ; j < len(s); j++ {
			if j+len(old) > len(s) {
				return s
			}
			if s[j:j+len(old)] == old {
				return s[:j] + new + s[j+len(old):]
			}
		}
		if j >= len(s) {
			return s
		}
		i = j + 1
	}
}
