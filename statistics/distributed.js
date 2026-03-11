// 分布式管理页面 JavaScript

// ============ CDN URLs 常量 ============
const CDN_URLS = {
    MONACO_EDITOR_PATH: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs'
};

// ============ 配置常量 ============
const CONFIG_DEFAULTS = {
    PROTOCOL: 'http',
    URL: 'https://api.example.com',
    METHOD: 'GET',
    CONCURRENCY: 2,
    REQUESTS: 10,
    DURATION: 60,
    TIMEOUT: 30,
    RAMP_UP: 0
};

const PROTOCOL_OPTIONS = ['http', 'grpc', 'websocket'];
const VALID_PROTOCOLS = ['http', 'https', 'grpc', 'websocket'];
const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

// ============ 常用 HTTP Headers 常量 ============
const COMMON_HEADERS = [
    'Accept',
    'Accept-Encoding',
    'Accept-Language',
    'Authorization',
    'Cache-Control',
    'Connection',
    'Content-Length',
    'Content-Type',
    'Cookie',
    'Host',
    'Origin',
    'Referer',
    'User-Agent',
    'X-Api-Key',
    'X-Auth-Token',
    'X-Requested-With',
    'X-Request-ID',
    'X-Trace-ID'
];

// ============ 编辑器配置常量 ============
const EDITOR_CONFIG = {
    LANGUAGE: 'yaml',
    THEME: 'vs',
    FONT_SIZE: 13,
    HEIGHT: 300
};

// ============ UI 文本常量 ============
const UI_TEXT = {
    // 提示消息
    CONFIRM_START_TASK: '确定要启动任务 {0} 吗？\n\n任务将立即分发到 Slave 节点并开始执行',
    CONFIRM_STOP_TASK: '确定要停止任务 {0} 吗？\n\n将通知所有 Slave 节点停止执行该任务',
    CONFIRM_RETRY_TASK: '确定要重试任务 {0} 吗？\n这将创建一个新的任务副本。',
    TASK_CREATED: '任务创建成功！任务ID: {0}',
    TASK_STARTED: '任务启动成功！',
    TASK_STOPPED: '任务已停止！',
    TASK_RETRIED: '✅ 任务重试成功！\n\n原任务 ID: {0}\n新任务 ID: {1}\n\n新任务已创建，状态为 pending，可在任务列表中查看。',
    
    // 错误消息
    ERROR_LOAD_SLAVES: '加载 Slave 列表失败',
    ERROR_LOAD_TASKS: '加载任务列表失败',
    ERROR_CREATE_TASK: '创建任务失败: {0}',
    ERROR_START_TASK: '启动任务失败: {0}',
    ERROR_STOP_TASK: '停止任务失败: {0}',
    ERROR_RETRY_TASK: '重试任务失败: {0}',
    ERROR_RETRY_TASK: '重试任务失败: {0}',
    ERROR_EDITOR_NOT_INIT: '编辑器未初始化',
    ERROR_CONFIG_EMPTY: '请输入配置内容',
    ERROR_CONFIG_FORMAT: '配置格式错误: {0}',
    ERROR_FORMAT_CONVERT: '格式转换失败: {0}',
    ERROR_UNSUPPORTED_FORMAT: '不支持的格式',
    ERROR_MISSING_FIELDS: '缺少必填字段: {0}',
    ERROR_INVALID_PROTOCOL: '无效的协议类型。支持: {0}',
    ERROR_INVALID_CONCURRENCY: '并发数必须大于 0',
    
    // 成功消息
    SUCCESS_CONFIG_FORMATTED: '配置已格式化',
    SUCCESS_CONFIG_VALID: '✅ 配置校验通过！所有字段格式正确。',
    SUCCESS_IMPORT_FROM_FORM: '✅ 已从表单导入配置',
    
    // 空状态
    EMPTY_SLAVES: '暂无 Slave 节点',
    EMPTY_SLAVES_TIP: '请启动 Slave 节点并连接到 Master',
    EMPTY_TASKS: '暂无任务',
    EMPTY_TASKS_FILTERED: '暂无{0}任务',
    
    // 其他
    DEFAULT_VALUE: '-',
    DEFAULT_REGION: 'default',
    AUTO_ASSIGN: '自动分配（全部节点）',
    
    // 按钮文本
    BTN_DETAIL: '详情',
    BTN_REALTIME_REPORT: '实时报告',
    BTN_START: '启动',
    BTN_STOP: '停止',
    
    // 图标
    ICON_SLAVE: '🤖',
    ICON_TASK: '📊'
};

// ============ API 端点常量 ============
const API_ENDPOINTS = {
    SLAVES: '/api/v1/slaves',
    TASKS: '/api/v1/tasks',
    TASK_START: '/api/v1/tasks/{0}/start',
    TASK_STOP: '/api/v1/tasks/{0}',
    TASK_RETRY: '/api/v1/tasks/{0}/retry',
    SLAVE_DETAIL: '/distributed/slaves/{0}',
    TASK_DETAIL: '/distributed/tasks/{0}',
    REALTIME_REPORT: '/realtime?slave_id={0}'
};

