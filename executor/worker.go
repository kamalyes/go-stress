/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 20:10:56
 * @FilePath: \go-stress\executor\worker.go
 * @Description: Worker实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-stress/statistics"
	"github.com/kamalyes/go-stress/verify"
)

// WorkerDependencyContext 每个 worker 的本地依赖上下文
type WorkerDependencyContext struct {
	extractedVars map[string]string // 本地提取的变量
	failedAPIs    map[string]bool   // 本地失败的 API
}

// copyHeaders 深拷贝 Headers map，避免并发修改
func copyHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}

	newHeaders := make(map[string]string, len(headers))
	for k, v := range headers {
		newHeaders[k] = v
	}
	return newHeaders
}

// NewWorkerDependencyContext 创建新的依赖上下文
func NewWorkerDependencyContext() *WorkerDependencyContext {
	return &WorkerDependencyContext{
		extractedVars: make(map[string]string),
		failedAPIs:    make(map[string]bool),
	}
}

// Worker 工作单元
type Worker struct {
	id          uint64
	client      Client
	handler     RequestHandler
	collector   *statistics.Collector
	reqCount    uint64
	reqBuilder  *RequestBuilder          // 单API模式使用
	apiSelector APISelector              // 多API模式使用
	varResolver *config.VariableResolver // 动态变量解析器
	controller  Controller               // 控制器
	depContext  *WorkerDependencyContext // 本地依赖上下文
}

// WorkerConfig Worker配置
type WorkerConfig struct {
	ID          uint64
	Client      Client
	Handler     RequestHandler
	Collector   *statistics.Collector
	ReqCount    uint64
	ReqBuilder  *RequestBuilder // 单API模式使用（可选）
	APISelector APISelector     // 多API模式使用（可选）
	Controller  Controller      // 控制器（可选）
}

// NewWorker 创建Worker
func NewWorker(cfg WorkerConfig, varResolver *config.VariableResolver) *Worker {
	ctrl := cfg.Controller
	if ctrl == nil {
		ctrl = &NoOpController{}
	}

	return &Worker{
		id:          cfg.ID,
		client:      cfg.Client,
		handler:     cfg.Handler,
		collector:   cfg.Collector,
		reqCount:    cfg.ReqCount,
		reqBuilder:  cfg.ReqBuilder,
		apiSelector: cfg.APISelector,
		varResolver: varResolver,
		controller:  ctrl,
		depContext:  NewWorkerDependencyContext(),
	}
}

// Run 运行Worker
func (w *Worker) Run(ctx context.Context) error {
	// 建立连接
	if err := w.client.Connect(ctx); err != nil {
		logger.Default.Errorf("❌ Worker %d: 连接失败: %v", w.id, err)
		return err
	}
	defer w.client.Close()

	// 判断是否是依赖链模式
	isDependencyMode := w.apiSelector != nil && w.apiSelector.HasDependencies()
	var executionOrder []string
	var resolver *DependencyResolver

	if isDependencyMode {
		// 获取依赖链的执行顺序
		resolver = w.apiSelector.GetDependencyResolver()
		if resolver != nil {
			executionOrder = resolver.GetExecutionOrder()
		}
	}

	// 执行请求
	for i := uint64(0); i < w.reqCount; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 检查是否停止
		if w.controller.IsStopped() {
			return nil
		}

		// 等待暂停结束
		if stopped := WaitWhilePaused(w.controller); stopped {
			return nil
		}

		// 每次新的请求序列，重置本地依赖上下文
		w.depContext = NewWorkerDependencyContext()

		// 计算分组ID（(Worker ID + 1) * 100000 + 请求序号，确保全局唯一）
		groupID := (w.id+1)*100000 + i + 1

		// 在依赖链模式下，按顺序执行完整的依赖链
		if isDependencyMode && len(executionOrder) > 0 {
			for _, apiName := range executionOrder {
				// 检查是否停止
				if w.controller.IsStopped() {
					return nil
				}

				// 等待暂停结束
				if stopped := WaitWhilePaused(w.controller); stopped {
					return nil
				}

				// 获取 API 配置以检查重复次数
				api := resolver.GetAPI(apiName)
				if api == nil {
					logger.Default.Errorf("Worker %d: 找不到 API [%s]", w.id, apiName)
					continue
				}

				// 确定重复次数（默认为1）
				repeatCount := 1
				if api.Repeat > 0 {
					repeatCount = api.Repeat
				}

				// 执行指定次数
				for r := 0; r < repeatCount; r++ {
					// 检查是否停止
					if w.controller.IsStopped() {
						return nil
					}

					// 等待暂停结束
					if stopped := WaitWhilePaused(w.controller); stopped {
						return nil
					}

					// 每次执行使用不同的 groupID（对于重复执行的API）
					currentGroupID := groupID
					if r > 0 {
						// 如果是重复执行，在原groupID基础上增加一个小的偏移
						currentGroupID = groupID + uint64(r)*100
					}

					// 直接按顺序执行指定的 API
					w.executeRequestByName(ctx, apiName, resolver, currentGroupID)
				}
			}
		} else {
			// 单API模式或其他模式，执行一次
			w.executeRequest(ctx)
		}
	}

	return nil
}

