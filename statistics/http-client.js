// HTTP 请求封装工具
// 提供统一的请求接口、日志追踪、错误处理

// ============ 配置常量 ============
const HTTP_CLIENT_CONFIG = {
    LOG_ENABLED: true,              // 是否启用日志
    LOG_REQUEST: true,              // 是否记录请求
    LOG_RESPONSE: true,             // 是否记录响应
    LOG_ERROR: true,                // 是否记录错误
    TIMEOUT: 30000,                 // 默认超时时间（毫秒）
    RETRY_COUNT: 0,                 // 默认重试次数
    RETRY_DELAY: 1000               // 重试延迟（毫秒）
};

// ============ 日志工具 ============
const Logger = {
    // 生成请求 ID
    generateRequestId() {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    },

    // 请求日志
    logRequest(requestId, method, url, options) {
        if (!HTTP_CLIENT_CONFIG.LOG_ENABLED || !HTTP_CLIENT_CONFIG.LOG_REQUEST) return;
        
        console.group(`📤 [${requestId}] ${method} ${url}`);
        console.log('⏰ 时间:', new Date().toLocaleTimeString());
        if (options.headers) {
            console.log('📋 Headers:', options.headers);
        }
        if (options.body) {
            try {
                const body = typeof options.body === 'string' ? JSON.parse(options.body) : options.body;
                console.log('📦 Body:', body);
            } catch (e) {
                console.log('📦 Body:', options.body);
            }
        }
        console.groupEnd();
    },

    // 响应日志
    logResponse(requestId, method, url, status, data, duration) {
        if (!HTTP_CLIENT_CONFIG.LOG_ENABLED || !HTTP_CLIENT_CONFIG.LOG_RESPONSE) return;
        
        const emoji = status >= 200 && status < 300 ? '✅' : '⚠️';
        console.group(`${emoji} [${requestId}] ${status} ${method} ${url}`);
        console.log('⏰ 时间:', new Date().toLocaleTimeString());
        console.log('⏱️ 耗时:', `${duration}ms`);
        console.log('📊 Status:', status);
        if (data !== null && data !== undefined) {
            console.log('📥 Response:', data);
        }
        console.groupEnd();
    },

    // 错误日志
    logError(requestId, method, url, error, duration) {
        if (!HTTP_CLIENT_CONFIG.LOG_ENABLED || !HTTP_CLIENT_CONFIG.LOG_ERROR) return;
        
        console.group(`❌ [${requestId}] ERROR ${method} ${url}`);
        console.log('⏰ 时间:', new Date().toLocaleTimeString());
        console.log('⏱️ 耗时:', `${duration}ms`);
        console.error('💥 Error:', error);
        console.groupEnd();
    }
};

// ============ HTTP 客户端 ============
class HttpClient {
    constructor(config = {}) {
        this.config = { ...HTTP_CLIENT_CONFIG, ...config };
    }

    // 通用请求方法
    async request(method, url, options = {}) {
        const requestId = Logger.generateRequestId();
        const startTime = Date.now();

        // 合并配置
        const fetchOptions = {
            method: method.toUpperCase(),
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        };

        // 记录请求日志
        Logger.logRequest(requestId, method, url, fetchOptions);

        // 执行请求（支持重试）
        let lastError = null;
        const retryCount = options.retry ?? this.config.RETRY_COUNT;

        for (let attempt = 0; attempt <= retryCount; attempt++) {
            try {
                // 超时控制
                const timeout = options.timeout ?? this.config.TIMEOUT;
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), timeout);

                const response = await fetch(url, {
                    ...fetchOptions,
                    signal: controller.signal
                });

                clearTimeout(timeoutId);

                const duration = Date.now() - startTime;

                // 解析响应
                let data = null;
                const contentType = response.headers.get('content-type');
                
                if (contentType?.includes('application/json')) {
                    data = await response.json();
                } else if (contentType?.includes('text/')) {
                    data = await response.text();
                } else {
                    data = await response.blob();
                }

                // 记录响应日志
                Logger.logResponse(requestId, method, url, response.status, data, duration);

                // 检查 HTTP 状态
                if (!response.ok) {
                    throw new HttpError(
                        data?.message || data || `HTTP ${response.status}`,
                        response.status,
                        data,
                        requestId
                    );
                }

                return {
                    ok: true,
                    status: response.status,
                    data: data,
                    headers: response.headers,
                    requestId: requestId,
                    duration: duration
                };

            } catch (error) {
                lastError = error;
                const duration = Date.now() - startTime;

                // 超时错误
                if (error.name === 'AbortError') {
                    lastError = new HttpError(
                        `请求超时 (${options.timeout ?? this.config.TIMEOUT}ms)`,
                        0,
                        null,
                        requestId
                    );
                }

                // 如果还有重试次数，延迟后继续
                if (attempt < retryCount) {
                    const delay = options.retryDelay ?? this.config.RETRY_DELAY;
                    console.warn(`🔄 [${requestId}] 重试 ${attempt + 1}/${retryCount}，延迟 ${delay}ms`);
                    await new Promise(resolve => setTimeout(resolve, delay));
                    continue;
                }

                // 记录错误日志
                Logger.logError(requestId, method, url, lastError, duration);

                throw lastError;
            }
        }
    }

    // GET 请求
    async get(url, options = {}) {
        return this.request('GET', url, options);
    }

    // POST 请求
    async post(url, body, options = {}) {
        return this.request('POST', url, {
            ...options,
            body: JSON.stringify(body)
        });
    }

    // PUT 请求
    async put(url, body, options = {}) {
        return this.request('PUT', url, {
            ...options,
            body: JSON.stringify(body)
        });
    }

    // DELETE 请求
    async delete(url, options = {}) {
        return this.request('DELETE', url, options);
    }

    // PATCH 请求
    async patch(url, body, options = {}) {
        return this.request('PATCH', url, {
            ...options,
            body: JSON.stringify(body)
        });
    }
}

// ============ 自定义错误类 ============
class HttpError extends Error {
    constructor(message, status, data, requestId) {
        super(message);
        this.name = 'HttpError';
        this.status = status;
        this.data = data;
        this.requestId = requestId;
    }

    toString() {
        return `[${this.requestId}] ${this.name}: ${this.message} (HTTP ${this.status})`;
    }
}

// ============ 导出 ============
// 默认实例
const httpClient = new HttpClient();

// 便捷方法（兼容旧代码）
const http = {
    get: (url, options) => httpClient.get(url, options),
    post: (url, body, options) => httpClient.post(url, body, options),
    put: (url, body, options) => httpClient.put(url, body, options),
    delete: (url, options) => httpClient.delete(url, options),
    patch: (url, body, options) => httpClient.patch(url, body, options),
    
    // 创建新实例
    create: (config) => new HttpClient(config),
    
    // 配置
    config: HTTP_CLIENT_CONFIG,
    
    // 错误类
    HttpError: HttpError
};

// 支持浏览器和 Node.js
if (typeof window !== 'undefined') {
    window.httpClient = httpClient;
    window.http = http;
    window.HttpError = HttpError;
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { httpClient, http, HttpClient, HttpError };
}