// ============ CSS 类名常量 ============
const CSS_CLASSES = {
    ACTIVE: 'active',
    SLAVE_CARD: 'slave-card',
    TASK_ITEM: 'task-item',
    STATUS_PREFIX: 'status-',
    EMPTY_STATE: 'empty-state',
    VALIDATION_SUCCESS: 'validation-message success',
    VALIDATION_ERROR: 'validation-message error',
    KEY_VALUE_ITEM: 'key-value-item',
    VERIFY_ITEM: 'verify-item',
    EXTRACTOR_ITEM: 'extractor-item'
};

// ============ DOM 选择器常量 ============
const SELECTORS = {
    FILTER_BTN: '.filter-btn',
    MODE_BTN_FORM: '[data-mode="form"]',
    MODE_BTN_CODE: '[data-mode="code"]',
    FORM_REQUIRED: '#formMode input[required], #formMode select[required], #formMode textarea[required]',
    HEADERS_LIST: '#headersList .key-value-item',
    VERIFY_LIST: '#verifyList .verify-item',
    EXTRACTOR_LIST: '#extractorsList .extractor-item'
};

// ============ Monaco Editor 配置 ============
function initMonacoEditor() {
    require.config({ paths: { vs: CDN_URLS.MONACO_EDITOR_PATH } });
    
    require(['vs/editor/editor.main'], function() {
        // 创建编辑器实例
        configEditor = monaco.editor.create(document.getElementById('configEditor'), {
            value: getDefaultConfig(),
            language: EDITOR_CONFIG.LANGUAGE,
            theme: EDITOR_CONFIG.THEME,
            automaticLayout: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            fontSize: EDITOR_CONFIG.FONT_SIZE,
            lineNumbers: 'on',
            roundedSelection: true,
            readOnly: false,
            cursorStyle: 'line',
            wordWrap: 'on',
            folding: true,
            formatOnPaste: true,
            formatOnType: true
        });
        
        // 监听内容变化，自动校验
        configEditor.onDidChangeModelContent(() => {
            clearValidationMessage();
        });
    });
}

// 获取默认配置模板
function getDefaultConfig() {
    return `protocol: ${CONFIG_DEFAULTS.PROTOCOL}
url: ${CONFIG_DEFAULTS.URL}
method: ${CONFIG_DEFAULTS.METHOD}
concurrency: ${CONFIG_DEFAULTS.CONCURRENCY}
requests: ${CONFIG_DEFAULTS.REQUESTS}
duration: ${CONFIG_DEFAULTS.DURATION}s
timeout: ${CONFIG_DEFAULTS.TIMEOUT}s
headers:
  Content-Type: application/json`;
}

// 切换编辑器语言
function switchEditorLanguage(lang) {
    if (!configEditor) return;
    
    const currentValue = configEditor.getValue();
    const currentLang = monaco.editor.getModel(configEditor.getModel().uri).getLanguageId();
    
    // 如果目标语言和当前语言相同，无需转换
    if (currentLang === lang) return;
    
    let newValue = currentValue;
    
    try {
        if (lang === 'yaml' && currentLang === 'json') {
            // JSON 转 YAML
            const jsonObj = JSON.parse(currentValue);
            newValue = jsonToYaml(jsonObj);
        } else if (lang === 'json' && currentLang === 'yaml') {
            // YAML 转 JSON（提示用户 YAML 库可能未加载）
            if (typeof jsyaml !== 'undefined') {
                const yamlObj = jsyaml.load(currentValue);
                newValue = JSON.stringify(yamlObj, null, 2);
            } else {
                // YAML 库未加载，提示用户手动编辑或继续使用 YAML
                if (!confirm('YAML 解析库未加载，无法自动转换为 JSON。\n\n您可以：\n1. 点击"确定"切换到 JSON 编辑器手动输入\n2. 点击"取消"继续使用 YAML（后端支持 YAML 解析）')) {
                    return; // 用户选择继续使用 YAML
                }
                // 用户选择切换到 JSON，清空内容让其手动输入
                newValue = '{\n  \n}';
            }
        }
        
        monaco.editor.setModelLanguage(configEditor.getModel(), lang);
        configEditor.setValue(newValue);
    } catch (e) {
        showValidationError(UI_TEXT.ERROR_FORMAT_CONVERT.replace('{0}', e.message));
    }
}

// 简单的 JSON 转 YAML
function jsonToYaml(obj, indent = 0) {
    const spaces = '  '.repeat(indent);
    let yaml = '';
    
    for (const [key, value] of Object.entries(obj)) {
        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
            yaml += `${spaces}${key}:\n${jsonToYaml(value, indent + 1)}`;
        } else if (Array.isArray(value)) {
            yaml += `${spaces}${key}:\n`;
            value.forEach(item => {
                if (typeof item === 'object') {
                    yaml += `${spaces}  -\n${jsonToYaml(item, indent + 2)}`;
                } else {
                    yaml += `${spaces}  - ${item}\n`;
                }
            });
        } else {
            yaml += `${spaces}${key}: ${value}\n`;
        }
    }
    
    return yaml;
}

