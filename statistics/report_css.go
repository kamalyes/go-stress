/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-09 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-09 11:15:00
 * @FilePath: \go-stress\statistics\report_css.go
 * @Description: 报告样式表模板
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package statistics

const reportCSS = `* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: 20px;
    color: #333;
}

.container {
    max-width: 1600px;
    margin: 0 auto;
    background: white;
    border-radius: 12px;
    box-shadow: 0 10px 40px rgba(0,0,0,0.1);
    overflow: hidden;
}

.header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 30px 40px;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.header h1 {
    font-size: 2em;
    text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
}

.status-badge {
    background: rgba(255,255,255,0.2);
    padding: 10px 20px;
    border-radius: 20px;
    font-size: 1.1em;
    display: flex;
    align-items: center;
    gap: 10px;
}

.status-dot {
    width: 12px;
    height: 12px;
    background: #38ef7d;
    border-radius: 50%;
    animation: pulse 2s infinite;
}

@keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
}

.info-bar {
    background: #f8f9fa;
    padding: 20px 40px;
    display: flex;
    justify-content: space-between;
    border-bottom: 2px solid #e9ecef;
    flex-wrap: wrap;
    gap: 20px;
}

.info-item {
    text-align: center;
    min-width: 150px;
}

.info-label {
    color: #6c757d;
    font-size: 0.9em;
    margin-bottom: 5px;
}

.info-value {
    font-size: 1.2em;
    font-weight: bold;
    color: #495057;
}

.config-info {
    background: linear-gradient(135deg, #f0f4ff 0%, #e6f0ff 100%);
    padding: 20px 25px;
    margin: 20px 0;
    border-radius: 12px;
    border-left: 4px solid #667eea;
}

.config-info h3 {
    margin: 0 0 15px 0;
    color: #495057;
    font-size: 1.1em;
}

.config-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 15px;
}

.config-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 15px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.05);
}

.config-label {
    font-weight: 600;
    color: #6c757d;
    min-width: 90px;
}

.config-value {
    color: #495057;
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    word-break: break-all;
}

.protocol-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 12px;
    border-radius: 20px;
    font-size: 0.9em;
    font-weight: 600;
}

.protocol-http { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; }
.protocol-https { background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); color: white; }
.protocol-grpc { background: linear-gradient(135deg, #ff6b6b 0%, #feca57 100%); color: white; }
.protocol-websocket { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); color: white; }
.protocol-wss { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); color: white; }

.metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: 15px;
    padding: 25px;
    background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
}

.metric-card {
    background: white;
    padding: 16px 12px;
    border-radius: 8px;
    box-shadow: 0 2px 6px rgba(0,0,0,0.08);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    border-left: 3px solid #667eea;
}

.metric-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(102, 126, 234, 0.2);
    border-left-color: #764ba2;
}

.metric-label {
    font-size: 0.75em;
    color: #6c757d;
    margin-bottom: 6px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 600;
}

.metric-value {
    font-size: 1.6em;
    font-weight: 700;
    color: #667eea;
    line-height: 1.2;
}

.metric-value.success {
    color: #38ef7d;
}

.metric-value.error {
    color: #f45c43;
}

.content {
    padding: 30px;
}

.section {
    margin-bottom: 30px;
}

.section-title {
    font-size: 1.5em;
    color: #495057;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 2px solid #667eea;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.chart-container {
    background: white;
    padding: 20px;
    border-radius: 10px;
    box-shadow: 0 2px 10px rgba(0,0,0,0.05);
    margin-bottom: 20px;
}

.chart {
    width: 100%;
    height: 300px;
}

table {
    width: 100%;
    border-collapse: collapse;
    background: white;
    border-radius: 10px;
    overflow: hidden;
    box-shadow: 0 2px 10px rgba(0,0,0,0.05);
}

thead {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
}

th, td {
    padding: 12px;
    text-align: left;
    font-size: 0.9em;
}

th {
    font-weight: 600;
    text-transform: uppercase;
    font-size: 0.8em;
    letter-spacing: 0.5px;
}

tbody tr {
    border-bottom: 1px solid #e9ecef;
    transition: background 0.2s;
}

tbody tr:hover {
    background: #f8f9fa;
}

tbody tr:last-child {
    border-bottom: none;
}

.status-success {
    color: #38ef7d;
    font-weight: bold;
}

.status-error {
    color: #f45c43;
    font-weight: bold;
}

.status-warning {
    color: #ffa502;
    font-weight: bold;
}

/* HTTP 方法标签样式 (Swagger风格) */
.http-method {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 700;
    text-align: center;
    min-width: 60px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.http-method-get {
    background-color: #61affe;
    color: white;
    border: 1px solid #4a90e2;
}

.http-method-post {
    background-color: #49cc90;
    color: white;
    border: 1px solid #3cb371;
}

.http-method-put {
    background-color: #fca130;
    color: white;
    border: 1px solid #e89020;
}

.http-method-delete {
    background-color: #f93e3e;
    color: white;
    border: 1px solid #e02020;
}

.http-method-patch {
    background-color: #50e3c2;
    color: white;
    border: 1px solid #40c9aa;
}

.http-method-head {
    background-color: #9012fe;
    color: white;
    border: 1px solid #7a0ad4;
}

.http-method-options {
    background-color: #0d5aa7;
    color: white;
    border: 1px solid #094a8a;
}

.http-method-default {
    background-color: #6c757d;
    color: white;
    border: 1px solid #5a6268;
}

/* 操作按钮和下拉菜单样式 */
.action-dropdown {
    position: relative;
    display: inline-block;
}

.action-dropdown-btn {
    padding: 8px 16px;
    background: #667eea;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.3s;
    min-width: 120px;
    justify-content: center;
    height: 36px;
}

.action-dropdown-btn:hover {
    background: #5568d3;
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.action-dropdown-menu {
    display: none;
    position: fixed;
    background: white;
    border-radius: 8px;
    box-shadow: 0 8px 32px rgba(0,0,0,0.2);
    min-width: 220px;
    max-width: 300px;
    z-index: 10000;
    border: 1px solid #e9ecef;
    max-height: 80vh;
    overflow-y: auto;
}

.action-dropdown-menu::-webkit-scrollbar {
    width: 6px;
}

.action-dropdown-menu::-webkit-scrollbar-track {
    background: #f1f1f1;
    border-radius: 8px;
}

.action-dropdown-menu::-webkit-scrollbar-thumb {
    background: #c1c1c1;
    border-radius: 8px;
}

.action-dropdown-menu::-webkit-scrollbar-thumb:hover {
    background: #a8a8a8;
}

.action-dropdown-menu-item {
    padding: 12px 16px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    gap: 10px;
    color: #495057;
    font-size: 14px;
    border-bottom: 1px solid #f8f9fa;
}

.action-dropdown-menu-item:last-child {
    border-bottom: none;
}

.action-dropdown-menu-item:hover {
    background: #f8f9fa;
    color: #667eea;
}

.action-dropdown-menu-item span {
    font-size: 16px;
}

.action-menu-section {
    padding: 8px 16px;
    font-size: 12px;
    color: #6c757d;
    font-weight: 600;
    background: #f8f9fa;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.progress-bar {
    width: 100%;
    height: 8px;
    background: #e9ecef;
    border-radius: 4px;
    overflow: hidden;
    margin-top: 5px;
}

.progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
    transition: width 0.3s ease;
}

.error-message {
    word-break: break-all;
    max-width: 400px;
    font-size: 0.85em;
}

.tab-btn {
    padding: 12px 24px;
    background: #ffffff;
    border: none;
    border-bottom: 3px solid transparent;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    color: #6c757d;
    transition: all 0.3s;
    position: relative;
}

.tab-btn:hover {
    color: #667eea;
    background: #f0f0f0;
}

.tab-btn.active {
    color: #667eea;
    border-bottom-color: #667eea;
    font-weight: 600;
    background: #ffffff;
}

.detail-row {
    display: none;
    background: #f8f9fa;
}

.detail-row.show {
    display: table-row;
}

.detail-btn {
    background: #667eea;
    color: white;
    border: none;
    padding: 5px 12px;
    border-radius: 5px;
    cursor: pointer;
    font-size: 0.85em;
    transition: background 0.2s;
}

.detail-btn:hover {
    background: #5568d3;
}

.detail-content {
    padding: 15px;
    max-height: 500px;
    overflow-y: auto;
    overflow-x: hidden;
    word-wrap: break-word;
    word-break: break-word;
}

.detail-section {
    margin-bottom: 15px;
}

.detail-section pre {
    white-space: pre-wrap;
    word-wrap: break-word;
    word-break: break-word;
    overflow-wrap: break-word;
    max-width: 100%;
}

.detail-section-title {
    font-weight: bold;
    color: #495057;
    margin-bottom: 8px;
    font-size: 0.9em;
}

.detail-tabs-container {
    margin-top: 10px;
}

.detail-tabs-header {
    display: flex;
    gap: 5px;
    border-bottom: 2px solid #e9ecef;
    margin-bottom: 15px;
}

.detail-tab-btn {
    padding: 10px 20px;
    background: transparent;
    border: none;
    border-bottom: 3px solid transparent;
    cursor: pointer;
    font-size: 14px;
    color: #6c757d;
    transition: all 0.2s;
    font-weight: 500;
}

.detail-tab-btn:hover {
    color: #667eea;
    background: #f8f9fa;
}

.detail-tab-btn.active {
    color: #667eea;
    border-bottom-color: #667eea;
    background: #f8f9fa;
}

.detail-tabs-content {
    position: relative;
}

.detail-tab-content {
    display: none;
}

.detail-tab-content.active {
    display: block;
}

.detail-table {
    width: 100%;
    background: white;
    border-radius: 5px;
    overflow: hidden;
    font-size: 0.85em;
}

.detail-table td {
    padding: 6px 10px;
    border-bottom: 1px solid #e9ecef;
}

.detail-table td:first-child {
    font-weight: bold;
    color: #6c757d;
    width: 120px;
}

.detail-code {
    background: white;
    padding: 10px;
    border-radius: 5px;
    overflow-x: auto;
    font-family: 'Courier New', monospace;
    font-size: 0.85em;
    max-height: 200px;
    overflow-y: auto;
    white-space: pre-wrap;
    word-break: break-all;
}

.footer {
    background: #f8f9fa;
    padding: 20px;
    text-align: center;
    color: #6c757d;
    border-top: 2px solid #e9ecef;
}

.file-loader {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: 40px;
    text-align: center;
    border-radius: 10px;
    margin: 20px 0;
    color: white;
}

.file-loader h3 {
    margin: 0 0 20px 0;
    font-size: 1.5em;
}

.file-loader p {
    margin: 0 0 20px 0;
    opacity: 0.9;
}

.file-input-wrapper {
    display: inline-block;
    position: relative;
    overflow: hidden;
    background: white;
    color: #667eea;
    padding: 12px 30px;
    border-radius: 5px;
    cursor: pointer;
    font-weight: bold;
    transition: all 0.3s ease;
}

.file-input-wrapper:hover {
    transform: translateY(-2px);
    box-shadow: 0 5px 15px rgba(0,0,0,0.3);
}

.file-input-wrapper input[type="file"] {
    position: absolute;
    left: -9999px;
}

.file-name {
    margin-top: 15px;
    font-size: 0.9em;
    opacity: 0.8;
}

.pagination {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 10px;
    margin: 20px 0;
    padding: 15px;
    background: #f8f9fa;
    border-radius: 8px;
}

.pagination button {
    padding: 8px 15px;
    border: 1px solid #dee2e6;
    background: white;
    border-radius: 5px;
    cursor: pointer;
    transition: all 0.3s ease;
    font-size: 0.9em;
}

.pagination button:hover:not(:disabled) {
    background: #667eea;
    color: white;
    border-color: #667eea;
}

.pagination button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.pagination select {
    padding: 8px 12px;
    border: 1px solid #dee2e6;
    border-radius: 5px;
    background: white;
    cursor: pointer;
}

.pagination-info {
    color: #6c757d;
    font-size: 0.9em;
}

@media (max-width: 768px) {
    .metrics-grid {
        grid-template-columns: 1fr;
    }
    
    .info-bar {
        flex-direction: column;
    }
}
`
