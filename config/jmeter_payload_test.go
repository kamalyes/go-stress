/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 09:19:56
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 09:22:01
 * @FilePath: \go-stress\config\jmeter_payload_test.go
 * @Description: JMeter 原始请求体互转的逐字节一致性测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package config

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJMeterRawBodyPreservesWhitespace(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"leading and trailing spaces", "  payload  "},
		{"trailing newline", "first line\nsecond line\n"},
		{"whitespace-only payload", " \t\r\n "},
		{"CRLF around JSON", "\r\n {\r\n  \"value\": \"文本\"\r\n }\r\n"},
		{"empty body", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{URL: "https://example.test/payload", Method: http.MethodPost,
				Body: tc.body, Concurrency: 1, Requests: 1}
			back, _, _ := roundTrip(t, cfg)
			require.Len(t, back.APIs, 1)
			assert.Equal(t, tc.body, back.APIs[0].Body, "原始报文应逐字节还原，不能裁剪空白")
		})
	}
}
