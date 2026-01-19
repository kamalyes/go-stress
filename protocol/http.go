/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 00:00:00
 * @FilePath: \go-stress\protocol\http.go
 * @Description: 协议客户端实现 - HTTP
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package protocol

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-toolbox/pkg/httpx"
)

// HTTPClient HTTP协议客户端
type HTTPClient struct {
	config *config.Config
	client *httpx.Client
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(cfg *config.Config) (*HTTPClient, error) {
	if cfg.HTTP == nil {
		cfg.HTTP = &config.HTTPConfig{
			MaxConnsPerHost: 100,
			FollowRedirects: true,
		}
	}

	// 使用 go-toolbox 的 httpx 客户端
	var client *httpx.Client
	if cfg.HTTP.KeepAlive {
		// 自定义配置的HTTP客户端，支持长连接
		client = httpx.NewCustomDefaultClient()
	} else {
		// 默认客户端
		client = httpx.NewDefaultHttpClient()
	}

	return &HTTPClient{
		config: cfg,
		client: client,
	}, nil
}

// Connect HTTP无需显式连接
func (h *HTTPClient) Connect(ctx context.Context) error {
	return nil
}

// Send 发送HTTP请求
func (h *HTTPClient) Send(ctx context.Context, req *types.Request) (*types.Response, error) {
	startTime := time.Now()

	// 使用 go-toolbox 的 httpx 构建请求
	httpReq := h.client.NewRequest(req.Method, req.URL)

	// 设置超时
	if h.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.config.Timeout)
		defer cancel()
	}

	// 设置Headers
	for k, v := range req.Headers {
		httpReq.SetHeader(k, v)
	}

	// 设置Body - 使用SetBodyRaw发送原始字节，避免二次JSON编码
	if req.Body != "" {
		httpReq.SetBodyRaw([]byte(req.Body))
	}

	// 执行请求
	httpResp, err := httpReq.Send()
	duration := time.Since(startTime)

	// 解析Query参数
	var queryString string
	if u, parseErr := url.Parse(req.URL); parseErr == nil {
		queryString = u.RawQuery
	}

	if err != nil {
		return &types.Response{
			Duration:       duration,
			Error:          fmt.Errorf("HTTP请求失败: %w", err),
			RequestURL:     req.URL,
			RequestMethod:  req.Method,
			RequestHeaders: req.Headers,
			RequestBody:    req.Body,
			RequestQuery:   queryString,
		}, err
	}

	// 读取响应体 - httpx.Response.Body() 返回 ([]byte, error)
	body, err := httpResp.Body()
	if err != nil {
		return &types.Response{
			StatusCode:     httpResp.StatusCode,
			Duration:       duration,
			Error:          fmt.Errorf("读取响应失败: %w", err),
			RequestURL:     req.URL,
			RequestMethod:  req.Method,
			RequestHeaders: req.Headers,
			RequestBody:    req.Body,
			RequestQuery:   queryString,
		}, err
	}

	// 构建响应
	headers := make(map[string]string)
	for k, v := range httpResp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	response := &types.Response{
		StatusCode:     httpResp.StatusCode,
		Headers:        headers,
		Body:           body,
		Duration:       duration,
		RequestURL:     req.URL,
		RequestMethod:  req.Method,
		RequestHeaders: req.Headers,
		RequestBody:    req.Body,
		RequestQuery:   queryString,
	}

	return response, nil
}

// Close 关闭HTTP客户端
func (h *HTTPClient) Close() error {
	// HTTP客户端无需显式关闭
	return nil
}

// Type 返回协议类型
func (h *HTTPClient) Type() types.ProtocolType {
	return types.ProtocolHTTP
}
