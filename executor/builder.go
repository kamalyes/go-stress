/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 11:17:55
 * @FilePath: \go-stress\executor\builder.go
 * @Description: 请求构建器和结果构建器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"time"
)

// RequestBuilder 请求构建器
type RequestBuilder struct {
	url     string
	method  string
	headers map[string]string
	body    string
}

// NewRequestBuilder 创建请求构建器
func NewRequestBuilder(url, method string, headers map[string]string, body string) *RequestBuilder {
	return &RequestBuilder{
		url:     url,
		method:  method,
		headers: headers,
		body:    body,
	}
}

// Build 构建请求
func (rb *RequestBuilder) Build() *Request {
	return &Request{
		URL:     rb.url,
		Method:  rb.method,
		Headers: rb.headers,
		Body:    rb.body,
	}
}

// BuildRequestResult 构建请求结果
func BuildRequestResult(resp *Response, err error) *RequestResult {
	result := &RequestResult{
		Success:   err == nil,
		Timestamp: time.Now(),
		Error:     err,
	}

	if resp != nil {
		result.StatusCode = resp.StatusCode
		result.Duration = resp.Duration
		result.Size = float64(len(resp.Body))

		// 填充请求详情
		result.URL = resp.RequestURL
		result.Method = resp.RequestMethod
		result.Query = resp.RequestQuery
		result.Headers = resp.RequestHeaders
		result.Body = resp.RequestBody

		// 填充响应详情
		result.ResponseBody = string(resp.Body)
		result.ResponseHeaders = resp.Headers

		// 填充验证结果
		result.Verifications = resp.Verifications
	}

	return result
}