// 格式化配置
function formatConfig() {
    if (!configEditor) return;
    
    configEditor.getAction('editor.action.formatDocument').run();
    showValidationSuccess(UI_TEXT.SUCCESS_CONFIG_FORMATTED);
}

// 校验配置
function validateConfig() {
    if (!configEditor) return;
    
    const content = configEditor.getValue().trim();
    if (!content) {
        showValidationError(UI_TEXT.ERROR_CONFIG_EMPTY);
        return;
    }
    
    const lang = monaco.editor.getModel(configEditor.getModel().uri).getLanguageId();
    
    try {
        let configObj;
        
        if (lang === 'json') {
            configObj = JSON.parse(content);
        } else if (lang === 'yaml') {
            // YAML 校验（前端校验为可选，后端也会校验）
            if (typeof jsyaml !== 'undefined') {
                configObj = jsyaml.load(content);
            } else {
                console.warn('YAML 库未加载，跳过前端校验');
                showValidationSuccess('YAML 格式将由后端校验（前端库未加载）');
                return;
            }
        } else {
            throw new Error(UI_TEXT.ERROR_UNSUPPORTED_FORMAT);
        }
        
        // 基本字段校验
        const requiredFields = ['protocol', 'url'];
        const missingFields = requiredFields.filter(field => !configObj[field]);
        
        if (missingFields.length > 0) {
            showValidationError(UI_TEXT.ERROR_MISSING_FIELDS.replace('{0}', missingFields.join(', ')));
            return;
        }
        
        // 协议校验
        if (!VALID_PROTOCOLS.includes(configObj.protocol?.toLowerCase())) {
            showValidationError(UI_TEXT.ERROR_INVALID_PROTOCOL.replace('{0}', VALID_PROTOCOLS.join(', ')));
            return;
        }
        
        // 并发数校验
        if (configObj.concurrency && configObj.concurrency < 1) {
            showValidationError(UI_TEXT.ERROR_INVALID_CONCURRENCY);
            return;
        }
        
        showValidationSuccess(UI_TEXT.SUCCESS_CONFIG_VALID);
        
    } catch (e) {
        showValidationError(`${lang.toUpperCase()} ${UI_TEXT.ERROR_CONFIG_FORMAT.replace('{0}', e.message)}`);
    }
}

// 显示校验成功消息
function showValidationSuccess(message) {
    const msgEl = document.getElementById('validationMessage');
    msgEl.className = 'validation-message success';
    msgEl.textContent = message;
    
    setTimeout(() => {
        msgEl.style.display = 'none';
    }, 3000);
}

// 显示校验错误消息
function showValidationError(message) {
    const msgEl = document.getElementById('validationMessage');
    msgEl.className = 'validation-message error';
    msgEl.innerHTML = message.includes('\n') 
        ? `<strong>❌ 校验失败:</strong><pre>${message}</pre>`
        : `<strong>❌ 校验失败:</strong> ${message}`;
}

// 清除校验消息
function clearValidationMessage() {
    const msgEl = document.getElementById('validationMessage');
    msgEl.style.display = 'none';
}

// ============ 表单模式相关功能 ============

// 初始化表单模式
function initFormMode() {
    // 默认显示代码模式（因为更灵活）
    currentMode = 'code';
    switchMode('code');
}

// 切换编辑模式
function switchMode(mode) {
    currentMode = mode;
    
    const formMode = document.getElementById('formMode');
    const codeMode = document.getElementById('codeMode');
    const formBtn = document.querySelector(SELECTORS.MODE_BTN_FORM);
    const codeBtn = document.querySelector(SELECTORS.MODE_BTN_CODE);
    
    if (mode === 'form') {
        formMode.style.display = 'block';
        codeMode.style.display = 'none';
        formBtn.classList.add(CSS_CLASSES.ACTIVE);
        codeBtn.classList.remove(CSS_CLASSES.ACTIVE);
        
        // 启用表单字段的 required 验证
        enableFormValidation(true);
    } else {
        formMode.style.display = 'none';
        codeMode.style.display = 'block';
        formBtn.classList.remove(CSS_CLASSES.ACTIVE);
        codeBtn.classList.add(CSS_CLASSES.ACTIVE);
        
        // 禁用表单字段的 required 验证
        enableFormValidation(false);
        
        // 切换到代码模式时，从表单导入数据
        importFromForm();
    }
}

// 启用/禁用表单验证
function enableFormValidation(enable) {
    const formInputs = document.querySelectorAll(SELECTORS.FORM_REQUIRED);
    formInputs.forEach(input => {
        if (enable) {
            input.setAttribute('required', 'required');
        } else {
            input.removeAttribute('required');
        }
    });
}

