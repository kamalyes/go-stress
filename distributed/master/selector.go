/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\master\selector.go
 * @Description: Slave 选择策略实现
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package master

import (
	"math/rand"
	"sort"

	"github.com/kamalyes/go-stress/distributed/common"
)

// RandomSelector 随机选择器
type RandomSelector struct{}

// NewRandomSelector 创建随机选择器
func NewRandomSelector() *RandomSelector {
	return &RandomSelector{}
}

// Select 随机选择指定数量的 Slave
func (s *RandomSelector) Select(slaves []*common.SlaveInfo, count int) []*common.SlaveInfo {
	if count >= len(slaves) {
		return slaves
	}

	// 复制切片避免修改原始数据
	selected := make([]*common.SlaveInfo, len(slaves))
	copy(selected, slaves)

	// Fisher-Yates 洗牌算法
	rand.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	return selected[:count]
}

// LeastLoadedSelector 负载最低选择器
type LeastLoadedSelector struct{}

// NewLeastLoadedSelector 创建负载最低选择器
func NewLeastLoadedSelector() *LeastLoadedSelector {
	return &LeastLoadedSelector{}
}

// Select 选择负载最低的 Slave
func (s *LeastLoadedSelector) Select(slaves []*common.SlaveInfo, count int) []*common.SlaveInfo {
	if count >= len(slaves) {
		return slaves
	}

	// 复制并按负载排序
	sorted := make([]*common.SlaveInfo, len(slaves))
	copy(sorted, slaves)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CurrentLoad < sorted[j].CurrentLoad
	})

	return sorted[:count]
}

// LocationAwareSelector 地域感知选择器
type LocationAwareSelector struct {
	preferredRegions []string
}

// NewLocationAwareSelector 创建地域感知选择器
func NewLocationAwareSelector(regions []string) *LocationAwareSelector {
	return &LocationAwareSelector{
		preferredRegions: regions,
	}
}

// Select 优先选择指定地域的 Slave
func (s *LocationAwareSelector) Select(slaves []*common.SlaveInfo, count int) []*common.SlaveInfo {
	preferred := make([]*common.SlaveInfo, 0)
	others := make([]*common.SlaveInfo, 0)

	// 分类
	for _, slave := range slaves {
		if s.isPreferred(slave.Region) {
			preferred = append(preferred, slave)
		} else {
			others = append(others, slave)
		}
	}

	// 组合结果
	result := make([]*common.SlaveInfo, 0, count)
	result = append(result, preferred...)
	if len(result) < count && len(others) > 0 {
		need := count - len(result)
		if need > len(others) {
			need = len(others)
		}
		result = append(result, others[:need]...)
	}

	if len(result) > count {
		result = result[:count]
	}

	return result
}

func (s *LocationAwareSelector) isPreferred(region string) bool {
	for _, r := range s.preferredRegions {
		if r == region {
			return true
		}
	}
	return false
}

// RoundRobinSelector 轮询选择器
type RoundRobinSelector struct {
	index int
}

// NewRoundRobinSelector 创建轮询选择器
func NewRoundRobinSelector() *RoundRobinSelector {
	return &RoundRobinSelector{index: 0}
}

// Select 轮询选择 Slave
func (s *RoundRobinSelector) Select(slaves []*common.SlaveInfo, count int) []*common.SlaveInfo {
	if count >= len(slaves) {
		return slaves
	}

	result := make([]*common.SlaveInfo, 0, count)
	for i := 0; i < count; i++ {
		idx := (s.index + i) % len(slaves)
		result = append(result, slaves[idx])
	}

	s.index = (s.index + count) % len(slaves)
	return result
}

// GetSelector 根据策略获取选择器
func GetSelector(strategy common.SelectStrategy, regions []string) SlaveSelector {
	switch strategy {
	case common.SelectStrategyRandom:
		return NewRandomSelector()
	case common.SelectStrategyLeastLoaded:
		return NewLeastLoadedSelector()
	case common.SelectStrategyLocationAware:
		return NewLocationAwareSelector(regions)
	case common.SelectStrategyRoundRobin:
		return NewRoundRobinSelector()
	default:
		return NewRandomSelector()
	}
}
