// Go-Stress 报告脚本
console.log('========== 报告脚本已加载 ==========');
console.log('当前时间:', new Date().toLocaleString());

// ============ 元素ID常量 ============
const ELEMENT_IDS = {
  // 指标卡片
  TOTAL_REQUESTS: 'total-requests',
  SUCCESS_REQUESTS: 'success-requests',
  FAILED_REQUESTS: 'failed-requests',
  SKIPPED_REQUESTS: 'skipped-requests',
  SUCCESS_RATE: 'success-rate',
  QPS: 'qps',
  AVG_DURATION: 'avg-duration',
  MIN_DURATION: 'min-duration',
  MAX_DURATION: 'max-duration',
  P50: 'p50',
  P90: 'p90',
  P95: 'p95',
  P99: 'p99',
  ELAPSED: 'elapsed',
  TEST_DURATION: 'test-duration',
  
  // 容器元素
  FILE_LOADER: 'fileLoader',
  METRICS_GRID: 'metricsGrid',
  FILE_NAME: 'fileName',
  DETAILS_TBODY: 'details-tbody',
  
  // Tab标签
  TAB_ALL: 'tab-all',
  TAB_SUCCESS: 'tab-success',
  TAB_FAILED: 'tab-failed',
  TAB_SKIPPED: 'tab-skipped',
  COUNT_ALL: 'count-all',
  COUNT_SUCCESS: 'count-success',
  COUNT_FAILED: 'count-failed',
  COUNT_SKIPPED: 'count-skipped',
  
  // 筛选器
  SEARCH_PATH: 'searchPath',
  METHOD_FILTER: 'methodFilter',
  STATUS_FILTER: 'statusFilter',
  DURATION_FILTER: 'durationFilter',
  
  // 分页
  PAGINATION: 'pagination',
  CURRENT_PAGE: 'currentPage',
  TOTAL_PAGES: 'totalPages',
  TOTAL_RECORDS: 'totalRecords',
  PAGE_SIZE_SELECT: 'pageSizeSelect',
  FIRST_BTN: 'firstBtn',
  PREV_BTN: 'prevBtn',
  NEXT_BTN: 'nextBtn',
  LAST_BTN: 'lastBtn',
  
  // 控制按钮（实时模式）
  PAUSE_BTN: 'pauseBtn',
  STOP_BTN: 'stopBtn',
  STATUS_TEXT: 'statusText',
  STATUS_DOT: 'statusDot',
  
  // 图表
  DURATION_CHART: 'durationChart',
  STATUS_CHART: 'statusChart',
  ERROR_CHART: 'errorChart'
};

// ============ Tab名称常量 ============
const TAB_NAMES = {
  ALL: 'all',
  SUCCESS: 'success',
  FAILED: 'failed',
  SKIPPED: 'skipped'
};

// ============ 验证状态常量 ============
const VERIFY_STATUS = {
  SKIPPED: {
    color: '#6c757d',
    bg: '#f8f9fa',
    border: '#dee2e6',
    text: '未执行',
    icon: '⏭',
    class: 'status-warning'
  },
  SUCCESS: {
    color: '#38ef7d',
    bg: '#f0fdf4',
    border: '#86efac',
    text: '验证通过',
    icon: '✓',
    class: 'status-success'
  },
  FAILED: {
    color: '#f45c43',
    bg: '#fff5f5',
    border: '#feb2b2',
    text: '验证失败',
    icon: '✗',
    class: 'status-error'
  }
};

// ============ 协议图标和样式映射 ============
const PROTOCOL_STYLES = {
  http: { icon: '🌐', class: 'protocol-http', name: 'HTTP' },
  https: { icon: '🔒', class: 'protocol-https', name: 'HTTPS' },
  grpc: { icon: '⚡', class: 'protocol-grpc', name: 'gRPC' },
  websocket: { icon: '📡', class: 'protocol-websocket', name: 'WebSocket' },
  ws: { icon: '📡', class: 'protocol-websocket', name: 'WebSocket' },
  wss: { icon: '🔐', class: 'protocol-wss', name: 'WebSocket (Secure)' }
};

// ============ HTTP 方法样式映射 (Swagger风格) ============
const HTTP_METHOD_STYLES = {
  GET: 'http-method-get',
  POST: 'http-method-post',
  PUT: 'http-method-put',
  DELETE: 'http-method-delete',
  PATCH: 'http-method-patch',
  HEAD: 'http-method-head',
  OPTIONS: 'http-method-options'
};

// ============ 控制状态常量 ============
const CONTROL_STATUS = {
  RUNNING: {
    pauseBtn: { text: '⏸ 暂停', bg: '#ffc107', color: '#333' },
    statusText: '实时监控中',
    statusDot: { bg: '#38ef7d', animation: 'pulse 2s infinite' }
  },
  PAUSED: {
    pauseBtn: { text: '▶ 恢复', bg: '#28a745', color: 'white' },
    statusText: '已暂停',
    statusDot: { bg: '#ffc107', animation: 'none' }
  },
  STOPPED: {
    statusText: '已停止',
    statusDot: { bg: '#dc3545', animation: 'none' }
  }
};

// ============ API 端点常量 ============
const API_ENDPOINTS = {
  PAUSE: '/api/pause',
  RESUME: '/api/resume',
  STOP: '/api/stop'
};

// ============ 格式化常量 ============
const FORMAT_CONFIG = {
  DECIMAL_PLACES: 2,           // 小数位数
  BYTES_UNIT: 1024,           // 字节单位换算基数
  MS_TO_NS: 1000000,          // 毫秒转纳秒
  MS_TO_SEC: 1000,            // 毫秒转秒
  RESPONSE_TRUNCATE: 2000,    // 响应内容截断长度
  DEFAULT_PAGE_SIZE: 20       // 默认分页大小
};

// ============ 文件大小单位 ============
const SIZE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'];

// 格式化 HTTP 方法为带样式的标签
function formatHttpMethod(method) {
  if (!method) return '<span class="http-method http-method-default">N/A</span>';
  const upperMethod = method.toUpperCase();
  const className = HTTP_METHOD_STYLES[upperMethod] || 'http-method-default';
  return '<span class="http-method ' + className + '">' + upperMethod + '</span>';
}

let durationChart, statusChart, errorChart;
const isRealtime = (typeof IS_REALTIME_PLACEHOLDER !== 'undefined' && IS_REALTIME_PLACEHOLDER) || false;
const jsonFilename = "JSON_FILENAME_PLACEHOLDER" || "index.json";
let serverTotal = 0; // 服务器返回的真实总数（用于实时模式分页显示）

// 从 URL 获取参数
const urlParams = new URLSearchParams(window.location.search);
const slaveId = urlParams.get('slave_id') || ''; // 分布式模式下的 slave_id
const realtimeUrl = urlParams.get('realtime_url') || ''; // Slave 实时报告服务器地址（如 http://localhost:8088）

console.log('运行模式:', isRealtime ? '实时模式' : '静态模式');
if (slaveId) {
  console.log('Slave ID:', slaveId);
}
if (realtimeUrl) {
  console.log('实时报告服务器:', realtimeUrl);
}

// 全局变量存储所有数据
let allDetailsData = [];
let currentTab = TAB_NAMES.ALL;
let currentPage = 1;
let pageSize = FORMAT_CONFIG.DEFAULT_PAGE_SIZE;
let filteredData = [];
let isPaused = false;

