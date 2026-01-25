// Slave 详情页面 JavaScript

// ============ 常量定义 ============
const ELEMENT_IDS = {
    SLAVE_ID: 'slaveId',
    SLAVE_STATE: 'slaveState',
    SLAVE_ADDRESS: 'slaveAddress',
    SLAVE_REGION: 'slaveRegion',
    SLAVE_REGISTERED: 'slaveRegistered',
    SLAVE_HEARTBEAT: 'slaveHeartbeat',
    SLAVE_HEALTH: 'slaveHealth',
    SLAVE_CURRENT_TASK: 'slaveCurrentTask',
    SLAVE_CPU_CORES: 'slaveCpuCores',
    SLAVE_CPU_USAGE: 'slaveCpuUsage',
    SLAVE_MEMORY_TOTAL: 'slaveMemoryTotal',
    SLAVE_MEMORY_USAGE: 'slaveMemoryUsage',
    TASK_COUNT: 'taskCount',
    TASK_LIST: 'taskList'
};

const SLAVE_STATE_MAP = {
    'idle': '空闲',
    'running': '运行中',
    'busy': '繁忙',
    'offline': '离线',
    'error': '错误',
    'unreachable': '不可达',
    'stopping': '停止中',
    'overloaded': '过载'
};

const HEALTH_THRESHOLDS = {
    WARNING: 3,
    CRITICAL: 5
};

const DEFAULT_VALUES = {
    EMPTY: '-',
    NO_TASK: '无',
    DEFAULT_REGION: '默认'
};

// ============ 工具函数 ============
function getSlaveIdFromURL() {
    const path = window.location.pathname;
    const parts = path.split('/');
    return parts[parts.length - 1];
}

// 页面加载时初始化
document.addEventListener('DOMContentLoaded', function() {
    const slaveId = getSlaveIdFromURL();
    if (slaveId) {
        loadSlaveDetail(slaveId);
    } else {
        showError('无效的 Slave ID');
    }
});

// 加载 Slave 详情
async function loadSlaveDetail(slaveId) {
    try {
        const res = await http.get(`/api/v1/slaves/${slaveId}`);
        const slave = res.data;
        renderSlaveDetail(slave);
        
    } catch (error) {
        console.error('加载 Slave 详情失败:', error);
        showError('加载 Slave 详情失败: ' + error.message);
    }
}

