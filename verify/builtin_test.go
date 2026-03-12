/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 22:56:39
 * @FilePath: \go-stress\verify\builtin_test.go
 * @Description: 内置验证器测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package verify

import (
	"strconv"
	"testing"
	"time"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseTimeUsesMeasuredDuration(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration time.Duration
		expect   any
		operator types.ExpectOperator
		pass     bool
	}{
		{"slow response fails", 120 * time.Millisecond, 50, types.OpLTE, false},
		{"fast response passes", 30 * time.Millisecond, 50, types.OpLTE, true},
		{"default operator is lte", 120 * time.Millisecond, 50, "", false},
		{"exact boundary", 50 * time.Millisecond, 50, types.OpLTE, true},
		{"fractional milliseconds are retained", 50500 * time.Microsecond, 50.0, types.OpLTE, false},
		{"float threshold", 50500 * time.Microsecond, 50.5, types.OpEQ, true},
		{"greater than", 120 * time.Millisecond, 50, types.OpGT, true},
		{"invalid expectation", time.Millisecond, "not-a-number", types.OpLTE, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &Response{StatusCode: 200, Duration: tc.duration}
			v := NewHTTPVerifier(&config.VerifyConfig{
				Type: VerifyTypeResponseTime, Operator: tc.operator, Expect: tc.expect,
			})
			pass, err := v.Verify(resp)
			assert.Equal(t, tc.pass, pass)
			if tc.pass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			require.Len(t, resp.Verifications, 1)
			assert.Equal(t, pass, resp.Verifications[0].Success)
			if _, invalid := tc.expect.(string); !invalid {
				actual, err := strconv.ParseFloat(resp.Verifications[0].Actual, 64)
				require.NoError(t, err)
				assert.InDelta(t, float64(tc.duration)/float64(time.Millisecond), actual, 1e-9)
			}
		})
	}
}
