/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 17:29:55
 * @FilePath: \go-stress\protocol\grpc.go
 * @Description: gRPC协议客户端实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package protocol

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/kamalyes/go-stress/config"
)

// GRPCClient gRPC协议客户端
type GRPCClient struct {
	config    *config.Config
	conn      *grpc.ClientConn
	reflector *GRPCReflector // 反射辅助
}

// NewGRPCClient 创建gRPC客户端
func NewGRPCClient(cfg *config.Config) (*GRPCClient, error) {
	if cfg.GRPC == nil {
		return nil, fmt.Errorf("gRPC配置不能为空")
	}

	client := &GRPCClient{
		config: cfg,
	}

	// 如果启用反射，创建反射辅助
	if cfg.GRPC.UseReflection {
		client.reflector = NewGRPCReflector()
	}

	return client, nil
}

// Connect 建立gRPC连接
func (g *GRPCClient) Connect(ctx context.Context) error {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// 支持 TLS 配置
	if g.config.GRPC.TLS != nil && g.config.GRPC.TLS.Enabled {
		var tlsConfig *tls.Config

		if g.config.GRPC.TLS.InsecureSkipVerify {
			// 不验证服务器证书（仅用于测试环境）
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			// 加载 CA 证书
			if g.config.GRPC.TLS.CAFile != "" {
				certPool := x509.NewCertPool()
				ca, err := os.ReadFile(g.config.GRPC.TLS.CAFile)
				if err != nil {
					return fmt.Errorf("读取 CA 证书失败: %w", err)
				}
				if !certPool.AppendCertsFromPEM(ca) {
					return fmt.Errorf("添加 CA 证书失败")
				}
				tlsConfig = &tls.Config{
					RootCAs: certPool,
				}
			}

			// 加载客户端证书（双向认证）
			if g.config.GRPC.TLS.CertFile != "" && g.config.GRPC.TLS.KeyFile != "" {
				cert, err := tls.LoadX509KeyPair(g.config.GRPC.TLS.CertFile, g.config.GRPC.TLS.KeyFile)
				if err != nil {
					return fmt.Errorf("加载客户端证书失败: %w", err)
				}
				if tlsConfig == nil {
					tlsConfig = &tls.Config{}
				}
				tlsConfig.Certificates = []tls.Certificate{cert}
			}
		}

		if tlsConfig != nil {
			// 替换为 TLS 凭证
			opts = []grpc.DialOption{
				grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
			}
		}
	}

	conn, err := grpc.NewClient(g.config.URL, opts...)
	if err != nil {
		return fmt.Errorf("gRPC连接失败: %w", err)
	}

	g.conn = conn

	// 如果使用反射，初始化反射客户端
	if g.reflector != nil {
		if err := g.reflector.Init(conn); err != nil {
			g.conn.Close()
			return fmt.Errorf("初始化gRPC反射失败: %w", err)
		}
	}

	return nil
}

// Send 发送gRPC请求
func (g *GRPCClient) Send(ctx context.Context, req *Request) (*Response, error) {
	startTime := time.Now()

	// 设置超时
	if g.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.config.Timeout)
		defer cancel()
	}

	// 设置metadata
	if len(g.config.GRPC.Metadata) > 0 {
		md := metadata.New(g.config.GRPC.Metadata)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	var respData []byte
	var err error

	// 使用反射或普通调用
	if g.reflector != nil {
		// 反射调用
		respData, err = g.reflector.Invoke(
			ctx,
			g.conn,
			g.config.GRPC.Service,
			g.config.GRPC.Method,
			[]byte(req.Body),
		)
	} else {
		// 普通调用（需要生成的代码）
		return nil, fmt.Errorf("未启用反射模式，暂不支持")
	}

	duration := time.Since(startTime)

	if err != nil {
		return &Response{
			Duration: duration,
			Error:    fmt.Errorf("gRPC调用失败: %w", err),
		}, err
	}

	return &Response{
		StatusCode: 0, // gRPC没有HTTP状态码
		Body:       respData,
		Duration:   duration,
	}, nil
}

// Close 关闭gRPC连接
func (g *GRPCClient) Close() error {
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}

// Type 返回协议类型
func (g *GRPCClient) Type() ProtocolType {
	return ProtocolGRPC
}

// parseRequestBody 解析请求体为map
func parseRequestBody(body string) (map[string]any, error) {
	var data map[string]any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, fmt.Errorf("解析请求体失败: %w", err)
	}
	return data, nil
}
