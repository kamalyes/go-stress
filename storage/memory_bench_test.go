/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 19:05:11
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 19:22:38
 * @FilePath: \go-stress\storage\memory_bench_test.go
 * @Description: 内存存储写入路径的基准
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/types"
)

// benchDetail 造一条典型记录
func benchDetail(i int) *RequestResult {
	return &types.RequestResult{
		ID:         fmt.Sprintf("bench-%d", i),
		NodeID:     "local",
		Success:    i%5 != 0,
		StatusCode: 200,
		Duration:   time.Millisecond,
		Timestamp:  time.Now(),
	}
}

// BenchmarkMemoryWrite 连续写入 N 条的单条均摊成本
//
// 这个基准的意义在于「单条成本会不会随已写入量增长」：
// 若写入是把新记录插到切片头部，每次都要重建整条切片，
// 单条成本随 N 线性上升，总成本是 O(N²)
func BenchmarkMemoryWrite(b *testing.B) {
	for _, n := range []int{1000, 10000, 50000} {
		b.Run(fmt.Sprintf("已有%d条", n), func(b *testing.B) {
			s := NewMemoryStorage("local", logger.NewEmptyLogger())
			for i := 0; i < n; i++ {
				s.Write(benchDetail(i))
			}
			details := make([]*RequestResult, b.N)
			for i := range details {
				details[i] = benchDetail(n + i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Write(details[i])
			}
		})
	}
}
