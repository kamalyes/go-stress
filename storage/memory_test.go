/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-12 19:28:50
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-12 19:52:16
 * @FilePath: \go-stress\storage\memory_test.go
 * @Description: 内存存储的顺序、分页与隔离测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package storage

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-stress/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMem 建一个内存存储
func newMem() *MemoryStorage {
	return NewMemoryStorage("local", logger.NewEmptyLogger())
}

// detailWith 造一条带指定归属的记录
func detailWith(id, nodeID, taskID string, success, skipped bool) *RequestResult {
	return &types.RequestResult{
		ID:        id,
		NodeID:    nodeID,
		TaskID:    taskID,
		Success:   success,
		Skipped:   skipped,
		Duration:  time.Millisecond,
		Timestamp: time.Now(),
	}
}

// idsOf 取出一组记录的 ID，便于断言顺序
func idsOf(details []*RequestResult) []string {
	out := make([]string, len(details))
	for i, d := range details {
		out[i] = d.ID
	}
	return out
}

// TestMemoryQueryNewestFirst 查询必须最新在前
//
// 回归用：写入从「插入头部」改成「尾部追加」后，顺序语义由写侧移到了读侧，
// 这个不变量是报告明细表的呈现基础 —— 搞反了页面会从最早的请求开始列
func TestMemoryQueryNewestFirst(t *testing.T) {
	s := newMem()
	for i := 1; i <= 5; i++ {
		s.Write(detailWith(fmt.Sprintf("r%d", i), "local", "", true, false))
	}

	got, err := s.Query(0, 10, StatusFilterAll, "", "")
	require.NoError(t, err)
	assert.Equal(t, []string{"r5", "r4", "r3", "r2", "r1"}, idsOf(got))
}

// TestMemoryPaginationNoOverlap 翻页既不重叠也不漏
func TestMemoryPaginationNoOverlap(t *testing.T) {
	s := newMem()
	for i := 1; i <= 10; i++ {
		s.Write(detailWith(fmt.Sprintf("r%d", i), "local", "", true, false))
	}

	p1, err := s.Query(0, 3, StatusFilterAll, "", "")
	require.NoError(t, err)
	p2, err := s.Query(3, 3, StatusFilterAll, "", "")
	require.NoError(t, err)
	p3, err := s.Query(6, 3, StatusFilterAll, "", "")
	require.NoError(t, err)
	p4, err := s.Query(9, 3, StatusFilterAll, "", "")
	require.NoError(t, err)

	assert.Equal(t, []string{"r10", "r9", "r8"}, idsOf(p1))
	assert.Equal(t, []string{"r7", "r6", "r5"}, idsOf(p2))
	assert.Equal(t, []string{"r4", "r3", "r2"}, idsOf(p3))
	assert.Equal(t, []string{"r1"}, idsOf(p4), "末页只剩一条")

	// 越界的 offset 返回空而不是报错，也不该 panic
	p5, err := s.Query(100, 3, StatusFilterAll, "", "")
	require.NoError(t, err)
	assert.Empty(t, p5)
}

// TestMemoryStatusFilters 分状态切片各自独立且同样最新在前
func TestMemoryStatusFilters(t *testing.T) {
	s := newMem()
	s.Write(detailWith("ok1", "local", "", true, false))
	s.Write(detailWith("bad1", "local", "", false, false))
	s.Write(detailWith("skip1", "local", "", false, true))
	s.Write(detailWith("ok2", "local", "", true, false))
	s.Write(detailWith("bad2", "local", "", false, false))

	cases := []struct {
		filter StatusFilter
		want   []string
	}{
		{StatusFilterAll, []string{"bad2", "ok2", "skip1", "bad1", "ok1"}},
		{StatusFilterSuccess, []string{"ok2", "ok1"}},
		{StatusFilterFailed, []string{"bad2", "bad1"}},
		{StatusFilterSkipped, []string{"skip1"}},
	}
	for _, tc := range cases {
		got, err := s.Query(0, 10, tc.filter, "", "")
		require.NoError(t, err)
		assert.Equal(t, tc.want, idsOf(got), "filter=%v", tc.filter)

		n, err := s.Count(tc.filter, "", "")
		require.NoError(t, err)
		assert.Equal(t, len(tc.want), n, "filter=%v 的计数应与查询一致", tc.filter)
	}
}