// ============ 全局函数（供HTML内联调用） ============
// 控制函数（暂停/恢复/停止）
window.togglePause = function() {
  const endpoint = isPaused ? API_ENDPOINTS.RESUME : API_ENDPOINTS.PAUSE;
  const pauseBtn = document.getElementById(ELEMENT_IDS.PAUSE_BTN);
  const statusText = document.getElementById(ELEMENT_IDS.STATUS_TEXT);
  const statusDot = document.getElementById(ELEMENT_IDS.STATUS_DOT);
  
  fetch(endpoint, { method: 'POST' })
    .then(response => response.json())
    .then(data => {
      if (data.success) {
        isPaused = !isPaused;
        const status = isPaused ? CONTROL_STATUS.PAUSED : CONTROL_STATUS.RUNNING;
        
        pauseBtn.textContent = status.pauseBtn.text;
        pauseBtn.style.background = status.pauseBtn.bg;
        pauseBtn.style.color = status.pauseBtn.color;
        statusText.textContent = status.statusText;
        statusDot.style.background = status.statusDot.bg;
        statusDot.style.animation = status.statusDot.animation;
      }
    })
    .catch(err => console.error('控制操作失败:', err));
};

window.stopMonitoring = function() {
  if (!confirm('确定要停止压测吗？停止后将无法恢复！')) {
    return;
  }
  
  const stopBtn = document.getElementById(ELEMENT_IDS.STOP_BTN);
  const pauseBtn = document.getElementById(ELEMENT_IDS.PAUSE_BTN);
  const statusText = document.getElementById(ELEMENT_IDS.STATUS_TEXT);
  const statusDot = document.getElementById(ELEMENT_IDS.STATUS_DOT);
  
  fetch(API_ENDPOINTS.STOP, { method: 'POST' })
    .then(response => response.json())
    .then(data => {
      if (data.success) {
        stopBtn.disabled = true;
        pauseBtn.disabled = true;
        stopBtn.style.opacity = '0.5';
        pauseBtn.style.opacity = '0.5';
        statusText.textContent = CONTROL_STATUS.STOPPED.statusText;
        statusDot.style.background = CONTROL_STATUS.STOPPED.statusDot.bg;
        statusDot.style.animation = CONTROL_STATUS.STOPPED.statusDot.animation;
        alert('压测已停止！');
      }
    })
    .catch(err => console.error('停止失败:', err));
};

// 处理文件选择
window.handleFileSelect = function (event) {
  const file = event.target.files[0];
  if (!file) return;

  const fileNameElem = document.getElementById(ELEMENT_IDS.FILE_NAME);
  if (fileNameElem) {
    fileNameElem.textContent = "正在加载: " + file.name;
  }

  const reader = new FileReader();
  reader.onload = function (e) {
    try {
      const data = JSON.parse(e.target.result);
      if (window.loadReportData) {
        window.loadReportData(data);
      }
    } catch (error) {
      console.error("JSON 解析错误:", error);
      alert("文件格式错误,请选择正确的 JSON 文件");
      if (fileNameElem) {
        fileNameElem.textContent = "加载失败: " + error.message;
      }
    }
  };

  reader.onerror = function () {
    console.error("文件读取错误");
    alert("文件读取失败");
    if (fileNameElem) {
      fileNameElem.textContent = "读取失败";
    }
  };

  reader.readAsText(file);
};

// ============ 静态模式专用函数 ============
if (!isRealtime) {
  // 加载数据的统一处理函数
  window.loadReportData = function (data) {
    try {
      document.getElementById(ELEMENT_IDS.FILE_LOADER).style.display = "none";
      const metricsGrid = document.getElementById(ELEMENT_IDS.METRICS_GRID);
      if (metricsGrid) {
        metricsGrid.style.display = "grid";
      }

      updateStaticMetrics(data);
      updateChartsFromData(data);

      allDetailsData = data.all_details || data.request_details || [];
      updateTabCounts();
      filterDetails();

      console.log("数据加载成功:", data);
    } catch (error) {
      console.error("数据处理错误:", error);
      alert("数据处理失败: " + error.message);
      document.getElementById(ELEMENT_IDS.FILE_NAME).textContent =
        "加载失败: " + error.message;
    }
  };

  // 自动加载JSON文件
  window.autoLoadJSON = function () {
    const jsonUrl = jsonFilename;
    document.getElementById(ELEMENT_IDS.FILE_NAME).textContent =
      "正在自动加载: " + jsonUrl;

    fetch(jsonUrl)
      .then((response) => {
        if (!response.ok) {
          throw new Error("无法加载文件,请手动选择");
        }
        return response.json();
      })
      .then((data) => {
        loadReportData(data);
      })
      .catch((error) => {
        console.warn("自动加载失败:", error);
        document.getElementById(ELEMENT_IDS.FILE_NAME).textContent =
          "⚠️ 自动加载失败,请手动选择JSON文件";
      });
  };
}

// ============ 图表初始化 ============
function initCharts() {
  const durationChartDom = document.getElementById(ELEMENT_IDS.DURATION_CHART);
  const statusChartDom = document.getElementById(ELEMENT_IDS.STATUS_CHART);
  const errorChartDom = document.getElementById(ELEMENT_IDS.ERROR_CHART);

  if (durationChartDom) {
    durationChart = echarts.init(durationChartDom);
    durationChart.setOption({
      title: { text: "响应时间趋势", left: "center" },
      tooltip: { trigger: "axis" },
      xAxis: { type: "category", data: [] },
      yAxis: { type: "value", name: "响应时间 (ms)" },
      series: [
        {
          data: [],
          type: "line",
          smooth: true,
          areaStyle: { color: "rgba(102, 126, 234, 0.2)" },
          lineStyle: { color: "#667eea", width: 2 },
        },
      ],
    });
  } else {
    console.error("durationChartDom not found!");
  }

  if (statusChartDom) {
    statusChart = echarts.init(statusChartDom);
    statusChart.setOption({
      title: { text: "状态码分布", left: "center" },
      tooltip: { trigger: "axis" },
      xAxis: { type: "category", data: [] },
      yAxis: { type: "value" },
      series: [{ data: [], type: "bar", itemStyle: { color: "#667eea" } }],
    });
  } else {
    console.error("statusChartDom not found!");
  }

  if (errorChartDom) {
    errorChart = echarts.init(errorChartDom);
    errorChart.setOption({
      title: { text: "Top错误", left: "center" },
      tooltip: { trigger: "item" },
      series: [{ type: "pie", radius: "60%", data: [] }],
    });
  } else {
    console.error("errorChartDom not found!");
  }

  window.addEventListener("resize", () => {
    if (durationChart) durationChart.resize();
    if (statusChart) statusChart.resize();
    if (errorChart) errorChart.resize();
  });
}