// 从表单生成配置对象
function generateConfigFromForm() {
    const config = {
        protocol: document.getElementById('protocol').value,
        method: document.getElementById('method').value,
        url: document.getElementById('url').value,
        concurrency: parseInt(document.getElementById('concurrency').value),
        requests: parseInt(document.getElementById('requests').value),
        duration: `${parseInt(document.getElementById('duration').value)}s`, // 转换为Go的time.Duration格式
        timeout: `${parseInt(document.getElementById('timeout').value)}s`    // 转换为Go的time.Duration格式
    };
    
    // 路径
    const path = document.getElementById('path').value.trim();
    if (path) {
        config.path = path;
    }
    
    // 渐进启动
    const rampUp = parseInt(document.getElementById('rampUp').value);
    if (rampUp > 0) {
        if (!config.advanced) {
            config.advanced = {};
        }
        config.advanced.ramp_up = `${rampUp}s`; // 转换为Go的time.Duration格式
    }
    
    // Headers
    const headers = {};
    document.querySelectorAll(SELECTORS.HEADERS_LIST).forEach(item => {
        const inputs = item.querySelectorAll('input');
        const key = inputs[0].value.trim();
        const value = inputs[1].value.trim();
        if (key) {
            headers[key] = value;
        }
    });
    if (Object.keys(headers).length > 0) {
        config.headers = headers;
    }
    
    // Body
    const body = document.getElementById('body').value.trim();
    if (body) {
        try {
            config.body = JSON.parse(body);
        } catch (e) {
            config.body = body;
        }
    }
    
    // 校验规则
    // 注意：VerifyConfig 的 JSON 字段是 type/jsonpath/expect，期望值统一写 expect，
    // 类型值必须用后端常量（STATUS_CODE/JSONPATH/CONTAINS），否则后端走 default 分支静默放行
    const verifyRules = [];
    document.querySelectorAll(SELECTORS.VERIFY_LIST).forEach(item => {
        const type = item.querySelector('[name="verifyType"]').value;
        const rule = { type: type };

        // 三种类型各只有一个输入框参与期望值，统一落到 expect
        if (type === 'STATUS_CODE') {
            const statusCode = item.querySelector('[name="statusCode"]')?.value;
            if (statusCode) rule.expect = parseInt(statusCode, 10);
        } else if (type === 'JSONPATH') {
            const jsonPath = item.querySelector('[name="jsonPath"]')?.value;
            if (jsonPath) rule.jsonpath = jsonPath;

            const expectedValue = item.querySelector('[name="expectedValue"]')?.value;
            if (expectedValue) rule.expect = expectedValue;
        } else if (type === 'CONTAINS') {
            const contains = item.querySelector('[name="contains"]')?.value;
            if (contains) rule.expect = contains;
        }

        verifyRules.push(rule);
    });
    if (verifyRules.length > 0) {
        config.verify = verifyRules;
    }
    
    // 提取器
    const extractors = [];
    document.querySelectorAll(SELECTORS.EXTRACTOR_LIST).forEach(item => {
        const extractor = {
            name: item.querySelector('[name="extractorName"]').value,
            type: item.querySelector('[name="extractorType"]').value
        };
        
        const jsonPath = item.querySelector('[name="extractorJsonPath"]')?.value;
        if (jsonPath) extractor.jsonpath = jsonPath;
        
        const regex = item.querySelector('[name="extractorRegex"]')?.value;
        if (regex) extractor.regex = regex;
        
        const header = item.querySelector('[name="extractorHeader"]')?.value;
        if (header) extractor.header = header;
        
        extractors.push(extractor);
    });
    if (extractors.length > 0) {
        config.extractors = extractors;
    }
    
    return config;
}

// 从表单导入到编辑器
function importFromForm() {
    if (!configEditor) return;
    
    const config = generateConfigFromForm();
    const lang = monaco.editor.getModel(configEditor.getModel().uri).getLanguageId();
    
    if (lang === 'yaml') {
        configEditor.setValue(jsonToYaml(config));
    } else {
        configEditor.setValue(JSON.stringify(config, null, 2));
    }
    
    showValidationSuccess(UI_TEXT.SUCCESS_IMPORT_FROM_FORM);
}

// 预览生成的配置
function previewConfig() {
    const config = generateConfigFromForm();
    const yaml = jsonToYaml(config);
    const json = JSON.stringify(config, null, 2);
    
    // 创建模态框显示预览
    const modal = document.createElement('div');
    modal.style.cssText = `
        position: fixed; top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(0,0,0,0.5); z-index: 9999;
        display: flex; align-items: center; justify-content: center;
    `;
    
    const content = document.createElement('div');
    content.style.cssText = `
        background: white; padding: 30px; border-radius: 12px;
        max-width: 800px; max-height: 80vh; overflow: auto;
        box-shadow: 0 10px 40px rgba(0,0,0,0.3);
    `;
    
    content.innerHTML = `
        <h3 style="margin-bottom: 20px;">📋 配置预览</h3>
        <div style="display: flex; gap: 10px; margin-bottom: 15px;">
            <button onclick="this.parentElement.parentElement.querySelector('#previewJson').style.display='block';
                            this.parentElement.parentElement.querySelector('#previewYaml').style.display='none';"
                    style="padding: 8px 16px; border: 1px solid #667eea; background: white; color: #667eea; border-radius: 4px; cursor: pointer;">
                JSON
            </button>
            <button onclick="this.parentElement.parentElement.querySelector('#previewJson').style.display='none';
                            this.parentElement.parentElement.querySelector('#previewYaml').style.display='block';"
                    style="padding: 8px 16px; border: 1px solid #667eea; background: white; color: #667eea; border-radius: 4px; cursor: pointer;">
                YAML
            </button>
        </div>
        <pre id="previewJson" style="background: #f5f5f5; padding: 15px; border-radius: 6px; overflow: auto; max-height: 400px;">${json}</pre>
        <pre id="previewYaml" style="background: #f5f5f5; padding: 15px; border-radius: 6px; overflow: auto; max-height: 400px; display: none;">${yaml}</pre>
        <div style="margin-top: 20px; text-align: right;">
            <button onclick="this.closest('[style*=fixed]').remove()"
                    style="padding: 10px 20px; background: #667eea; color: white; border: none; border-radius: 6px; cursor: pointer;">
                关闭
            </button>
        </div>
    `;
    
    modal.appendChild(content);
    document.body.appendChild(modal);
    
    modal.onclick = (e) => {
        if (e.target === modal) modal.remove();
    };
}