// 渲染 Slave 详情
function renderSlaveDetail(slave) {
    console.log('Slave 数据:', slave); // 调试日志
    
    // 基本信息
    document.getElementById(ELEMENT_IDS.SLAVE_ID).textContent = slave.id || DEFAULT_VALUES.EMPTY;
    
    const stateText = SLAVE_STATE_MAP[slave.state] || slave.state || DEFAULT_VALUES.EMPTY;
    const stateEl = document.getElementById(ELEMENT_IDS.SLAVE_STATE);
    stateEl.innerHTML = `<span class="state-badge state-${slave.state}">${stateText}</span>`;
    
    // 地址: IP:Port 或 hostname
    const address = slave.ip && slave.grpc_port 
        ? `${slave.ip}:${slave.grpc_port}` 
        : (slave.hostname || DEFAULT_VALUES.EMPTY);
    document.getElementById(ELEMENT_IDS.SLAVE_ADDRESS).textContent = address;
    
    document.getElementById(ELEMENT_IDS.SLAVE_REGION).textContent = slave.region || DEFAULT_VALUES.DEFAULT_REGION;
    document.getElementById(ELEMENT_IDS.SLAVE_REGISTERED).textContent = formatTime(slave.registered_at);
    document.getElementById(ELEMENT_IDS.SLAVE_HEARTBEAT).textContent = formatTimeAgo(slave.last_heartbeat);
    
    // 健康状态
    const healthEl = document.getElementById(ELEMENT_IDS.SLAVE_HEALTH);
    const failures = slave.health_check_fail || slave.consecutive_failures || 0;
    const healthClass = getHealthClass(failures);
    const healthText = getHealthText(failures);
    healthEl.innerHTML = `<span class="health-indicator ${healthClass}"></span>${healthText}`;
    
    // 当前任务
    const currentTaskEl = document.getElementById(ELEMENT_IDS.SLAVE_CURRENT_TASK);
    const currentTaskId = slave.current_task_id || slave.currentTaskID;
    if (currentTaskId) {
        currentTaskEl.innerHTML = `<a href="/distributed/tasks/${currentTaskId}">${currentTaskId}</a>`;
    } else {
        currentTaskEl.textContent = DEFAULT_VALUES.NO_TASK;
    }
    
    // 系统资源 - 优先使用 resource_usage,其次 metrics
    const resources = slave.resource_usage || slave.metrics || {};
    const cpuCores = slave.cpu_cores || resources.cpu_cores || DEFAULT_VALUES.EMPTY;
    const cpuUsage = resources.cpu_usage;
    const memoryTotal = slave.memory || resources.memory_total;
    const memoryUsage = resources.memory_usage;
    
    document.getElementById(ELEMENT_IDS.SLAVE_CPU_CORES).textContent = cpuCores;
    document.getElementById(ELEMENT_IDS.SLAVE_CPU_USAGE).textContent = cpuUsage 
        ? `${cpuUsage.toFixed(1)}%` 
        : DEFAULT_VALUES.EMPTY;
    document.getElementById(ELEMENT_IDS.SLAVE_MEMORY_TOTAL).textContent = formatBytes(memoryTotal);
    document.getElementById(ELEMENT_IDS.SLAVE_MEMORY_USAGE).textContent = memoryUsage 
        ? `${memoryUsage.toFixed(1)}%` 
        : DEFAULT_VALUES.EMPTY;
    
    // 任务历史
    const tasks = slave.task_history || slave.running_tasks || [];
    document.getElementById(ELEMENT_IDS.TASK_COUNT).textContent = tasks.length;
    
    const taskList = document.getElementById(ELEMENT_IDS.TASK_LIST);
    if (tasks.length > 0) {
        // 如果是字符串数组(running_tasks),转换为对象
        const taskItems = typeof tasks[0] === 'string' 
            ? tasks.map(taskId => ({ id: taskId, state: 'running' }))
            : tasks;
            
        taskList.innerHTML = taskItems.map(task => `
            <div class="task-card">
                <div class="task-card-header">
                    <strong>
                        <a href="/distributed/tasks/${task.id}" style="text-decoration: none; color: #4285f4;">
                            ${task.id}
                        </a>
                    </strong>
                    <span class="status-badge status-${task.state || 'running'}">${task.state || 'running'}</span>
                </div>
                ${task.started_at ? `
                    <div style="color: #666; font-size: 14px;">
                        ${formatTime(task.started_at)} ~ ${formatTime(task.completed_at)}
                    </div>
                ` : ''}
            </div>
        `).join('');
    } else {
        taskList.innerHTML = '<p style="color: #999;">暂无任务历史</p>';
    }
}

// 获取健康状态样式
function getHealthClass(failures) {
    if (failures === 0) return 'health-healthy';
    if (failures < HEALTH_THRESHOLDS.WARNING) return 'health-warning';
    return 'health-critical';
}

// 获取健康状态文本
function getHealthText(failures) {
    if (failures === 0) return '健康';
    if (failures < HEALTH_THRESHOLDS.WARNING) return `警告 (${failures} 次失败)`;
    return `严重 (${failures} 次失败)`;
}

// 格式化时间
function formatTime(timestamp) {
    if (!timestamp) return '-';
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

// 格式化相对时间
function formatTimeAgo(timestamp) {
    if (!timestamp) return '-';
    const now = new Date();
    const date = new Date(timestamp);
    const seconds = Math.floor((now - date) / 1000);
    
    if (seconds < 60) return `${seconds}秒前`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟前`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}小时前`;
    return `${Math.floor(seconds / 86400)}天前`;
}

// 格式化字节大小
function formatBytes(bytes) {
    if (!bytes || bytes === 0) return DEFAULT_VALUES.EMPTY;
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    
    while (size >= 1024 && unitIndex < units.length - 1) {
        size /= 1024;
        unitIndex++;
    }
    
    return `${size.toFixed(2)} ${units[unitIndex]}`;
}

// 显示错误
function showError(message) {
    const container = document.querySelector('.detail-container');
    container.innerHTML = `
        <div style="text-align: center; padding: 60px 20px;">
            <div style="font-size: 48px; margin-bottom: 20px;">⚠️</div>
            <h2 style="color: #f45c43; margin-bottom: 12px;">加载失败</h2>
            <p style="color: #666; margin-bottom: 24px;">${message}</p>
            <a href="/distributed" class="btn btn-primary">返回列表</a>
        </div>
    `;
}
