/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-02-10 17:30:15
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-02-10 17:52:08
 * @FilePath: \go-stress\executor\jmeter_duration_test.go
 * @Description: JMeter 耗时断言
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/statistics"
	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJMeterDurationAssertionCountsAsFailure(t *testing.T) {
	const delay = 25 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(delay)
		fmt.Fprint(w, "payload")
	}))
	t.Cleanup(srv.Close)

	data, _, err := config.ExportJMeterPlan(&config.Config{
		Protocol: types.ProtocolHTTP, Concurrency: 1, Requests: 1, Timeout: time.Second,
		APIs: []config.APIConfig{{
			Name: "slow_body", URL: srv.URL, Method: http.MethodGet,
			Verify: []config.VerifyConfig{{Type: types.VerifyTypeResponseTime, Operator: types.OpLTE, Expect: 1}},
		}},
	})
	require.NoError(t, err)
	require.Contains(t, string(data), "DurationAssertion")
	path := filepath.Join(t.TempDir(), "duration.jmx")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	cfg, _, err := config.LoadJMeterPlan(path, false)
	require.NoError(t, err)

	result := runWithAPIs(t, cfg.APIs, 1).result
	assert.Equal(t, uint64(1), result.Report.TotalRequests)
	assert.Equal(t, uint64(1), result.Report.FailedRequests, "超时断言失败应计入失败统计")
	assert.Zero(t, result.Report.SuccessRequests)
	details := result.Executor.GetCollector().GetRequestDetails(0, 10, statistics.StatusFilterAll, "", "")
	require.Len(t, details, 1)
	require.Len(t, details[0].Verifications, 1)
	verification := details[0].Verifications[0]
	assert.Equal(t, types.VerifyTypeResponseTime, verification.Type)
	assert.False(t, verification.Success)
	actualMS, err := strconv.ParseFloat(verification.Actual, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, actualMS, float64(delay.Milliseconds()))
}