// 添加 Header
function addHeader() {
    const container = document.getElementById('headersList');
    const item = document.createElement('div');
    item.className = 'key-value-item';
    
    // 复用全局的 datalist
    item.innerHTML = `
        <input type="text" list="headerKeyList-default" placeholder="选择或输入 Key">
        <input type="text" placeholder="Value">
        <button type="button" class="btn-remove" onclick="removeItem(this)">×</button>
    `;
    container.appendChild(item);
}

// 添加校验规则
function addVerify() {
    const container = document.getElementById('verifyList');
    const item = document.createElement('div');
    item.className = 'verify-item';
    item.innerHTML = `
        <button type="button" class="btn-remove" onclick="removeItem(this.parentElement)">×</button>
        <div class="form-row">
            <div class="form-group">
                <label>校验类型</label>
                <select name="verifyType" onchange="toggleVerifyFields(this)">
                    <option value="STATUS_CODE">状态码</option>
                    <option value="JSONPATH">JSON路径</option>
                    <option value="CONTAINS">包含文本</option>
                </select>
            </div>
            <div class="form-group verify-status">
                <label>期望状态码</label>
                <input type="number" name="statusCode" value="200" placeholder="200">
            </div>
        </div>
        <div class="form-group verify-jsonpath" style="display: none;">
            <label>JSON路径</label>
            <input type="text" name="jsonPath" placeholder="$.data.status">
        </div>
        <div class="form-group verify-jsonpath" style="display: none;">
            <label>期望值</label>
            <input type="text" name="expectedValue" placeholder="success">
        </div>
        <div class="form-group verify-contains" style="display: none;">
            <label>包含文本</label>
            <input type="text" name="contains" placeholder="success">
        </div>
    `;
    container.appendChild(item);
}

// 切换校验字段显示
function toggleVerifyFields(select) {
    const item = select.closest('.verify-item');
    const type = select.value;
    
    item.querySelectorAll('.verify-status, .verify-jsonpath, .verify-contains').forEach(el => {
        el.style.display = 'none';
    });
    
    if (type === 'STATUS_CODE') {
        item.querySelectorAll('.verify-status').forEach(el => el.style.display = 'block');
    } else if (type === 'JSONPATH') {
        item.querySelectorAll('.verify-jsonpath').forEach(el => el.style.display = 'block');
    } else if (type === 'CONTAINS') {
        item.querySelectorAll('.verify-contains').forEach(el => el.style.display = 'block');
    }
}

// 添加提取器
function addExtractor() {
    const container = document.getElementById('extractorsList');
    const item = document.createElement('div');
    item.className = 'extractor-item';
    item.innerHTML = `
        <button type="button" class="btn-remove" onclick="removeItem(this.parentElement)">×</button>
        <div class="form-row">
            <div class="form-group">
                <label>变量名称</label>
                <input type="text" name="extractorName" placeholder="user_id" required>
            </div>
            <div class="form-group">
                <label>提取类型</label>
                <select name="extractorType" onchange="toggleExtractorFields(this)">
                    <option value="jsonpath">JSON路径</option>
                    <option value="regex">正则表达式</option>
                    <option value="header">响应头</option>
                </select>
            </div>
        </div>
        <div class="form-group extractor-jsonpath">
            <label>JSON路径</label>
            <input type="text" name="extractorJsonPath" placeholder="$.data.user_id">
        </div>
        <div class="form-group extractor-regex" style="display: none;">
            <label>正则表达式</label>
            <input type="text" name="extractorRegex" placeholder="user_id=(\\d+)">
        </div>
        <div class="form-group extractor-header" style="display: none;">
            <label>响应头名称</label>
            <input type="text" name="extractorHeader" placeholder="X-Request-ID">
        </div>
    `;
    container.appendChild(item);
}

// 切换提取器字段显示
function toggleExtractorFields(select) {
    const item = select.closest('.extractor-item');
    const type = select.value;
    
    item.querySelectorAll('.extractor-jsonpath, .extractor-regex, .extractor-header').forEach(el => {
        el.style.display = 'none';
    });
    
    if (type === 'jsonpath') {
        item.querySelectorAll('.extractor-jsonpath').forEach(el => el.style.display = 'block');
    } else if (type === 'regex') {
        item.querySelectorAll('.extractor-regex').forEach(el => el.style.display = 'block');
    } else if (type === 'header') {
        item.querySelectorAll('.extractor-header').forEach(el => el.style.display = 'block');
    }
}