// ============ 静态模式数据更新 ============
function updateStaticMetrics(data) {
  const setTextContent = (id, value) => {
    const elem = document.getElementById(id);
    if (elem) elem.textContent = value;
  };

  // 使用与实时报告相同的元素ID
  setTextContent(ELEMENT_IDS.TOTAL_REQUESTS, data.total_requests || 0);
  setTextContent(ELEMENT_IDS.SUCCESS_REQUESTS, data.success_requests || 0);
  setTextContent(ELEMENT_IDS.FAILED_REQUESTS, data.failed_requests || 0);
  setTextContent(ELEMENT_IDS.SKIPPED_REQUESTS, data.skipped_requests || 0);
  setTextContent(ELEMENT_IDS.SUCCESS_RATE, (data.success_rate || 0).toFixed(2) + "%");
  setTextContent(ELEMENT_IDS.QPS, (data.qps || 0).toFixed(2));
  setTextContent(ELEMENT_IDS.AVG_DURATION, (data.avg_latency || 0).toFixed(2) + "ms");
  
  // 响应时间统计
  setTextContent(ELEMENT_IDS.MIN_DURATION, (data.min_latency || 0).toFixed(2) + "ms");
  setTextContent(ELEMENT_IDS.MAX_DURATION, (data.max_latency || 0).toFixed(2) + "ms");
  
  // 百分位统计
  setTextContent(ELEMENT_IDS.P50, (data.p50_latency || 0).toFixed(2) + "ms");
  setTextContent(ELEMENT_IDS.P90, (data.p90_latency || 0).toFixed(2) + "ms");
  setTextContent(ELEMENT_IDS.P95, (data.p95_latency || 0).toFixed(2) + "ms");
  setTextContent(ELEMENT_IDS.P99, (data.p99_latency || 0).toFixed(2) + "ms");
  
  // 静态报告特有的：测试时长（使用total_time）
  const totalTimeSec = data.total_time_ms ? (data.total_time_ms / 1000).toFixed(2) : 0;
  setTextContent(ELEMENT_IDS.TEST_DURATION, totalTimeSec + "s");
}

function updateChartsFromData(data) {
  if (data.request_details && data.request_details.length > 0 && durationChart) {
    const recentDetails = data.request_details.slice(-1000);
    const durations = recentDetails.map((d) => d.duration / 1000000);
    const indices = durations.map((_, i) => i + 1);

    durationChart.setOption({
      xAxis: { data: indices },
      series: [{ data: durations }],
    });
  }

  if (data.status_codes && Object.keys(data.status_codes).length > 0 && statusChart) {
    const statusCodes = Object.keys(data.status_codes).sort();
    const statusCounts = statusCodes.map((code) => data.status_codes[code]);

    statusChart.setOption({
      xAxis: { data: statusCodes },
      series: [{ data: statusCounts }],
    });
  }

  if (data.errors && Object.keys(data.errors).length > 0 && errorChart) {
    const errorList = Object.entries(data.errors)
      .map(([error, count]) => ({ name: error.substring(0, 50), value: count }))
      .sort((a, b) => b.value - a.value)
      .slice(0, 10);

    errorChart.setOption({
      series: [{ data: errorList }],
    });
  }
}

// ============ 明细渲染 ============
function renderStaticDetails(details) {
  const tbody = document.getElementById(ELEMENT_IDS.DETAILS_TBODY);
  if (!details || details.length === 0) {
    tbody.innerHTML =
      '<tr><td colspan="12" style="text-align:center;">无请求数据</td></tr>';
    return;
  }

  tbody.innerHTML = details
    .map((req, index) => {
      const statusClass = req.skipped ? "status-warning" : (req.success ? "status-success" : "status-error");
      const statusText = req.skipped ? "⏭ 跳过" : (req.success ? "✓ 成功" : "✗ 失败");
      const detailsId = "details-" + index;
      
      // 验证状态：考虑跳过、成功、失败三种情况
      let verifyStatus = "-";
      let verifyClass = "";
      if (req.verifications && req.verifications.length > 0) {
        const allSkipped = req.verifications.every((v) => v.skipped);
        const allSuccess = req.verifications.every((v) => v.success || v.skipped);
        
        if (allSkipped) {
          verifyStatus = VERIFY_STATUS.SKIPPED.icon + " " + VERIFY_STATUS.SKIPPED.text;
          verifyClass = VERIFY_STATUS.SKIPPED.class;
        } else if (allSuccess) {
          verifyStatus = VERIFY_STATUS.SUCCESS.icon + " " + VERIFY_STATUS.SUCCESS.text;
          verifyClass = VERIFY_STATUS.SUCCESS.class;
        } else {
          verifyStatus = VERIFY_STATUS.FAILED.icon + " " + VERIFY_STATUS.FAILED.text;
          verifyClass = VERIFY_STATUS.FAILED.class;
        }
      }

      let html = '<tr style="cursor:pointer;" onclick="toggleDetails(\'' + detailsId + '\')">';
      html += "<td>" + (index + 1) + "</td>";
      
      // 根据运行模式决定是否显示GroupID和APIName
      if (window.reportData && window.reportData.run_mode != 'cli') {
        html += "<td>" + (req.group_id || "-") + "</td>";
        html += "<td>" + (req.api_name || "-") + "</td>";
      }
      
      html += "<td>" + (req.timestamp ? new Date(req.timestamp).toLocaleTimeString() : "-") + "</td>";
      html +=
        '<td style="max-width:300px;overflow:hidden;text-overflow:ellipsis;" title="' +
        (req.url || req.request_url || "") +
        '">' +
        (req.url || req.request_url || "") +
        "</td>";
      html += "<td>" + formatHttpMethod(req.method || req.request_method) + "</td>";
      html += "<td>" + ((req.duration ? req.duration / 1000000 : req.duration_ms) || 0).toFixed(2) + "ms</td>";
      html += "<td>" + (req.skipped ? '-' : (req.status_code || 0)) + "</td>";
      html +=
        '<td class="' +
        statusClass +
        '">' +
        statusText +
        "</td>";
      html += '<td class="' + verifyClass + '">' + verifyStatus + "</td>";
      html += "<td>" + formatBytes(req.size || 0) + "</td>";
      html +=
        '<td onclick="event.stopPropagation();"><button class="detail-btn" onclick="toggleDetails(\'' +
        detailsId +
        "')\">查看详情</button></td>";
      html += "</tr>";
      html +=
        '<tr id="' + detailsId + '" class="detail-row" style="display:none;">';
      html += '<td colspan="12"><div class="detail-content">';
      html += generateDetailContent(req);
      html += "</div></td></tr>";
      return html;
    })
    .join("");
}

// ============ Tab和筛选 ============
function switchTab(tab) {
  currentTab = tab;
  document
    .querySelectorAll(".tab-btn")
    .forEach((btn) => btn.classList.remove("active"));
  document.getElementById('tab-' + tab).classList.add("active");
  
  // 收起所有展开的详情行
  const detailRows = document.querySelectorAll('.detail-row');
  detailRows.forEach(row => {
    row.style.display = 'none';
    row.classList.remove('show');
  });
  
  // 清空实时模式的展开记录
  if (typeof openDetails !== 'undefined') {
    openDetails.clear();
  }
  
  // 重置到第一页
  currentPage = 1;
  
  // 实时模式：直接从服务器加载新Tab的数据
  if (isRealtime) {
    loadRealtimePageData();
  } else {
    // 静态模式：客户端筛选
    filterDetails();
  }
}

