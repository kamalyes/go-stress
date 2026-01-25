// 生成完整请求信息（包含请求、响应、验证等所有信息）
function generateFullRequest(req) {
  const method = (req.method || req.request_method || 'GET').toUpperCase();
  const url = req.url || req.request_url || '';
  const headers = req.headers || req.request_headers || {};
  const body = req.body || req.request_body || '';
  const query = req.query || req.request_query || '';
  
  let text = '========================================\n';
  text += '           完整请求/响应信息\n';
  text += '========================================\n\n';
  
  // === 基本信息 ===
  text += '【基本信息】\n';
  text += 'API名称: ' + (req.api_name || '-') + '\n';
  text += 'Group ID: ' + (req.group_id || '-') + '\n';
  text += '请求时间: ' + (req.timestamp ? new Date(req.timestamp).toLocaleString() : '-') + '\n';
  text += '状态: ' + (req.skipped ? '⏭ 跳过' : (req.success ? '✓ 成功' : '✗ 失败')) + '\n';
  if (req.skip_reason) {
    text += '跳过原因: ' + req.skip_reason + '\n';
  }
  text += '\n';
  
  // === 请求信息 ===
  text += '【请求信息】\n';
  text += 'Method: ' + method + '\n';
  text += 'URL: ' + url + '\n';
  
  if (query) {
    text += 'Query: ' + query + '\n';
  }
  
  text += '\n【请求 Headers】\n';
  if (typeof headers === 'object' && headers !== null && Object.keys(headers).length > 0) {
    Object.entries(headers).forEach(([key, value]) => {
      text += '  ' + key + ': ' + value + '\n';
    });
  } else if (headers) {
    text += headers + '\n';
  } else {
    text += '  (无)\n';
  }
  
  if (body) {
    text += '\n【请求 Body】\n';
    try {
      const parsed = JSON.parse(body);
      text += JSON.stringify(parsed, null, 2) + '\n';
    } catch (e) {
      text += body + '\n';
    }
  }
  
  // === 响应信息 ===
  if (!req.skipped) {
    text += '\n【响应信息】\n';
    text += 'Status Code: ' + (req.status_code || 0) + '\n';
    text += 'Duration: ' + ((req.duration ? req.duration / 1000000 : req.duration_ms) || 0).toFixed(2) + ' ms\n';
    text += 'Size: ' + formatBytes(req.size || 0) + '\n';
    
    if (req.response_body) {
      text += '\n【响应 Body】\n';
      try {
        const parsed = JSON.parse(req.response_body);
        text += JSON.stringify(parsed, null, 2) + '\n';
      } catch (e) {
        // 如果太长就截断
        const body = req.response_body;
        if (body.length > 2000) {
          text += body.substring(0, 2000) + '\n... (内容过长，已截断)\n';
        } else {
          text += body + '\n';
        }
      }
    }
  }
  
  // === 提取变量 ===
  if (req.extracted_vars && Object.keys(req.extracted_vars).length > 0) {
    text += '\n【提取变量】\n';
    Object.entries(req.extracted_vars).forEach(([key, value]) => {
      text += '  ' + key + ' = ' + JSON.stringify(value) + '\n';
    });
  }
  
  // === 验证结果 ===
  if (req.verifications && req.verifications.length > 0) {
    text += '\n【验证结果】(' + req.verifications.length + ' 项)\n';
    req.verifications.forEach((verify, idx) => {
      const status = verify.skipped ? '⏭ 未执行' : (verify.success ? '✓ 通过' : '✗ 失败');
      text += '\n  [' + (idx + 1) + '] ' + status + '\n';
      if (verify.type) text += '    类型: ' + verify.type + '\n';
      if (verify.field) text += '    字段: ' + verify.field + '\n';
      if (verify.operator) text += '    操作符: ' + verify.operator + '\n';
      if (verify.description) text += '    描述: ' + verify.description + '\n';
      if (verify.expect !== undefined) text += '    期望值: ' + JSON.stringify(verify.expect) + '\n';
      if (verify.actual !== undefined) text += '    实际值: ' + JSON.stringify(verify.actual) + '\n';
      if (verify.message) text += '    消息: ' + verify.message + '\n';
    });
  }
  
  // === 错误信息 ===
  if (req.error) {
    text += '\n【错误信息】\n';
    text += req.error + '\n';
  }
  
  text += '\n========================================\n';
  return text;
}

// 辅助函数：格式化字节
function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
}

// 生成 curl (bash) 命令
function generateCurlBash(req) {
  const method = (req.method || req.request_method || 'GET').toUpperCase();
  const url = req.url || req.request_url || '';
  const headers = req.headers || req.request_headers || {};
  const body = req.body || req.request_body || '';
  
  let curl = 'curl -X ' + method;
  
  // 添加 Headers
  if (typeof headers === 'object' && headers !== null) {
    Object.entries(headers).forEach(([key, value]) => {
      curl += ' \\\n  -H "' + key + ': ' + value + '"';
    });
  }
  
  // 添加 Body
  if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
    const escapedBody = body.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\n/g, '');
    curl += ' \\\n  -d "' + escapedBody + '"';
  }
  
  curl += ' \\\n  "' + url + '"';
  return curl;
}

