/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 13:00:00
 * @FilePath: \go-stress\executor\middleware.go
 * @Description: 中间件实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"context"
	"fmt"

	"github.com/kamalyes/go-stress/verify"
	"github.com/kamalyes/go-toolbox/pkg/breaker"
	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// MiddlewareChain 中间件链
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewMiddlewareChain 创建中间件链
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]Middleware, 0),
	}
}

// Use 添加中间件
func (mc *MiddlewareChain) Use(m Middleware) *MiddlewareChain {
	mc.middlewares = append(mc.middlewares, m)
	return mc
}

// Build 构建最终的处理器
func (mc *MiddlewareChain) Build(handler RequestHandler) RequestHandler {
	// 从后往前包装中间件
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		handler = mc.middlewares[i](handler)
	}
	return handler
}

// BreakerMiddleware 熔断中间件
func BreakerMiddleware(circuit *breaker.Circuit) Middleware {
	return func(next RequestHandler) RequestHandler {
		return func(ctx context.Context, req *Request) (*Response, error) {
			var resp *Response
			var err error

			breakerErr := circuit.Execute(func() error {
				resp, err = next(ctx, req)
				return err
			})

			if breakerErr != nil {
				return nil, fmt.Errorf("熔断器拦截: %w", breakerErr)
			}

			return resp, err
		}
	}
}

// RetryMiddleware 重试中间件
func RetryMiddleware(retrier *retry.Runner[error]) Middleware {
	return func(next RequestHandler) RequestHandler {
		return func(ctx context.Context, req *Request) (*Response, error) {
			var resp *Response
			var err error

			_, retryErr := retrier.Run(func(retryCtx context.Context) (error, error) {
				resp, err = next(ctx, req)
				return err, err
			})

			if retryErr != nil {
				return resp, retryErr
			}

			return resp, err
		}
	}
}

// VerifyMiddleware 验证中间件
func VerifyMiddleware(verifier verify.Verifier) Middleware {
	return func(next RequestHandler) RequestHandler {
		return func(ctx context.Context, req *Request) (*Response, error) {
			resp, err := next(ctx, req)
			if err != nil {
				return resp, err
			}

			// 验证响应并记录验证结果
			if isValid, verifyErr := verifier.Verify(resp); !isValid {
				if verifyErr != nil {
					return resp, fmt.Errorf("响应验证失败: %w", verifyErr)
				}
				return resp, fmt.Errorf("响应验证失败")
			}

			return resp, nil
		}
	}
}

// ClientMiddleware 客户端执行中间件（最底层）
func ClientMiddleware(client Client) RequestHandler {
	return func(ctx context.Context, req *Request) (*Response, error) {
		return client.Send(ctx, req)
	}
}