function filterDetails() {
  const searchValue = document.getElementById(ELEMENT_IDS.SEARCH_PATH).value.toLowerCase();
  const methodFilter = document.getElementById(ELEMENT_IDS.METHOD_FILTER).value;
  const statusFilter = document.getElementById(ELEMENT_IDS.STATUS_FILTER).value;
  const durationFilter = document.getElementById(ELEMENT_IDS.DURATION_FILTER).value;

  filteredData = allDetailsData;

  if (currentTab === TAB_NAMES.SUCCESS) {
    filteredData = filteredData.filter((d) => d.success && !d.skipped);
  } else if (currentTab === TAB_NAMES.FAILED) {
    filteredData = filteredData.filter((d) => !d.success && !d.skipped);
  } else if (currentTab === TAB_NAMES.SKIPPED) {
    filteredData = filteredData.filter((d) => d.skipped);
  }

  if (searchValue) {
    filteredData = filteredData.filter(
      (d) =>
        (d.url || "").toLowerCase().includes(searchValue) ||
        (d.request_url || "").toLowerCase().includes(searchValue)
    );
  }

  if (methodFilter) {
    filteredData = filteredData.filter(
      (d) => (d.method || d.request_method || "").toUpperCase() === methodFilter
    );
  }

  if (statusFilter) {
    filteredData = filteredData.filter((d) => {
      const code = d.status_code || 0;
      if (statusFilter === "2xx") return code >= 200 && code < 300;
      if (statusFilter === "3xx") return code >= 300 && code < 400;
      if (statusFilter === "4xx") return code >= 400 && code < 500;
      if (statusFilter === "5xx") return code >= 500 && code < 600;
      return true;
    });
  }

  if (durationFilter) {
    filteredData = filteredData.filter((d) => {
      const durationMs =
        d.duration_ms || (d.duration ? d.duration / 1000000 : 0);
      if (durationFilter === "<100") return durationMs < 100;
      if (durationFilter === "100-500")
        return durationMs >= 100 && durationMs < 500;
      if (durationFilter === "500-1000")
        return durationMs >= 500 && durationMs < 1000;
      if (durationFilter === ">1000") return durationMs >= 1000;
      return true;
    });
  }

  updateTabCounts();
  currentPage = 1;
  renderPage();
}

function clearFilters() {
  document.getElementById(ELEMENT_IDS.SEARCH_PATH).value = "";
  document.getElementById(ELEMENT_IDS.METHOD_FILTER).value = "";
  document.getElementById(ELEMENT_IDS.STATUS_FILTER).value = "";
  document.getElementById(ELEMENT_IDS.DURATION_FILTER).value = "";
  filterDetails();
}

function updateTabCounts() {
  // 实时模式使用服务器返回的真实统计数据，静态模式使用客户端数据
  if (isRealtime && window.realtimeStats) {
    // 使用从 /api/details 接口获取的真实统计数据
    document.getElementById(ELEMENT_IDS.COUNT_ALL).textContent = window.realtimeStats.total_requests || 0;
    document.getElementById(ELEMENT_IDS.COUNT_SUCCESS).textContent = window.realtimeStats.success_count || 0;
    document.getElementById(ELEMENT_IDS.COUNT_FAILED).textContent = window.realtimeStats.failed_count || 0;
    document.getElementById(ELEMENT_IDS.COUNT_SKIPPED).textContent = window.realtimeStats.skipped_count || 0;
  } else {
    // 静态模式使用客户端加载的详情数据计算
    const total = allDetailsData.length;
    const skipped = allDetailsData.filter((d) => d.skipped).length;
    const success = allDetailsData.filter((d) => d.success && !d.skipped).length;
    const failed = allDetailsData.filter((d) => !d.success && !d.skipped).length;

    document.getElementById(ELEMENT_IDS.COUNT_ALL).textContent = total;
    document.getElementById(ELEMENT_IDS.COUNT_SUCCESS).textContent = success;
    document.getElementById(ELEMENT_IDS.COUNT_FAILED).textContent = failed;
    document.getElementById(ELEMENT_IDS.COUNT_SKIPPED).textContent = skipped;
  }
}

// ============ 分页 ============
function renderPage() {
  // 实时模式：从服务器加载数据（支持真正的分页）
  if (isRealtime) {
    loadRealtimePageData();
  } else {
    // 静态模式：使用客户端内存分页
    const start = (currentPage - 1) * pageSize;
    const end = start + pageSize;
    const pageData = filteredData.slice(start, end);
    renderStaticDetails(pageData);
    updatePaginationControls();
    
    const paginationEl = document.getElementById(ELEMENT_IDS.PAGINATION);
    if (paginationEl && filteredData.length > pageSize) {
      paginationEl.style.display = "flex";
    } else if (paginationEl) {
      paginationEl.style.display = "none";
    }
  }
}

function updatePaginationControls() {
  // 实时模式使用服务器返回的真实总数和计数器总数
  let displayTotal;
  if (isRealtime && window.realtimeStats) {
    // 根据当前 tab 显示对应的总数
    switch (currentTab) {
      case TAB_NAMES.SUCCESS:
        displayTotal = window.realtimeStats.success_count || 0;
        break;
      case TAB_NAMES.FAILED:
        displayTotal = window.realtimeStats.failed_count || 0;
        break;
      case TAB_NAMES.SKIPPED:
        displayTotal = window.realtimeStats.skipped_count || 0;
        break;
      default:
        displayTotal = window.realtimeStats.total_requests || 0;
    }
  } else {
    // 静态模式使用客户端筛选后的总数
    displayTotal = filteredData.length;
  }
  
  const totalPages = Math.ceil(displayTotal / pageSize) || 1;

  document.getElementById(ELEMENT_IDS.CURRENT_PAGE).textContent = currentPage;
  document.getElementById(ELEMENT_IDS.TOTAL_PAGES).textContent = totalPages;
  document.getElementById(ELEMENT_IDS.TOTAL_RECORDS).textContent = displayTotal;

  document.getElementById(ELEMENT_IDS.FIRST_BTN).disabled = currentPage === 1;
  document.getElementById(ELEMENT_IDS.PREV_BTN).disabled = currentPage === 1;
  document.getElementById(ELEMENT_IDS.NEXT_BTN).disabled = currentPage >= totalPages;
  document.getElementById(ELEMENT_IDS.LAST_BTN).disabled = currentPage >= totalPages;
  
  // 显示分页控件
  const paginationEl = document.getElementById(ELEMENT_IDS.PAGINATION);
  if (paginationEl && displayTotal > pageSize) {
    paginationEl.style.display = "flex";
  } else if (paginationEl) {
    paginationEl.style.display = "none";
  }
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
  // 实时模式：使用服务器总数，静态模式：使用客户端筛选后的总数
  let displayTotal;
  if (isRealtime && window.realtimeStats) {
    switch (currentTab) {
      case TAB_NAMES.SUCCESS:
        displayTotal = window.realtimeStats.success_count || 0;
        break;
      case TAB_NAMES.FAILED:
        displayTotal = window.realtimeStats.failed_count || 0;
        break;
      case TAB_NAMES.SKIPPED:
        displayTotal = window.realtimeStats.skipped_count || 0;
        break;
      default:
        displayTotal = window.realtimeStats.total_requests || 0;
    }
  } else {
    displayTotal = filteredData.length;
  }
  
  const totalPages = Math.ceil(displayTotal / pageSize);
  if (currentPage < totalPages) {
    currentPage++;
    renderPage();
  }
}

function goToLastPage() {
  // 实时模式：使用服务器总数，静态模式：使用客户端筛选后的总数
  let displayTotal;
  if (isRealtime && window.realtimeStats) {
    switch (currentTab) {
      case TAB_NAMES.SUCCESS:
        displayTotal = window.realtimeStats.success_count || 0;
        break;
      case TAB_NAMES.FAILED:
        displayTotal = window.realtimeStats.failed_count || 0;
        break;
      case TAB_NAMES.SKIPPED:
        displayTotal = window.realtimeStats.skipped_count || 0;
        break;
      default:
        displayTotal = window.realtimeStats.total_requests || 0;
    }
  } else {
    displayTotal = filteredData.length;
  }
  
  const totalPages = Math.ceil(displayTotal / pageSize) || 1;
  currentPage = totalPages;
  renderPage();
}

function changePageSize() {
  pageSize = parseInt(document.getElementById(ELEMENT_IDS.PAGE_SIZE_SELECT).value);
  currentPage = 1;
  renderPage();
}

