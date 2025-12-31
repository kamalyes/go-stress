/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-30 10:05:47
 * @FilePath: \go-stress\protocol\grpc_reflection.go
 * @Description: gRPC反射支持
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package protocol

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// GRPCReflector gRPC反射辅助
type GRPCReflector struct {
	client   grpc_reflection_v1alpha.ServerReflectionClient
	services map[string]*serviceDesc
}

// serviceDesc 服务描述
type serviceDesc struct {
	descriptor protoreflect.ServiceDescriptor
	methods    map[string]*methodDesc
}

// methodDesc 方法描述
type methodDesc struct {
	descriptor protoreflect.MethodDescriptor
	inputType  protoreflect.MessageDescriptor
	outputType protoreflect.MessageDescriptor
}

// NewGRPCReflector 创建gRPC反射辅助
func NewGRPCReflector() *GRPCReflector {
	return &GRPCReflector{
		services: make(map[string]*serviceDesc),
	}
}

// Init 初始化反射客户端
func (r *GRPCReflector) Init(conn *grpc.ClientConn) error {
	r.client = grpc_reflection_v1alpha.NewServerReflectionClient(conn)
	return nil
}

// Invoke 通过反射调用gRPC方法
func (r *GRPCReflector) Invoke(
	ctx context.Context,
	conn *grpc.ClientConn,
	service string,
	method string,
	requestJSON []byte,
) ([]byte, error) {
	// 获取方法描述
	methodDesc, err := r.getMethodDescriptor(ctx, service, method)
	if err != nil {
		return nil, fmt.Errorf("获取方法描述失败: %w", err)
	}

	// 创建动态请求消息
	reqMsg := dynamicpb.NewMessage(methodDesc.inputType)

	// 解析JSON到proto消息
	if err := protojson.Unmarshal(requestJSON, reqMsg); err != nil {
		return nil, fmt.Errorf("解析请求JSON失败: %w", err)
	}

	// 创建动态响应消息
	respMsg := dynamicpb.NewMessage(methodDesc.outputType)

	// 构建完整的方法名
	fullMethod := fmt.Sprintf("/%s/%s", service, method)

	// 调用gRPC方法
	err = conn.Invoke(ctx, fullMethod, reqMsg, respMsg)
	if err != nil {
		return nil, fmt.Errorf("调用gRPC方法失败: %w", err)
	}

	// 将响应转换为JSON
	responseJSON, err := protojson.Marshal(respMsg)
	if err != nil {
		return nil, fmt.Errorf("序列化响应失败: %w", err)
	}

	return responseJSON, nil
}

// getMethodDescriptor 获取方法描述符
func (r *GRPCReflector) getMethodDescriptor(ctx context.Context, service, method string) (*methodDesc, error) {
	// 检查缓存
	if svcDesc, ok := r.services[service]; ok {
		if methodDesc, ok := svcDesc.methods[method]; ok {
			return methodDesc, nil
		}
	}

	// 通过反射获取服务描述
	fileDesc, err := r.getFileDescriptor(ctx, service)
	if err != nil {
		return nil, err
	}

	// 查找服务描述符
	services := fileDesc.Services()
	var svcDescriptor protoreflect.ServiceDescriptor
	for i := 0; i < services.Len(); i++ {
		svc := services.Get(i)
		if string(svc.FullName()) == service {
			svcDescriptor = svc
			break
		}
	}

	if svcDescriptor == nil {
		return nil, fmt.Errorf("服务不存在: %s", service)
	}

	// 查找方法描述符
	methods := svcDescriptor.Methods()
	var methodDescriptor protoreflect.MethodDescriptor
	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		if string(m.Name()) == method {
			methodDescriptor = m
			break
		}
	}

	if methodDescriptor == nil {
		return nil, fmt.Errorf("方法不存在: %s", method)
	}

	// 创建方法描述
	desc := &methodDesc{
		descriptor: methodDescriptor,
		inputType:  methodDescriptor.Input(),
		outputType: methodDescriptor.Output(),
	}

	// 缓存
	if _, ok := r.services[service]; !ok {
		r.services[service] = &serviceDesc{
			descriptor: svcDescriptor,
			methods:    make(map[string]*methodDesc),
		}
	}
	r.services[service].methods[method] = desc

	return desc, nil
}

// getFileDescriptor 通过反射获取文件描述符
func (r *GRPCReflector) getFileDescriptor(ctx context.Context, symbol string) (protoreflect.FileDescriptor, error) {
	// 创建反射流
	stream, err := r.client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建反射流失败: %w", err)
	}
	defer stream.CloseSend()

	// 发送请求
	req := &grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}

	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("发送反射请求失败: %w", err)
	}

	// 接收响应
	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("接收反射响应失败: %w", err)
	}

	// 解析文件描述符
	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		return nil, fmt.Errorf("未获取到文件描述符")
	}

	// 解析proto文件
	fdProtoList := fdResp.GetFileDescriptorProto()
	if len(fdProtoList) == 0 {
		return nil, fmt.Errorf("文件描述符为空")
	}

	// 解析第一个文件描述符
	var fdProto descriptorpb.FileDescriptorProto
	if err := proto.Unmarshal(fdProtoList[0], &fdProto); err != nil {
		return nil, fmt.Errorf("解析文件描述符失败: %w", err)
	}

	// 构建文件描述符
	fileDesc, err := protodesc.NewFile(&fdProto, protoregistry.GlobalFiles)
	if err != nil {
		return nil, fmt.Errorf("创建文件描述符失败: %w", err)
	}

	return fileDesc, nil
}

// ListServices 列出所有服务（可选功能）
func (r *GRPCReflector) ListServices(ctx context.Context) ([]string, error) {
	stream, err := r.client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建反射流失败: %w", err)
	}
	defer stream.CloseSend()

	req := &grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	}

	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("接收响应失败: %w", err)
	}

	listResp := resp.GetListServicesResponse()
	if listResp == nil {
		return nil, fmt.Errorf("未获取到服务列表")
	}

	services := make([]string, 0, len(listResp.Service))
	for _, svc := range listResp.Service {
		services = append(services, svc.Name)
	}

	return services, nil
}
