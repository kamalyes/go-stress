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
	"time"

	"github.com/google/uuid"
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

var tokens = make(map[string]string) // token -> userID

func main() {
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/user/info", handleGetUserInfo)
	http.HandleFunc("/api/user/update", handleUpdateUser)
	http.HandleFunc("/api/health", handleHealth)

	fmt.Println("🚀 测试服务器启动在 http://localhost:3000")
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
	tokens[token] = userID

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

	userID, exists := tokens[token]
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

	userID, exists := tokens[token]
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