// ============ 工具函数 ============
function escapeHtml(text) {
  if (!text) return "";
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

function toggleDetails(detailsId) {
  const row = document.getElementById(detailsId);
  if (row) {
    row.style.display = row.style.display === "none" ? "table-row" : "none";
  }
}

function formatBytes(bytes) {
  if (bytes === 0) return "0B";
  const k = FORMAT_CONFIG.BYTES_UNIT;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return (bytes / Math.pow(k, i)).toFixed(FORMAT_CONFIG.DECIMAL_PLACES) + SIZE_UNITS[i];
}

function copyToClipboard(elementId, btnElement) {
  const element = document.getElementById(elementId);
  if (element) {
    const text = element.textContent;
    navigator.clipboard.writeText(text).then(() => {
      if (btnElement) {
        const originalText = btnElement.textContent;
        btnElement.textContent = '✓ 已复制';
        btnElement.style.background = '#38ef7d';
        setTimeout(() => {
          btnElement.textContent = originalText;
          btnElement.style.background = '#667eea';
        }, 2000);
      }
    }).catch(err => {
      console.error('复制失败:', err);
      alert('复制失败,请手动复制');
    });
  }
}

function formatCodeBlock(content, label) {
  if (!content) return '';
  
  const trimmed = content.trim();
  let isJson = false;
  let formatted = content;
  
  // 尝试解析为JSON
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      const parsed = JSON.parse(content);
      formatted = JSON.stringify(parsed);
      isJson = true;
    } catch (e) {
      // 不是有效的JSON,保持原样
    }
  }
  
  const blockId = 'code-' + Math.random().toString(36).substr(2, 9);
  return '<div class="detail-section"><strong>' + label + (isJson ? ' (JSON)' : '') + ':</strong>' +
    '<div style="position:relative;">' +
    '<button onclick="copyToClipboard(\'' + blockId + '\', this)" ' +
    'style="position:absolute;right:8px;top:8px;padding:6px 12px;background:#667eea;color:white;border:none;border-radius:4px;cursor:pointer;font-size:12px;z-index:10;">' +
    '📋 复制</button>' +
    '<pre id="' + blockId + '" style="background:#f8f9fa;padding:12px;border-radius:4px;border:1px solid #e9ecef;margin-top:8px;max-height:400px;overflow-y:auto;overflow-x:hidden;white-space:pre-wrap;word-wrap:break-word;word-break:break-all;">' +
    escapeHtml(formatted) +
    '</pre></div></div>';
}