// executeRequest 执行单次请求
func (w *Worker) executeRequest(ctx context.Context) {
	// 构建请求
	var req *Request
	var apiCfg *APIRequestConfig

	if w.apiSelector != nil {
		// 多API模式：从选择器获取下一个API
		apiCfg = w.apiSelector.Next()
		if apiCfg == nil {
			logger.Default.Error("API选择器返回空配置")
			return
		}

		// 检查是否被标记为跳过（使用本地上下文）
		if w.shouldSkipAPI(apiCfg.Name) {
			// 替换提取的变量（保留完整配置）
			apiCfg = w.replaceExtractedVarsLocal(apiCfg)
			apiCfg = w.resolveAPIConfigVariables(apiCfg)

			// 找出具体失败的依赖API
			failedDeps := w.getFailedDependencies(apiCfg.Name)
			skipReason := fmt.Sprintf("依赖的API失败: %s", strings.Join(failedDeps, ", "))

			// 跳过该API，记录完整配置但标记为跳过
			result := &RequestResult{
				Success:    false,
				Skipped:    true,
				SkipReason: skipReason,
				GroupID:    0, // 非依赖模式下 GroupID 为 0
				APIName:    apiCfg.Name,
				StatusCode: 0,
				Duration:   0,
				Error:      fmt.Errorf("%s", skipReason),
				Timestamp:  time.Now(),
				URL:        apiCfg.URL,
				Method:     apiCfg.Method,
				Headers:    apiCfg.Headers,
				Body:       apiCfg.Body,
				// 记录配置的验证规则（虽未执行，但有助于排查）
				Verifications: w.buildPlannedVerifications(apiCfg),
			}
			w.collector.Collect(result)
			logger.Default.Warnf("⏭️  Worker %d: 跳过 API [%s]，%s", w.id, apiCfg.Name, skipReason)
			return
		}

		// 如果有依赖关系，需要替换提取的变量（使用本地上下文）
		if w.apiSelector.HasDependencies() {
			apiCfg = w.replaceExtractedVarsLocal(apiCfg)
		}

		// 解析动态变量
		apiCfg = w.resolveAPIConfigVariables(apiCfg)
	} else if w.reqBuilder != nil {
		// 单API模式：使用请求构建器
		req = w.reqBuilder.Build()

		// 解析动态变量
		req = w.resolveRequestVariables(req)
	} else {
		logger.Default.Error("Worker既没有API选择器也没有请求构建器")
		return
	}

	if req == nil && apiCfg != nil {
		req = BuildRequest(apiCfg)
	}

	// 执行请求（通过中间件链）
	resp, err := w.handler(ctx, req)

	// 先提取变量（无论验证是否通过都提取）
	var extractedVars map[string]string
	if apiCfg != nil && len(apiCfg.Extractors) > 0 && resp != nil {
		extractedVars = w.extractAndStoreVarsLocal(apiCfg, req, resp)
	}

	// 标记验证是否成功
	verifySuccess := true

	// 如果请求本身失败，标记为失败
	if apiCfg != nil && err != nil {
		verifySuccess = false
		w.markAPIFailedLocal(apiCfg.Name)
		logger.Default.Errorf("❌ Worker %d: API [%s] 请求失败: %v，后续依赖的API将被跳过", w.id, apiCfg.Name, err)
	} else if apiCfg != nil && len(apiCfg.Verify) > 0 && resp != nil {
		// 如果有API级别的验证配置，执行验证
		verifyErr := w.executeVerifications(apiCfg, resp)
		if verifyErr != nil {
			err = verifyErr
			// 检查是否所有验证都设置了 continue_on_failure
			allContinueOnFailure := true
			for _, verify := range apiCfg.Verify {
				if !verify.ContinueOnFailure {
					allContinueOnFailure = false
					break
				}
			}

			if !allContinueOnFailure {
				verifySuccess = false
				// 标记为本地失败
				w.markAPIFailedLocal(apiCfg.Name)
				logger.Default.Errorf("❌ Worker %d: API [%s] 验证失败: %v，后续依赖的API将被跳过", w.id, apiCfg.Name, verifyErr)
			} else {
				// 所有验证都设置了忽略失败，记录警告但不阻断后续
				logger.Default.Warnf("⚠️  Worker %d: API [%s] 验证失败: %v，但已设置忽略错误，继续执行后续 API", w.id, apiCfg.Name, verifyErr)
				verifySuccess = true // 设置为成功，以便继续执行
				err = nil            // 清除错误，不影响后续执行
			}
		}
	}

	// 如果验证失败，依然使用提取的变量（可能为空或默认值）
	if !verifySuccess && len(extractedVars) > 0 {
		logger.Default.Warnf("⚠️  Worker %d: API [%s] 验证失败，但已提取 %d 个变量（可能为空或默认值）", w.id, apiCfg.Name, len(extractedVars))
	}

	// 记录结果（包含提取的变量）
	result := BuildRequestResult(resp, err)
	result.ExtractedVars = extractedVars
	// 设置 APIName（非依赖模式下 GroupID 保持为 0）
	if apiCfg != nil {
		result.APIName = apiCfg.Name
	}
	w.collector.Collect(result)
}

