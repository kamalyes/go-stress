/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-31 16:28:17
 * @FilePath: \go-stress\statistics\unified_template.go
 * @Description: 统一HTML模板（支持静态和实时模式）
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package statistics

const unifiedTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go-Stress {{if .IsRealtime}}实时{{end}}性能测试报告</title>
    <script src="https://cdn.jsdelivr.net/npm/echarts@5.4.3/dist/echarts.min.js"></script>
    <style>
        * {
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
        
        {{if .IsRealtime}}
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
        {{end}}
        
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
        
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 30px;
            background: #f8f9fa;
        }
        
        .metric-card {
            background: white;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }
        
        .metric-card:hover {
            transform: translateY(-3px);
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        }
        
        .metric-label {
            font-size: 0.85em;
            color: #6c757d;
            margin-bottom: 8px;
        }
        
        .metric-value {
            font-size: 1.8em;
            font-weight: bold;
            color: #667eea;
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
        
        .status-failure {
            color: #f45c43;
            font-weight: bold;
        }
        
        .status-warning {
            color: #ffa502;
            font-weight: bold;
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
        
        .detail-row {
            display: none;
            background: #f8f9fa;
        }
        
        .detail-row.show {
            display: table-row;
        }
        
        .detail-content {
            padding: 15px;
        }
        
        .detail-section {
            margin-bottom: 15px;
        }
        
        .detail-section-title {
            font-weight: bold;
            color: #495057;
            margin-bottom: 8px;
            font-size: 0.9em;
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
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚡ Go-Stress {{if .IsRealtime}}实时{{end}}性能测试报告</h1>
            {{if .IsRealtime}}
            <div class="status-badge">
                <div class="status-dot"></div>
                <span>实时监控中</span>
            </div>
            {{end}}
        </div>
        
        {{if not .IsRealtime}}
        <!-- 文件加载器 -->
        <div class="file-loader" id="fileLoader">
            <h3>📂 加载测试报告数据</h3>
            <p>请选择对应的 JSON 数据文件</p>
            <p style="font-size: 0.9em; opacity: 0.8; margin-top: -10px;">💡 提示: 请选择同目录下的 <strong>{{.JSONFilename}}</strong></p>
            <label class="file-input-wrapper">
                <input type="file" id="jsonFileInput" accept=".json" onchange="handleFileSelect(event)">
                选择 JSON 文件
            </label>
            <div class="file-name" id="fileName"></div>
        </div>
        
        <div class="info-bar" id="infoBar" style="display: none;">
            <div class="info-item">
                <div class="info-label">生成时间</div>
                <div class="info-value" id="generate-time">-</div>
            </div>
            <div class="info-item">
                <div class="info-label">测试时长</div>
                <div class="info-value" id="test-duration">-</div>
            </div>
            <div class="info-item">
                <div class="info-label">总请求数</div>
                <div class="info-value" id="static-total-requests">-</div>
            </div>
            <div class="info-item">
                <div class="info-label">成功率</div>
                <div class="info-value" id="static-success-rate">-</div>
            </div>
            <div class="info-item">
                <div class="info-label">QPS</div>
                <div class="info-value" id="static-qps">-</div>
            </div>
        </div>
        {{end}}
        
        {{if .IsRealtime}}
        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">总请求数</div>
                <div class="metric-value" id="total-requests">0</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">成功请求</div>
                <div class="metric-value success" id="success-requests">0</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">失败请求</div>
                <div class="metric-value error" id="failed-requests">0</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">成功率</div>
                <div class="metric-value" id="success-rate">0%</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">QPS</div>
                <div class="metric-value" id="qps">0</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">平均响应时间</div>
                <div class="metric-value" id="avg-duration">0ms</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">运行时间</div>
                <div class="metric-value" id="elapsed">0s</div>
            </div>
        </div>
        {{end}}
        
        <div class="content">
            <div class="section">
                <div class="section-title">📈 实时图表</div>
                <div class="chart-container">
                    <div id="durationChart" class="chart"></div>
                </div>
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
                    <div class="chart-container">
                        <div id="statusChart" class="chart"></div>
                    </div>
                    <div class="chart-container">
                        <div id="errorChart" class="chart"></div>
                    </div>
                </div>
            </div>
            
            <div class="section">
                <div class="section-title">
                    <span>📋 请求明细</span>
                </div>
                
                <!-- 高级筛选栏 -->
                <div style="padding: 20px; background: #f8f9fa; border-radius: 8px; margin-bottom: 30px; position: relative;">
                    <div style="display: grid; grid-template-columns: 2fr 1fr 1fr 1fr auto; gap: 15px; align-items: center;">
                        <input type="text" id="searchPath" placeholder="搜索 URL 路径..." style="padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;" onkeyup="filterDetails()">
                        
                        <select id="methodFilter" onchange="filterDetails()" style="padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;">
                            <option value="">所有方法</option>
                            <option value="GET">GET</option>
                            <option value="POST">POST</option>
                            <option value="PUT">PUT</option>
                            <option value="DELETE">DELETE</option>
                            <option value="PATCH">PATCH</option>
                        </select>
                        
                        <select id="statusFilter" onchange="filterDetails()" style="padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;">
                            <option value="">所有状态码</option>
                            <option value="2xx">2xx 成功</option>
                            <option value="3xx">3xx 重定向</option>
                            <option value="4xx">4xx 客户端错误</option>
                            <option value="5xx">5xx 服务端错误</option>
                        </select>
                        
                        <select id="durationFilter" onchange="filterDetails()" style="padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;">
                            <option value="">所有响应时间</option>
                            <option value="<100">&lt; 100ms</option>
                            <option value="100-500">100-500ms</option>
                            <option value="500-1000">500-1000ms</option>
                            <option value=">1000">&gt; 1000ms</option>
                        </select>
                        
                        <button onclick="clearFilters()" style="padding: 10px 20px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer; white-space: nowrap;">清除筛选</button>
                    </div>
                </div>
                
                <!-- Tab 切换 -->
                <div style="display: flex; gap: 10px; margin-bottom: 20px; border-bottom: 2px solid #e9ecef; background: white; position: relative;">
                    <button class="tab-btn active" onclick="switchTab('all')" id="tab-all">全部 (<span id="count-all">0</span>)</button>
                    <button class="tab-btn" onclick="switchTab('success')" id="tab-success">成功 (<span id="count-success">0</span>)</button>
                    <button class="tab-btn" onclick="switchTab('failed')" id="tab-failed">失败 (<span id="count-failed">0</span>)</button>
                    <button class="tab-btn" onclick="switchTab('skipped')" id="tab-skipped">跳过 (<span id="count-skipped">0</span>)</button>
                </div>
                
                <div style="overflow-x: auto;">
                    <table>
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>分组ID</th>
                                <th>API名称</th>
                                <th>时间</th>
                                <th>URL</th>
                                <th>方法</th>
                                <th>响应时间</th>
                                <th>状态码</th>
                                <th>状态</th>
                                <th>验证</th>
                                <th>大小</th>
                                <th>操作</th>
                            </tr>
                        </thead>
                        <tbody id="details-tbody">
                            <tr><td colspan="12" style="text-align:center;padding:40px;color:#6c757d;">加载中...</td></tr>
                        </tbody>
                    </table>
                    
                    <!-- 分页组件（实时和静态模式都支持） -->
                    <div class="pagination" id="pagination" style="display: none;">
                        <button onclick="goToFirstPage()" id="firstBtn">首页</button>
                        <button onclick="previousPage()" id="prevBtn">上一页</button>
                        <span class="pagination-info">
                            第 <strong id="currentPage">1</strong> 页 / 共 <strong id="totalPages">1</strong> 页
                            (共 <strong id="totalRecords">0</strong> 条记录)
                        </span>
                        <button onclick="nextPage()" id="nextBtn">下一页</button>
                        <button onclick="goToLastPage()" id="lastBtn">末页</button>
                        <select id="pageSizeSelect" onchange="changePageSize()">
                            <option value="10">10条/页</option>
                            <option value="20" selected>20条/页</option>
                            <option value="50">50条/页</option>
                            <option value="100">100条/页</option>
                            <option value="200">200条/页</option>
                        </select>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="footer">
            <p>由 Go-Stress 高性能压测工具生成 | © 2025 Kamalyes</p>
        </div>
    </div>
    
    <script>
        let durationChart, statusChart, errorChart;
        const isRealtime = {{.IsRealtime}};
        const jsonFilename = '{{.JSONFilename}}';
        
        {{if not .IsRealtime}}
        // 静态模式 - 加载数据的统一处理函数
        function loadReportData(data) {
            try {
                // 隐藏文件选择器，显示信息栏
                document.getElementById('fileLoader').style.display = 'none';
                document.getElementById('infoBar').style.display = 'flex';
                
                // 更新静态指标
                updateStaticMetrics(data);
                
                // 更新图表
                updateChartsFromData(data);
                
                // 保存详情数据
                allDetailsData = data.all_details || [];
                
                // 初始化显示
                updateTabCounts();
                filterDetails();
                
                console.log('数据加载成功:', data);
            } catch (error) {
                console.error('数据处理错误:', error);
                alert('数据处理失败: ' + error.message);
                document.getElementById('fileName').textContent = '加载失败: ' + error.message;
            }
        }
        
        // 自动加载JSON文件
        function autoLoadJSON() {
            const jsonUrl = jsonFilename;
            document.getElementById('fileName').textContent = '正在自动加载: ' + jsonUrl;
            
            fetch(jsonUrl)
                .then(response => {
                    if (!response.ok) {
                        throw new Error('无法加载文件，请手动选择');
                    }
                    return response.json();
                })
                .then(data => {
                    loadReportData(data);
                })
                .catch(error => {
                    console.warn('自动加载失败:', error);
                    // 自动加载失败时，文件加载器继续显示，让用户手动选择
                    document.getElementById('fileName').textContent = '⚠️ 自动加载失败，请手动选择JSON文件';
                });
        }
        
        // 处理文件选择（必须在DOM之前定义，因为HTML中有onchange引用）
        function handleFileSelect(event) {
            const file = event.target.files[0];
            if (!file) return;
            
            // 显示文件名
            document.getElementById('fileName').textContent = '正在加载: ' + file.name;
            
            const reader = new FileReader();
            reader.onload = function(e) {
                try {
                    const data = JSON.parse(e.target.result);
                    loadReportData(data);
                } catch (error) {
                    console.error('JSON 解析错误:', error);
                    alert('文件格式错误，请选择正确的 JSON 文件');
                    document.getElementById('fileName').textContent = '加载失败: ' + error.message;
                }
            };
            
            reader.onerror = function() {
                console.error('文件读取错误');
                alert('文件读取失败');
                document.getElementById('fileName').textContent = '读取失败';
            };
            
            reader.readAsText(file);
        }
        {{end}}
        
        function initCharts() {
            durationChart = echarts.init(document.getElementById('durationChart'));
            statusChart = echarts.init(document.getElementById('statusChart'));
            errorChart = echarts.init(document.getElementById('errorChart'));
            
            // 初始化空图表
            durationChart.setOption({
                title: { text: '响应时间趋势', left: 'center' },
                tooltip: { trigger: 'axis' },
                xAxis: { type: 'category', data: [] },
                yAxis: { type: 'value', name: '响应时间 (ms)' },
                series: [{ data: [], type: 'line', smooth: true, areaStyle: { color: 'rgba(102, 126, 234, 0.2)' }, lineStyle: { color: '#667eea', width: 2 } }]
            });
            
            statusChart.setOption({
                title: { text: '状态码分布', left: 'center' },
                tooltip: { trigger: 'axis' },
                xAxis: { type: 'category', data: [] },
                yAxis: { type: 'value' },
                series: [{ data: [], type: 'bar', itemStyle: { color: '#667eea' } }]
            });
            
            errorChart.setOption({
                title: { text: 'Top错误', left: 'center' },
                tooltip: { trigger: 'item' },
                series: [{ type: 'pie', radius: '60%', data: [] }]
            });
            
            window.addEventListener('resize', () => {
                durationChart.resize();
                statusChart.resize();
                errorChart.resize();
            });
        }
        
        function updateStaticMetrics(data) {
            const setTextContent = (id, value) => {
                const elem = document.getElementById(id);
                if (elem) elem.textContent = value;
            };
            
            setTextContent('total-requests', data.total_requests || 0);
            setTextContent('success-requests', data.success_requests || 0);
            setTextContent('failed-requests', data.failed_requests || 0);
            setTextContent('success-rate', (data.success_rate || 0).toFixed(2) + '%');
            setTextContent('qps', (data.qps || 0).toFixed(2));
            setTextContent('avg-duration', (data.avg_duration_ms || 0) + 'ms');
            setTextContent('elapsed', Math.floor((data.total_time_ms || 0) / 1000) + 's');
        }
        
        function updateChartsFromData(data) {
            // 从请求明细中提取响应时间数据（最多显示最近1000个）
            if (data.request_details && data.request_details.length > 0) {
                const recentDetails = data.request_details.slice(-1000);
                const durations = recentDetails.map(d => d.duration / 1000000); // 纳秒转毫秒
                const indices = durations.map((_, i) => i + 1);
                
                durationChart.setOption({
                    xAxis: { data: indices },
                    series: [{ data: durations }]
                });
            }
            
            // 更新状态码分布图表
            if (data.status_codes && Object.keys(data.status_codes).length > 0) {
                const statusCodes = Object.keys(data.status_codes).sort();
                const statusCounts = statusCodes.map(code => data.status_codes[code]);
                
                statusChart.setOption({
                    xAxis: { data: statusCodes },
                    series: [{ data: statusCounts }]
                });
            }
            
            // 更新错误分布图表（Top 10）
            if (data.errors && Object.keys(data.errors).length > 0) {
                const errorList = Object.entries(data.errors)
                    .map(([error, count]) => ({ name: error.substring(0, 50), value: count }))
                    .sort((a, b) => b.value - a.value)
                    .slice(0, 10);
                
                errorChart.setOption({
                    series: [{ data: errorList }]
                });
            }
        }
        
        function renderStaticDetails(details) {
            const tbody = document.getElementById('details-tbody');
            if (!details || details.length === 0) {
                tbody.innerHTML = '<tr><td colspan="12" style="text-align:center;">无请求数据</td></tr>';
                return;
            }
            
            tbody.innerHTML = details.map((req, index) => {
                const statusClass = req.skipped ? 'status-warning' : (req.success ? 'status-success' : 'status-failure');
                const statusText = req.skipped ? '⏭ 跳过' : (req.success ? '✓ 成功' : '✗ 失败');
                const detailsId = 'details-' + index;
                
                return ` + "`" + `
                    <tr>
                        <td>${index + 1}</td>
                        <td>${req.group_id || '-'}</td>
                        <td>${req.api_name || '-'}</td>
                        <td>${req.timestamp ? new Date(req.timestamp).toLocaleTimeString() : '-'}</td>
                        <td style="max-width:300px;overflow:hidden;text-overflow:ellipsis;" title="${req.request_url || ''}">${req.request_url || ''}</td>
                        <td>${req.request_method || ''}</td>
                        <td>${req.duration_ms || 0}ms</td>
                        <td>${req.status_code || (req.skipped ? '-' : 0)}</td>
                        <td class="${statusClass}">${statusText}</td>
                        <td class="${req.verifications && req.verifications.length > 0 ? (req.verifications.every(v => v.success) ? 'status-success' : 'status-error') : ''}">${req.verifications && req.verifications.length > 0 ? (req.verifications.every(v => v.success) ? '✓ 通过' : '✗ 失败') : '-'}</td>
                        <td>${formatBytes(req.size || 0)}</td>
                        <td><button onclick="toggleDetails('${detailsId}')">查看详情</button></td>
                    </tr>
                    <tr id="${detailsId}" class="details-row" style="display:none;">
                        <td colspan="12">
                            <div class="detail-content">
                                <div class="detail-section">
                                    <strong>请求Query:</strong>
                                    <pre>${escapeHtml(req.request_query || '')}</pre>
                                </div>
                                <div class="detail-section">
                                    <strong>请求Headers:</strong>
                                    <pre>${escapeHtml(JSON.stringify(req.request_headers || {}, null, 2))}</pre>
                                </div>
                                <div class="detail-section">
                                    <strong>请求Body:</strong>
                                    <pre>${escapeHtml(req.request_body || '')}</pre>
                                </div>
                                <div class="detail-section">
                                    <strong>响应Body:</strong>
                                    <pre>${escapeHtml(req.response_body || '')}</pre>
                                </div>
                                ${req.error ? ` + "`" + `<div class="detail-section"><strong>错误:</strong><pre style="color:red;">${escapeHtml(req.error)}</pre></div>` + "`" + ` : ''}
                                ${req.verifications && req.verifications.length > 0 ? ` + "`" + `
                                    <div class="detail-section">
                                        <strong>验证结果:</strong>
                                        <table style="width:100%; border-collapse: collapse; margin-top: 10px;">
                                            <thead>
                                                <tr style="background: #f8f9fa;">
                                                    <th style="padding: 8px; border: 1px solid #dee2e6;">类型</th>
                                                    <th style="padding: 8px; border: 1px solid #dee2e6;">状态</th>
                                                    <th style="padding: 8px; border: 1px solid #dee2e6;">期望值</th>
                                                    <th style="padding: 8px; border: 1px solid #dee2e6;">实际值</th>
                                                    <th style="padding: 8px; border: 1px solid #dee2e6;">消息</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                ${req.verifications.map(v => ` + "`" + `
                                                    <tr style="background: ${v.success ? '#f0fff4' : '#fff5f5'};">
                                                        <td style="padding: 8px; border: 1px solid #dee2e6;">${v.type}</td>
                                                        <td style="padding: 8px; border: 1px solid #dee2e6; color: ${v.success ? '#38ef7d' : '#f45c43'};">${v.success ? '✓ 通过' : '✗ 失败'}</td>
                                                        <td style="padding: 8px; border: 1px solid #dee2e6; max-width: 200px; overflow: hidden; text-overflow: ellipsis;">${escapeHtml(v.expect || '-')}</td>
                                                        <td style="padding: 8px; border: 1px solid #dee2e6; max-width: 200px; overflow: hidden; text-overflow: ellipsis;">${escapeHtml(v.actual || '-')}</td>
                                                        <td style="padding: 8px; border: 1px solid #dee2e6;">${escapeHtml(v.message || '-')}</td>
                                                    </tr>
                                                ` + "`" + `).join('')}
                                            </tbody>
                                        </table>
                                    </div>
                                ` + "`" + ` : ''}
                            </div>
                        </td>
                    </tr>
                ` + "`" + `;
            }).join('');
        }
        
        function escapeHtml(text) {
            if (!text) return '';
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        function toggleDetails(detailsId) {
            const row = document.getElementById(detailsId);
            if (row) {
                row.style.display = row.style.display === 'none' ? 'table-row' : 'none';
            }
        }
        
        // 全局变量存储所有数据
        let allDetailsData = [];
        let currentTab = 'all';
        let currentPage = 1;
        let pageSize = 20;
        let filteredData = [];
        
        // 页面加载后初始化计数
        document.addEventListener('DOMContentLoaded', function() {
            updateTabCounts();
        });
        
        // Tab 切换
        function switchTab(tab) {
            currentTab = tab;
            
            // 更新按钮状态
            document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
            document.getElementById('tab-' + tab).classList.add('active');
            
            // 重新渲染
            filterDetails();
        }
        
        // 搜索和过滤
        function filterDetails() {
            const searchValue = document.getElementById('searchPath').value.toLowerCase();
            const methodFilter = document.getElementById('methodFilter').value;
            const statusFilter = document.getElementById('statusFilter').value;
            const durationFilter = document.getElementById('durationFilter').value;
            
            filteredData = allDetailsData;
            
            // 根据Tab过滤
            if (currentTab === 'success') {
                filteredData = filteredData.filter(d => d.success && !d.skipped);
            } else if (currentTab === 'failed') {
                filteredData = filteredData.filter(d => !d.success && !d.skipped);
            } else if (currentTab === 'skipped') {
                filteredData = filteredData.filter(d => d.skipped);
            }
            
            // 根据搜索词过滤
            if (searchValue) {
                filteredData = filteredData.filter(d => 
                    (d.url || '').toLowerCase().includes(searchValue) ||
                    (d.request_url || '').toLowerCase().includes(searchValue)
                );
            }
            
            // 根据请求方法过滤
            if (methodFilter) {
                filteredData = filteredData.filter(d => 
                    (d.method || d.request_method || '').toUpperCase() === methodFilter
                );
            }
            
            // 根据状态码过滤
            if (statusFilter) {
                filteredData = filteredData.filter(d => {
                    const code = d.status_code || 0;
                    if (statusFilter === '2xx') return code >= 200 && code < 300;
                    if (statusFilter === '3xx') return code >= 300 && code < 400;
                    if (statusFilter === '4xx') return code >= 400 && code < 500;
                    if (statusFilter === '5xx') return code >= 500 && code < 600;
                    return true;
                });
            }
            
            // 根据响应时间过滤
            if (durationFilter) {
                filteredData = filteredData.filter(d => {
                    const durationMs = d.duration_ms || (d.duration ? d.duration / 1000000 : 0);
                    if (durationFilter === '<100') return durationMs < 100;
                    if (durationFilter === '100-500') return durationMs >= 100 && durationMs < 500;
                    if (durationFilter === '500-1000') return durationMs >= 500 && durationMs < 1000;
                    if (durationFilter === '>1000') return durationMs >= 1000;
                    return true;
                });
            }
            
            // 更新计数
            updateTabCounts();
            
            // 重置到第一页
            currentPage = 1;
            renderPage();
        }
        
        // 清除所有筛选
        function clearFilters() {
            document.getElementById('searchPath').value = '';
            document.getElementById('methodFilter').value = '';
            document.getElementById('statusFilter').value = '';
            document.getElementById('durationFilter').value = '';
            filterDetails();
        }
        
        // 更新Tab计数
        function updateTabCounts() {
            const total = allDetailsData.length;
            const skipped = allDetailsData.filter(d => d.skipped).length;
            const success = allDetailsData.filter(d => d.success && !d.skipped).length;
            const failed = allDetailsData.filter(d => !d.success && !d.skipped).length;
            
            document.getElementById('count-all').textContent = total;
            document.getElementById('count-success').textContent = success;
            document.getElementById('count-failed').textContent = failed;
            document.getElementById('count-skipped').textContent = skipped;
        }
        
        // 通用分页函数（实时和静态模式共用）
        function renderPage() {
            const start = (currentPage - 1) * pageSize;
            const end = start + pageSize;
            const pageData = filteredData.slice(start, end);
            
            // 根据模式渲染数据
            {{if .IsRealtime}}
            renderRealtimeDetails(pageData);
            {{else}}
            renderStaticDetails(pageData);
            {{end}}
            
            // 更新分页控件
            updatePaginationControls();
            
            // 显示/隐藏分页组件
            const paginationEl = document.getElementById('pagination');
            if (paginationEl && filteredData.length > pageSize) {
                paginationEl.style.display = 'flex';
            } else if (paginationEl) {
                paginationEl.style.display = 'none';
            }
        }
        
        function updatePaginationControls() {
            const totalPages = Math.ceil(filteredData.length / pageSize) || 1;
            
            document.getElementById('currentPage').textContent = currentPage;
            document.getElementById('totalPages').textContent = totalPages;
            document.getElementById('totalRecords').textContent = filteredData.length;
            
            // 更新按钮状态
            document.getElementById('firstBtn').disabled = currentPage === 1;
            document.getElementById('prevBtn').disabled = currentPage === 1;
            document.getElementById('nextBtn').disabled = currentPage >= totalPages;
            document.getElementById('lastBtn').disabled = currentPage >= totalPages;
        }
        
        function goToFirstPage() {
            currentPage = 1;
            renderPage();
        }
        
        function previousPage() {
            if (currentPage > 1) {
                currentPage--;
                renderPage();
            }
        }
        
        function nextPage() {
            const totalPages = Math.ceil(filteredData.length / pageSize);
            if (currentPage < totalPages) {
                currentPage++;
                renderPage();
            }
        }
        
        function goToLastPage() {
            currentPage = Math.ceil(filteredData.length / pageSize) || 1;
            renderPage();
        }
        
        function changePageSize() {
            pageSize = parseInt(document.getElementById('pageSizeSelect').value);
            currentPage = 1;
            renderPage();
        }
        
        {{if .IsRealtime}}
        // 实时模式 - SSE连接和数据更新逻辑
        function updateMetrics(data) {
            document.getElementById('total-requests').textContent = data.total_requests;
            document.getElementById('success-requests').textContent = data.success_requests;
            document.getElementById('failed-requests').textContent = data.failed_requests;
            document.getElementById('success-rate').textContent = data.success_rate.toFixed(2) + '%';
            document.getElementById('qps').textContent = data.qps.toFixed(2);
            document.getElementById('avg-duration').textContent = data.avg_duration_ms + 'ms';
            document.getElementById('elapsed').textContent = data.elapsed_seconds + 's';
        }
        
        function updateCharts(data) {
            if (data.recent_durations && data.recent_durations.length > 0) {
                const indices = data.recent_durations.map((_, i) => i + 1);
                durationChart.setOption({
                    xAxis: { data: indices },
                    series: [{ data: data.recent_durations }]
                });
            }
            
            if (data.status_codes) {
                const codes = Object.keys(data.status_codes).sort();
                const values = codes.map(code => data.status_codes[code]);
                statusChart.setOption({
                    xAxis: { data: codes },
                    series: [{
                        data: values.map((v, i) => ({
                            value: v,
                            itemStyle: {
                                color: codes[i].startsWith('2') ? '#38ef7d' :
                                       codes[i].startsWith('4') ? '#f45c43' :
                                       codes[i].startsWith('5') ? '#eb3349' : '#667eea'
                            }
                        }))
                    }]
                });
            }
            
            if (data.errors) {
                const errors = Object.entries(data.errors)
                    .map(([name, value]) => ({
                        name: name.substring(0, 30) + (name.length > 30 ? '...' : ''),
                        value: value
                    }))
                    .slice(0, 5);
                errorChart.setOption({
                    series: [{ data: errors }]
                });
            }
        }
        
        let lastDetailsCount = 0;
        const openDetails = new Set(); // 记录已展开的详情
        
        function loadDetails() {
            fetch('/api/details?offset=0&limit=100')
                .then(res => res.json())
                .then(data => {
                    // 只在数据数量变化时才更新
                    if (data.total === lastDetailsCount && lastDetailsCount > 0) {
                        return;
                    }
                    lastDetailsCount = data.total;
                    
                    // 存储到全局变量
                    allDetailsData = data.details || [];
                    
                    // 应用过滤
                    filterDetails();
                });
        }
        
        function renderRealtimeDetails(details) {
            const tbody = document.getElementById('details-tbody');
            tbody.innerHTML = '';
            
            if (details && details.length > 0) {
                details.forEach((detail, idx) => {
                            const row = tbody.insertRow();
                            row.innerHTML = ` + "`" + `
                                <td>${detail.id}</td>
                                <td>${detail.group_id || '-'}</td>
                                <td>${detail.api_name || '-'}</td>
                                <td>${new Date(detail.timestamp).toLocaleTimeString()}</td>
                                <td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="${detail.url || '-'}">${detail.url || '-'}</td>
                                <td>${detail.method || '-'}</td>
                                <td>${(detail.duration / 1000000).toFixed(2)}ms</td>
                                <td>${detail.skipped ? '-' : (detail.status_code || '-')}</td>
                                <td class="${detail.skipped ? 'status-warning' : (detail.success ? 'status-success' : 'status-error')}">${detail.skipped ? '⏭ 跳过' : (detail.success ? '✓ 成功' : '✗ 失败')}</td>
                                <td class="${detail.verifications && detail.verifications.length > 0 ? (detail.verifications.every(v => v.success) ? 'status-success' : 'status-error') : ''}">${detail.verifications && detail.verifications.length > 0 ? (detail.verifications.every(v => v.success) ? '✓ 通过' : '✗ 失败') : '-'}</td>
                                <td>${formatBytes(detail.size)}</td>
                                <td><button type="button" class="detail-btn" onclick="event.stopPropagation(); toggleRealtimeDetail(${idx})"> 查看详情</button></td>
                            ` + "`" + `;
                            
                            // 详情行
                            const detailRow = tbody.insertRow();
                            detailRow.className = 'detail-row';
                            detailRow.id = 'realtime-detail-' + idx;
                            let detailHTML = '<td colspan="12"><div class="detail-content">';
                            
                            if (detail.query) {
                                detailHTML += ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-section-title">🔍 Query参数</div>
                                        <div class="detail-code">${detail.query}</div>
                                    </div>
                                ` + "`" + `;
                            }
                            
                            if (detail.headers && Object.keys(detail.headers).length > 0) {
                                detailHTML += ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-section-title">📤 请求头</div>
                                        <table class="detail-table">
                                ` + "`" + `;
                                for (let [key, value] of Object.entries(detail.headers)) {
                                    detailHTML += ` + "`<tr><td>${key}</td><td>${value}</td></tr>`" + `;
                                }
                                detailHTML += '</table></div>';
                            }
                            
                            if (detail.body) {
                                detailHTML += ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-section-title">📦 请求体</div>
                                        <div class="detail-code">${detail.body}</div>
                                    </div>
                                ` + "`" + `;
                            }
                            
                            if (detail.response_headers && Object.keys(detail.response_headers).length > 0) {
                                detailHTML += ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-section-title">📥 响应头</div>
                                        <table class="detail-table">
                                ` + "`" + `;
                                for (let [key, value] of Object.entries(detail.response_headers)) {
                                    detailHTML += ` + "`<tr><td>${key}</td><td>${value}</td></tr>`" + `;
                                }
                                detailHTML += '</table></div>';
                            }
                            
                            if (detail.response_body) {
                                detailHTML += ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-section-title">📄 响应体</div>
                                        <div class="detail-code">${detail.response_body}</div>
                                    </div>
                                ` + "`" + `;
                            }
                            
                            if (detail.error) {
                                detailHTML += ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-section-title">❌ 错误信息</div>
                                        <div class="detail-code" style="color: #f45c43;">${detail.error}</div>
                                    </div>
                                ` + "`" + `;
                            }
                            
                            // 验证结果
                            if (detail.verifications && detail.verifications.length > 0) {
                                const allSuccess = detail.verifications.every(v => v.success);
                                detailHTML += ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-section-title">✓ 验证结果 <span style="color: ${allSuccess ? '#38ef7d' : '#f45c43'};">(${allSuccess ? '全部通过' : '部分失败'})</span></div>
                                        <table class="detail-table">
                                            <thead>
                                                <tr>
                                                    <th>类型</th>
                                                    <th>状态</th>
                                                    <th>期望值</th>
                                                    <th>实际值</th>
                                                    <th>消息</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                ` + "`" + `;
                                detail.verifications.forEach(v => {
                                    detailHTML += ` + "`" + `
                                        <tr style="background: ${v.success ? '#f0fff4' : '#fff5f5'};">
                                            <td>${v.type}</td>
                                            <td style="color: ${v.success ? '#38ef7d' : '#f45c43'};">${v.success ? '✓ 通过' : '✗ 失败'}</td>
                                            <td style="max-width: 200px; overflow: hidden; text-overflow: ellipsis;">${v.expect || '-'}</td>
                                            <td style="max-width: 200px; overflow: hidden; text-overflow: ellipsis;">${v.actual || '-'}</td>
                                            <td>${v.message || '-'}</td>
                                        </tr>
                                    ` + "`" + `;
                                });
                                detailHTML += '</tbody></table></div>';
                            }
                            
                            detailHTML += '</div></td>';
                            detailRow.innerHTML = detailHTML;
                            
                            // 恢复之前展开的状态
                            if (openDetails.has(idx)) {
                                detailRow.classList.add('show');
                            }
                        });
                    } else {
                        tbody.innerHTML = '<tr><td colspan="12" style="text-align:center;padding:40px;color:#6c757d;">暂无数据</td></tr>';
                    }
        }
        
        function formatBytes(bytes) {
            if (bytes === 0) return '0B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return (bytes / Math.pow(k, i)).toFixed(2) + sizes[i];
        }
        
        function connectSSE() {
            const eventSource = new EventSource('/stream');
            
            eventSource.onmessage = function(event) {
                const data = JSON.parse(event.data);
                updateMetrics(data);
                updateCharts(data);
                loadDetails();
            };
            
            eventSource.onerror = function() {
                console.error('SSE连接错误，5秒后重连...');
                eventSource.close();
                setTimeout(connectSSE, 5000);
            };
        }
        
        initCharts();
        connectSSE();
        loadDetails();
        
        function toggleDetail(idx) {
            const detailRow = document.getElementById('detail-' + idx);
            if (detailRow) {
                detailRow.classList.toggle('show');
            }
        }
        
        function toggleRealtimeDetail(idx) {
            const detailRow = document.getElementById('realtime-detail-' + idx);
            if (detailRow) {
                const wasOpen = detailRow.classList.contains('show');
                detailRow.classList.toggle('show');
                
                // 记录状态
                if (!wasOpen) {
                    openDetails.add(idx);
                } else {
                    openDetails.delete(idx);
                }
            }
        }
        {{else}}
        // 静态模式 - 等待 DOM 加载完成后初始化
        document.addEventListener('DOMContentLoaded', function() {
            initCharts();
            // 自动尝试加载JSON文件
            autoLoadJSON();
        });
        {{end}}
        
        function escapeHtml(text) {
            if (!text) return '';
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        function toggleDetails(detailsId) {
            const row = document.getElementById(detailsId);
            if (row) {
                row.style.display = row.style.display === 'none' ? 'table-row' : 'none';
            }
        }
        
    </script>
</body>
</html>
`
