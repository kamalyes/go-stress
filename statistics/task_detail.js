// 任务详情页面 JavaScript

// ============ 常量定义 ============
const ELEMENT_IDS = {
    TASK_ID: 'taskId',
    TASK_STATUS: 'taskStatus',
    TASK_PROTOCOL: 'taskProtocol',
    TASK_WORKERS: 'taskWorkers',
    TASK_DURATION: 'taskDuration',
    TASK_CREATED: 'taskCreated',
    TASK_STARTED: 'taskStarted',
    SLAVE_COUNT: 'slaveCount',
    SLAVE_LIST: 'slaveList',
    TASK_CONFIG: 'taskConfig',
    SLAVE_SELECTOR: 'slaveSelector',
    BTN_STOP: 'btnStop',
    AUTO_REFRESH_SLAVES: 'autoRefreshSlaves',
    REFRESH_INTERVAL: 'refreshInterval',
    SLAVE_CHECKBOXES: 'slaveCheckboxes',
    REGION_SELECT: 'regionSelect',
    SLAVE_ALL: 'slaveAll',
    SLAVE_SPECIFIC: 'slaveSpecific',
    SLAVE_REGION: 'slaveRegion'
};

const TASK_STATUS_MAP = {
    'pending': '待执行',
    'running': '运行中',
    'completed': '已完成',
    'failed': '失败',
    'stopped': '已停止',
    'cancelled': '已取消'
};

const TASK_STATES = {
    PENDING: 'pending',
    RUNNING: 'running',
    COMPLETED: 'completed',
    FAILED: 'failed',
    STOPPED: 'stopped',
    CANCELLED: 'cancelled'
};

const SLAVE_STATES = {
    IDLE: 'idle',
    RUNNING: 'running',
    OFFLINE: 'offline',
    ERROR: 'error'
};

const REFRESH_INTERVALS = {
    THREE_SECONDS: 3000,
    FIVE_SECONDS: 5000,
    TEN_SECONDS: 10000,
    THIRTY_SECONDS: 30000
};

const API_ENDPOINTS = {
    TASKS: '/api/v1/tasks',
    SLAVES: '/api/v1/slaves',
    TASK_START: '/api/v1/tasks/{id}/start',
    TASK_STOP: '/api/v1/tasks/{id}'
};

const UI_TEXT = {
    INVALID_TASK_ID: '无效的任务 ID',
    LOAD_DETAIL_FAILED: '加载任务详情失败',
    LOAD_SLAVES_FAILED: '加载 Slave 列表失败',
    NO_SLAVES_AVAILABLE: '暂无可用 Slave',
    LOADING: '加载中...',
    CONFIRM_START: '确定要启动这个任务吗？\n策略：{0}',
    CONFIRM_STOP: '确定要停止这个任务吗?',
    TASK_STARTED: '任务启动成功!',
    TASK_STOPPED: '任务已停止!',
    START_FAILED: '启动任务失败',
    STOP_FAILED: '停止任务失败',
    AUTO_REFRESH_STARTED: '✅ Slave 自动刷新已启动，间隔: {0}ms',
    AUTO_REFRESH_STOPPED: '⏹️ Slave 自动刷新已停止',
    SELECTION_ALL: '使用所有可用 Slave',
    SELECTION_SPECIFIC: '指定节点：{0}',
    SELECTION_REGION: '区域：{0}',
    SELECTION_DEFAULT: '默认'
};

const CSS_CLASSES = {
    STATUS_BADGE: 'status-badge',
    STATUS_PREFIX: 'status-',
    SLAVE_CHECKBOX_ITEM: 'slave-checkbox-item',
    SLAVE_STATE: 'slave-state',
    LOADING: 'loading',
    ERROR: 'error'
};

const SELECTORS = {
    SLAVE_SELECTION_RADIO: 'input[name="slaveSelection"]:checked',
    SLAVE_CHECKBOXES_CHECKED: '#slaveCheckboxes input[type="checkbox"]:checked',
    DETAIL_CONTAINER: '.detail-container'
};

