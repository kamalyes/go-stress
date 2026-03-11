/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 13:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:15:57
 * @FilePath: \go-stress\testserver\test_server.go
 * @Description: 测试服务器 - 用于验证依赖和数据提取功能
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type UserInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type UpdateRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Ticket 相关结构
type CreateTicketRequest struct {
	UserID      string                 `json:"user_id"`
	Subject     string                 `json:"subject"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type Ticket struct {
	TicketID  string `json:"ticket_id"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	AgentID   string `json:"agent_id"`
	Status    string `json:"status"`
	Subject   string `json:"subject"`
}

type CreateTicketResponse struct {
	Ticket Ticket `json:"ticket"`
}

// Message 相关结构
type SendMessageRequest struct {
	SessionID    string                 `json:"session_id"`
	SenderID     string                 `json:"sender_id"`
	SenderType   int                    `json:"sender_type"`
	ReceiverID   string                 `json:"receiver_id"`
	ReceiverType int                    `json:"receiver_type"`
	MsgType      int                    `json:"msg_type"`
	Content      string                 `json:"content"`
	ContentExtra map[string]interface{} `json:"content_extra"`
	SeqNo        string                 `json:"seq_no"`
	Priority     int                    `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type SendMessageResponse struct {
	Data struct {
		MessageID  string `json:"message_id"`
		Status     string `json:"status"`
		SessionID  string `json:"session_id"`
		SenderID   string `json:"sender_id"`
		ReceiverID string `json:"receiver_id"`
		MsgType    string `json:"msg_type"`
		SeqNo      string `json:"seq_no"`
	} `json:"data"`
}

// tokens/sessions 会被多个 HTTP handler 并发读写，必须加锁保护，
// 否则高并发下触发 runtime 的 concurrent map writes fatal error
var (
	tokensMu   sync.RWMutex
	tokens     = make(map[string]string) // token -> userID
	sessionsMu sync.RWMutex
	sessions   = make(map[string]string) // sessionID -> ticketID
)

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源,测试用
	},
}

// WebSocket 消息类型
type WSMessage struct {
	Action    string                 `json:"action"`
	Data      map[string]interface{} `json:"data,omitempty"`
	MessageID int64                  `json:"message_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

type WSResponse struct {
	Success   bool                   `json:"success"`
	Action    string                 `json:"action"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Message   string                 `json:"message,omitempty"`
	MessageID int64                  `json:"message_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

func main() {
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/user/info", handleGetUserInfo)
	http.HandleFunc("/api/user/update", handleUpdateUser)
	http.HandleFunc("/api/health", handleHealth)

	// Ticket 和 Message 接口
	http.HandleFunc("/v1/tickets", handleCreateTicket)
	http.HandleFunc("/v1/messages/send", handleSendMessage)

	// WebSocket 接口
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/ws/echo", handleWebSocketEcho)
	http.HandleFunc("/ws/chat", handleWebSocketChat)

	fmt.Println("🚀 测试服务器启动在 http://localhost:3000")
	fmt.Println("📡 WebSocket 端点:")
	fmt.Println("   - ws://localhost:3000/ws (通用)")
	fmt.Println("   - ws://localhost:3000/ws/echo (回声)")
	fmt.Println("   - ws://localhost:3000/ws/chat (聊天)")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ JSON解析失败: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求"})
		return
	}

	// 模拟登录验证
	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "用户名和密码不能为空",
		})
		return
	}

	// 生成token和userID
	token := uuid.New().String()
	userID := uuid.New().String()
	tokensMu.Lock()
	tokens[token] = userID
	tokensMu.Unlock()

	// 设置session header
	w.Header().Set("X-Session-ID", fmt.Sprintf("sess_%d", time.Now().Unix()))

	resp := LoginResponse{
		Success: true,
		Token:   token,
		UserID:  userID,
		Message: "登录成功",
	}

	log.Printf("✅ 登录成功: user=%s, token=%s", req.Username, token)
	json.NewEncoder(w).Encode(resp)
}

func handleGetUserInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 验证token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "缺少Authorization"})
		return
	}

	// 提取token (Bearer xxx)
	var token string
	fmt.Sscanf(authHeader, "Bearer %s", &token)

	tokensMu.RLock()
	userID, exists := tokens[token]
	tokensMu.RUnlock()
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的token"})
		return
	}

	sessionID := r.Header.Get("X-Session-ID")

	resp := UserInfo{
		UserID:   userID,
		Username: "test_user",
		Email:    "test@example.com",
		Role:     "admin",
	}

	log.Printf("✅ 获取用户信息: userID=%s, session=%s", userID, sessionID)
	json.NewEncoder(w).Encode(resp)
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 验证token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "缺少Authorization"})
		return
	}

	var token string
	fmt.Sscanf(authHeader, "Bearer %s", &token)

	tokensMu.RLock()
	userID, exists := tokens[token]
	tokensMu.RUnlock()
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的token"})
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求"})
		return
	}

	resp := UpdateResponse{
		Success: true,
		Message: "更新成功",
		Data: map[string]interface{}{
			"user_id": userID,
			"email":   req.Email,
			"role":    req.Role,
		},
	}

	log.Printf("✅ 更新用户信息: userID=%s, email=%s", userID, req.Email)
	json.NewEncoder(w).Encode(resp)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "test-api",
	})
}

func handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ JSON解析失败: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求"})
		return
	}

	// 生成 ticket 数据
	ticketID := uuid.New().String()
	// uuid.ID() 返回 uint32，%x 最多只有 8 个字符，再切 [:32] 必然越界 panic，
	// 导致创建工单接口从未成功过。改用去掉连字符的完整 UUID：恰好 32 位
	// 小写十六进制，与下游校验规则 ^[a-z0-9]{32}$ 对齐
	sessionID := strings.ReplaceAll(uuid.New().String(), "-", "")
	agentID := "owner"

	// 存储 session
	sessionsMu.Lock()
	sessions[sessionID] = ticketID
	sessionsMu.Unlock()

	ticket := Ticket{
		TicketID:  ticketID,
		UserID:    req.UserID,
		SessionID: sessionID,
		AgentID:   agentID,
		Status:    "open",
		Subject:   req.Subject,
	}

	resp := CreateTicketResponse{
		Ticket: ticket,
	}

	log.Printf("✅ 创建工单: ticketID=%s, sessionID=%s, userID=%s", ticketID, sessionID, req.UserID)
	json.NewEncoder(w).Encode(resp)
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ JSON解析失败: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求"})
		return
	}

	// 验证 session
	sessionsMu.RLock()
	ticketID, exists := sessions[req.SessionID]
	sessionsMu.RUnlock()
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "session不存在"})
		return
	}

	// 生成消息 ID
	messageID := uuid.New().String()

	// 回显请求关键字段：ticket-and-send.yaml 的校验规则靠它们做链路一致性
	// 检查（消息会话 ID 必须等于工单会话 ID 等），此前响应只有 message_id
	// 和 status，4 条交叉校验必然失败
	var resp SendMessageResponse
	resp.Data.MessageID = messageID
	resp.Data.Status = "sent"
	resp.Data.SessionID = req.SessionID
	resp.Data.SenderID = req.SenderID
	resp.Data.ReceiverID = req.ReceiverID
	resp.Data.SeqNo = req.SeqNo
	if req.MsgType == 1 {
		resp.Data.MsgType = "MESSAGE_TYPE_TEXT"
	} else {
		resp.Data.MsgType = fmt.Sprintf("MESSAGE_TYPE_%d", req.MsgType)
	}

	log.Printf("✅ 发送消息: messageID=%s, sessionID=%s, ticketID=%s, content=%s",
		messageID, req.SessionID, ticketID, req.Content)
	json.NewEncoder(w).Encode(resp)
}

// handleWebSocket 处理通用 WebSocket 连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket 升级失败: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("🔌 WebSocket 客户端连接: %s", r.RemoteAddr)

	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket 读取错误: %v", err)
			} else {
				log.Printf("❌ WebSocket 消息解析失败（连接将关闭）: %v", err)
			}
			break
		}

		log.Printf("📥 收到消息: action=%s, data=%v", msg.Action, msg.Data)

		// 构造响应
		resp := WSResponse{
			Success:   true,
			Action:    msg.Action,
			MessageID: msg.MessageID,
			Timestamp: time.Now().Unix(),
		}

		// 根据不同的 action 处理
		switch msg.Action {
		case "ping":
			resp.Data = map[string]interface{}{
				"pong": true,
			}
		case "echo":
			resp.Data = msg.Data
		case "info":
			resp.Data = map[string]interface{}{
				"server":    "go-stress-testserver",
				"version":   "1.0.0",
				"timestamp": time.Now().Unix(),
			}
		default:
			resp.Data = map[string]interface{}{
				"received": msg.Action,
				"echo":     msg.Data,
			}
		}

		// 发送响应
		err = conn.WriteJSON(resp)
		if err != nil {
			log.Printf("❌ WebSocket 写入错误: %v", err)
			break
		}

		log.Printf("📤 发送响应: action=%s, success=%v", resp.Action, resp.Success)
	}

	log.Printf("🔌 WebSocket 客户端断开: %s", r.RemoteAddr)
}

// handleWebSocketEcho 回声服务器
func handleWebSocketEcho(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket 升级失败: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("🔌 Echo 客户端连接: %s", r.RemoteAddr)

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Echo 读取错误: %v", err)
			}
			break
		}

		log.Printf("📥 Echo 收到: %s", string(message))

		// 直接回送原消息
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Printf("❌ Echo 写入错误: %v", err)
			break
		}

		log.Printf("📤 Echo 发送: %s", string(message))
	}

	log.Printf("🔌 Echo 客户端断开: %s", r.RemoteAddr)
}

// handleWebSocketChat 模拟聊天服务器
func handleWebSocketChat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket 升级失败: %v", err)
		return
	}
	defer conn.Close()

	userID := uuid.New().String()[:8]
	log.Printf("🔌 Chat 客户端连接: %s (userID=%s)", r.RemoteAddr, userID)

	// 发送欢迎消息
	welcome := WSResponse{
		Success:   true,
		Action:    "welcome",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"user_id": userID,
			"message": "欢迎来到聊天室",
		},
	}
	conn.WriteJSON(welcome)

	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Chat 读取错误: %v", err)
			} else {
				// JSON 解析失败也要留痕：压测客户端发的 body 本身是坏 JSON 时，
				// 静默断连只会让对端表现出一堆莫名其妙的连接错误
				log.Printf("❌ Chat 消息解析失败（连接将关闭）: %v", err)
			}
			break
		}

		log.Printf("📥 Chat 收到: userID=%s, action=%s, data=%v", userID, msg.Action, msg.Data)

		// 构造聊天响应
		resp := WSResponse{
			Success:   true,
			Action:    "message",
			MessageID: msg.MessageID,
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"user_id":    userID,
				"message_id": uuid.New().String(),
				"content":    msg.Data["content"],
				"echo":       true,
			},
		}

		// 发送响应
		err = conn.WriteJSON(resp)
		if err != nil {
			log.Printf("❌ Chat 写入错误: %v", err)
			break
		}

		log.Printf("📤 Chat 发送: userID=%s, messageID=%v", userID, resp.Data["message_id"])
	}

	log.Printf("🔌 Chat 客户端断开: %s (userID=%s)", r.RemoteAddr, userID)
}
