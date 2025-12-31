// Go-Stress 报告脚本
console.log('========== 报告脚本已加载 ==========');
console.log('当前时间:', new Date().toLocaleString());

let durationChart, statusChart, errorChart;
const isRealtime = (typeof IS_REALTIME_PLACEHOLDER !== 'undefined' && IS_REALTIME_PLACEHOLDER) || false;
const jsonFilename = "JSON_FILENAME_PLACEHOLDER" || "index.json";

console.log('运行模式:', isRealtime ? '实时模式' : '静态模式');

// 全局变量存储所有数据
let allDetailsData = [];
let currentTab = "all";
let currentPage = 1;
let pageSize = 20;
let filteredData = [];
let isPaused = false;

// ============ 全局函数（供HTML内联调用） ============
// 控制函数（暂停/恢复/停止）
window.togglePause = function() {
  const endpoint = isPaused ? '/api/resume' : '/api/pause';
  const pauseBtn = document.getElementById('pauseBtn');
  const statusText = document.getElementById('statusText');
  const statusDot = document.getElementById('statusDot');
  
  fetch(endpoint, { method: 'POST' })
    .then(response => response.json())
    .then(data => {
      if (data.success) {
        isPaused = !isPaused;
        if (isPaused) {
          pauseBtn.textContent = '▶ 恢复';
          pauseBtn.style.background = '#28a745';
          pauseBtn.style.color = 'white';
          statusText.textContent = '已暂停';
          statusDot.style.background = '#ffc107';
          statusDot.style.animation = 'none';
        } else {
          pauseBtn.textContent = '⏸ 暂停';
          pauseBtn.style.background = '#ffc107';
          pauseBtn.style.color = '#333';
          statusText.textContent = '实时监控中';
          statusDot.style.background = '#38ef7d';
          statusDot.style.animation = 'pulse 2s infinite';
        }
      }
    })
    .catch(err => console.error('控制操作失败:', err));
};

window.stopMonitoring = function() {
  if (!confirm('确定要停止压测吗？停止后将无法恢复！')) {
    return;
  }
  
  const stopBtn = document.getElementById('stopBtn');
  const pauseBtn = document.getElementById('pauseBtn');
  const statusText = document.getElementById('statusText');
  const statusDot = document.getElementById('statusDot');
  
  fetch('/api/stop', { method: 'POST' })
    .then(response => response.json())
    .then(data => {
      if (data.success) {
        stopBtn.disabled = true;
        pauseBtn.disabled = true;
        stopBtn.style.opacity = '0.5';
        pauseBtn.style.opacity = '0.5';
        statusText.textContent = '已停止';
        statusDot.style.background = '#dc3545';
        statusDot.style.animation = 'none';
        alert('压测已停止！');
      }
    })
    .catch(err => console.error('停止失败:', err));
};

// 处理文件选择
window.handleFileSelect = function (event) {
  const file = event.target.files[0];
  if (!file) return;

  const fileNameElem = document.getElementById("fileName");
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
      document.getElementById("fileLoader").style.display = "none";
      document.getElementById("infoBar").style.display = "flex";

      updateStaticMetrics(data);
      updateChartsFromData(data);

      allDetailsData = data.all_details || data.request_details || [];
      updateTabCounts();
      filterDetails();

      console.log("数据加载成功:", data);
    } catch (error) {
      console.error("数据处理错误:", error);
      alert("数据处理失败: " + error.message);
      document.getElementById("fileName").textContent =
        "加载失败: " + error.message;
    }
  };

  // 自动加载JSON文件
  window.autoLoadJSON = function () {
    const jsonUrl = jsonFilename;
    document.getElementById("fileName").textContent =
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
        document.getElementById("fileName").textContent =
          "⚠️ 自动加载失败,请手动选择JSON文件";
      });
  };
}

