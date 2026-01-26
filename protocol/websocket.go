/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 16:00:00
 * @FilePath: \go-stress\protocol\websocket.go
 * @Description: WebSocket 协议客户端实现
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package protocol

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kamalyes/go-stress/config"
)

// WebSocketClient WebSocket 客户端
type WebSocketClient struct {
	config  *config.Config
	conn    *websocket.Conn
	dialer  *websocket.Dialer
	headers http.Header
	mu      sync.Mutex // 保护并发读写
}

// NewWebSocketClient 创建 WebSocket 客户端
func NewWebSocketClient(cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// 验证 URL
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid websocket URL: %w", err)
	}

	// WebSocket URL 必须是 ws:// 或 wss://
	if u.Scheme != "ws" && u.Scheme != "wss" {
		// 自动转换 http/https 为 ws/wss
		switch u.Scheme {
		case "http":
			u.Scheme = "ws"
		case "https":
			u.Scheme = "wss"
		default:
			return nil, fmt.Errorf("invalid websocket scheme: %s (expected ws or wss)", u.Scheme)
		}
		cfg.URL = u.String()
	}

	// 构建请求头
	headers := make(http.Header)
	for k, v := range cfg.Headers {
		headers.Set(k, v)
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: time.Duration(cfg.Timeout) * time.Second,
		Proxy:            http.ProxyFromEnvironment,
	}

	client := &WebSocketClient{
		config:  cfg,
		dialer:  dialer,
		headers: headers,
	}

	return client, nil
}

// Connect 建立 WebSocket 连接
func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // 已连接
	}

	conn, httpResp, err := c.dialer.DialContext(ctx, c.config.URL, c.headers)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	c.conn = conn

	if httpResp != nil {
		httpResp.Body.Close()
	}

	return nil
}

// Send 发送 WebSocket 请求
func (c *WebSocketClient) Send(ctx context.Context, req *Request) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	startTime := time.Now()

	// 确保连接已建立
	if c.conn == nil {
		conn, httpResp, err := c.dialer.DialContext(ctx, c.config.URL, c.headers)
		if err != nil {
			return nil, fmt.Errorf("websocket dial failed: %w", err)
		}
		c.conn = conn
		if httpResp != nil {
			httpResp.Body.Close()
		}
	}

	// 发送消息
	messageType := websocket.TextMessage
	if req.Metadata != nil {
		if mt, ok := req.Metadata["message_type"].(int); ok {
			messageType = mt
		}
	}

	if err := c.conn.WriteMessage(messageType, []byte(req.Body)); err != nil {
		// 连接断开,关闭并返回错误
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		return nil, fmt.Errorf("websocket write failed: %w", err)
	}

	// 接收响应
	_, respBody, err := c.conn.ReadMessage()
	if err != nil {
		// 检查是否为正常关闭或异常关闭
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			// WebSocket 连接已关闭，这是正常情况（特别是在长连接测试中）
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}

			duration := time.Since(startTime)
			headers := make(map[string]string)
			for k, v := range c.headers {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}

			// 返回成功响应，表示连接已正常处理
			return &Response{
				StatusCode:     200,
				Body:           []byte{},
				Duration:       duration,
				RequestURL:     c.config.URL,
				RequestMethod:  "WEBSOCKET",
				RequestHeaders: headers,
				RequestBody:    req.Body,
			}, nil
		}

		// 其他错误才视为失败
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		return nil, fmt.Errorf("websocket read failed: %w", err)
	}

	duration := time.Since(startTime)

	// 转换 http.Header 为 map[string]string
	headers := make(map[string]string)
	for k, v := range c.headers {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// 构造响应
	response := &Response{
		StatusCode:     200, // WebSocket 成功时状态码固定为 200
		Body:           respBody,
		Duration:       duration,
		RequestURL:     c.config.URL,
		RequestMethod:  "WEBSOCKET",
		RequestHeaders: headers,
		RequestBody:    req.Body,
	}

	return response, nil
}

// Type 返回协议类型
func (c *WebSocketClient) Type() ProtocolType {
	return ProtocolWebSocket
}

// Close 关闭客户端连接
func (c *WebSocketClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