const DEFAULT_VALUES = {
    DEFAULT_INTERVAL: 5000,
    RELOAD_DELAY: 500,
    PLACEHOLDER: '-',
    ZERO: '0',
    INDENT_SPACES: 2
};

const UI_STYLES = {
    DISPLAY_BLOCK: 'block',
    DISPLAY_INLINE_BLOCK: 'inline-block',
    DISPLAY_NONE: 'none',
    COLOR_GRAY: '#999',
    COLOR_ERROR: '#f45c43',
    COLOR_DARK_GRAY: '#666',
    COLOR_PRIMARY: '#667eea',
    TEXT_DECORATION_NONE: 'none',
    TEXT_DECORATION_LINE_THROUGH: 'line-through'
};

const INPUT_TYPES = {
    CHECKBOX: 'checkbox'
};

const ELEMENT_TYPES = {
    DIV: 'div',
    INPUT: 'input',
    LABEL: 'label',
    SPAN: 'span',
    OPTION: 'option'
};

const EMOJI = {
    ROBOT: '🤖',
    CHART: '📊',
    WAITING: '⏳',
    CHECK: '✓',
    WARNING: '⚠️',
    SUCCESS: '✅',
    STOP: '⛔'
};

const HTML_TEMPLATES = {
    PENDING_HINT: '<p style="color: #999;">⏳ 任务待启动,请在上方选择 Slave 节点后启动</p>',
    NO_SLAVES: '<p style="color: #999;">暂未分配 Slave 节点</p>',
    NO_CONFIG: '无配置数据',
    LOADING_TEXT: '<span class="loading">加载中...</span>',
    ERROR_NO_SLAVES: '<span class="error">暂无可用 Slave</span>',
    ERROR_LOAD_FAILED: '<span class="error">加载失败</span>',
    REGION_PLACEHOLDER: '<option value="">-- 选择区域 --</option>'
};

const ALERT_MESSAGES = {
    SELECT_AT_LEAST_ONE: '请至少选择一个 Slave 节点',
    SELECT_REGION: '请选择一个区域'
};

// ============ 全局变量 ============
let slaveRefreshInterval = null; // Slave 列表刷新定时器

// 从 URL 获取任务 ID
function getTaskIdFromURL() {
    const path = window.location.pathname;
    const parts = path.split('/');
    return parts[parts.length - 1];
}

// 页面加载时初始化
document.addEventListener('DOMContentLoaded', function() {
    // 检查 http 客户端是否加载
    if (typeof http === 'undefined') {
        console.error('❌ http-client.js 未正确加载！');
        showError('HTTP 客户端加载失败，请刷新页面重试');
        return;
    }
    
    const taskId = getTaskIdFromURL();
    if (taskId) {
        loadTaskDetail(taskId);
        initSlaveRefreshControls();
    } else {
        showError(UI_TEXT.INVALID_TASK_ID);
    }
});

// 初始化 Slave 刷新控制
function initSlaveRefreshControls() {
    const autoRefreshCheckbox = document.getElementById('autoRefreshSlaves');
    const refreshIntervalSelect = document.getElementById('refreshInterval');
    
    if (!autoRefreshCheckbox || !refreshIntervalSelect) return;
    
    // 监听自动刷新复选框
    autoRefreshCheckbox.addEventListener('change', function() {
        if (this.checked) {
            startSlaveAutoRefresh();
        } else {
            stopSlaveAutoRefresh();
        }
    });
    
    // 监听刷新间隔变化
    refreshIntervalSelect.addEventListener('change', function() {
        if (autoRefreshCheckbox.checked) {
            stopSlaveAutoRefresh();
            startSlaveAutoRefresh();
        }
    });
}