// ============ 图表初始化 ============
function initCharts() {
  const durationChartDom = document.getElementById("durationChart");
  const statusChartDom = document.getElementById("statusChart");
  const errorChartDom = document.getElementById("errorChart");

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

  setTextContent("generate-time", data.generate_time || new Date().toLocaleString('zh-CN'));
  setTextContent("test-duration", data.test_duration || (data.total_time_ms ? data.total_time_ms + 'ms' : '-'));
  setTextContent("static-total-requests", data.total_requests || 0);
  setTextContent(
    "static-success-rate",
    (data.success_rate || 0).toFixed(2) + "%"
  );
  setTextContent("static-qps", (data.qps || 0).toFixed(2));
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
  const tbody = document.getElementById("details-tbody");
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
      const verifyStatus =
        req.verifications && req.verifications.length > 0
          ? req.verifications.every((v) => v.success)
            ? "✓ 通过"
            : "✗ 失败"
          : "-";
      const verifyClass =
        req.verifications && req.verifications.length > 0
          ? req.verifications.every((v) => v.success)
            ? "status-success"
            : "status-error"
          : "";

      let html = '<tr style="cursor:pointer;" onclick="toggleDetails(\'' + detailsId + '\')">';
      html += "<td>" + (index + 1) + "</td>";
      html += "<td>" + (req.group_id || "-") + "</td>";
      html += "<td>" + (req.api_name || "-") + "</td>";
      html += "<td>" + (req.timestamp ? new Date(req.timestamp).toLocaleTimeString() : "-") + "</td>";
      html +=
        '<td style="max-width:300px;overflow:hidden;text-overflow:ellipsis;" title="' +
        (req.url || req.request_url || "") +
        '">' +
        (req.url || req.request_url || "") +
        "</td>";
      html += "<td>" + (req.method || req.request_method || "") + "</td>";
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
  document.getElementById("tab-" + tab).classList.add("active");
  
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
  
  filterDetails();
}

