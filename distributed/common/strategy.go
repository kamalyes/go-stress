/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-23 10:00:00
 * @FilePath: \go-stress\distributed\common\strategy.go
 * @Description: 策略枚举定义
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package common

// SplitStrategy 任务分片策略
type SplitStrategy string

const (
	SplitStrategyEqual    SplitStrategy = "equal"    // 平均分配
	SplitStrategyWeighted SplitStrategy = "weighted" // 按权重分配
	SplitStrategyCustom   SplitStrategy = "custom"   // 自定义分配
)

// SelectStrategy Slave 选择策略
type SelectStrategy string

const (
	SelectStrategyRandom        SelectStrategy = "random"         // 随机选择
	SelectStrategyLeastLoaded   SelectStrategy = "least_loaded"   // 负载最低
	SelectStrategyLocationAware SelectStrategy = "location_aware" // 地域感知
	SelectStrategyRoundRobin    SelectStrategy = "round_robin"    // 轮询
)

// FlushPolicy 刷新策略
type FlushPolicy string

const (
	FlushPolicyTime FlushPolicy = "time" // 按时间刷新
	FlushPolicySize FlushPolicy = "size" // 按大小刷新
	FlushPolicyBoth FlushPolicy = "both" // 两者都满足
)
