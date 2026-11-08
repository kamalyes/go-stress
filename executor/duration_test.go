/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-02-10 19:05:12
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-02-10 19:20:38
 * @FilePath: \go-stress\executor\duration_test.go
 * @Description: 按时长压测测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunTaskByDuration 按时长压测必须真的跑满时长并正常收尾
//
// 回归用：config.Duration 此前只有校验与分布式协议读过，独立模式的执行器从不理它
//   - 配置里写 duration: 3s 会被完全忽略，跑的是 DefaultConfig 的「每并发 1 个请求」
//   - 补上实现后又因 Worker 返回 ctx.Err() 而以 DeadlineExceeded 收场，
//     一次正常的定时压测被判为 FATAL，报告也不生成
func TestRunTaskByDuration(t *testing.T) {
	var served atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served.Add(1)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	const runFor = 600 * time.Millisecond

	start := time.Now()
	result := RunTask(RunOptions{
		Logger:      logger.NewEmptyLogger(),
		StorageMode: StorageModeMemory,
		NoReport:    true,
		NoPrint:     true,
		NoWait:      true,
		ConfigFunc: func() *config.Config {
			cfg := config.DefaultConfig()
			cfg.URL = srv.URL
			cfg.Method = http.MethodGet
			cfg.Concurrency = 4
			cfg.Requests = 0
			cfg.Duration = runFor
			cfg.Timeout = 5 * time.Second
			cfg.Advanced.DisableRealtime = true
			return cfg
		},
	})
	elapsed := time.Since(start)

	require.NoError(t, result.Error, "按时长跑完是正常完成，不该作为错误返回")
	require.NotNil(t, result.Report, "正常完成必须给出报告")

	assert.GreaterOrEqual(t, elapsed, runFor, "应当真的跑满设定时长，实际 %s", elapsed)
	assert.Greater(t, result.Report.TotalRequests, uint64(1),
		"按时长压测不应退化成「每并发一个请求」，实际只发了 %d 个", result.Report.TotalRequests)
	// 服务端实收与报告数允许有不超过并发数的偏差，方向不定
	//
	// 时长截止点上两边都可能多算：ctx 到期时某个请求已被服务端处理并计数，
	// 而结果还没进采集器（实收偏多）；或该请求在客户端侧因 ctx 到期而失败，
	// 计入了报告的失败数却未必到过服务端（报告偏多）
	// 每个并发最多各有一个在途请求，故偏差上界是并发数 ——
	// 断言「两者相等」或任一方向的大小关系都会偶发失败
	recorded, servedN := int64(result.Report.TotalRequests), served.Load()
	diff := recorded - servedN
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff, int64(4),
		"实收与报告的偏差不应超过并发数（实收 %d，报告 %d）", servedN, recorded)
}

// TestInFlightCancellationIsNotCountedAsFailure 轮次结束时在途的请求不算失败
//
// 时长模式把 Duration 实现为 context.WithTimeout，在途请求继承这个 ctx，于是时间一到，
// 每个 Worker 手上正在飞的那条请求都会以 context deadline exceeded 返回。
// 原先它们被当成真失败，后果是每轮固定多出「并发数」条失败记录，而且 Errors 里那条
// context deadline exceeded 与真正的请求级超时长得一模一样 ——
// 压测台最需要区分的恰恰是「被压对象真的失败了」和「这一轮结束了」
//
// 两个子测试是一对的：第一个证明在途取消不再计入，第二个证明真超时仍然计入。
// 只有前者会让「把所有 context 错误都丢掉」这种修过头的实现也通过
//
// [EN] A pair of subtests: the first proves an in-flight cancellation is no longer counted,
// the second proves a genuine per-request timeout still is. With only the first, an
// over-broad fix that discards every context error would also pass
func TestInFlightCancellationIsNotCountedAsFailure(t *testing.T) {
	t.Run("轮次结束的在途请求不计失败", func(t *testing.T) {
		// 每个请求 80ms，跑 300ms：轮次截止时每个 Worker 必然有一条请求在飞
		// [EN] 80ms per request over a 300ms round guarantees every worker is mid-request
		// when the round ends
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(80 * time.Millisecond)
			fmt.Fprint(w, `{"ok":true}`)
		}))
		defer srv.Close()

		const concurrency = 4
		result := RunTask(RunOptions{
			Logger:      logger.NewEmptyLogger(),
			StorageMode: StorageModeMemory,
			NoReport:    true,
			NoPrint:     true,
			NoWait:      true,
			ConfigFunc: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.URL = srv.URL
				cfg.Method = http.MethodGet
				cfg.Concurrency = concurrency
				cfg.Requests = 0
				cfg.Duration = 300 * time.Millisecond
				// 请求级超时放得足够宽：这个子测试里不该有任何真超时
				// [EN] A generous per-request timeout: no real timeout should occur here
				cfg.Timeout = 5 * time.Second
				cfg.Advanced.DisableRealtime = true
				return cfg
			},
		})

		require.NoError(t, result.Error)
		require.NotNil(t, result.Report)
		require.Greater(t, result.Report.TotalRequests, uint64(0), "一个请求都没发出，这条测试是空的")

		assert.Zero(t, result.Report.FailedRequests,
			"轮次结束时在途的请求被当成了失败：每轮会固定多出至多 %d 条（并发数），实际 %d 条，错误明细 %v",
			concurrency, result.Report.FailedRequests, result.Report.Errors)
		assert.Empty(t, result.Report.Errors,
			"Errors 里不该出现任何条目，实际 %v —— 它与真正的请求超时无法分辨", result.Report.Errors)
	})

	t.Run("真正的请求级超时仍然计失败", func(t *testing.T) {
		// 每个请求 500ms 而请求级超时 100ms：轮次还活着的时候就会超时
		// [EN] 500ms per request against a 100ms per-request timeout: it times out while the
		// round is still alive
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			fmt.Fprint(w, `{"ok":true}`)
		}))
		defer srv.Close()

		result := RunTask(RunOptions{
			Logger:      logger.NewEmptyLogger(),
			StorageMode: StorageModeMemory,
			NoReport:    true,
			NoPrint:     true,
			NoWait:      true,
			ConfigFunc: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.URL = srv.URL
				cfg.Method = http.MethodGet
				cfg.Concurrency = 2
				cfg.Requests = 0
				cfg.Duration = 450 * time.Millisecond
				cfg.Timeout = 100 * time.Millisecond
				cfg.Advanced.DisableRealtime = true
				return cfg
			},
		})

		require.NoError(t, result.Error)
		require.NotNil(t, result.Report)
		assert.Greater(t, result.Report.FailedRequests, uint64(0),
			"请求级超时必须算失败，否则「在途取消不计失败」就修过头了，把真失败一起丢了")
		assert.NotEmpty(t, result.Report.Errors, "真超时必须在 Errors 里留下条目")
	})
}

