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
  
  postControl(endpoint)
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
      } else {
        alert(data.message || '操作未生效');
      }
    })
    .catch(err => {
      console.error('控制操作失败:', err);
      alert('控制操作失败: ' + err.message);
    });
};

window.stopMonitoring = function() {
  if (!confirm('确定要停止压测吗？停止后将无法恢复！')) {
    return;
  }
  
  const stopBtn = document.getElementById(ELEMENT_IDS.STOP_BTN);
  const pauseBtn = document.getElementById(ELEMENT_IDS.PAUSE_BTN);
  const statusText = document.getElementById(ELEMENT_IDS.STATUS_TEXT);
  const statusDot = document.getElementById(ELEMENT_IDS.STATUS_DOT);
  
  postControl(API_ENDPOINTS.STOP)
    .then(data => {
      if (data.success) {
        stopBtn.disabled = true;
        pauseBtn.disabled = true;
        stopBtn.style.opacity = '0.5';
        pauseBtn.style.opacity = '0.5';
        statusText.textContent = CONTROL_STATUS.STOPPED.statusText;
        statusDot.style.background = CONTROL_STATUS.STOPPED.statusDot.bg;
        statusDot.style.animation = CONTROL_STATUS.STOPPED.statusDot.animation;
        alert(data.message || '压测已停止');
      } else {
        alert(data.message || '停止未生效');
      }
    })
    .catch(err => {
      console.error('停止失败:', err);
      alert('停止失败: ' + err.message);
    });
};

