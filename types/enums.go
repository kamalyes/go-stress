/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 12:12:39
 * @FilePath: \go-stress\types\enums.go
 * @Description: 枚举类型定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package types

import "github.com/kamalyes/go-toolbox/pkg/validator"

// HTTPMethod HTTP请求方法
type HTTPMethod string

const (
	MethodGet     HTTPMethod = "GET"
	MethodPost    HTTPMethod = "POST"
	MethodPut     HTTPMethod = "PUT"
	MethodDelete  HTTPMethod = "DELETE"
	MethodPatch   HTTPMethod = "PATCH"
	MethodHead    HTTPMethod = "HEAD"
	MethodOptions HTTPMethod = "OPTIONS"
	MethodTrace   HTTPMethod = "TRACE"
	MethodConnect HTTPMethod = "CONNECT"
)

// ExtractorType 提取器类型
type ExtractorType string

const (
	ExtractorTypeJSONPath ExtractorType = "JSONPATH" // JSONPath提取
	ExtractorTypeRegex    ExtractorType = "REGEX"    // 正则表达式提取
	ExtractorTypeHeader   ExtractorType = "HEADER"   // 响应头提取
)

// AuthType 认证类型
type AuthType string

const (
	AuthTypeNone   AuthType = "NONE"   // 无认证
	AuthTypeBasic  AuthType = "BASIC"  // Basic认证
	AuthTypeBearer AuthType = "BEARER" // Bearer Token认证
	AuthTypeSign   AuthType = "SIGN"   // 签名认证
)

// RunMode 运行模式
type RunMode string

const (
	RunModeMaster        RunMode = "master"
	RunModeSlave         RunMode = "slave"
	RunModeStandaloneCLI RunMode = "cli" // 独立模式
)

// RunMode 实现 flag.Value 接口
func (s *RunMode) String() string {
	if s == nil {
		return string(RunModeStandaloneCLI)
	}
	return string(*s)
}

func (s *RunMode) Set(value string) error {
	*s = RunMode(value)
	return nil
}

// VerifyType 验证类型
type VerifyType string

const (
	// 基础验证
	VerifyTypeStatusCode VerifyType = "STATUS_CODE" // 状态码验证（支持操作符：=, !=, >, <, >=, <=）
	VerifyTypeJSONPath   VerifyType = "JSONPATH"    // JSONPath验证（支持操作符：=, !=, >, <, >=, <=, contains, hasPrefix, hasSuffix）
	VerifyTypeContains   VerifyType = "CONTAINS"    // 包含字符串验证
	VerifyTypeRegex      VerifyType = "REGEX"       // 正则表达式验证

	// JSON 相关验证
	VerifyTypeJSONSchema VerifyType = "JSON_SCHEMA" // JSON Schema 验证
	VerifyTypeJSONValid  VerifyType = "JSON_VALID"  // JSON 格式验证

	// HTTP 相关验证
	VerifyTypeHeader       VerifyType = "HEADER"        // HTTP 响应头验证
	VerifyTypeResponseTime VerifyType = "RESPONSE_TIME" // 响应时间验证（毫秒）
	VerifyTypeResponseSize VerifyType = "RESPONSE_SIZE" // 响应大小验证（字节）

	// 数据格式验证
	VerifyTypeEmail  VerifyType = "EMAIL"  // Email 格式验证
	VerifyTypeIP     VerifyType = "IP"     // IP 地址验证（支持 IPv4/IPv6）
	VerifyTypeURL    VerifyType = "URL"    // URL 格式验证
	VerifyTypeUUID   VerifyType = "UUID"   // UUID 格式验证
	VerifyTypeBase64 VerifyType = "BASE64" // Base64 编码验证

	// 字符串验证
	VerifyTypeLength   VerifyType = "LENGTH"    // 字符串长度验证
	VerifyTypePrefix   VerifyType = "PREFIX"    // 前缀验证
	VerifyTypeSuffix   VerifyType = "SUFFIX"    // 后缀验证
	VerifyTypeEmpty    VerifyType = "EMPTY"     // 空值验证
	VerifyTypeNotEmpty VerifyType = "NOT_EMPTY" // 非空验证

	// 自定义
	VerifyTypeCustom VerifyType = "CUSTOM" // 自定义验证
)

// ExpectOperator 比较操作符（用于 STATUS_CODE/JSONPATH 等）
// 使用 go-toolbox/validator 提供的 CompareOperator
type ExpectOperator = validator.CompareOperator

// 操作符常量 - 直接引用 validator 中的定义
const (
	OpEQ                 = validator.OpEqual
	OpNE                 = validator.OpNotEqual
	OpGT                 = validator.OpGreaterThan
	OpGTE                = validator.OpGreaterThanOrEqual
	OpLT                 = validator.OpLessThan
	OpLTE                = validator.OpLessThanOrEqual
	OpContains           = validator.OpContains
	OpNotContains        = validator.OpNotContains
	OpHasPrefix          = validator.OpHasPrefix
	OpHasSuffix          = validator.OpHasSuffix
	OpEmpty              = validator.OpEmpty
	OpNotEmpty           = validator.OpNotEmpty
	OpRegex              = validator.OpRegex
	OpEqual              = validator.OpSymbolEqual
	OpDoubleEqual        = validator.OpSymbolEqual
	OpNotEqual           = validator.OpSymbolNotEqual
	OpGreaterThan        = validator.OpSymbolGreaterThan
	OpGreaterThanOrEqual = validator.OpSymbolGreaterThanOrEqual
	OpLessThan           = validator.OpSymbolLessThan
	OpLessThanOrEqual    = validator.OpSymbolLessThanOrEqual
	OpRegexp             = validator.OpRegex
)

// ToString
func (vt VerifyType) ToString() string {
	return string(vt)
}

// Verifier 验证器接口
type Verifier interface {
	// Verify 验证响应
	Verify(resp *Response) (bool, error)
}

// ContentType 内容类型
type ContentType string

const (
	ContentTypeJSON          ContentType = "application/json"
	ContentTypeXML           ContentType = "application/xml"
	ContentTypeForm          ContentType = "application/x-www-form-urlencoded"
	ContentTypeMultipartForm ContentType = "multipart/form-data"
	ContentTypeText          ContentType = "text/plain"
	ContentTypeHTML          ContentType = "text/html"
	ContentTypeOctetStream   ContentType = "application/octet-stream"
	ContentTypeJavaScript    ContentType = "application/javascript"
	ContentTypeProtobuf      ContentType = "application/protobuf"
	ContentTypeMsgpack       ContentType = "application/msgpack"
)

// StorageMode 存储模式
type StorageMode string

const (
	// StorageModeMemory 内存模式 - 数据存储在内存中，速度快但程序退出后丢失
	StorageModeMemory StorageMode = "memory"

	// StorageModeSQLite 文件模式 - 数据持久化到SQLite，支持海量数据
	StorageModeSQLite StorageMode = "sqlite"
)

// StorageMode 实现 flag.Value 接口
func (s *StorageMode) String() string {
	if s == nil {
		return string(StorageModeMemory)
	}
	return string(*s)
}

func (s *StorageMode) Set(value string) error {
	*s = StorageMode(value)
	return nil
}