// 启动 Slave 自动刷新
function startSlaveAutoRefresh() {
    stopSlaveAutoRefresh(); // 先清除旧的定时器
    
    const intervalSelect = document.getElementById(ELEMENT_IDS.REFRESH_INTERVAL);
    const interval = intervalSelect ? parseInt(intervalSelect.value) : DEFAULT_VALUES.DEFAULT_INTERVAL;
    
    slaveRefreshInterval = setInterval(() => {
        loadAvailableSlaves();
    }, interval);
    
    console.log(UI_TEXT.AUTO_REFRESH_STARTED.replace('{0}', interval));
}

// 停止 Slave 自动刷新
function stopSlaveAutoRefresh() {
    if (slaveRefreshInterval) {
        clearInterval(slaveRefreshInterval);
        slaveRefreshInterval = null;
        console.log(UI_TEXT.AUTO_REFRESH_STOPPED);
    }
}

// 页面卸载时清理
window.addEventListener('beforeunload', function() {
    stopSlaveAutoRefresh();
});

// 加载任务详情
async function loadTaskDetail(taskId) {
    try {
        const res = await http.get(`${API_ENDPOINTS.TASKS}/${taskId}`);
        const task = res.data;
        renderTaskDetail(task);
        
    } catch (error) {
        console.error(UI_TEXT.LOAD_DETAIL_FAILED, error);
        showError(`${UI_TEXT.LOAD_DETAIL_FAILED}: ${error.message}`);
    }
}