// 生成 curl (cmd) 命令 - Windows 格式
function generateCurlCmd(req) {
  const method = (req.method || req.request_method || 'GET').toUpperCase();
  const url = req.url || req.request_url || '';
  const headers = req.headers || req.request_headers || {};
  const body = req.body || req.request_body || '';
  
  let curl = 'curl -X ' + method;
  
  // 添加 Headers
  if (typeof headers === 'object' && headers !== null) {
    Object.entries(headers).forEach(([key, value]) => {
      curl += ' ^\n  -H "' + key + ': ' + value + '"';
    });
  }
  
  // 添加 Body
  if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
    const escapedBody = body.replace(/"/g, '\\"');
    curl += ' ^\n  -d "' + escapedBody + '"';
  }
  
  curl += ' ^\n  "' + url + '"';
  return curl;
}

// 生成 go-stress 命令
function generateGoStress(req) {
  const method = (req.method || req.request_method || 'GET').toUpperCase();
  let url = req.url || req.request_url || '';
  const headers = req.headers || req.request_headers || {};
  const body = req.body || req.request_body || '';
  const query = req.query || req.request_query || '';
  
  // 拼接 query 参数到 URL
  if (query) {
    url += (url.includes('?') ? '&' : '?') + query;
  }
  
  let cmd = 'go-stress';
  
  // URL（必需）
  cmd += ' -url "' + url + '"';
  
  // 请求方法
  if (method !== 'GET') {
    cmd += ' -method ' + method;
  }
  
  // 添加 headers（使用 -H）
  if (typeof headers === 'object' && headers !== null) {
    Object.entries(headers).forEach(([key, value]) => {
      cmd += ' -H "' + key + ': ' + value + '"';
    });
  }
  
  // 添加 body（使用 -data）
  if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
    const escapedBody = body.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\n/g, '');
    cmd += ' -data "' + escapedBody + '"';
  }
  
  return cmd;
}

