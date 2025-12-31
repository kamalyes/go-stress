/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-12-31 13:25:00
 * @FilePath: \go-stress\types\enums.go
 * @Description: 枚举类型定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package types

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

// VerifyType 验证类型
type VerifyType string

const (
	VerifyTypeStatusCode VerifyType = "STATUS_CODE" // 状态码验证（支持操作符：=, !=, >, <, >=, <=）
	VerifyTypeJSONPath   VerifyType = "JSONPATH"    // JSONPath验证（支持操作符：=, !=, >, <, >=, <=, contains, hasPrefix, hasSuffix）
	VerifyTypeContains   VerifyType = "CONTAINS"    // 包含字符串验证
	VerifyTypeRegex      VerifyType = "REGEX"       // 正则表达式验证
	VerifyTypeCustom     VerifyType = "CUSTOM"      // 自定义验证
)

// ExpectOperator 比较操作符（用于 STATUS_CODE/JSONPATH 等）
// 说明：该类型序列化为字符串，配置里仍然填写字符串即可。
type ExpectOperator string

const (
	OpEQ          ExpectOperator = "eq"           // 等于
	OpNE          ExpectOperator = "ne"           // 不等于
	OpGT          ExpectOperator = "gt"           // 大于
	OpGTE         ExpectOperator = "gte"          // 大于等于
	OpLT          ExpectOperator = "lt"           // 小于
	OpLTE         ExpectOperator = "lte"          // 小于等于
	OpContains    ExpectOperator = "contains"     // 字符串包含
	OpNotContains ExpectOperator = "not_contains" // 字符串不包含
	OpHasPrefix   ExpectOperator = "has_prefix"   // 字符串前缀
	OpHasSuffix   ExpectOperator = "has_suffix"   // 字符串后缀
	OpEmpty       ExpectOperator = "empty"        // 为空
	OpNotEmpty    ExpectOperator = "not_empty"    // 不为空
	OpRegex       ExpectOperator = "regex"        // 正则表达式匹配

	// 符号形式的操作符（兼容性）
	OpEqual              ExpectOperator = "="      // 等于（符号）
	OpDoubleEqual        ExpectOperator = "=="     // 等于（符号）
	OpNotEqual           ExpectOperator = "!="     // 不等于（符号）
	OpGreaterThan        ExpectOperator = ">"      // 大于（符号）
	OpGreaterThanOrEqual ExpectOperator = ">="     // 大于等于（符号）
	OpLessThan           ExpectOperator = "<"      // 小于（符号）
	OpLessThanOrEqual    ExpectOperator = "<="     // 小于等于（符号）
	OpRegexp             ExpectOperator = "regexp" // 正则表达式（别名）
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
