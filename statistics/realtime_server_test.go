/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 09:19:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 13:56:00
 * @FilePath: \go-stress\statistics\realtime_server_test.go
 * @Description: 实时报告端口冲突与启动生命周期回归测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/storage"
	"github.com/stretchr/testify/require"
)

func TestRealtimeServerPortConflict(t *testing.T) {
	// 旧进程继续提供 health 报告，新任务必须同步获知绑定失败。
	old := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "health")
	}))
	// 与报告服务使用相同的通配监听地址，避免 IPv4/IPv6 各自绑定导致未发生冲突。
	require.NoError(t, old.Listener.Close())
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	old.Listener = listener
	old.Start()
	defer old.Close()

	port := old.Listener.Addr().(*net.TCPAddr).Port
	server := NewRealtimeServer(newTestCollector(), port, logger.NewEmptyLogger())
	defer server.Stop()
	err = server.Start()
	require.ErrorIs(t, err, syscall.EADDRINUSE)
	require.ErrorContains(t, err, "advanced.realtime_port")

	resp, err := old.Client().Get("http://localhost:" + strconv.Itoa(port))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "health", string(body))
}

func TestRealtimeServerStartAndStop(t *testing.T) {
	log := logger.NewEmptyLogger()
	store := storage.NewMemoryStorage("test", log)
	defer store.Close()
	server := NewRealtimeServer(NewCollector(store, log), 0, log)
	require.NoError(t, server.Start())
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	require.Positive(t, server.GetPort())

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:" + strconv.Itoa(server.GetPort()) + "/api/data")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var data map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&data))
	require.Equal(t, float64(0), data["total_requests"])

	require.NoError(t, server.Stop())
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", strconv.Itoa(server.GetPort())), time.Second)
	if conn != nil {
		conn.Close()
	}
	require.Error(t, err, "Stop 后应释放监听端口")
}