// postControl 发起一次控制请求并归一化返回值
//
// 原先两处都是 fetch(...).then(r => r.json())，而失败时服务端返回的是
// http.Error 的纯文本，r.json() 直接抛异常落进 catch，catch 里只有
// console.error —— 点下去页面毫无反应，使用者无从知道发生了什么
// 分布式模式下 Master 此前连这三个路由都没注册，必然 404，全程静默
function postControl(endpoint) {
  return fetch(endpoint, { method: 'POST' }).then(function (response) {
    return response
      .json()
      .catch(function () {
        // 非 JSON 响应（404 的纯文本等）转成同样形状，调用方只看 success
        return { success: false, message: 'HTTP ' + response.status + ' ' + response.statusText };
      })
      .then(function (data) {
        if (typeof data.success === 'undefined') {
          data.success = response.ok;
        }
        return data;
      });
  });
}

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
      renderAPIStats(data);

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
      title: { text: "时序趋势", left: "center" },
      tooltip: { trigger: "axis" },
      legend: { data: ["P50", "P95", "P99", "QPS"], bottom: 0 },
      xAxis: { type: "category", data: [] },
      yAxis: [
        { type: "value", name: "响应时间 (ms)" },
        { type: "value", name: "QPS" },
      ],
      series: [
        {
          name: "P50",
          data: [],
          type: "line",
          smooth: true,
          showSymbol: false,
          lineStyle: { color: "#38ef7d", width: 2 },
        },
        {
          name: "P95",
          data: [],
          type: "line",
          smooth: true,
          showSymbol: false,
          lineStyle: { color: "#ffc107", width: 2 },
        },
        {
          name: "P99",
          data: [],
          type: "line",
          smooth: true,
          showSymbol: false,
          lineStyle: { color: "#eb3349", width: 2 },
        },
        {
          name: "QPS",
          data: [],
          type: "line",
          yAxisIndex: 1,
          smooth: true,
          showSymbol: false,
          lineStyle: { color: "#667eea", width: 2, type: "dashed" },
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

// ============ 时序趋势图（静态与实时共用） ============
// 用时序数据刷新趋势图；数据不存在（旧版 JSON / 实时未启动记录）返回 false，由调用方降级
function updateTimeseriesChart(data) {
  const ts = data && data.timeseries;
  if (!durationChart || !ts || !ts.points || ts.points.length === 0) {
    return false;
  }
  const labels = ts.points.map((p) => new Date(p.t * 1000).toLocaleTimeString());
  durationChart.setOption({
    xAxis: { type: "category", data: labels },
    series: [
      { name: "P50", data: ts.points.map((p) => round2(p.p50)) },
      { name: "P95", data: ts.points.map((p) => round2(p.p95)) },
      { name: "P99", data: ts.points.map((p) => round2(p.p99)) },
      { name: "QPS", data: ts.points.map((p) => round2(p.qps)) },
    ],
  });
  return true;
}

// 降级视图：没有时序数据时（旧版报告 JSON）退化为「最近请求耗时」单线图
function showDegradedTrendChart(values) {
  if (!durationChart) return;
  durationChart.setOption({
    title: { text: "最近请求耗时", left: "center" },
    legend: { show: false },
    xAxis: { type: "category", data: values.map((_, i) => i + 1) },
    yAxis: [
      { type: "value", name: "响应时间 (ms)" },
      { type: "value", name: "QPS" },
    ],
    series: [
      {
        name: "耗时",
        data: values,
        type: "line",
        smooth: true,
        showSymbol: false,
        areaStyle: { color: "rgba(102, 126, 234, 0.2)" },
        lineStyle: { color: "#667eea", width: 2 },
      },
      { data: [] },
      { data: [] },
      { data: [] },
    ],
  });
}

// 保留两位小数（避免图表轴标签出现长浮点）
function round2(v) {
  return Math.round((v || 0) * 100) / 100;
}

// ============ API 汇总：Swagger 风格折叠项（静态与实时共用） ============
//
// 原先是一张 11 列平表，窄屏必须横向滚动，而真正要找的是「哪个接口慢、
// 哪个在报错」。改成按接口折叠：行首是方法与路径，关键指标压进同一行，
// 明细按需展开

// apiItemsOpen 记住展开状态
//
// 实时模式每秒重渲染一次，若不记住状态，用户刚展开的接口会被立刻合上
const apiItemsOpen = new Set();

// methodOf 取方法名并归一为大写；缺失时按 — 展示而不是留空
function methodOf(s) {
  const m = (s.method || "").toUpperCase();
  return m || "—";
}

// pathOf 只取 URL 的路径与查询部分
//
// 行首展示完整 URL 会被域名占满，同一服务的多个接口看起来几乎一样；
// 完整地址在展开后的 api-url 里给出
function pathOf(s) {
  const raw = s.url || "";
  if (!raw) return escapeHtml(s.name || "-");
  try {
    const u = new URL(raw);
    return escapeHtml(u.pathname + (u.search || "")) || "/";
  } catch (e) {
    // 地址里含未替换的变量占位符时 URL 解析会失败，退回原串
    return escapeHtml(raw);
  }
}

// rateClass 成功率的着色档位
function rateClass(rate, failed) {
  if (failed > 0 && rate < 95) return "bad";
  if (failed > 0) return "warn";
  return "ok";
}

// statusCodeClass 按状态码首位着色
function statusCodeClass(code) {
  const n = parseInt(code, 10);
  if (n >= 200 && n < 300) return "c2xx";
  if (n >= 300 && n < 400) return "c3xx";
  if (n >= 400 && n < 500) return "c4xx";
  if (n >= 500) return "c5xx";
  return "";
}

// renderStatusCodes 状态码分布，按码值升序
//
// 每个码是一个下钻入口：点「404 · 11」就在原地列出这 11 条里的真实请求，
// 只给出一个数字说明不了「怎么个 404 法」
function renderStatusCodes(codes, name, drill) {
  const entries = Object.entries(codes || {});
  if (entries.length === 0) return "";
  const chips = entries
    .sort((a, b) => parseInt(a[0], 10) - parseInt(b[0], 10))
    .map(([code, n]) => {
      const on =
        drill && drill.kind === "code" && String(drill.key) === String(code)
          ? " is-active"
          : "";
      return `<button type="button" class="api-code ${statusCodeClass(code)}${on}"
          data-drill="code" data-key="${escapeAttr(code)}" data-api="${escapeAttr(name)}"
          title="查看这 ${n} 条请求的明细">${escapeHtml(code)} · ${n}</button>`;
    })
    .join("");
  return `<div class="api-block">
      <div class="api-sub-title">状态码分布<span class="api-sub-hint">点任意状态码查看命中的请求</span></div>
      <div class="api-codes">${chips}</div>
    </div>`;
}

// renderAPIErrors 错误归类，按次数降序，最多展示 8 条
function renderAPIErrors(errors, name, drill) {
  const entries = Object.entries(errors || {});
  if (entries.length === 0) return "";
  const sorted = entries.sort((a, b) => b[1] - a[1]);
  const shown = sorted.slice(0, 8);
  const rest = sorted.length - shown.length;
  const rows = shown
    .map(([msg, n]) => {
      const on =
        drill && drill.kind === "error" && drill.key === msg ? " is-active" : "";
      return `<button type="button" class="api-error${on}"
        data-drill="error" data-key="${escapeAttr(msg)}" data-api="${escapeAttr(name)}"
        title="查看这 ${n} 条请求的明细">
        <span class="api-error-count">${n}×</span>
        <span class="api-error-msg">${escapeHtml(msg)}</span>
      </button>`;
    })
    .join("");
  const more = rest > 0 ? `<div class="api-error-msg" style="font-size:11px;color:#6c757d;">另有 ${rest} 类错误，完整列表见请求明细</div>` : "";
  return `<div class="api-block">
      <div class="api-sub-title">错误归类</div>
      <div class="api-errors">${rows}${more}</div>
    </div>`;
}

// ---- 下钻：从「某接口的某个状态码 / 某类错误」落到真实请求 ----

// apiDrillOpen 记住每个接口当前钻取的条件，apiSampleOpen 记住展开到底的样本
//
// 两份状态都必须留在渲染函数之外：实时模式每秒整体重渲染一次，
// 不记住的话用户刚点开的 404 样本下一秒就被合上了
const apiDrillOpen = new Map(); // name -> {kind: "code"|"error", key: string}
const apiSampleOpen = new Set(); // name + "\u0000" + 明细 id

// API_SAMPLE_LIMIT 一次下钻最多列出的样本条数
//
// 命中几万条时全列出来既无意义也会把页面卡住；看清「错成什么样」
// 十几条足够，超出的数量照实写在标题上
const API_SAMPLE_LIMIT = 20;

// errorTextOf 取明细的错误文本，口径与 APIStat.Errors 的聚合键一致
function errorTextOf(req) {
  return req.error || req.error_msg || "";
}

// collectSamples 从已加载的明细里找出命中下钻条件的请求
//
// 返回 total 与 list 两个值：明细本身可能被 -details-limit 截断过，
// 此时 total 会小于状态码分布上的计数，照实显示而不是假装取全了
function collectSamples(name, drill) {
  const out = { list: [], total: 0 };
  if (!drill || !Array.isArray(allDetailsData)) return out;
  for (let i = 0; i < allDetailsData.length; i++) {
    const req = allDetailsData[i];
    if ((req.api_name || "") !== name) continue;
    if (drill.kind === "code") {
      if (String(req.status_code || "") !== String(drill.key)) continue;
    } else if (errorTextOf(req) !== drill.key) {
      continue;
    }
    out.total++;
    if (out.list.length < API_SAMPLE_LIMIT) out.list.push(req);
  }
  return out;
}

// sampleTime 样本行的时间戳，精确到毫秒
function sampleTime(req) {
  if (!req.timestamp) return "-";
  const d = new Date(req.timestamp);
  if (isNaN(d.getTime())) return "-";
  const ms = String(d.getMilliseconds()).padStart(3, "0");
  return d.toLocaleTimeString("en-GB", { hour12: false }) + "." + ms;
}

// sampleReason 样本行的一句话原因：优先取没通过的那条验证规则
//
// 验证规则带着 expect / actual，比「验证失败」这个归类文本有用得多
function sampleReason(req) {
  const rules = req.verifications || [];
  for (let i = 0; i < rules.length; i++) {
    const v = rules[i];
    if (v.success || v.skipped) continue;
    const detail = v.message || `期望 ${v.expect}，实际 ${v.actual}`;
    return `${v.type}: ${detail}`;
  }
  return errorTextOf(req);
}

// renderSample 单条样本：一行摘要 + 按需展开的完整明细
function renderSample(name, req, idx) {
  const key = name + "\u0000" + (req.id || idx);
  const isOpen = apiSampleOpen.has(key);
  const code = req.status_code || 0;
  const ms = ((req.duration || 0) / 1e6).toFixed(2);
  const reason = sampleReason(req);
  // 样本行优先显示服务端回带的 ID：拿它去对方日志系统里查最直接，
  // 对方没回带时才退回本工具发出去的那个
  const trace = req.server_trace_id || req.trace_id || "";
  const traceFrom = req.server_trace_id
    ? "服务端 " + (req.server_trace_header || "trace id")
    : "trace_id（本工具发出）";

  return `<div class="api-sample${isOpen ? " is-open" : ""}">
      <button type="button" class="api-sample-head" data-sample="${escapeAttr(req.id || idx)}" data-api="${escapeAttr(name)}">
        <span class="api-sample-idx">#${idx + 1}</span>
        <span class="api-sample-time">${sampleTime(req)}</span>
        <span class="api-sample-dur">${ms}ms</span>
        <span class="api-code ${statusCodeClass(code)}">${code || "-"}</span>
        ${trace ? `<span class="api-sample-trace${req.server_trace_id ? " is-server" : ""}" title="${escapeAttr(traceFrom)}">${escapeHtml(trace)}</span>` : ""}
        ${reason ? `<span class="api-sample-reason">${escapeHtml(reason)}</span>` : ""}
        <span class="api-caret">▶</span>
      </button>
      <div class="api-sample-body">${isOpen ? generateDetailContent(req) : ""}</div>
    </div>`;
}

// drillEmptyHint 下钻取不到样本时的说明
//
// 两种模式的成因完全不同，笼统写一句会把人引到错误的方向：
//   - 静态报告：明细是一次性导出的，取不到多半是被条数上限截断了
//   - 实时模式：只加载了当前这一页，命中的请求可能在别的页上
function drillEmptyHint(noDataAtAll) {
  if (isRealtime) {
    return noDataAtAll
      ? "明细还在加载中，稍候再点一次；实时模式下只加载当前页，命中的请求可能在其他页"
      : "当前页的明细里没有命中的请求，到下方「请求明细」翻页或切换状态筛选后再试";
  }
  return noDataAtAll
    ? "本报告未导出请求明细，无法下钻（--details-limit 设为 0 时不导出明细）"
    : "明细中没有命中的请求，可能已被明细条数上限截断";
}

// renderDrill 下钻面板；未钻取时不占版面
function renderDrill(s) {
  const drill = apiDrillOpen.get(s.name);
  if (!drill) return "";

  const label =
    drill.kind === "code"
      ? `状态码 ${escapeHtml(String(drill.key))}`
      : `错误「${escapeHtml(drill.key)}」`;

  if (!Array.isArray(allDetailsData) || allDetailsData.length === 0) {
    return `<div class="api-drill">
        <div class="api-drill-title">${label} 的请求样本</div>
        <div class="api-drill-empty">${drillEmptyHint(true)}</div>
      </div>`;
  }

  const found = collectSamples(s.name, drill);
  if (found.total === 0) {
    return `<div class="api-drill">
        <div class="api-drill-title">${label} 的请求样本</div>
        <div class="api-drill-empty">${drillEmptyHint(false)}</div>
      </div>`;
  }

  const more =
    found.total > found.list.length
      ? `<span class="api-drill-more">命中 ${found.total} 条，展示前 ${found.list.length} 条</span>`
      : `<span class="api-drill-more">命中 ${found.total} 条</span>`;

  return `<div class="api-drill">
      <div class="api-drill-title">${label} 的请求样本${more}
        <button type="button" class="api-btn api-drill-close" data-drill="close" data-api="${escapeAttr(s.name)}">收起</button>
      </div>
      <div class="api-samples">${found.list.map((r, i) => renderSample(s.name, r, i)).join("")}</div>
    </div>`;
}

// renderAPIItem 渲染单个接口的折叠项
function renderAPIItem(s) {
  const method = methodOf(s);
  const failed = s.failed || 0;
  const skipped = s.skipped || 0;
  const rate = s.success_rate || 0;
  const cls = rateClass(rate, failed);
  const open = apiItemsOpen.has(s.name) ? " is-open" : "";
  const failing = failed > 0 ? " has-failure" : "";

  const chips = [
    `<span class="api-chip">${s.requests} 次</span>`,
    `<span class="api-chip ${cls}">${rate.toFixed(2)}%</span>`,
    `<span class="api-chip">P95 ${s.p95_ms.toFixed(1)}ms</span>`,
  ];
  if (failed > 0) chips.push(`<span class="api-chip bad">失败 ${failed}</span>`);
  if (skipped > 0) chips.push(`<span class="api-chip warn">跳过 ${skipped}</span>`);

  const metrics = [
    ["请求数", s.requests, ""],
    ["成功", s.success, "ok"],
    ["失败", failed, failed > 0 ? "bad" : ""],
    ["跳过", skipped, ""],
    // 成功率在头部已有 chip，这里仍保留一格：指标共 12 项时恰好铺满 6 列两行，
    // 不会像 11 项那样在末行留下一个孤零零的格子
    ["成功率", round2(s.success_rate) + "%", rateClass(s.success_rate, failed)],
    ["QPS", round2(s.qps), ""],
    ["平均", s.avg_ms.toFixed(2) + "ms", ""],
    ["最小", s.min_ms.toFixed(2) + "ms", ""],
    ["最大", s.max_ms.toFixed(2) + "ms", ""],
    ["P50", s.p50_ms.toFixed(2) + "ms", ""],
    ["P95", s.p95_ms.toFixed(2) + "ms", ""],
    ["P99", s.p99_ms.toFixed(2) + "ms", ""],
  ]
    .map(
      ([label, value, k]) => `<div class="api-metric">
        <div class="api-metric-label">${label}</div>
        <div class="api-metric-value ${k}">${value}</div>
      </div>`
    )
    .join("");

  const urlBlock = s.url
    ? `<div class="api-url">${escapeHtml(s.url)}</div>`
    : "";

  const drill = apiDrillOpen.get(s.name);

  return `<div class="api-item${open}${failing}" data-method="${escapeAttr(method)}" data-api="${escapeAttr(s.name)}">
      <button type="button" class="api-head">
        <span class="api-method">${escapeHtml(method)}</span>
        <span class="api-path">${pathOf(s)}</span>
        <span class="api-name">${escapeHtml(s.name)}</span>
        <span class="api-chips">${chips.join("")}</span>
        <span class="api-caret">▶</span>
      </button>
      <div class="api-body">
        ${urlBlock}
        <div class="api-metrics">${metrics}</div>
        ${renderStatusCodes(s.status_codes, s.name, drill)}
        ${renderAPIErrors(s.errors, s.name, drill)}
        ${renderDrill(s)}
      </div>
    </div>`;
}

// apiStatsCache 供筛选与展开/收起复用，避免重新取数
let apiStatsCache = [];

// applyAPIFilter 按关键字筛选接口（匹配名称、方法、地址）
function applyAPIFilter() {
  const input = document.getElementById("apiFilter");
  const list = document.getElementById("api-list");
  const empty = document.getElementById("apiEmpty");
  if (!list) return;

  const kw = ((input && input.value) || "").trim().toLowerCase();
  const matched = kw
    ? apiStatsCache.filter((s) => {
        const hay = [s.name, s.method, s.url].join(" ").toLowerCase();
        return hay.indexOf(kw) !== -1;
      })
    : apiStatsCache;

  list.innerHTML = matched.map(renderAPIItem).join("");
  if (empty) empty.style.display = matched.length === 0 ? "" : "none";
}

function renderAPIStats(data) {
  const section = document.getElementById("apiSection");
  const list = document.getElementById("api-list");
  if (!section || !list) return;

  const stats = (data && data.api_stats) || [];
  if (stats.length === 0) {
    section.style.display = "none";
    return;
  }

  section.style.display = "";
  apiStatsCache = stats;
  applyAPIFilter();
}

// 事件用委托绑定在容器上
//
// 折叠项每次渲染都会重建，逐个绑定 onclick 会在实时模式下反复失效；
// 委托到不被重建的容器上则一次绑定长期有效
document.addEventListener("click", function (e) {
  if (!e.target.closest) return;

  // 下钻入口：状态码、错误行、收起按钮
  const trigger = e.target.closest("[data-drill]");
  if (trigger) {
    const name = trigger.getAttribute("data-api");
    const kind = trigger.getAttribute("data-drill");
    if (kind === "close") {
      apiDrillOpen.delete(name);
    } else {
      const key = trigger.getAttribute("data-key");
      const cur = apiDrillOpen.get(name);
      // 再点一次同一个入口则收起，省去专门去找收起按钮
      if (cur && cur.kind === kind && cur.key === key) {
        apiDrillOpen.delete(name);
      } else {
        apiDrillOpen.set(name, { kind: kind, key: key });
      }
    }
    applyAPIFilter();
    return;
  }

  // 样本行展开/收起
  const sampleHead = e.target.closest(".api-sample-head");
  if (sampleHead) {
    const key =
      sampleHead.getAttribute("data-api") +
      "\u0000" +
      sampleHead.getAttribute("data-sample");
    if (apiSampleOpen.has(key)) {
      apiSampleOpen.delete(key);
    } else {
      apiSampleOpen.add(key);
    }
    applyAPIFilter();
    return;
  }

  const head = e.target.closest(".api-head");
  if (head) {
    const item = head.parentElement;
    const name = item.getAttribute("data-api");
    if (item.classList.toggle("is-open")) {
      apiItemsOpen.add(name);
    } else {
      apiItemsOpen.delete(name);
    }
    return;
  }

  if (e.target.id === "apiExpandAll") {
    apiStatsCache.forEach((s) => apiItemsOpen.add(s.name));
    applyAPIFilter();
  } else if (e.target.id === "apiCollapseAll") {
    apiItemsOpen.clear();
    applyAPIFilter();
  }
});

document.addEventListener("input", function (e) {
  if (e.target.id === "apiFilter") applyAPIFilter();
});

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
  if (data.request_details && data.request_details.length > 0 && durationChart && !updateTimeseriesChart(data)) {
    // 旧版 JSON 没有时序数据：退化为「最后 1000 条请求耗时」
    const recentDetails = data.request_details.slice(-1000);
    const durations = recentDetails.map((d) => d.duration / 1000000);

    showDegradedTrendChart(durations);
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

// escapeAttr 供双引号属性值使用
//
// escapeHtml 走的是 textContent -> innerHTML，这条路径不转义引号：
// 错误文本里只要出现一个 " 就能把属性提前闭合，整段标记就歪了
function escapeAttr(text) {
  if (text === null || text === undefined) return "";
  return String(text)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
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

// REQUEST_STORE_LIMIT 请求数据暂存的条数上限
//
// 这份暂存只服务于「复制为 curl」「重放」这类按钮，而按钮只可能作用在
// 当前看得见的行上，几百条绰绰有余
const REQUEST_STORE_LIMIT = 500;
const requestStoreOrder = [];

// rememberRequest 暂存一条请求数据，超出上限时淘汰最旧的
//
// 原先是直接 window.requestDataStore[reqId] = req，键是每次渲染新取的随机串，
// 且从不删除。实时模式每秒重渲染一整页明细，一页 20 行就是每秒 20 条新条目，
// 每条还挂着完整的响应体 —— 跑一小时攒下七万多条，页面越开越卡
function rememberRequest(reqId, req) {
  if (!window.requestDataStore) {
    window.requestDataStore = {};
  }
  if (!(reqId in window.requestDataStore)) {
    requestStoreOrder.push(reqId);
  }
  window.requestDataStore[reqId] = req;
  while (requestStoreOrder.length > REQUEST_STORE_LIMIT) {
    delete window.requestDataStore[requestStoreOrder.shift()];
  }
}

function generateDetailContent(req) {
  const tabId = 'tab-' + Math.random().toString(36).substr(2, 9);
  const reqId = 'req-' + Math.random().toString(36).substr(2, 9);
  const menuId = 'menu-' + reqId;
  
  // 将请求数据保存到全局变量供按钮使用（在生成HTML之前）
  rememberRequest(reqId, req);
  
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
  // 两个追踪 ID 分开列：trace_id 是本工具发出去的，server_trace_id 是对端回带的，
  // 查对方日志时该拿哪一个取决于对方有没有接 W3C 链路
  if (req.trace_id) {
    html += '<div class="detail-section"><strong>Trace ID (本工具发出):</strong><pre>' + escapeHtml(req.trace_id) + '</pre></div>';
  }
  if (req.server_trace_id) {
    const from = req.server_trace_header ? ' · 来自响应头 ' + escapeHtml(req.server_trace_header) : '';
    html += '<div class="detail-section"><strong>服务端 Trace ID' + from + ':</strong><pre>' + escapeHtml(req.server_trace_id) + '</pre></div>';
  }
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
      
      // 后端 VerifyResult 的字段名是 expect，不是 expected
      if (verify.expect !== undefined && verify.expect !== null) {
        html += '<div style="margin-bottom:8px;"><strong>✓ 期望值:</strong> <code style="background:#f8f9fa;padding:2px 6px;border-radius:3px;">' +
          escapeHtml(String(verify.expect)) + '</code></div>';
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
    // 时序趋势图；旧版实时数据（无时序）降级为最近 20 次请求耗时
    if (!updateTimeseriesChart(data) && data.recent_durations && data.recent_durations.length > 0 && durationChart) {
      showDegradedTrendChart(data.recent_durations);
    }

    renderAPIStats(data);

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