// executeRequestByName 按名称执行指定的 API（用于依赖链模式）
func (w *Worker) executeRequestByName(ctx context.Context, apiName string, resolver *DependencyResolver, groupID uint64) {
	// 从 resolver 获取 API 配置（即使要跳过也需要配置信息）
	api := resolver.GetAPI(apiName)
	if api == nil {
		logger.Default.Errorf("Worker %d: 找不到 API [%s]", w.id, apiName)
		return
	}

	// 构建 API 请求配置（深拷贝 Headers 避免并发问题）
	apiCfg := &APIRequestConfig{
		Name:       api.Name,
		URL:        api.URL,
		Method:     api.Method,
		Headers:    copyHeaders(api.Headers),
		Body:       api.Body,
		Verify:     api.Verify,
		Extractors: api.Extractors,
	}

	// 检查是否应该跳过
	if w.shouldSkipAPI(apiName) {
		// 替换提取的变量（保留完整配置）
		apiCfg = w.replaceExtractedVarsLocal(apiCfg)
		apiCfg = w.resolveAPIConfigVariables(apiCfg)

		// 找出具体失败的依赖API
		failedDeps := w.getFailedDependencies(apiName)
		skipReason := fmt.Sprintf("依赖的API失败: %s", strings.Join(failedDeps, ", "))

		// 跳过该API，记录完整配置但标记为跳过
		result := &RequestResult{
			Success:    false,
			Skipped:    true,
			SkipReason: skipReason,
			GroupID:    groupID,
			APIName:    apiName,
			StatusCode: 0,
			Duration:   0,
			Error:      fmt.Errorf("%s", skipReason),
			Timestamp:  time.Now(),
			URL:        apiCfg.URL,
			Method:     apiCfg.Method,
			Headers:    apiCfg.Headers,
			Body:       apiCfg.Body,
			// 记录配置的验证规则
			Verifications: w.buildPlannedVerifications(apiCfg),
		}
		w.collector.Collect(result)
		logger.Default.Warnf("⏭️  Worker %d: 跳过 API [%s]，%s", w.id, apiName, skipReason)
		return
	}

	// 从 resolver 获取 API 配置
	api = resolver.GetAPI(apiName)
	if api == nil {
		logger.Default.Errorf("Worker %d: 找不到 API [%s]", w.id, apiName)
		return
	}

	// 构建 API 请求配置（深拷贝 Headers 避免并发问题）
	apiCfg = &APIRequestConfig{
		Name:       api.Name,
		URL:        api.URL,
		Method:     api.Method,
		Headers:    copyHeaders(api.Headers),
		Body:       api.Body,
		Verify:     api.Verify,
		Extractors: api.Extractors,
	}

	// 替换提取的变量（使用本地上下文）
	apiCfg = w.replaceExtractedVarsLocal(apiCfg)

	// 解析动态变量
	apiCfg = w.resolveAPIConfigVariables(apiCfg)

	// 构建请求
	req := BuildRequest(apiCfg)

	// 执行请求
	resp, err := w.handler(ctx, req)

	// 先提取变量（无论验证是否通过都提取）
	var extractedVars map[string]string
	if len(apiCfg.Extractors) > 0 && resp != nil {
		extractedVars = w.extractAndStoreVarsLocal(apiCfg, req, resp)
	}

	// 标记验证是否成功
	verifySuccess := true

	// 如果请求本身失败，标记为失败
	if err != nil {
		verifySuccess = false
		w.markAPIFailedLocal(apiCfg.Name)
		logger.Default.Errorf("❌ Worker %d: API [%s] 请求失败: %v，后续依赖的API将被跳过", w.id, apiCfg.Name, err)
	} else if len(apiCfg.Verify) > 0 && resp != nil {
		// 如果有验证配置，执行验证
		verifyErr := w.executeVerifications(apiCfg, resp)
		if verifyErr != nil {
			err = verifyErr
			// 检查是否所有验证都设置了 continue_on_failure
			allContinueOnFailure := true
			for _, verify := range apiCfg.Verify {
				if !verify.ContinueOnFailure {
					allContinueOnFailure = false
					break
				}
			}

			if !allContinueOnFailure {
				verifySuccess = false
				// 标记为本地失败
				w.markAPIFailedLocal(apiCfg.Name)
				logger.Default.Errorf("❌ Worker %d: API [%s] 验证失败: %v，后续依赖的API将被跳过", w.id, apiCfg.Name, verifyErr)
			} else {
				// 所有验证都设置了忽略失败，记录警告但不阻断后续
				logger.Default.Warnf("⚠️  Worker %d: API [%s] 验证失败: %v，但已设置忽略错误，继续执行后续 API", w.id, apiCfg.Name, verifyErr)
				verifySuccess = true // 设置为成功，以便继续执行
				err = nil            // 清除错误，不影响后续执行
			}
		}
	}

	// 如果验证失败，依然使用提取的变量（可能为空或默认值）
	if !verifySuccess && len(extractedVars) > 0 {
		logger.Default.Warnf("⚠️  Worker %d: API [%s] 验证失败，但已提取 %d 个变量（可能为空或默认值）", w.id, apiCfg.Name, len(extractedVars))
	}

	// 记录结果（包含提取的变量）
	result := BuildRequestResult(resp, err)
	result.GroupID = groupID
	result.APIName = apiName
	result.ExtractedVars = extractedVars
	w.collector.Collect(result)
}