function filterDetails() {
  const searchValue = document.getElementById("searchPath").value.toLowerCase();
  const methodFilter = document.getElementById("methodFilter").value;
  const statusFilter = document.getElementById("statusFilter").value;
  const durationFilter = document.getElementById("durationFilter").value;

  filteredData = allDetailsData;

  if (currentTab === "success") {
    filteredData = filteredData.filter((d) => d.success && !d.skipped);
  } else if (currentTab === "failed") {
    filteredData = filteredData.filter((d) => !d.success && !d.skipped);
  } else if (currentTab === "skipped") {
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
  document.getElementById("searchPath").value = "";
  document.getElementById("methodFilter").value = "";
  document.getElementById("statusFilter").value = "";
  document.getElementById("durationFilter").value = "";
  filterDetails();
}

function updateTabCounts() {
  const total = allDetailsData.length;
  const skipped = allDetailsData.filter((d) => d.skipped).length;
  const success = allDetailsData.filter((d) => d.success && !d.skipped).length;
  const failed = allDetailsData.filter((d) => !d.success && !d.skipped).length;

  document.getElementById("count-all").textContent = total;
  document.getElementById("count-success").textContent = success;
  document.getElementById("count-failed").textContent = failed;
  document.getElementById("count-skipped").textContent = skipped;
}

// ============ 分页 ============
function renderPage() {
  const start = (currentPage - 1) * pageSize;
  const end = start + pageSize;
  const pageData = filteredData.slice(start, end);

  if (isRealtime) {
    renderRealtimeDetails(pageData);
  } else {
    renderStaticDetails(pageData);
  }

  updatePaginationControls();

  const paginationEl = document.getElementById("pagination");
  if (paginationEl && filteredData.length > pageSize) {
    paginationEl.style.display = "flex";
  } else if (paginationEl) {
    paginationEl.style.display = "none";
  }
}

function updatePaginationControls() {
  const totalPages = Math.ceil(filteredData.length / pageSize) || 1;

  document.getElementById("currentPage").textContent = currentPage;
  document.getElementById("totalPages").textContent = totalPages;
  document.getElementById("totalRecords").textContent = filteredData.length;

  document.getElementById("firstBtn").disabled = currentPage === 1;
  document.getElementById("prevBtn").disabled = currentPage === 1;
  document.getElementById("nextBtn").disabled = currentPage >= totalPages;
  document.getElementById("lastBtn").disabled = currentPage >= totalPages;
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
  pageSize = parseInt(document.getElementById("pageSizeSelect").value);
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
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return (bytes / Math.pow(k, i)).toFixed(2) + sizes[i];
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
  let html = '<div class="detail-tabs-container">';
  
  // Tab按钮
  html += '<div class="detail-tabs-header">';
  html += '<button class="detail-tab-btn active" onclick="switchDetailTab(event, \'' + tabId + '-url\')">请求信息</button>';
  
  if (req.headers || req.request_headers) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-headers\')">Headers</button>';
  }
  
  if (req.body || req.request_body) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-reqbody\')">请求Body</button>';
  }
  
  if (req.response_body) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-respbody\')">响应Body</button>';
  }
  
  if (req.extracted_vars && Object.keys(req.extracted_vars).length > 0) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-extracted\')" style="color:#667eea;">📦 提取变量</button>';
  }
  
  if (req.verifications && req.verifications.length > 0) {
    const allPassed = req.verifications.every(v => v.success);
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-verify\')" style="color:' + (allPassed ? '#38ef7d' : '#f45c43') + ';">' + 
      (allPassed ? '✓ ' : '✗ ') + '验证结果</button>';
  }
  
  if (req.error) {
    html += '<button class="detail-tab-btn" onclick="switchDetailTab(event, \'' + tabId + '-error\')">错误</button>';
  }
  
  html += '</div>';
  
  // Tab内容
  html += '<div class="detail-tabs-content">';
  
  // 请求信息Tab
  html += '<div id="' + tabId + '-url" class="detail-tab-content active">';
  html += '<div class="detail-section"><strong>请求URL:</strong><pre>' + escapeHtml(req.url || req.request_url || "") + '</pre></div>';
  if (req.query || req.request_query) {
    html += '<div class="detail-section"><strong>请求Query:</strong><pre>' + escapeHtml(req.query || req.request_query) + '</pre></div>';
  }
  html += '<div class="detail-section"><strong>请求方法:</strong><pre>' + escapeHtml(req.method || req.request_method || "") + '</pre></div>';
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
      const statusColor = verify.success ? '#38ef7d' : '#f45c43';
      const statusBg = verify.success ? '#f0fdf4' : '#fff5f5';
      const statusBorder = verify.success ? '#86efac' : '#feb2b2';
      
      html += '<div style="background:white;padding:15px;border-radius:8px;margin-bottom:10px;border:2px solid ' + statusBorder + ';">';
      html += '<div style="display:flex;align-items:center;gap:10px;margin-bottom:10px;">';
      html += '<span style="font-size:20px;">' + (verify.success ? '✓' : '✗') + '</span>';
      html += '<strong style="color:' + statusColor + ';">' + (verify.success ? '验证通过' : '验证失败') + '</strong>';
      html += '</div>';
      
      if (verify.name) {
        html += '<div style="margin-bottom:8px;"><strong>验证名称:</strong> ' + escapeHtml(verify.name) + '</div>';
      }
      
      if (verify.type) {
        html += '<div style="margin-bottom:8px;"><strong>验证类型:</strong> ' + escapeHtml(verify.type) + '</div>';
      }
      
      if (verify.expected !== undefined) {
        html += '<div style="margin-bottom:8px;"><strong>期望值:</strong> <code style="background:#f8f9fa;padding:2px 6px;border-radius:3px;">' + 
          escapeHtml(String(verify.expected)) + '</code></div>';
      }
      
      if (verify.actual !== undefined) {
        html += '<div style="margin-bottom:8px;"><strong>实际值:</strong> <code style="background:#f8f9fa;padding:2px 6px;border-radius:3px;">' + 
          escapeHtml(String(verify.actual)) + '</code></div>';
      }
      
      if (verify.message) {
        html += '<div style="margin-top:10px;padding:10px;background:' + statusBg + ';border-radius:4px;color:' + statusColor + ';">' + 
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
    document.getElementById("total-requests").textContent = data.total_requests;
    document.getElementById("success-requests").textContent =
      data.success_requests;
    document.getElementById("failed-requests").textContent =
      data.failed_requests;
    document.getElementById("success-rate").textContent =
      data.success_rate.toFixed(2) + "%";
    document.getElementById("qps").textContent = data.qps.toFixed(2);
    document.getElementById("avg-duration").textContent =
      data.avg_duration_ms + "ms";
    document.getElementById("elapsed").textContent = data.elapsed_seconds + "s";
    
    // 检查任务状态并更新按钮
    const pauseBtn = document.getElementById('pauseBtn');
    const stopBtn = document.getElementById('stopBtn');
    const statusText = document.getElementById('statusText');
    const statusDot = document.getElementById('statusDot');
    
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

  // 加载详情数据
  window.loadDetails = function () {
    fetch("/api/details?offset=0&limit=100")
      .then((res) => res.json())
      .then((data) => {
        if (data.total === lastDetailsCount && lastDetailsCount > 0) {
          return;
        }
        lastDetailsCount = data.total;
        allDetailsData = data.details || [];
        filterDetails();
      });
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
        
        const verifyStatus =
          detail.verifications && detail.verifications.length > 0
            ? detail.verifications.every((v) => v.success)
              ? "✓ 通过"
              : "✗ 失败"
            : "-";
        const verifyClass =
          detail.verifications && detail.verifications.length > 0
            ? detail.verifications.every((v) => v.success)
              ? "status-success"
              : "status-error"
            : "";

        row.innerHTML = `
                    <td>${detail.id}</td>
                    <td>${detail.group_id || '-'}</td>
                    <td>${detail.api_name || '-'}</td>
                    <td>${new Date(detail.timestamp).toLocaleTimeString()}</td>
                    <td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="${
                      detail.url || "-"
                    }">${detail.url || "-"}</td>
                    <td>${detail.method || "-"}</td>
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

        // 不自动恢复展开状态,保持收起
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

    eventSource.onmessage = function (event) {
      const data = JSON.parse(event.data);
      updateMetrics(data);
      updateCharts(data);
      loadDetails();
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
