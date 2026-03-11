/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-03-11 21:08:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-03-11 21:08:00
 * @FilePath: \go-stress\executor\variable_replacer_test.go
 * @Description: 变量替换器测试 - 重点覆盖模板解析失败时保留原文的行为
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"strings"
	"testing"

	"github.com/kamalyes/go-stress/config"
	"github.com/stretchr/testify/assert"
)

// TestReplaceStringKeepsOriginalOnUnknownFunction 未注册函数导致模板解析失败时必须保留原文
//
// 这是 chat 场景排障的教训：模板里出现未注册函数（如 randomSentence）时，
// text/template 在 Parse 阶段整体失败，{{seq}} 等正常占位符也一并原样保留。
// 若此时行为悄悄改变（比如清空 body 或半渲染），测试会在此处拦住
func TestReplaceStringKeepsOriginalOnUnknownFunction(t *testing.T) {
	resolver := config.NewVariableResolver()
	replacer := NewVariableReplacer(resolver, nil, nil)

	input := `{"message_id": {{seq}}, "content": "{{randomSentence}}"}`
	got := replacer.ReplaceString(input)

	assert.Equal(t, input, got, "模板解析失败时应保留原文，不能产生半渲染结果")
}

// TestReplaceStringResolvesRegisteredFunctions 已注册函数正常渲染
func TestReplaceStringResolvesRegisteredFunctions(t *testing.T) {
	resolver := config.NewVariableResolver()
	replacer := NewVariableReplacer(resolver, nil, nil)

	got := replacer.ReplaceString(`{"content": "{{randomString 16}}"}`)
	assert.NotContains(t, got, "{{", "注册函数应被渲染，不应残留模板占位符")
	assert.True(t, strings.Contains(got, `"content": "`), "模板外的结构必须原样保留")
}

// TestReplaceStringReplacesExtractedVars 第一步提取变量替换仍生效
func TestReplaceStringReplacesExtractedVars(t *testing.T) {
	resolver := config.NewVariableResolver()
	extracted := map[string]string{"create_ticket.session_id": "abc123"}
	replacer := NewVariableReplacer(resolver, extracted, nil)

	got := replacer.ReplaceString(`{"session_id": "{{.create_ticket.session_id}}"}`)
	assert.Equal(t, `{"session_id": "abc123"}`, got)
}