// 移除项
function removeItem(element) {
    element.remove();
}

// ============ 元素 ID 常量 ============
const ELEMENT_IDS = {
    // 统计
    TOTAL_SLAVES: 'totalSlaves',
    IDLE_COUNT: 'idleCount',
    RUNNING_COUNT: 'runningCount',
    OFFLINE_COUNT: 'offlineCount',
    ERROR_COUNT: 'errorCount',
    
    // 容器
    SLAVE_GRID: 'slaveGrid',
    TASK_LIST: 'taskList',
    TASK_FORM: 'taskForm',
    
    // 表单
    CONFIG_FILE: 'configFile'
};

// ============ 状态映射常量 ============
const SLAVE_STATE_MAP = {
    'idle': '空闲',
    'running': '运行中',
    'stopping': '停止中',
    'error': '错误',
    'offline': '离线',
    'busy': '繁忙',
    'overloaded': '过载',
    'unreachable': '不可达'
};

const OFFLINE_STATES = ['offline', 'error', 'unreachable'];

const TASK_STATUS_MAP = {
    'pending': '待执行',
    'running': '运行中',
    'completed': '已完成',
    'failed': '失败',
    'stopped': '已停止',
    'cancelled': '已取消'
};

const FILTER_LABELS = {
    'all': '全部',
    'running': '运行中',
    'completed': '已完成',
    'failed': '失败',
    'stopped': '已停止'
};

// ============ 时间常量 ============
const TIME_CONSTANTS = {
    REFRESH_INTERVAL: 5000,      // 自动刷新间隔(毫秒)
    JUST_NOW_THRESHOLD: 60,      // "刚刚"阈值(秒)
    MINUTES_THRESHOLD: 3600,     // "分钟前"阈值(秒)
    HOURS_THRESHOLD: 86400       // "小时前"阈值(秒)
};

let currentFilter = 'all';
let refreshInterval = null;
let configEditor = null; // Monaco Editor 实例
let currentMode = 'code'; // 当前模式: form | code

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    initMonacoEditor();
    initTaskForm();
    initFilters();
    initFormMode();
    loadData();
    startAutoRefresh();
    
    // 默认代码模式，禁用表单验证
    enableFormValidation(false);
});

// 初始化任务表单
function initTaskForm() {
    const form = document.getElementById(ELEMENT_IDS.TASK_FORM);
    form.addEventListener('submit', async function(e) {
        e.preventDefault();
        await createTask();
    });
}

// 初始化筛选器
function initFilters() {
    const filters = document.querySelectorAll(SELECTORS.FILTER_BTN);
    filters.forEach(btn => {
        btn.addEventListener('click', function() {
            filters.forEach(b => b.classList.remove(CSS_CLASSES.ACTIVE));
            this.classList.add(CSS_CLASSES.ACTIVE);
            currentFilter = this.dataset.filter;
            loadTasks();
        });
    });
}

// 加载所有数据
async function loadData() {
    await Promise.all([
        loadSlaves(),
        loadTasks()
    ]);
}

// 加载 Slave 列表
async function loadSlaves() {
    try {
        const res = await http.get(API_ENDPOINTS.SLAVES);
        const data = res.data;
        
        updateSlaveStats(data.stats || {});
        renderSlaves(data.slaves || []);
    } catch (error) {
        console.error('加载 Slave 列表失败:', error);
        showError(UI_TEXT.ERROR_LOAD_SLAVES);
    }
}

// 更新 Slave 统计（直接使用后端返回的统计数据）
function updateSlaveStats(stats) {
    const total = (stats.idle || 0) + (stats.running || 0) + (stats.offline || 0) + (stats.error || 0);
    document.getElementById(ELEMENT_IDS.TOTAL_SLAVES).textContent = total;
    document.getElementById(ELEMENT_IDS.IDLE_COUNT).textContent = stats.idle || 0;
    document.getElementById(ELEMENT_IDS.RUNNING_COUNT).textContent = stats.running || 0;
    document.getElementById(ELEMENT_IDS.OFFLINE_COUNT).textContent = stats.offline || 0;
    document.getElementById(ELEMENT_IDS.ERROR_COUNT).textContent = stats.error || 0;
}