function generateDetailContent(req) {
  const tabId = 'tab-' + Math.random().toString(36).substr(2, 9);
  const reqId = 'req-' + Math.random().toString(36).substr(2, 9);
  const menuId = 'menu-' + reqId;
  
  // 将请求数据保存到全局变量供按钮使用（在生成HTML之前）
  if (!window.requestDataStore) {
    window.requestDataStore = {};
  }
  window.requestDataStore[reqId] = req;
  
  let html = '<div class="detail-tabs-container">';
  
  // 跳过提示
  if (req.skipped) {
    html += '<div style="padding:12px;background:#fff3cd;border-left:4px solid #ffc107;color:#856404;margin-bottom:15px;border-radius:4px;">';
    html += '<strong>⚠️ 此请求已跳过:</strong> ' + escapeHtml(req.skip_reason || '依赖的API失败');
    html += '<div style="margin-top:5px;font-size:13px;">下方显示的是配置的请求信息和验证规则（未实际执行）</div>';
    html += '</div>';
  }
  
  // Tab按钮和更多操作在同一行
  html += '<div style="display:flex;justify-content:space-between;align-items:center;border-bottom:2px solid #e9ecef;background:white;margin-bottom:20px;">';
  
  // 左侧：Tab按钮
  html += '<div class="detail-tabs-header" style="border-bottom:none;flex:1;">';
  html += '<button class="detail-tab-btn active" onclick="switchDetailTab(event, \'' + tabId + '-url\')">请求信息</button>';
  
  if (req.headers || req.request_headers) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-headers\')">Headers</button>';
  }
  
  if (req.body || req.request_body) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-reqbody\')">请求Body</button>';
  }
  
  // 只有非跳过请求才显示响应Body
  if (!req.skipped && req.response_body) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-respbody\')">响应Body</button>';
  }
  
  if (req.extracted_vars && Object.keys(req.extracted_vars).length > 0) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-extracted\')" style="color:#667eea;">📦 提取变量</button>';
  }
  
  if (req.verifications && req.verifications.length > 0) {
    // 判断验证状态：跳过、全部通过、有失败
    const allSkipped = req.verifications.every(v => v.skipped);
    const allPassed = req.verifications.every(v => v.success || v.skipped);
    
    let statusConfig;
    if (allSkipped) {
      statusConfig = VERIFY_STATUS.SKIPPED;
    } else if (allPassed) {
      statusConfig = VERIFY_STATUS.SUCCESS;
    } else {
      statusConfig = VERIFY_STATUS.FAILED;
    }
    
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-verify\')" style="color:' + statusConfig.color + ';">' + 
      statusConfig.icon + ' ' + statusConfig.text + '</button>';
  }
  
  if (req.error) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-error\')">错误</button>';
  }
  
  html += '</div>'; // 结束 detail-tabs-header
  
  // 右侧：更多操作下拉菜单
  html += '<div class="action-dropdown" style="margin:0 10px;">';
  html += '  <button class="action-dropdown-btn" onclick="toggleActionMenu(\''+menuId+'\', event)">';
  html += '    <span>⚙️</span> 更多操作 <span style="margin-left:auto;">▼</span>';
  html += '  </button>';
  html += '  <div id="'+menuId+'" class="action-dropdown-menu">';
  html += '    <div class="action-dropdown-menu-item" onclick="copyAs(window.requestDataStore[\''+reqId+'\'], \'full-request\', this)" style="font-weight:600;color:#667eea;">';
  html += '      <span>📄</span> 复制完整请求';
  html += '    </div>';
  html += '    <div class="action-menu-section">复制为代码</div>';
  html += '    <div class="action-dropdown-menu-item" onclick="copyAs(window.requestDataStore[\''+reqId+'\'], \'go-stress\', this)" style="font-weight:600;color:#10b981;">';
  html += '      <span>🚀</span> go-stress';
  html += '    </div>';
  html += '    <div class="action-dropdown-menu-item" onclick="copyAs(window.requestDataStore[\''+reqId+'\'], \'curl-bash\', this)">';
  html += '      <span>📋</span> curl (bash)';
  html += '    </div>';
  html += '    <div class="action-dropdown-menu-item" onclick="copyAs(window.requestDataStore[\''+reqId+'\'], \'curl-cmd\', this)">';
  html += '      <span>📋</span> curl (cmd)';
  html += '    </div>';
  html += '    <div class="action-dropdown-menu-item" onclick="copyAs(window.requestDataStore[\''+reqId+'\'], \'powershell\', this)">';
  html += '      <span>💻</span> PowerShell';
  html += '    </div>';
  html += '    <div class="action-menu-section">操作</div>';
  html += '    <div class="action-dropdown-menu-item" onclick="replayRequest(window.requestDataStore[\''+reqId+'\'], this)">';
  html += '      <span>🔄</span> 重放请求';
  html += '    </div>';
  html += '  </div>';
  html += '</div>';
  
  html += '</div>'; // 结束 flex 容器
  
  // Tab内容
  html += '<div class="detail-tabs-content">';
  
  // 请求信息Tab
  html += '<div id="' + tabId + '-url" class="detail-tab-content active">';
  html += '<div class="detail-section"><strong>请求URL:</strong><pre>' + escapeHtml(req.url || req.request_url || "") + '</pre></div>';
  if (req.query || req.request_query) {
    html += '<div class="detail-section"><strong>请求Query:</strong><pre>' + escapeHtml(req.query || req.request_query) + '</pre></div>';
  }
  html += '<div class="detail-section"><strong>请求方法:</strong> ' + formatHttpMethod(req.method || req.request_method) + '</div>';
  html += '<div class="detail-section"><strong>响应时间:</strong><pre>' + ((req.duration ? req.duration / 1000000 : req.duration_ms) || 0).toFixed(2) + 'ms</pre></div>';
  html += '<div class="detail-section"><strong>状态码:</strong><pre>' + (req.status_code || 0) + '</pre></div>';
  html += '</div>';
  
  // Headers Tab
  if (req.headers || req.request_headers) {
    const headers = req.headers || req.request_headers;
    const headerId = 'headers-' + Math.random().toString(36).substr(2, 9);
    let headersText = '';
    
    if (typeof headers === 'object' && headers !== null) {
      headersText = Object.entries(headers)
        .map(([key, value]) => `${key}: ${value}`)
        .join('\n');
    } else {
      headersText = headers;
    }
    
    html += '<div id="' + tabId + '-headers" class="detail-tab-content">';
    html += '<div style="position:relative;">';
    html += '<button onclick="copyToClipboard(\'' + headerId + '\', this)" ' +
      'style="position:absolute;right:8px;top:8px;padding:6px 12px;background:#667eea;color:white;border:none;border-radius:4px;cursor:pointer;font-size:12px;z-index:10;">' +
      '📋 复制</button>';
    html += '<pre id="' + headerId + '" style="background:#f8f9fa;padding:12px;border-radius:4px;border:1px solid #e9ecef;margin-top:8px;max-height:500px;overflow-y:auto;overflow-x:hidden;white-space:pre-wrap;word-wrap:break-word;word-break:break-all;">' +
      escapeHtml(headersText) + '</pre></div></div>';
  }
  
  // 请求Body Tab
  if (req.body || req.request_body) {
    html += '<div id="' + tabId + '-reqbody" class="detail-tab-content">';
    html += formatCodeBlock(req.body || req.request_body, '请求Body').replace('<div class="detail-section"><strong>请求Body', '<div style="margin:0"><strong>请求Body');
    html += '</div>';
  }
  
  // 响应Body Tab
  if (req.response_body) {
    const responseBody = req.response_body;
    const trimmed = responseBody.trim();
    const isHtml = trimmed.toLowerCase().startsWith('<!doctype html') || trimmed.toLowerCase().startsWith('<html');
    
    html += '<div id="' + tabId + '-respbody" class="detail-tab-content">';
    
    if (isHtml) {
      const htmlId = 'html-' + Math.random().toString(36).substr(2, 9);
      html += '<iframe srcdoc="' + escapeHtml(responseBody).replace(/"/g, '&quot;') + '" ' +
        'style="width:100%;height:450px;border:1px solid #ddd;border-radius:4px;background:white;margin-bottom:10px;"></iframe>';
      html += '<details><summary style="cursor:pointer;color:#667eea;user-select:none;">📄 查看HTML源码</summary>';
      html += '<div style="position:relative;margin-top:10px;">';
      html += '<button onclick="copyToClipboard(\'' + htmlId + '\', this)" ' +
        'style="position:absolute;right:8px;top:8px;padding:6px 12px;background:#667eea;color:white;border:none;border-radius:4px;cursor:pointer;font-size:12px;z-index:10;">' +
        '📋 复制</button>';
      html += '<pre id="' + htmlId + '" style="background:#f8f9fa;padding:12px;border-radius:4px;border:1px solid #e9ecef;max-height:400px;overflow:auto;">' +
        escapeHtml(responseBody) + '</pre></div></details>';
    } else {
      html += formatCodeBlock(responseBody, '响应Body').replace('<div class="detail-section"><strong>响应Body', '<div style="margin:0"><strong>响应Body');
    }
    
    html += '</div>';
  }
  
  // 提取变量Tab
  if (req.extracted_vars && Object.keys(req.extracted_vars).length > 0) {
    html += '<div id="' + tabId + '-extracted" class="detail-tab-content">';
    html += '<div style="background:#f8f9fa;padding:15px;border-radius:8px;">';
    html += '<div style="margin-bottom:15px;color:#667eea;font-weight:bold;font-size:14px;">📦 提取的变量 (' + Object.keys(req.extracted_vars).length + ' 个)</div>';
    
    Object.entries(req.extracted_vars).forEach(([key, value]) => {
      html += '<div style="background:white;padding:12px;border-radius:6px;margin-bottom:10px;border:1px solid #e9ecef;">';
      html += '<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px;">';
      html += '<strong style="color:#667eea;font-size:13px;">🔑 ' + escapeHtml(key) + '</strong>';
      html += '<button onclick="copyToClipboard(\'extracted-' + tabId + '-' + key.replace(/[^a-zA-Z0-9]/g, '_') + '\', this)" ' +
        'style="padding:4px 10px;background:#667eea;color:white;border:none;border-radius:3px;cursor:pointer;font-size:11px;">复制</button>';
      html += '</div>';
      html += '<pre id="extracted-' + tabId + '-' + key.replace(/[^a-zA-Z0-9]/g, '_') + '" style="background:#f8f9fa;padding:10px;border-radius:4px;margin:0;font-size:13px;word-break:break-all;white-space:pre-wrap;">' +
        escapeHtml(String(value)) + '</pre>';
      html += '</div>';
    });
    
    html += '</div></div>';
  }
  
  // 验证结果Tab
  if (req.verifications && req.verifications.length > 0) {
    html += '<div id="' + tabId + '-verify" class="detail-tab-content">';
    html += '<div style="background:#f8f9fa;padding:15px;border-radius:8px;">';
    
    req.verifications.forEach((verify, idx) => {
      // 根据验证状态获取样式配置
      let statusConfig;
      if (verify.skipped) {
        statusConfig = VERIFY_STATUS.SKIPPED;
      } else if (verify.success) {
        statusConfig = VERIFY_STATUS.SUCCESS;
      } else {
        statusConfig = VERIFY_STATUS.FAILED;
      }
      
      html += '<div style="background:white;padding:15px;border-radius:8px;margin-bottom:10px;border:2px solid ' + statusConfig.border + ';">';
      html += '<div style="display:flex;align-items:center;gap:10px;margin-bottom:10px;">';
      html += '<span style="font-size:20px;">' + statusConfig.icon + '</span>';
      html += '<strong style="color:' + statusConfig.color + ';">' + statusConfig.text + '</strong>';
      html += '</div>';
      
      if (verify.description) {
        html += '<div style="margin-bottom:8px;"><strong>📝 描述:</strong> ' + escapeHtml(verify.description) + '</div>';
      }
      
      if (verify.type) {
        html += '<div style="margin-bottom:8px;"><strong>🔍 验证类型:</strong> ' + escapeHtml(verify.type) + '</div>';
      }
      
      if (verify.field) {
        let fieldLabel = '字段';
        if (verify.type === 'JSONPATH') {
          fieldLabel = 'JSONPath';
        } else if (verify.type === 'HEADER') {
          fieldLabel = 'Header';
        } else if (verify.type === 'REGEX') {
          fieldLabel = '正则表达式';
        }
        html += '<div style="margin-bottom:8px;"><strong>📍 ' + fieldLabel + ':</strong> <code style="background:#f8f9fa;padding:2px 6px;border-radius:3px;">' + 
          escapeHtml(verify.field) + '</code></div>';
      }
      
      if (verify.operator) {
        const operatorMap = {
          'eq': '等于 (=)',
          'ne': '不等于 (≠)',
          'gt': '大于 (>)',
          'lt': '小于 (<)',
          'gte': '大于等于 (≥)',
          'lte': '小于等于 (≤)',
          'contains': '包含',
          'regex': '正则匹配',
          'hasPrefix': '前缀匹配',
          'hasSuffix': '后缀匹配'
        };
        const operatorText = operatorMap[verify.operator] || verify.operator;
        html += '<div style="margin-bottom:8px;"><strong>⚙️ 操作符:</strong> ' + escapeHtml(operatorText) + '</div>';
      }
      
      if (verify.expected !== undefined && verify.expected !== null) {
        html += '<div style="margin-bottom:8px;"><strong>✓ 期望值:</strong> <code style="background:#f8f9fa;padding:2px 6px;border-radius:3px;">' + 
          escapeHtml(String(verify.expected)) + '</code></div>';
      }
      
      if (verify.actual !== undefined && verify.actual !== null) {
        html += '<div style="margin-bottom:8px;"><strong>📊 实际值:</strong> <code style="background:#f8f9fa;padding:2px 6px;border-radius:3px;">' + 
          escapeHtml(String(verify.actual)) + '</code></div>';
      }
      
      if (verify.message) {
        html += '<div style="margin-top:10px;padding:10px;background:' + statusConfig.bg + ';border-radius:4px;color:' + statusConfig.color + ';">' + 
          escapeHtml(verify.message) + '</div>';
      }
      
      html += '</div>';
    });
    
    html += '</div></div>';
  }
  
  // 错误Tab
  if (req.error) {
    html += '<div id="' + tabId + '-error" class="detail-tab-content">';
    html += '<pre style="color:red;background:#fff5f5;padding:12px;border-radius:4px;border:1px solid #feb2b2;white-space:pre-wrap;word-wrap:break-word;">' +
      escapeHtml(req.error) + '</pre>';
    html += '</div>';
  }
  
  html += '</div></div>';
  
  return html;
}

