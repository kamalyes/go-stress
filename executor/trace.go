/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-02-10 16:20:08
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-02-10 16:50:12
 * @FilePath: \go-stress\executor\trace.go
 * @Description: 请求级 trace 标识生成（W3C trace-context）
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"strings"
)

// HeaderTraceparent W3C trace-context 的传播头
//
// 用标准头而非自造的 X-Trace-Id：被压服务只要接了任意 OTel SDK 或主流框架的
// 链路中间件，就会自动把它作为父 span 接上，服务端那条链路与此处的压测结果
// 共享同一个 trace-id，无需对方改一行代码
const HeaderTraceparent = "traceparent"

// HeaderRequestID 默认随请求带上的请求 ID 头
//
// traceparent 要求对端接了链路中间件才有意义，而 X-Request-Id 是网关、
// 反向代理与各框架访问日志的事实标准：两个一起带，对端是哪种形态都能对上
const HeaderRequestID = "X-Request-Id"

// DisableMarker 配置里填这个值表示关闭对应能力
const DisableMarker = "-"

// defaultTraceCaptureHeaders 默认从响应头里回捞的追踪字段，按优先级排列
//
// 顺序即优先级：自家服务通常用前两个；x-amzn-trace-id 与 x-cloud-trace-context
// 分别来自 ALB 与 GCP，过了这两家网关的请求必带；traceparent 放末位是因为
// 对端若原样回带，它多半只是我们自己发出去的那个值，信息量最低
var defaultTraceCaptureHeaders = []string{
	"X-Trace-Id",
	"X-Request-Id",
	"X-Correlation-Id",
	"Trace-Id",
	"Request-Id",
	"X-Amzn-Trace-Id",
	"X-Cloud-Trace-Context",
	"X-B3-TraceId",
	"traceparent",
}

// traceparentLen "00-" + 32 位 trace-id + "-" + 16 位 span-id + "-01"
const traceparentLen = 3 + 32 + 1 + 16 + 3

// traceCapture 按配置的字段名从响应头里回捞服务端的追踪 ID
//
// 每个 Worker 共享同一个实例，内部只读，故无锁
type traceCapture struct {
	names []string

	// 预筛用的两个位集，构造时一次算好
	//
	// 回捞走在每个请求的收尾路径上，而「一个候选字段都没命中」才是常态
	// （多数服务并不回带追踪头）。朴素写法要拿 9 个候选 × 响应头逐一
	// EqualFold，实测 316ns/请求全花在注定失败的比较上
	lenSet    uint64 // 允许的字段名长度，第 n 位为 1 表示有长度为 n 的候选
	firstSet  uint32 // 允许的首字母，第 i 位为 1 表示有以 'a'+i 开头的候选
	prefilter bool   // 候选名都能编进上面两个位集时才启用预筛
}

// newTraceCapture 构造回捞器；names 为空取默认列表，含 "-" 表示整体关闭
func newTraceCapture(names []string) *traceCapture {
	if len(names) == 0 {
		return newTraceCaptureFrom(defaultTraceCaptureHeaders)
	}
	cleaned := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == DisableMarker {
			return &traceCapture{}
		}
		if n != "" {
			cleaned = append(cleaned, n)
		}
	}
	if len(cleaned) == 0 {
		return newTraceCaptureFrom(defaultTraceCaptureHeaders)
	}
	return newTraceCaptureFrom(cleaned)
}

// newTraceCaptureFrom 以给定候选名建器并算好预筛位集
func newTraceCaptureFrom(names []string) *traceCapture {
	c := &traceCapture{names: names, prefilter: true}
	for _, n := range names {
		ch := lowerASCII(n[0])
		// 超长或首字符不是 ASCII 字母的候选编不进位集；与其悄悄漏掉
		// 用户配的字段，不如整体关掉预筛，退回逐一比对
		if len(n) >= 64 || ch < 'a' || ch > 'z' {
			c.prefilter = false
			return c
		}
		c.lenSet |= 1 << uint(len(n))
		c.firstSet |= 1 << uint(ch-'a')
	}
	return c
}

// enabled 是否需要回捞
func (c *traceCapture) enabled() bool { return len(c.names) > 0 }

// lowerASCII ASCII 转小写；非字母原样返回
func lowerASCII(b byte) byte { return b | 0x20 }