// 渲染 Slave 卡片
function renderSlaves(slaves) {
    const grid = document.getElementById(ELEMENT_IDS.SLAVE_GRID);
    
    if (slaves.length === 0) {
        grid.innerHTML = `
            <div class="${CSS_CLASSES.EMPTY_STATE}" style="grid-column: 1 / -1;">
                <div class="empty-state-icon">🤖</div>
                <p>${UI_TEXT.EMPTY_SLAVES}</p>
                <p style="color: #ccc; font-size: 0.9em; margin-top: 10px;">${UI_TEXT.EMPTY_SLAVES_TIP}</p>
            </div>
        `;
        return;
    }
    
    grid.innerHTML = slaves.map(slave => {
        // 判断在线状态: idle/running/busy 都算在线
        const isOnline = !OFFLINE_STATES.includes(slave.state);
        const stateText = SLAVE_STATE_MAP[slave.state] || slave.state;
        
        return `
        <div class="${CSS_CLASSES.SLAVE_CARD} ${slave.state}">
            <div class="slave-header">
                <span class="slave-id">${escapeHtml(slave.id)}</span>
                <span class="slave-status ${CSS_CLASSES.STATUS_PREFIX}${isOnline ? 'online' : 'offline'}">
                    ${stateText}
                </span>
            </div>
            <div class="slave-info">
                <div><span>📍 区域:</span><span>${escapeHtml(slave.region || UI_TEXT.DEFAULT_REGION)}</span></div>
                <div><span>📊 负载:</span><span>${slave.load || 0}</span></div>
                <div><span>🔢 任务数:</span><span>${slave.running_tasks?.length || 0}</span></div>
                <div><span>💓 心跳:</span><span>${formatTime(slave.last_heartbeat)}</span></div>
            </div>
            <div class="slave-actions">
                <button class="btn-detail" onclick="viewSlaveDetail('${slave.id}')">
                    详情
                </button>
                <button class="btn-report" onclick='viewSlaveReport(${JSON.stringify(slave).replace(/'/g, "&apos;")})'>
                    实时报告
                </button>
            </div>
        </div>
    `;
    }).join('');
}

// 加载任务列表
async function loadTasks() {
    try {
        const res = await http.get(API_ENDPOINTS.TASKS);
        const data = res.data;
        
        renderTasks(data.tasks || []);
    } catch (error) {
        console.error('加载任务列表失败:', error);
        showError(UI_TEXT.ERROR_LOAD_TASKS);
    }
}

// 渲染任务列表
function renderTasks(tasks) {
    const list = document.getElementById(ELEMENT_IDS.TASK_LIST);
    
    // 过滤任务
    let filtered = tasks;
    if (currentFilter !== 'all') {
        filtered = tasks.filter(t => t.state === currentFilter);
    }
    
    if (filtered.length === 0) {
        const message = currentFilter === 'all' 
            ? UI_TEXT.EMPTY_TASKS 
            : UI_TEXT.EMPTY_TASKS_FILTERED.replace('{0}', getFilterLabel(currentFilter));
        list.innerHTML = `
            <div class="${CSS_CLASSES.EMPTY_STATE}">
                <div class="empty-state-icon">📊</div>
                <p>${message}</p>
            </div>
        `;
        return;
    }
    
    list.innerHTML = filtered.map(task => `
        <div class="${CSS_CLASSES.TASK_ITEM}">
            <div class="task-header" onclick="viewTaskDetail('${task.id}')">
                <span class="task-id">${escapeHtml(task.id)}</span>
                <span class="task-status ${task.state}">${getStatusLabel(task.state)}</span>
            </div>
            <div class="task-info" onclick="viewTaskDetail('${task.id}')">
                <div>协议: ${escapeHtml(task.protocol || UI_TEXT.DEFAULT_VALUE)} | 总并发: ${task.total_workers || 0}</div>
                <div>分配节点: ${task.assigned_slaves?.length || 0} 个</div>
            </div>
            <div class="task-actions">
                ${task.state === 'pending' ? `
                    <button class="btn-start" onclick="event.stopPropagation(); startTask('${task.id}')">
                        ▶️ 启动
                    </button>
                ` : ''}
                ${task.state === 'running' ? `
                    <button class="btn-stop" onclick="event.stopPropagation(); stopTask('${task.id}')">
                        ⏸️ 停止
                    </button>
                ` : ''}
                ${task.state === 'completed' || task.state === 'failed' || task.state === 'stopped' ? `
                    <button class="btn-retry" onclick="event.stopPropagation(); retryTask('${task.id}')">
                        🔄 重试
                    </button>
                ` : ''}
            </div>
        </div>
    `).join('');
}

// 创建任务
async function createTask() {
    let config;
    
    // 根据当前模式获取配置
    if (currentMode === 'form') {
        // 从表单生成 JSON 配置
        const configObj = generateConfigFromForm();
        config = JSON.stringify(configObj);
        console.log('生成的配置:', configObj);
    } else {
        // 从编辑器获取配置（支持 JSON 或 YAML）
        if (!configEditor) {
            showError(UI_TEXT.ERROR_EDITOR_NOT_INIT);
            return;
        }
        
        config = configEditor.getValue().trim();
        if (!config) {
            showError(UI_TEXT.ERROR_CONFIG_EMPTY);
            return;
        }
        
        // 简单校验：JSON 模式下检查语法
        const lang = monaco.editor.getModel(configEditor.getModel().uri).getLanguageId();
        if (lang === 'json') {
            try {
                JSON.parse(config);
            } catch (e) {
                showError(UI_TEXT.ERROR_CONFIG_FORMAT.replace('{0}', e.message));
                return;
            }
        }
        // YAML 模式直接发送原始内容，由后端解析
    }
    
    try {
        const requestBody = {
            config_file: config
        };
        
        const res = await http.post(API_ENDPOINTS.TASKS, requestBody);
        const data = res.data;
        
        showSuccess(UI_TEXT.TASK_CREATED.replace('{0}', data.task_id));
        
        // 清空表单
        if (currentMode === 'form') {
            document.getElementById('taskForm').reset();
            document.getElementById('headersList').innerHTML = `
                <div class="key-value-item">
                    <input type="text" list="headerKeyList-default" placeholder="选择或输入 Key" value="Content-Type">
                    <input type="text" placeholder="Value" value="application/json">
                    <button type="button" class="btn-remove" onclick="removeItem(this)">×</button>
                </div>
            `;
            document.getElementById('verifyList').innerHTML = '';
            document.getElementById('extractorsList').innerHTML = '';
        } else {
            configEditor.setValue(getDefaultConfig());
        }
        
        // 刷新任务列表
        await loadTasks();
        
        // 可选：跳转到任务详情
        setTimeout(() => viewTaskDetail(data.task_id), 1500);
        
    } catch (error) {
        console.error('创建任务失败:', error);
        showError(UI_TEXT.ERROR_CREATE_TASK.replace('{0}', error.message));
    }
}