// resolveAPIConfigVariables 解析API配置中的动态变量
func (w *Worker) resolveAPIConfigVariables(apiCfg *APIRequestConfig) *APIRequestConfig {
	if apiCfg == nil || w.varResolver == nil {
		return apiCfg
	}

	w.resolveString(&apiCfg.URL)
	w.resolveString(&apiCfg.Body)
	apiCfg.Headers = w.resolveHeaders(apiCfg.Headers)

	return apiCfg
}

// resolveRequestVariables 解析请求中的动态变量
func (w *Worker) resolveRequestVariables(req *Request) *Request {
	if req == nil || w.varResolver == nil {
		return req
	}

	w.resolveString(&req.URL)
	w.resolveString(&req.Body)
	req.Headers = w.resolveHeaders(req.Headers)

	return req
}

// resolveString 解析单个字符串变量
func (w *Worker) resolveString(s *string) {
	if *s != "" {
		if resolved, err := w.varResolver.Resolve(*s); err == nil {
			*s = resolved
		}
	}
}

// resolveHeaders 解析Headers中的变量（返回新的map，不修改原map）
func (w *Worker) resolveHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}

	// 创建新的 map，避免并发写入问题
	newHeaders := make(map[string]string, len(headers))
	for k, v := range headers {
		if resolved, err := w.varResolver.Resolve(v); err == nil {
			newHeaders[k] = resolved
		} else {
			newHeaders[k] = v
		}
	}
	return newHeaders
}