// TestRunTaskDurationWithQPS 时长与定速可以共存
func TestRunTaskDurationWithQPS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	const (
		runFor = time.Second
		qps    = 120.0
	)

	result := RunTask(RunOptions{
		Logger:      logger.NewEmptyLogger(),
		StorageMode: StorageModeMemory,
		NoReport:    true,
		NoPrint:     true,
		NoWait:      true,
		ConfigFunc: func() *config.Config {
			cfg := config.DefaultConfig()
			cfg.URL = srv.URL
			cfg.Method = http.MethodGet
			cfg.Concurrency = 8
			cfg.Requests = 0
			cfg.Duration = runFor
			cfg.Timeout = 5 * time.Second
			cfg.Advanced.DisableRealtime = true
			cfg.Advanced.TargetQPS = qps
			return cfg
		},
	})

	require.NoError(t, result.Error)
	require.NotNil(t, result.Report)

	// 这里只断言「限速把速率压下来了」，不断言下界
	//
	// 速率精度由 TestRateLimiterPacesAcrossWorkers 以纯单测覆盖（无 HTTP、
	// 无其他包争用），那里才是验证精度的地方
	//
	// 此处要跑真实 HTTP，与整套测试并行时机器被占满，完成数会掉到任何预设
	// 下界以下 —— 那是调度现象而非缺陷，为它设下界只会得到一个时不时变红的
	// 测试，而那比没有测试更糟：它训练人忽略红色
	//
	// 上界方向则是稳健的：负载只会让完成数更少，不会让限速失效而变多。
	// 不限速时 8 个并发打本地 httptest 一秒可达数万请求，落在目标速率的
	// 数倍以内即可证明限速生效
	want := qps * runFor.Seconds()
	got := float64(result.Report.TotalRequests)
	assert.Positive(t, got, "限速不应把发压完全卡死")
	assert.Less(t, got, want*4,
		"请求数远超目标速率，限速未生效：目标约 %.0f 个，实际 %d", want, result.Report.TotalRequests)
}

// TestValidateConfigAllowsDurationWithoutRequests 校验必须放行按时长压测
func TestValidateConfigAllowsDurationWithoutRequests(t *testing.T) {
	base := func() *config.Config {
		cfg := config.DefaultConfig()
		cfg.URL = "http://127.0.0.1:1/x"
		cfg.Concurrency = 1
		return cfg
	}

	t.Run("请求数与时长都为零时报错", func(t *testing.T) {
		cfg := base()
		cfg.Requests, cfg.Duration = 0, 0
		assert.Error(t, validateConfig(cfg))
	})

	t.Run("只给时长应放行", func(t *testing.T) {
		cfg := base()
		cfg.Requests, cfg.Duration = 0, 10*time.Second
		assert.NoError(t, validateConfig(cfg), "按时长压测时请求数本就为 0")
	})

	t.Run("只给请求数应放行", func(t *testing.T) {
		cfg := base()
		cfg.Requests, cfg.Duration = 10, 0
		assert.NoError(t, validateConfig(cfg))
	})
}

// TestWorkerByDurationLoopsUntilContextDone reqCount 为 0 时由 ctx 决定何时停止
func TestWorkerByDurationLoopsUntilContextDone(t *testing.T) {
	var calls atomic.Int64
	w := &Worker{
		id:       0,
		reqCount: 0, // 0 表示按时长跑
		client:   stubClient{},
		handler: func(ctx context.Context, req *Request) (*Response, error) {
			calls.Add(1)
			return &Response{StatusCode: 200}, nil
		},
		controller:     &NoOpController{},
		depContext:     NewWorkerDependencyContext(),
		logger:         logger.NewEmptyLogger(),
		think:          newThinkTimer(),
		trace:          newTraceGen(),
		extractorCache: map[string]*ExtractorManager{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := w.Run(ctx)
	elapsed := time.Since(start)

	assert.ErrorIs(t, err, context.DeadlineExceeded, "到点应返回 ctx 的错误，由上层判定是否正常")
	assert.GreaterOrEqual(t, elapsed, 150*time.Millisecond, "应持续到 ctx 结束")
	assert.Zero(t, calls.Load(), "缺少 APISelector 时不应真的发请求")
}

// stubClient 满足 Client 接口的空实现
type stubClient struct{}

func (stubClient) Connect(ctx context.Context) error { return nil }
func (stubClient) Send(ctx context.Context, req *Request) (*Response, error) {
	return &Response{StatusCode: 200}, nil
}
func (stubClient) Close() error       { return nil }
func (stubClient) Type() ProtocolType { return ProtocolHTTP }