// 查看 Slave 详情
function viewSlaveDetail(slaveId) {
    window.location.href = API_ENDPOINTS.SLAVE_DETAIL.replace('{0}', slaveId);
}

// 查看 Slave 实时报告
function viewSlaveReport(slave) {
    // 构建实时报告 URL，包含 slave_id 和 realtime_url 参数
    const slaveId = typeof slave === 'string' ? slave : slave.id;
    const realtimeUrl = typeof slave === 'object' && slave.realtime_port 
        ? `http://${slave.ip}:${slave.realtime_port}` 
        : '';
    
    let url = `/realtime?slave_id=${slaveId}`;
    if (realtimeUrl) {
        url += `&realtime_url=${encodeURIComponent(realtimeUrl)}`;
    }
    
    window.open(url, '_blank');
}

// 查看任务详情
function viewTaskDetail(taskId) {
    window.location.href = API_ENDPOINTS.TASK_DETAIL.replace('{0}', taskId);
}

// 启动任务
async function startTask(taskId) {
    if (!confirm(UI_TEXT.CONFIRM_START_TASK.replace('{0}', taskId))) {
        return;
    }
    
    try {
        await http.post(API_ENDPOINTS.TASK_START.replace('{0}', taskId), {});
        
        showSuccess(UI_TEXT.TASK_STARTED);
        await loadTasks();
        
    } catch (error) {
        console.error('启动任务失败:', error);
        showError(UI_TEXT.ERROR_START_TASK.replace('{0}', error.message));
    }
}

// 停止任务
async function stopTask(taskId) {
    if (!confirm(UI_TEXT.CONFIRM_STOP_TASK.replace('{0}', taskId))) {
        return;
    }
    
    try {
        await http.delete(API_ENDPOINTS.TASK_STOP.replace('{0}', taskId));
        
        showSuccess(UI_TEXT.TASK_STOPPED);
        await loadTasks();
        
    } catch (error) {
        console.error('停止任务失败:', error);
        showError(UI_TEXT.ERROR_STOP_TASK.replace('{0}', error.message));
    }
}

// 重试任务
async function retryTask(taskId) {
    if (!confirm(UI_TEXT.CONFIRM_RETRY_TASK.replace('{0}', taskId))) {
        return;
    }
    
    try {
        const response = await http.post(API_ENDPOINTS.TASK_RETRY.replace('{0}', taskId), {});
        const data = response.data;
        const newTaskId = data.new_task_id || data.newTaskId;
        const originalTaskId = data.original_task_id || taskId;
        
        showSuccess(UI_TEXT.TASK_RETRIED.replace('{0}', originalTaskId).replace('{1}', newTaskId));
        await loadTasks();
        
    } catch (error) {
        console.error('重试任务失败:', error);
        showError(UI_TEXT.ERROR_RETRY_TASK.replace('{0}', error.message));
    }
}

// 启动自动刷新
function startAutoRefresh() {
    refreshInterval = setInterval(loadData, TIME_CONSTANTS.REFRESH_INTERVAL);
}

// 停止自动刷新
function stopAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
}

// 工具函数
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatTime(timestamp) {
    if (!timestamp) return '-';
    const date = new Date(timestamp);
    const now = new Date();
    const diff = (now - date) / 1000; // 秒
    
    if (diff < TIME_CONSTANTS.JUST_NOW_THRESHOLD) return '刚刚';
    if (diff < TIME_CONSTANTS.MINUTES_THRESHOLD) return Math.floor(diff / 60) + '分钟前';
    if (diff < TIME_CONSTANTS.HOURS_THRESHOLD) return Math.floor(diff / 3600) + '小时前';
    return date.toLocaleString('zh-CN');
}

function getFilterLabel(filter) {
    return FILTER_LABELS[filter] || filter;
}

function getStatusLabel(status) {
    return TASK_STATUS_MAP[status] || status;
}

function showSuccess(message) {
    // 简单的提示实现，可以替换为更好的 UI 组件
    alert('✅ ' + message);
}

function showError(message) {
    alert('❌ ' + message);
}

// 页面卸载时清理
window.addEventListener('beforeunload', function() {
    stopAutoRefresh();
});
