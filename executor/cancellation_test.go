/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 09:19:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 13:56:00
 * @FilePath: \go-stress\executor\cancellation_test.go
 * @Description: 渐进启动、暂停等待及重复中断信号的退出回归测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/statistics"
	"github.com/kamalyes/go-stress/storage"
	"github.com/stretchr/testify/require"
)

func cancellationScheduler(t *testing.T, workers uint64, ramp time.Duration, handler RequestHandler) *Scheduler {
	t.Helper()
	log := logger.NewEmptyLogger()
	store := storage.NewMemoryStorage("test", log)
	t.Cleanup(func() { _ = store.Close() })
	return NewScheduler(SchedulerConfig{
		WorkerCount: workers, RequestPerWorker: 1, RampUpDuration: ramp,
		ClientPool: NewClientPool(func() (Client, error) { return stubClient{}, nil }, int(workers)),
		Handler:    handler, Collector: statistics.NewCollector(store, log),
		APISelector: NewRoundRobinSelector([]config.APIConfig{{Name: "echo", URL: "ws://localhost/ws"}}),
		VarResolver: config.NewVariableResolver(), Logger: log,
	})
}

func TestSchedulerCancelDuringRampUp(t *testing.T) {
	started := make(chan struct{}, 1)
	var calls atomic.Int64
	s := cancellationScheduler(t, 6, 2*time.Second, func(ctx context.Context, req *Request) (*Response, error) {
		calls.Add(1)
		started <- struct{}{}
		return &Response{StatusCode: 200}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("首个 Worker 未启动")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("取消后仍在等待 ramp_up")
	}
	require.EqualValues(t, 1, calls.Load(), "取消后不应继续启动剩余 Worker")
}

func TestSchedulerRampUpUsesOneWindow(t *testing.T) {
	var calls atomic.Int64
	s := cancellationScheduler(t, 6, 600*time.Millisecond, func(ctx context.Context, req *Request) (*Response, error) {
		calls.Add(1)
		return &Response{StatusCode: 200}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	start := time.Now()
	require.NoError(t, s.Run(ctx), "启动时间不应按 Worker 序号逐次累加")
	require.EqualValues(t, 6, calls.Load())
	require.GreaterOrEqual(t, time.Since(start), 500*time.Millisecond)
}

type alwaysPausedController struct{}

func (alwaysPausedController) IsPaused() bool  { return true }
func (alwaysPausedController) IsStopped() bool { return false }

func TestPausedWorkerWaitRespondsToCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan bool, 1)
	go func() { done <- WaitWhilePausedContext(ctx, alwaysPausedController{}) }()
	cancel()
	select {
	case stopped := <-done:
		require.True(t, stopped)
	case <-time.After(time.Second):
		t.Fatal("暂停等待未响应取消")
	}
}

func TestStandaloneSecondInterruptExits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("子进程 SIGINT 回归测试仅用于 Unix")
	}
	const helperEnv = "GO_STRESS_INTERRUPT_TEST_HELPER"
	if os.Getenv(helperEnv) == "1" {
		strategy := NewStandaloneStrategy(logger.NewEmptyLogger(), "", true, true, true)
		ctx, cancel, _ := strategy.PrepareContext(context.Background())
		defer cancel()
		process, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)
		require.NoError(t, process.Signal(os.Interrupt))
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
			t.Fatal("首次 Ctrl+C 未取消任务")
		}
		fmt.Println("first interrupt canceled context")
		require.NoError(t, process.Signal(os.Interrupt))
		time.Sleep(time.Second)
		t.Fatal("第二次 Ctrl+C 被吞掉")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStandaloneSecondInterruptExits$")
	cmd.Env = append(os.Environ(), helperEnv+"=1")
	out, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "中断测试子进程超时: %s", out)
	require.Contains(t, string(out), "first interrupt canceled context")
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, -1, exitErr.ExitCode(), "第二次中断应通过信号终止子进程: %s", out)
}