// 渲染任务详情
function renderTaskDetail(task) {
    // 基本信息
    document.getElementById(ELEMENT_IDS.TASK_ID).textContent = task.id || DEFAULT_VALUES.PLACEHOLDER;
    
    const statusText = TASK_STATUS_MAP[task.state] || task.state || DEFAULT_VALUES.PLACEHOLDER;
    const statusEl = document.getElementById(ELEMENT_IDS.TASK_STATUS);
    statusEl.innerHTML = `<span class="${CSS_CLASSES.STATUS_BADGE} ${CSS_CLASSES.STATUS_PREFIX}${task.state}">${statusText}</span>`;
    
    // 显示 Slave 选择器或停止按钮
    const slaveSelector = document.getElementById(ELEMENT_IDS.SLAVE_SELECTOR);
    const btnStop = document.getElementById(ELEMENT_IDS.BTN_STOP);
    
    if (task.state === TASK_STATES.PENDING) {
        // pending 状态：显示 Slave 选择器
        slaveSelector.style.display = UI_STYLES.DISPLAY_BLOCK;
        btnStop.style.display = UI_STYLES.DISPLAY_NONE;
        loadAvailableSlaves(); // 加载可用的 Slave 列表
        
        // 检查是否启用自动刷新
        const autoRefreshCheckbox = document.getElementById(ELEMENT_IDS.AUTO_REFRESH_SLAVES);
        if (autoRefreshCheckbox && autoRefreshCheckbox.checked) {
            startSlaveAutoRefresh();
        }
    } else if (task.state === TASK_STATES.RUNNING) {
        // running 状态：显示停止按钮
        slaveSelector.style.display = UI_STYLES.DISPLAY_NONE;
        btnStop.style.display = UI_STYLES.DISPLAY_INLINE_BLOCK;
        stopSlaveAutoRefresh(); // 停止刷新
    } else {
        // 其他状态：都隐藏
        slaveSelector.style.display = UI_STYLES.DISPLAY_NONE;
        btnStop.style.display = UI_STYLES.DISPLAY_NONE;
        stopSlaveAutoRefresh(); // 停止刷新
    }
    
    document.getElementById(ELEMENT_IDS.TASK_PROTOCOL).textContent = task.protocol || DEFAULT_VALUES.PLACEHOLDER;
    document.getElementById(ELEMENT_IDS.TASK_WORKERS).textContent = task.total_workers || DEFAULT_VALUES.ZERO;
    
    // 持续时间显示：
    // - 如果任务已完成或失败，显示实际运行时间（completed_at - started_at）
    // - 如果任务运行中，显示已运行时间（now - started_at）
    // - 否则显示配置的持续时间
    let durationText = DEFAULT_VALUES.PLACEHOLDER;
    if (task.completed_at && task.started_at) {
        // 任务已完成，计算实际运行时间
        const startTime = new Date(task.started_at);
        const endTime = new Date(task.completed_at);
        const durationSeconds = Math.floor((endTime - startTime) / 1000);
        durationText = `${durationSeconds}s`;
    } else if (task.state === 'running' && task.started_at) {
        // 任务运行中，显示已运行时间
        const startTime = new Date(task.started_at);
        const now = new Date();
        const durationSeconds = Math.floor((now - startTime) / 1000);
        durationText = `${durationSeconds}s (运行中)`;
    } else if (task.duration) {
        // 显示配置的持续时间
        durationText = `${task.duration}s (预计)`;
    }
    document.getElementById(ELEMENT_IDS.TASK_DURATION).textContent = durationText;
    
    document.getElementById(ELEMENT_IDS.TASK_CREATED).textContent = formatTime(task.created_at);
    document.getElementById(ELEMENT_IDS.TASK_STARTED).textContent = formatTime(task.started_at);
    
    // 分配的 Slave 节点
    const slaves = task.assigned_slaves || [];
    document.getElementById(ELEMENT_IDS.SLAVE_COUNT).textContent = slaves.length;
    
    const slaveList = document.getElementById(ELEMENT_IDS.SLAVE_LIST);
    if (task.state === TASK_STATES.PENDING) {
        // pending 状态显示提示信息
        slaveList.innerHTML = HTML_TEMPLATES.PENDING_HINT;
    } else if (slaves.length > 0) {
        const taskId = task.id;
        slaveList.innerHTML = slaves.map(slaveId => `
            <div class="slave-badge">
                <a href="/distributed/slaves/${slaveId}" style="text-decoration: ${UI_STYLES.TEXT_DECORATION_NONE}; color: inherit;">
                    ${EMOJI.ROBOT} ${slaveId}
                </a>
                <a href="/realtime?slave_id=${slaveId}&task_id=${taskId}" 
                   target="_blank" 
                   title="查看 ${slaveId} 的实时报告"
                   style="margin-left: 8px; color: ${UI_STYLES.COLOR_PRIMARY}; text-decoration: ${UI_STYLES.TEXT_DECORATION_NONE};">
                    ${EMOJI.CHART}
                </a>
            </div>
        `).join('');
    } else {
        slaveList.innerHTML = HTML_TEMPLATES.NO_SLAVES;
    }
    
    // 任务配置
    const configEl = document.getElementById(ELEMENT_IDS.TASK_CONFIG);
    if (task.config_data) {
        try {
            let config = task.config_data;
            
            // 如果是 Base64 编码的字符串，先解码
            if (typeof config === 'string' && !config.startsWith('{')) {
                try {
                    // 尝试 Base64 解码
                    const decoded = atob(config);
                    // 验证是否为 JSON
                    const parsed = JSON.parse(decoded);
                    config = JSON.stringify(parsed, null, DEFAULT_VALUES.INDENT_SPACES);
                } catch (decodeErr) {
                    // 如果解码失败，直接显示原始字符串
                    config = task.config_data;
                }
            } else if (typeof config === 'object') {
                config = JSON.stringify(config, null, DEFAULT_VALUES.INDENT_SPACES);
            }
            
            configEl.textContent = config;
        } catch (e) {
            configEl.textContent = String(task.config_data);
        }
    } else {
        configEl.textContent = HTML_TEMPLATES.NO_CONFIG;
    }
}