// TestMemoryFilterByNodeAndTask 带归属过滤时顺序与分页同样成立
func TestMemoryFilterByNodeAndTask(t *testing.T) {
	s := newMem()
	s.Write(detailWith("a1", "slave-a", "t1", true, false))
	s.Write(detailWith("b1", "slave-b", "t1", true, false))
	s.Write(detailWith("a2", "slave-a", "t2", true, false))
	s.Write(detailWith("a3", "slave-a", "t1", true, false))

	got, err := s.Query(0, 10, StatusFilterAll, "slave-a", "")
	require.NoError(t, err)
	assert.Equal(t, []string{"a3", "a2", "a1"}, idsOf(got))

	got, err = s.Query(0, 10, StatusFilterAll, "slave-a", "t1")
	require.NoError(t, err)
	assert.Equal(t, []string{"a3", "a1"}, idsOf(got), "node 与 task 是与的关系")

	// 过滤下的分页
	got, err = s.Query(1, 1, StatusFilterAll, "slave-a", "")
	require.NoError(t, err)
	assert.Equal(t, []string{"a2"}, idsOf(got))

	n, err := s.Count(StatusFilterAll, "slave-a", "t1")
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	// 对不上的归属返回空
	got, err = s.Query(0, 10, StatusFilterAll, "slave-zzz", "")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestMemoryQueryReturnsCopy 返回的切片不能是内部数组的窗口
//
// 原实现直接交出 source[offset:end]，而锁在 Query 返回时就已释放，
// 调用方实际握着存储自己的底层数组
func TestMemoryQueryReturnsCopy(t *testing.T) {
	s := newMem()
	for i := 1; i <= 3; i++ {
		s.Write(detailWith(fmt.Sprintf("r%d", i), "local", "", true, false))
	}

	got, err := s.Query(0, 3, StatusFilterAll, "", "")
	require.NoError(t, err)
	require.Len(t, got, 3)

	got[0] = detailWith("tampered", "local", "", true, false)

	again, err := s.Query(0, 3, StatusFilterAll, "", "")
	require.NoError(t, err)
	assert.Equal(t, []string{"r3", "r2", "r1"}, idsOf(again), "改动返回值不该影响存储")
}

// TestMemoryZeroAndNegativeLimit limit 非正时返回空而不是 panic
func TestMemoryZeroAndNegativeLimit(t *testing.T) {
	s := newMem()
	s.Write(detailWith("r1", "local", "", true, false))

	for _, limit := range []int{0, -1} {
		got, err := s.Query(0, limit, StatusFilterAll, "", "")
		require.NoError(t, err)
		assert.Empty(t, got, "limit=%d", limit)
	}

	got, err := s.Query(-1, 10, StatusFilterAll, "", "")
	require.NoError(t, err)
	assert.Empty(t, got, "负 offset 同样返回空")
}

// TestMemoryConcurrentWriteAndQuery 并发写入与查询不应触发竞态
func TestMemoryConcurrentWriteAndQuery(t *testing.T) {
	s := newMem()
	var wg sync.WaitGroup

	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				s.Write(detailWith(fmt.Sprintf("w%d-%d", w, i), "local", "", i%3 != 0, false))
			}
		}(w)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			if got, err := s.Query(0, 20, StatusFilterAll, "", ""); err == nil {
				for _, d := range got {
					_ = d.ID // 读一下，让竞态检测器有机会抓到写穿
				}
			}
		}
	}()
	wg.Wait()

	n, err := s.Count(StatusFilterAll, "", "")
	require.NoError(t, err)
	assert.Equal(t, 800, n)
}