// replaceExtractedVarsLocal 使用本地上下文替换API配置中的提取变量
func (w *Worker) replaceExtractedVarsLocal(apiCfg *APIRequestConfig) *APIRequestConfig {
	if len(w.depContext.extractedVars) == 0 {
		return apiCfg
	}

	// 复制配置避免修改原始数据
	newCfg := &APIRequestConfig{
		Name:       apiCfg.Name,
		URL:        replaceVars(apiCfg.URL, w.depContext.extractedVars),
		Method:     apiCfg.Method,
		Headers:    make(map[string]string),
		Body:       replaceVars(apiCfg.Body, w.depContext.extractedVars),
		Verify:     apiCfg.Verify,
		Extractors: apiCfg.Extractors,
	}

	// 替换headers中的变量
	for k, v := range apiCfg.Headers {
		newCfg.Headers[k] = replaceVars(v, w.depContext.extractedVars)
	}

	return newCfg
}

// markAPIFailedLocal 标记API在本地上下文中失败
func (w *Worker) markAPIFailedLocal(apiName string) {
	w.depContext.failedAPIs[apiName] = true
}

// shouldSkipAPI 检查是否应该跳过该API（基于本地上下文）
func (w *Worker) shouldSkipAPI(apiName string) bool {
	if !w.apiSelector.HasDependencies() {
		return false
	}

	resolver := w.apiSelector.GetDependencyResolver()
	if resolver == nil {
		return false
	}

	api := resolver.GetAPI(apiName)
	if api == nil {
		return false
	}

	// 检查所有依赖的API是否有失败的（本地上下文）
	for _, dep := range api.DependsOn {
		if w.depContext.failedAPIs[dep] {
			return true
		}
	}

	return false
}

// getFailedDependencies 获取失败的依赖API列表
func (w *Worker) getFailedDependencies(apiName string) []string {
	if !w.apiSelector.HasDependencies() {
		return nil
	}

	resolver := w.apiSelector.GetDependencyResolver()
	if resolver == nil {
		return nil
	}

	api := resolver.GetAPI(apiName)
	if api == nil {
		return nil
	}

	var failedDeps []string
	for _, dep := range api.DependsOn {
		if w.depContext.failedAPIs[dep] {
			failedDeps = append(failedDeps, dep)
		}
	}
	return failedDeps
}