// 生成 PowerShell 命令
function generatePowerShell(req) {
  const method = (req.method || req.request_method || 'GET').toUpperCase();
  let url = req.url || req.request_url || '';
  const headers = req.headers || req.request_headers || {};
  const body = req.body || req.request_body || '';
  const query = req.query || req.request_query || '';
  
  // 拼接 query 参数到 URL
  if (query) {
    url += (url.includes('?') ? '&' : '?') + query;
  }
  
  let ps = '$headers = @{\n';
  
  if (typeof headers === 'object' && headers !== null) {
    Object.entries(headers).forEach(([key, value]) => {
      ps += '    "' + key + '" = "' + value + '"\n';
    });
  }
  ps += '}\n\n';
  
  if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
    const escapedBody = body.replace(/"/g, '\`"').replace(/\$/g, '\`$');
    ps += '$body = @"\n' + escapedBody + '\n"@\n\n';
    ps += 'Invoke-RestMethod -Uri "' + url + '" \`\n';
    ps += '    -Method ' + method + ' \`\n';
    ps += '    -Headers $headers \`\n';
    ps += '    -Body $body';
  } else {
    ps += 'Invoke-RestMethod -Uri "' + url + '" \`\n';
    ps += '    -Method ' + method + ' \`\n';
    ps += '    -Headers $headers';
  }
  
  return ps;
}

// 复制代码到剪贴板
function copyCode(code, btnElement, format) {
  navigator.clipboard.writeText(code).then(() => {
    if (btnElement) {
      const originalHtml = btnElement.innerHTML;
      btnElement.innerHTML = '✓ 已复制';
      btnElement.style.background = '#38ef7d';
      setTimeout(() => {
        btnElement.innerHTML = originalHtml;
        btnElement.style.background = '';
      }, 2000);
    }
    console.log('已复制 ' + format + ' 格式代码');
  }).catch(err => {
    console.error('复制失败:', err);
    alert('复制失败，请手动复制');
  });
}

// 复制为指定格式
function copyAs(req, format, btnElement) {
  let code = '';
  switch(format) {
    case 'full-request':
      code = generateFullRequest(req);
      break;
    case 'curl-bash':
      code = generateCurlBash(req);
      break;
    case 'curl-cmd':
      code = generateCurlCmd(req);
      break;
    case 'powershell':
      code = generatePowerShell(req);
      break;
    case 'go-stress':
      code = generateGoStress(req);
      break;
    default:
      code = generateCurlBash(req);
  }
  copyCode(code, btnElement, format);
}

// 重放请求
function replayRequest(req, btnElement) {
  const method = (req.method || req.request_method || 'GET').toUpperCase();
  const url = req.url || req.request_url || '';
  
  // 检查是否可能遇到 CORS 问题
  try {
    const currentOrigin = window.location.origin;
    const targetUrl = new URL(url, currentOrigin);
    const isCrossOrigin = targetUrl.origin !== currentOrigin;
    
    if (isCrossOrigin) {
      const message = 
        '⚠️ 跨域请求限制\n\n' +
        '由于浏览器的 CORS 安全策略，无法直接重放跨域请求。\n\n' +
        '建议方案：\n' +
        '1. 使用"复制为"功能，选择合适的格式在终端或代码中执行\n' +
        '2. 使用 Postman 等 API 测试工具\n' +
        '3. 在服务器端配置 CORS 允许跨域访问';
      
      alert(message);
      return;
    }
  } catch (e) {
    console.error('URL 解析失败:', e);
  }
  
  // 同源请求，可以直接发送
  if (btnElement) {
    btnElement.disabled = true;
    const originalHtml = btnElement.innerHTML;
    btnElement.innerHTML = '🔄 发送中...';
    btnElement.style.background = '#ffa502';
  }
  
  const headers = req.headers || req.request_headers || {};
  const body = req.body || req.request_body || '';
  
  const startTime = Date.now();
  
  const fetchOptions = {
    method: method,
    headers: typeof headers === 'object' ? headers : {},
    mode: 'cors',
    credentials: 'omit'
  };
  
  if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
    fetchOptions.body = body;
  }
  
  fetch(url, fetchOptions)
    .then(response => {
      const duration = Date.now() - startTime;
      return response.text().then(text => ({
        status: response.status,
        statusText: response.statusText,
        headers: Object.fromEntries(response.headers.entries()),
        body: text,
        duration: duration
      }));
    })
    .then(result => {
      console.log('重放请求成功:', result);
      if (btnElement) {
        btnElement.innerHTML = '✓ 成功 (' + result.duration + 'ms)';
        btnElement.style.background = '#38ef7d';
        setTimeout(() => {
          btnElement.innerHTML = '🔄 重放请求';
          btnElement.style.background = '';
          btnElement.disabled = false;
        }, 3000);
      }
      
      const resultHtml = 
        '状态码: ' + result.status + ' ' + result.statusText + '\n' +
        '响应时间: ' + result.duration + 'ms\n\n' +
        '响应内容:\n' +
        result.body.substring(0, 500) + (result.body.length > 500 ? '...' : '');
      alert('重放请求成功！\n\n' + resultHtml);
    })
    .catch(error => {
      console.error('重放请求失败:', error);
      if (btnElement) {
        btnElement.innerHTML = '✗ 失败';
        btnElement.style.background = '#f45c43';
        setTimeout(() => {
          btnElement.innerHTML = '🔄 重放请求';
          btnElement.style.background = '';
          btnElement.disabled = false;
        }, 3000);
      }
      
      let errorMsg = error.message;
      if (error.message.includes('CORS') || error.message.includes('NetworkError')) {
        errorMsg = 'CORS 跨域限制，建议使用"复制为"功能在终端执行';
      }
      alert('重放请求失败：' + errorMsg);
    });
}

// 切换下拉菜单显示
function toggleActionMenu(menuId, event) {
  event.stopPropagation();
  const menu = document.getElementById(menuId);
  const button = event.currentTarget;
  const isVisible = menu.style.display === 'block';
  
  // 关闭所有其他菜单
  document.querySelectorAll('.action-dropdown-menu').forEach(m => {
    m.style.display = 'none';
  });
  
  if (!isVisible) {
    // 先显示菜单以获取实际高度
    menu.style.display = 'block';
    menu.style.visibility = 'hidden'; // 临时隐藏
    
    // 计算按钮位置
    const rect = button.getBoundingClientRect();
    const menuRect = menu.getBoundingClientRect();
    const viewportHeight = window.innerHeight;
    const viewportWidth = window.innerWidth;
    
    // 计算菜单可用空间
    const spaceBelow = viewportHeight - rect.bottom - 10;
    const spaceAbove = rect.top - 10;
    
    let top, left;
    
    // 决定菜单显示位置（上方或下方）
    if (spaceBelow >= menuRect.height || spaceBelow >= spaceAbove) {
      // 显示在按钮下方
      top = rect.bottom + 5;
      // 如果下方空间不足，限制最大高度并添加滚动
      if (menuRect.height > spaceBelow) {
        menu.style.maxHeight = spaceBelow + 'px';
        menu.style.overflowY = 'auto';
      }
    } else {
      // 显示在按钮上方
      top = rect.top - menuRect.height - 5;
      // 如果上方空间也不足，限制最大高度
      if (menuRect.height > spaceAbove) {
        menu.style.maxHeight = spaceAbove + 'px';
        menu.style.overflowY = 'auto';
        top = 10; // 从顶部留出一点空间
      }
    }
    
    // 水平位置（优先右对齐按钮）
    left = rect.right - menuRect.width;
    
    // 如果左侧超出视口，改为左对齐按钮
    if (left < 10) {
      left = rect.left;
    }
    
    // 如果右侧超出视口，贴右边
    if (left + menuRect.width > viewportWidth - 10) {
      left = viewportWidth - menuRect.width - 10;
    }
    
    // 应用位置
    menu.style.top = top + 'px';
    menu.style.left = Math.max(10, left) + 'px';
    menu.style.visibility = 'visible'; // 恢复可见
  }
}

// 点击页面其他地方关闭菜单
document.addEventListener('click', function() {
  document.querySelectorAll('.action-dropdown-menu').forEach(m => {
    m.style.display = 'none';
  });
});