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

//go:embed  report.js
var reportJS string

//go:embed report_actions.js
var reportActionsJS string