// buildPlannedVerifications 构建计划的验证规则（虽未执行，但记录配置）
func (w *Worker) buildPlannedVerifications(apiCfg *APIRequestConfig) []VerificationResult {
	if len(apiCfg.Verify) == 0 {
		return nil
	}

	var verifications []VerificationResult
	for _, v := range apiCfg.Verify {
		verifications = append(verifications, VerificationResult{
			Type:    v.Type,
			Success: false, // 未执行
			Skipped: true,  // 标记为跳过
			Message: "未执行（请求被跳过）",
			Expect:  fmt.Sprintf("%v", v.Expect),
			Actual:  "-",
		})
	}
	return verifications
}

// replaceVars 替换字符串中的变量占位符 {{.apiName.varName}}
func replaceVars(text string, vars map[string]string) string {
	result := text
	for key, value := range vars {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// extractAndStoreVarsLocal 提取响应数据并存储到本地上下文，并返回提取的原始变量
func (w *Worker) extractAndStoreVarsLocal(apiCfg *APIRequestConfig, req *Request, resp *Response) map[string]string {
	// 构建默认值映射
	defaultValues := make(map[string]string)
	for _, extCfg := range apiCfg.Extractors {
		if extCfg.Default != "" {
			defaultValues[extCfg.Name] = extCfg.Default
		}
	}

	// 创建提取器管理器
	manager, err := NewExtractorManager(apiCfg.Extractors)
	if err != nil {
		logger.Default.Errorf("Worker %d: 创建提取器失败 [%s]: %v", w.id, apiCfg.Name, err)
		return nil
	}

	// 构造提取器上下文（传递请求和响应）
	extractCtx := &ExtractorContext{
		Request:   req,
		Response:  resp,
		Variables: w.depContext.extractedVars,
	}

	// 提取所有变量
	extractedVars := manager.ExtractAll(extractCtx, defaultValues)

	// 存储到本地上下文
	if len(extractedVars) > 0 {
		for k, v := range extractedVars {
			// 使用 apiName.varName 作为key
			key := fmt.Sprintf("%s.%s", apiCfg.Name, k)
			w.depContext.extractedVars[key] = v
		}
		logger.Default.Infof("📦 Worker %d: API [%s] 提取了 %d 个变量", w.id, apiCfg.Name, len(extractedVars))
	}

	return extractedVars
}

// executeVerifications 执行API级别的验证
func (w *Worker) executeVerifications(apiCfg *APIRequestConfig, resp *Response) error {
	for _, verifyCfg := range apiCfg.Verify {
		// 复制验证配置，以便修改而不影响原配置
		verifyConfig := verifyCfg

		// 解析验证配置中的变量（特别是 expect 字段）
		if verifyConfig.Expect != nil {
			// 如果是字符串类型，才进行变量替换
			if expectStr, ok := verifyConfig.Expect.(string); ok {
				// 先用 varResolver 解析配置变量（如 {{.session_id}}）
				if w.varResolver != nil {
					if resolved, err := w.varResolver.Resolve(expectStr); err == nil {
						expectStr = resolved
					}
				}
				// 再替换依赖变量占位符（如 {{.send_message.message_id}}）
				resolvedExpect := replaceVars(expectStr, w.depContext.extractedVars)
				verifyConfig.Expect = resolvedExpect
			}
			// 如果是其他类型（int, float64等），保持原样
		}

		// 解析 JSONPath 中的变量
		if verifyConfig.JSONPath != "" {
			// 先用 varResolver 解析
			if w.varResolver != nil {
				if resolved, err := w.varResolver.Resolve(verifyConfig.JSONPath); err == nil {
					verifyConfig.JSONPath = resolved
				}
			}
			// 再替换依赖变量
			verifyConfig.JSONPath = replaceVars(verifyConfig.JSONPath, w.depContext.extractedVars)
		}

		// 直接创建HTTP验证器（使用 verify 模块）
		httpVerifier := verify.NewHTTPVerifier(&verifyConfig)

		// 执行验证
		isValid, verifyErr := httpVerifier.Verify(resp)
		if !isValid {
			if verifyErr != nil {
				return fmt.Errorf("响应验证失败: %w", verifyErr)
			}
			return fmt.Errorf("响应验证失败")
		}
	}
	return nil
}