// 格式化时间
function formatTime(timestamp) {
    if (!timestamp) return DEFAULT_VALUES.PLACEHOLDER;
    const date = new Date(timestamp);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

// 启动任务
async function startTask() {
    const taskId = getTaskIdFromURL();
    if (!taskId) return;
    
    // 获取选择的 Slave 策略
    const selection = document.querySelector(SELECTORS.SLAVE_SELECTION_RADIO).value;
    let requestBody = {};
    
    if (selection === 'specific') {
        // 指定 Slave ID
        const checkboxes = document.querySelectorAll(SELECTORS.SLAVE_CHECKBOXES_CHECKED);
        const slaveIds = Array.from(checkboxes).map(cb => cb.value);
        
        if (slaveIds.length === 0) {
            alert(ALERT_MESSAGES.SELECT_AT_LEAST_ONE);
            return;
        }
        
        requestBody.slave_ids = slaveIds;
    } else if (selection === 'region') {
        // 按区域选择
        const region = document.getElementById(ELEMENT_IDS.REGION_SELECT).value;
        if (!region) {
            alert(ALERT_MESSAGES.SELECT_REGION);
            return;
        }
        requestBody.slave_region = region;
    }
    // selection === 'all' 时，requestBody 为空对象，使用默认策略
    
    console.log('启动任务请求体:', JSON.stringify(requestBody, null, DEFAULT_VALUES.INDENT_SPACES));
    
    if (!confirm(UI_TEXT.CONFIRM_START.replace('{0}', getSelectionText(selection)))) {
        return;
    }
    
    try {
        const res = await http.post(API_ENDPOINTS.TASK_START.replace('{id}', taskId), requestBody);
        const result = res.data;
        
        alert(UI_TEXT.TASK_STARTED);
        // 重新加载任务详情
        setTimeout(() => loadTaskDetail(taskId), DEFAULT_VALUES.RELOAD_DELAY);
        
    } catch (error) {
        console.error(UI_TEXT.START_FAILED, error);
        alert(`${UI_TEXT.START_FAILED}: ${error.message}`);
    }
}

// 加载可用的 Slave 列表
async function loadAvailableSlaves() {
    const checkboxesContainer = document.getElementById(ELEMENT_IDS.SLAVE_CHECKBOXES);
    const regionSelect = document.getElementById(ELEMENT_IDS.REGION_SELECT);
    
    checkboxesContainer.innerHTML = HTML_TEMPLATES.LOADING_TEXT;
    
    try {
        const res = await http.get(API_ENDPOINTS.SLAVES);
        const data = res.data;
        const slaves = data.slaves || [];
        
        if (slaves.length === 0) {
            checkboxesContainer.innerHTML = HTML_TEMPLATES.ERROR_NO_SLAVES;
            return;
        }
        
        // 渲染 Slave 复选框
        checkboxesContainer.innerHTML = '';
        const regions = new Set();
        
        slaves.forEach(slave => {
            const item = document.createElement(ELEMENT_TYPES.DIV);
            item.className = CSS_CLASSES.SLAVE_CHECKBOX_ITEM;
            
            const checkbox = document.createElement(ELEMENT_TYPES.INPUT);
            checkbox.type = INPUT_TYPES.CHECKBOX;
            checkbox.id = `slave-${slave.id}`;
            checkbox.value = slave.id;
            checkbox.disabled = slave.state !== SLAVE_STATES.IDLE;
            
            // 点击复选框时自动勾选 "指定 Slave 节点" radio
            checkbox.addEventListener('change', function() {
                // 检查是否有任何复选框被选中
                const anyChecked = document.querySelectorAll(SELECTORS.SLAVE_CHECKBOXES_CHECKED).length > 0;
                if (anyChecked) {
                    document.getElementById(ELEMENT_IDS.SLAVE_SPECIFIC).checked = true;
                }
            });
            
            const label = document.createElement(ELEMENT_TYPES.LABEL);
            label.htmlFor = `slave-${slave.id}`;
            label.textContent = slave.id;
            
            if (slave.state === SLAVE_STATES.IDLE) {
                const stateSpan = document.createElement(ELEMENT_TYPES.SPAN);
                stateSpan.className = CSS_CLASSES.SLAVE_STATE;
                stateSpan.textContent = EMOJI.CHECK;
                label.appendChild(stateSpan);
            } else {
                label.style.color = UI_STYLES.COLOR_GRAY;
                label.style.textDecoration = UI_STYLES.TEXT_DECORATION_LINE_THROUGH;
            }
            
            item.appendChild(checkbox);
            item.appendChild(label);
            checkboxesContainer.appendChild(item);
            
            // 收集区域
            if (slave.region) {
                regions.add(slave.region);
            }
        });
        
        // 渲染区域下拉框（先克隆元素来移除所有旧的事件监听器）
        const oldRegionSelect = regionSelect;
        const newRegionSelect = oldRegionSelect.cloneNode(false);
        oldRegionSelect.parentNode.replaceChild(newRegionSelect, oldRegionSelect);
        
        newRegionSelect.innerHTML = HTML_TEMPLATES.REGION_PLACEHOLDER;
        regions.forEach(region => {
            const option = document.createElement(ELEMENT_TYPES.OPTION);
            option.value = region;
            option.textContent = region;
            newRegionSelect.appendChild(option);
        });
        
        // 选择区域时自动勾选 "按区域选择" radio
        newRegionSelect.addEventListener('change', function() {
            if (this.value) {
                document.getElementById(ELEMENT_IDS.SLAVE_REGION).checked = true;
            }
        });
        
    } catch (error) {
        console.error(UI_TEXT.LOAD_SLAVES_FAILED, error);
        checkboxesContainer.innerHTML = HTML_TEMPLATES.ERROR_LOAD_FAILED;
    }
}

// 获取选择策略的文本描述
function getSelectionText(selection) {
    if (selection === 'all') {
        return UI_TEXT.SELECTION_ALL;
    } else if (selection === 'specific') {
        const checkboxes = document.querySelectorAll(SELECTORS.SLAVE_CHECKBOXES_CHECKED);
        const slaveIds = Array.from(checkboxes).map(cb => cb.value);
        return UI_TEXT.SELECTION_SPECIFIC.replace('{0}', slaveIds.join(', '));
    } else if (selection === 'region') {
        const region = document.getElementById(ELEMENT_IDS.REGION_SELECT).value;
        return UI_TEXT.SELECTION_REGION.replace('{0}', region);
    }
    return UI_TEXT.SELECTION_DEFAULT;
}

// 停止任务
async function stopTask() {
    const taskId = getTaskIdFromURL();
    if (!taskId) return;
    
    if (!confirm(UI_TEXT.CONFIRM_STOP)) return;
    
    try {
        await http.delete(API_ENDPOINTS.TASK_STOP.replace('{id}', taskId));
        
        alert(UI_TEXT.TASK_STOPPED);
        // 重新加载任务详情
        loadTaskDetail(taskId);
        
    } catch (error) {
        console.error(UI_TEXT.STOP_FAILED, error);
        alert(`${UI_TEXT.STOP_FAILED}: ${error.message}`);
    }
}

// 显示错误
function showError(message) {
    const container = document.querySelector(SELECTORS.DETAIL_CONTAINER);
    container.innerHTML = `
        <div style="text-align: center; padding: 60px 20px;">
            <div style="font-size: 48px; margin-bottom: 20px;">${EMOJI.WARNING}</div>
            <h2 style="color: ${UI_STYLES.COLOR_ERROR}; margin-bottom: 12px;">${UI_TEXT.LOAD_DETAIL_FAILED}</h2>
            <p style="color: ${UI_STYLES.COLOR_DARK_GRAY}; margin-bottom: 24px;">${message}</p>
            <a href="/distributed" class="btn btn-primary">返回列表</a>
        </div>
    `;
}
