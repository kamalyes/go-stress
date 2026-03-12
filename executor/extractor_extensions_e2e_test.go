/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 09:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 13:19:06
 * @FilePath: \go-stress\executor\extractor_extensions_e2e_test.go
 * @Description: 扩展提取器到下游依赖请求的端到端测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
)

func TestExtendedExtractorsFeedDependentRequest(t *testing.T) {
	for _, kind := range []types.ExtractorType{types.ExtractorTypeBoundary, types.ExtractorTypeRegex} {
		t.Run(string(kind), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/capture":
					w.Header().Set("X-Trace-Id", "trace-123")
					fmt.Fprint(w, "token=[first] token=[second]")
				case "/use":
					if r.Header.Get("X-Token") != "second" || r.Header.Get("X-Trace") != "trace-123" {
						http.Error(w, "extracted variables were not passed", http.StatusUnauthorized)
						return
					}
					fmt.Fprint(w, "ok")
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(srv.Close)
			report := runWithAPIs(t, []config.APIConfig{
				{Name: "capture", URL: srv.URL + "/capture", Method: http.MethodGet,
					Extractors: []config.ExtractorConfig{
						{Name: "token", Type: kind, LeftBoundary: "token=[", RightBoundary: "]",
							Regex: `token=\[([^]]*)\]`, MatchNumber: 2},
						{Name: "trace", Type: types.ExtractorTypeHeader, Header: "X-TRACE-ID"},
					}},
				{Name: "use", URL: srv.URL + "/use", Method: http.MethodGet,
					DependsOn: []string{"capture"}, Headers: map[string]string{
						"X-Token": "{{.capture.token}}", "X-Trace": "{{.capture.trace}}",
					}},
			}, 1).result.Report
			assert.Equal(t, uint64(2), report.TotalRequests)
			assert.Equal(t, uint64(2), report.SuccessRequests)
			assert.Zero(t, report.FailedRequests)
		})
	}
}