window.switchDetailTab = function(event, tabId) {
  const btn = event.currentTarget;
  const container = btn.closest('.detail-tabs-container');
  
  if (!container) {
    console.error('未找到容器元素');
    return;
  }
  
  // 移除所有active类
  const allBtns = container.querySelectorAll('.detail-tab-btn');
  const allContents = container.querySelectorAll('.detail-tab-content');
  
  console.log('找到按钮数:', allBtns.length, '找到内容数:', allContents.length);
  
  allBtns.forEach(b => b.classList.remove('active'));
  allContents.forEach(c => c.classList.remove('active'));
  
  // 添加active类
  btn.classList.add('active');
  const tabContent = document.getElementById(tabId);
  if (tabContent) {
    tabContent.classList.add('active');
    console.log('激活标签:', tabId);
  } else {
    console.error('未找到标签内容元素:', tabId);
  }
};

// ============ 实时模式专用函数 ============
if (isRealtime) {
  // 实时模式 - 更新指标
  window.updateMetrics = function (data) {
    document.getElementById(ELEMENT_IDS.TOTAL_REQUESTS).textContent = data.total_requests || 0;
    document.getElementById(ELEMENT_IDS.SUCCESS_REQUESTS).textContent =
      data.success_requests || 0;
    document.getElementById(ELEMENT_IDS.FAILED_REQUESTS).textContent =
      data.failed_requests || 0;
    document.getElementById(ELEMENT_IDS.SKIPPED_REQUESTS).textContent =
      data.skipped_requests || 0;
    document.getElementById(ELEMENT_IDS.SUCCESS_RATE).textContent =
      (data.success_rate || 0).toFixed(2) + "%";
    document.getElementById(ELEMENT_IDS.QPS).textContent = (data.qps || 0).toFixed(2);
    document.getElementById(ELEMENT_IDS.AVG_DURATION).textContent =
      (data.avg_latency || 0).toFixed(2) + "ms";
    
    // 响应时间统计
    document.getElementById(ELEMENT_IDS.MIN_DURATION).textContent =
      (data.min_latency || 0).toFixed(2) + "ms";
    document.getElementById(ELEMENT_IDS.MAX_DURATION).textContent =
      (data.max_latency || 0).toFixed(2) + "ms";
    
    // 百分位统计
    document.getElementById(ELEMENT_IDS.P50).textContent = (data.p50_latency || 0).toFixed(2) + "ms";
    document.getElementById(ELEMENT_IDS.P90).textContent = (data.p90_latency || 0).toFixed(2) + "ms";
    document.getElementById(ELEMENT_IDS.P95).textContent = (data.p95_latency || 0).toFixed(2) + "ms";
    document.getElementById(ELEMENT_IDS.P99).textContent = (data.p99_latency || 0).toFixed(2) + "ms";
    
    document.getElementById(ELEMENT_IDS.ELAPSED).textContent = (data.elapsed_seconds || 0) + "s";
    
    // 检查任务状态并更新按钮
    const pauseBtn = document.getElementById(ELEMENT_IDS.PAUSE_BTN);
    const stopBtn = document.getElementById(ELEMENT_IDS.STOP_BTN);
    const statusText = document.getElementById(ELEMENT_IDS.STATUS_TEXT);
    const statusDot = document.getElementById(ELEMENT_IDS.STATUS_DOT);
    
    if (data.is_completed) {
      // 任务已完成 - 隐藏控制按钮
      if (pauseBtn) pauseBtn.style.display = 'none';
      if (stopBtn) stopBtn.style.display = 'none';
      if (statusText) statusText.textContent = '已完成';
      if (statusDot) {
        statusDot.style.background = '#28a745';
        statusDot.style.animation = 'none';
      }
    } else if (data.is_stopped) {
      // 已停止 - 隐藏控制按钮
      if (pauseBtn) pauseBtn.style.display = 'none';
      if (stopBtn) stopBtn.style.display = 'none';
      if (statusText) statusText.textContent = '已停止';
      if (statusDot) {
        statusDot.style.background = '#dc3545';
        statusDot.style.animation = 'none';
      }
    } else if (data.is_paused) {
      // 已暂停
      if (pauseBtn) {
        pauseBtn.textContent = '▶ 恢复';
        pauseBtn.style.background = '#28a745';
        pauseBtn.style.color = 'white';
      }
      if (statusText) statusText.textContent = '已暂停';
      if (statusDot) {
        statusDot.style.background = '#ffc107';
        statusDot.style.animation = 'none';
      }
      isPaused = true;
    } else {
      // 运行中
      if (pauseBtn && !pauseBtn.disabled) {
        pauseBtn.textContent = '⏸ 暂停';
        pauseBtn.style.background = '#ffc107';
        pauseBtn.style.color = '#333';
      }
      if (statusText) statusText.textContent = '实时监控中';
      if (statusDot) {
        statusDot.style.background = '#38ef7d';
        statusDot.style.animation = 'pulse 2s infinite';
      }
      isPaused = false;
    }
  };

  // 更新实时图表
  window.updateCharts = function (data) {
    if (data.recent_durations && data.recent_durations.length > 0 && durationChart) {
      const indices = data.recent_durations.map((_, i) => i + 1);
      durationChart.setOption({
        xAxis: { data: indices },
        series: [{ data: data.recent_durations }],
      });
    }

    if (data.status_codes && statusChart) {
      const codes = Object.keys(data.status_codes).sort();
      const values = codes.map((code) => data.status_codes[code]);
      statusChart.setOption({
        xAxis: { data: codes },
        series: [
          {
            data: values.map((v, i) => ({
              value: v,
              itemStyle: {
                color: codes[i].startsWith("2")
                  ? "#38ef7d"
                  : codes[i].startsWith("4")
                  ? "#f45c43"
                  : codes[i].startsWith("5")
                  ? "#eb3349"
                  : "#667eea",
              },
            })),
          },
        ],
      });
    }

    if (data.errors && errorChart) {
      const errors = Object.entries(data.errors)
        .map(([name, value]) => ({
          name: name.substring(0, 30) + (name.length > 30 ? "..." : ""),
          value: value,
        }))
        .slice(0, 5);
      errorChart.setOption({
        series: [{ data: errors }],
      });
    }
  };

  let lastDetailsCount = 0;
  const openDetails = new Set();

  // 实时模式：从服务器加载分页数据
  window.loadRealtimePageData = function() {
    const offset = (currentPage - 1) * pageSize;
    const status = currentTab === TAB_NAMES.ALL ? TAB_NAMES.ALL : currentTab; // all/success/failed/skipped
    
    // 构建查询参数
    const params = new URLSearchParams({
      status: status,
      offset: offset,
      limit: pageSize
    });
    
    // 如果有 slave_id，添加到查询参数
    if (slaveId) {
      params.append('slave_id', slaveId);
    }
    
    // 构建 API 地址：优先使用 realtime_url（分布式模式），否则使用相对路径（单机模式）
    const apiUrl = realtimeUrl 
      ? `${realtimeUrl}/api/details?${params.toString()}`
      : `/api/details?${params.toString()}`;
    
    fetch(apiUrl)
      .then((res) => res.json())
      .then((data) => {
        // 更新全局统计计数器（用于Tab标签显示和分页计算）
        if (!window.realtimeStats) {
          window.realtimeStats = {};
        }
        window.realtimeStats.total_requests = data.total_requests || 0;
        window.realtimeStats.success_count = data.success_count || 0;
        window.realtimeStats.failed_count = data.failed_count || 0;
        window.realtimeStats.skipped_count = data.skipped_count || 0;
        
        // 保存当前页的详细数据，供"查看详情"按钮使用
        const details = data.details || [];
        allDetailsData = details;
        
        // 渲染当前页数据
        renderRealtimeDetails(details);
        
        // 更新分页控件
        updatePaginationControls();
        
        // 更新Tab标签的计数显示
        updateTabCounts();
      })
      .catch((err) => {
        console.error('加载分页数据失败:', err);
      });
  };

  // 加载详情数据（初始加载时使用）
  window.loadDetails = function () {
    loadRealtimePageData();
  };

  // 渲染实时明细
  window.renderRealtimeDetails = function (details) {
    const tbody = document.getElementById("details-tbody");
    tbody.innerHTML = "";

    if (details && details.length > 0) {
      details.forEach((detail, idx) => {
        const row = tbody.insertRow();
        row.style.cursor = "pointer";
        row.onclick = () => toggleRealtimeDetail(idx);
        
        // 验证状态：考虑跳过、成功、失败三种情况
        let verifyStatus = "-";
        let verifyClass = "";
        if (detail.verifications && detail.verifications.length > 0) {
          const allSkipped = detail.verifications.every((v) => v.skipped);
          const allSuccess = detail.verifications.every((v) => v.success || v.skipped);
          
          if (allSkipped) {
            verifyStatus = VERIFY_STATUS.SKIPPED.icon + " " + VERIFY_STATUS.SKIPPED.text;
            verifyClass = VERIFY_STATUS.SKIPPED.class;
          } else if (allSuccess) {
            verifyStatus = VERIFY_STATUS.SUCCESS.icon + " " + VERIFY_STATUS.SUCCESS.text;
            verifyClass = VERIFY_STATUS.SUCCESS.class;
          } else {
            verifyStatus = VERIFY_STATUS.FAILED.icon + " " + VERIFY_STATUS.FAILED.text;
            verifyClass = VERIFY_STATUS.FAILED.class;
          }
        }

        row.innerHTML = `
                    <td>${detail.id}</td>
                    <td>${detail.group_id || '-'}</td>
                    <td>${detail.api_name || '-'}</td>
                    <td>${new Date(detail.timestamp).toLocaleTimeString()}</td>
                    <td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="${
                      detail.url || "-"
                    }">${detail.url || "-"}</td>
                    <td>${formatHttpMethod(detail.method)}</td>
                    <td>${(detail.duration / 1000000).toFixed(2)}ms</td>
                    <td>${detail.skipped ? '-' : (detail.status_code || "-")}</td>
                    <td class="${
                      detail.skipped ? "status-warning" : (detail.success ? "status-success" : "status-error")
                    }">${detail.skipped ? "⏭ 跳过" : (detail.success ? "✓ 成功" : "✗ 失败")}</td>
                    <td class="${verifyClass}">${verifyStatus}</td>
                    <td>${formatBytes(detail.size)}</td>
                    <td onclick="event.stopPropagation();"><button type="button" class="detail-btn" onclick="toggleRealtimeDetail(${idx})">查看详情</button></td>
                `;

        const detailRow = tbody.insertRow();
        detailRow.className = "detail-row";
        detailRow.id = "realtime-detail-" + idx;
        detailRow.innerHTML =
          '<td colspan="12"><div class="detail-content">明细内容...</div></td>';
      });
    } else {
      tbody.innerHTML =
        '<tr><td colspan="12" style="text-align:center;padding:40px;">暂无数据</td></tr>';
    }
  };

  window.toggleRealtimeDetail = function (idx) {
    const detailRow = document.getElementById("realtime-detail-" + idx);
    if (detailRow) {
      const wasOpen = detailRow.classList.contains("show");
      detailRow.classList.toggle("show");

      if (!wasOpen) {
        openDetails.add(idx);
        // 加载详细内容
        if (allDetailsData && allDetailsData[idx]) {
          const detail = allDetailsData[idx];
          const detailContent = detailRow.querySelector('.detail-content');
          detailContent.innerHTML = generateDetailContent(detail);
        }
      } else {
        openDetails.delete(idx);
      }
    }
  };

  // SSE连接
  window.connectSSE = function () {
    const eventSource = new EventSource("/stream");
    let lastTotalRequests = 0;

    eventSource.onmessage = function (event) {
      const data = JSON.parse(event.data);
      updateMetrics(data);
      updateCharts(data);
      
      // 只有当总请求数变化时才重新加载数据
      if (data.total_requests !== lastTotalRequests) {
        lastTotalRequests = data.total_requests;
        // 只在第一页时才自动刷新
        if (currentPage === 1) {
          loadDetails();
        }
      }
    };

    eventSource.onerror = function () {
      console.error("SSE连接错误,5秒后重连...");
      eventSource.close();
      setTimeout(connectSSE, 5000);
    };
  };

  // 实时模式初始化
  document.addEventListener("DOMContentLoaded", function () {
    initCharts();
    connectSSE();
    loadDetails();
    updateTabCounts();
  });
} else {
  // 静态模式初始化
  document.addEventListener("DOMContentLoaded", function () {
    initCharts();
    updateTabCounts();
    autoLoadJSON();
  });
}
