/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 09:19:27
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 09:56:55
 * @FilePath: \go-stress\config\jmeter_extractor_extensions_test.go
 * @Description: JMeter 边界提取与第 N 次匹配的双向转换测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func extractorExtensionConfig(ex ExtractorConfig) *Config {
	return &Config{Protocol: types.ProtocolHTTP, Concurrency: 1, Requests: 1,
		APIs: []APIConfig{{Name: "extract", URL: "https://example.test/", Method: "GET", Extractors: []ExtractorConfig{ex}}}}
}

func TestJMeterBoundaryAndRegexMatchRoundTrip(t *testing.T) {
	for _, kind := range []types.ExtractorType{types.ExtractorTypeBoundary, types.ExtractorTypeRegex} {
		for _, number := range []int{0, 1, 3} {
			t.Run(fmt.Sprintf("%s/%d", kind, number), func(t *testing.T) {
				ex := ExtractorConfig{Name: "token", Type: kind, MatchNumber: number,
					Regex: ` ([^]]*) `, LeftBoundary: " [\r\n", RightBoundary: "\n] ", Default: "  missing  "}
				cfg := extractorExtensionConfig(ex)
				back, exported, imported := roundTrip(t, cfg)
				require.Empty(t, exported.Drops)
				require.Empty(t, imported.Drops)
				require.Len(t, back.APIs[0].Extractors, 1)
				got := back.APIs[0].Extractors[0]
				assert.Equal(t, kind, got.Type)
				assert.Equal(t, ex.Name, got.Name)
				assert.Equal(t, max(number, 1), got.MatchNumber)
				assert.Equal(t, ex.Default, got.Default)
				if kind == types.ExtractorTypeBoundary {
					assert.Equal(t, ex.LeftBoundary, got.LeftBoundary)
					assert.Equal(t, ex.RightBoundary, got.RightBoundary)
				} else {
					assert.Equal(t, ex.Regex, got.Regex)
				}
			})
		}
	}
}

func TestJMeterExtractorUnsupportedModesReported(t *testing.T) {
	for _, kind := range []types.ExtractorType{types.ExtractorTypeBoundary, types.ExtractorTypeRegex} {
		cfg := extractorExtensionConfig(ExtractorConfig{Name: "token", Type: kind,
			LeftBoundary: "[", RightBoundary: "]", Regex: `\[([^]]*)\]`, MatchNumber: 1})
		data, _, err := ExportJMeterPlan(cfg)
		require.NoError(t, err)
		tag := "BoundaryExtractor"
		if kind == types.ExtractorTypeRegex {
			tag = "RegexExtractor"
		}
		for _, tc := range []struct{ name, from, to, reason string }{
			{"random match", tag + `.match_number">1<`, tag + `.match_number">0<`, "match_number"},
			{"all matches", tag + `.match_number">1<`, tag + `.match_number">-1<`, "match_number"},
			{"dynamic match", tag + `.match_number">1<`, tag + `.match_number">${number}<`, "match_number"},
			{"headers", tag + `.useHeaders">false<`, tag + `.useHeaders">true<`, "useHeaders"},
			{"variable scope", "</" + tag + ">", `<stringProp name="Sample.scope">variable</stringProp></` + tag + ">", "作用域"},
		} {
			t.Run(string(kind)+"/"+tc.name, func(t *testing.T) {
				body := strings.Replace(string(data), tc.from, tc.to, 1)
				require.NotEqual(t, string(data), body)
				path := filepath.Join(t.TempDir(), "unsupported.jmx")
				require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
				back, report, err := LoadJMeterPlan(path, false)
				require.NoError(t, err)
				normalizeAPIs(back)
				assert.Empty(t, back.APIs[0].Extractors, "不应默默改成第一组或响应体提取")
				assert.Contains(t, report.String(), tc.reason)
				assert.Contains(t, report.String(), "WARN")
			})
		}
	}
}

func TestJMeterBoundaryExportLossReported(t *testing.T) {
	t.Run("request body has no equivalent postprocessor", func(t *testing.T) {
		data, report, err := ExportJMeterPlan(extractorExtensionConfig(ExtractorConfig{
			Name: "token", Type: types.ExtractorTypeBoundary, Source: ExtractorSourceRequest,
			LeftBoundary: "[", RightBoundary: "]",
		}))
		require.NoError(t, err)
		assert.NotContains(t, string(data), "<BoundaryExtractor")
		assert.Contains(t, report.String(), "source")
	})
	t.Run("transform pipeline is reported", func(t *testing.T) {
		data, report, err := ExportJMeterPlan(extractorExtensionConfig(ExtractorConfig{
			Name: "token", Type: types.ExtractorTypeBoundary, LeftBoundary: "[", RightBoundary: "]",
			Transforms: []TransformConfig{{Function: "upper"}},
		}))
		require.NoError(t, err)
		assert.Contains(t, string(data), "<BoundaryExtractor")
		assert.Contains(t, report.String(), "transforms")
	})
}