// mayMatch 预筛：长度与首字母都对不上的响应头不可能命中任何候选
//
// 两次位测试即可排除 Content-Type、Date、Server 这类绝大多数响应头，
// 省掉后面的逐一 EqualFold
func (c *traceCapture) mayMatch(k string) bool {
	n := len(k)
	if n == 0 || n >= 64 || c.lenSet&(1<<uint(n)) == 0 {
		return false
	}
	ch := lowerASCII(k[0])
	if ch < 'a' || ch > 'z' {
		return false
	}
	return c.firstSet&(1<<uint(ch-'a')) != 0
}

// find 返回命中的响应头原名与取值，都为空表示没捞到
//
// 匹配不区分大小写：HTTP/1.1 经 Go 规范化成 X-Trace-Id，HTTP/2 在线路上
// 全小写，而自建网关回带的又可能是 X-TRACE-ID，按字面比对必然漏
//
// 外层遍历响应头而非候选名：这样每个响应头只需过一次预筛。代价是不能
// 一命中就返回（map 遍历顺序随机，先遇到的未必是优先级最高的那个），
// 改为记住当前最优的候选下标，并把内层上界收窄到它
func (c *traceCapture) find(headers map[string]string) (string, string) {
	if len(headers) == 0 || len(c.names) == 0 {
		return "", ""
	}

	best := len(c.names)
	var bestKey, bestVal string
	for k, v := range headers {
		if v == "" {
			continue
		}
		if c.prefilter && !c.mayMatch(k) {
			continue
		}
		for i := 0; i < best; i++ {
			want := c.names[i]
			if len(k) == len(want) && strings.EqualFold(k, want) {
				best, bestKey, bestVal = i, k, v
				break
			}
		}
		if best == 0 {
			break // 已经是最高优先级，剩下的响应头不可能更优
		}
	}
	return bestKey, bestVal
}

// traceGen 为单个 Worker 生成 W3C trace-context 标识
//
// 每个 Worker 独占一个实例，故全程无锁、无原子操作：trace-id 取「Worker 创建时
// 一次性随机的 8 字节前缀 + 8 字节自增序号」，前缀保证跨 Worker 与跨次压测不撞，
// 序号保证同一 Worker 内唯一
//
// 不逐请求调 crypto/rand：这里的标识只用于把压测结果与被压服务的日志对上，
// 不承担任何安全语义，而压测工具自身绝不能成为被测系统的瓶颈
type traceGen struct {
	prefix uint64
	seq    uint64
}

// newTraceGen 创建生成器，随机前缀在此一次性取得
// crypto/rand 失败时回落到一个非零常量混入指针无关的序号，仍能保证格式合法
func newTraceGen() *traceGen {
	var b [8]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		return &traceGen{prefix: 0x9e3779b97f4a7c15}
	}
	prefix := binary.BigEndian.Uint64(b[:])
	if prefix == 0 {
		prefix = 0x9e3779b97f4a7c15 // trace-id 不得全零
	}
	return &traceGen{prefix: prefix}
}

// next 返回本次请求的 trace-id（32 位十六进制）与完整的 traceparent 头值
//
// 两者共用一次缓冲区填充，但各自转成 string 时仍需各一次分配：
// Go 字符串不可变，无法让 trace-id 复用 traceparent 的底层数组
func (g *traceGen) next() (traceID, traceparent string) {
	g.seq++

	var buf [traceparentLen]byte
	buf[0], buf[1], buf[2] = '0', '0', '-'
	putHex64(buf[3:19], g.prefix)
	putHex64(buf[19:35], g.seq)
	buf[35] = '-'
	// span-id 取序号的 splitmix64 终结函数，避免与 trace-id 尾部逐位相同
	putHex64(buf[36:52], spanIDFor(g.prefix, g.seq))
	buf[52], buf[53], buf[54] = '-', '0', '1' // flags=01 表示已采样

	return string(buf[3:35]), string(buf[:])
}

// spanIDFor 由前缀与序号导出 span-id，保证非零（W3C 要求 parent-id 不得全零）
func spanIDFor(prefix, seq uint64) uint64 {
	x := prefix ^ (seq * 0x9e3779b97f4a7c15)
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	if x == 0 {
		return 1
	}
	return x
}

const hexDigits = "0123456789abcdef"

// putHex64 把 v 以 16 位小写十六进制写入 dst（dst 长度必须为 16）
// 手写而非 strconv/hex：避免为每个请求产生中间切片与字符串
func putHex64(dst []byte, v uint64) {
	for i := 15; i >= 0; i-- {
		dst[i] = hexDigits[v&0xf]
		v >>= 4
	}
}
