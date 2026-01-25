/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-12-30 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-25 11:07:47
 * @FilePath: \go-stress\main.go
 * @Description: 压测工具主入口
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/kamalyes/go-stress/config"
	"github.com/kamalyes/go-stress/executor"
	"github.com/kamalyes/go-stress/logger"
	"github.com/kamalyes/go-stress/types"
	"github.com/kamalyes/go-toolbox/pkg/osx"
	"github.com/kamalyes/go-toolbox/pkg/units"
)

var (
	// 基础参数
	configFile  string
	curlFile    string
	protocol    string
	concurrency uint64
	requests    uint64
	url         string
	method      string
	timeout     time.Duration

	// HTTP参数
	http2     bool
	keepalive bool

	// gRPC参数
	grpcReflection bool
	grpcService    string
	grpcMethod     string

	// 其他
	body    string
	headers arrayFlags

	// 日志配置
	logLevel string
	logFile  string
	quiet    bool
	verbose  bool

	// 报告配置
	reportPrefix string            // 报告文件名前缀
	storageMode  types.StorageMode // 存储模式 (memory/db)

	// 内存限制
	maxMemory string // 内存使用阈值
)

// arrayFlags 数组flag
type arrayFlags []string

func (a *arrayFlags) String() string {
	return fmt.Sprintf("%v", *a)
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

// reportFile 报告文件信息
type reportFile struct {
	name    string
	modTime time.Time
}

func init() {
	// 设置默认值
	storageMode = types.StorageModeMemory

	// 基础参数
	flag.StringVar(&configFile, "config", "", "配置文件路径 (yaml/json)")
	flag.StringVar(&curlFile, "curl", "", "curl命令文件路径")
	flag.StringVar(&protocol, "protocol", "http", "协议类型 (http/grpc/websocket)")
	flag.Uint64Var(&concurrency, "c", 1, "并发数")
	flag.Uint64Var(&requests, "n", 1, "每个并发的请求数")
	flag.StringVar(&url, "url", "", "目标URL")
	flag.StringVar(&method, "method", "GET", "请求方法")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "请求超时时间")

	// HTTP参数
	flag.BoolVar(&http2, "http2", false, "使用HTTP/2")
	flag.BoolVar(&keepalive, "keepalive", false, "使用长连接")

	// gRPC参数
	flag.BoolVar(&grpcReflection, "grpc-reflection", false, "使用gRPC反射")
	flag.StringVar(&grpcService, "grpc-service", "", "gRPC服务名")
	flag.StringVar(&grpcMethod, "grpc-method", "", "gRPC方法名")

	// 其他
	flag.StringVar(&body, "data", "", "请求体数据")
	flag.Var(&headers, "H", "请求头 (可多次使用)")

	// 日志配置
	flag.StringVar(&logLevel, "log-level", "info", "日志级别 (debug/info/warn/error)")
	flag.StringVar(&logFile, "log-file", "", "日志文件路径")
	flag.BoolVar(&quiet, "quiet", false, "静默模式（仅错误）")
	flag.BoolVar(&verbose, "verbose", false, "详细模式（包含调试信息）")

	// 报告配置
	flag.StringVar(&reportPrefix, "report-prefix", "stress-report", "报告文件名前缀")
	flag.Var(&storageMode, "storage", "存储模式 (memory:内存模式 | sqlite:持久化到SQLite文件)")

	// 内存限制
	flag.StringVar(&maxMemory, "max-memory", "", "内存使用阈值，超过后自动停止测试 (如: 1GB, 512MB, 2048KB)")
}

func main() {
	flag.Parse()

	// 初始化日志器
	initLogger()

	// 如果没有任何参数，显示帮助信息
	if len(os.Args) == 1 {
		printBanner()
		printUsage()
		os.Exit(0)
	}

	// 打印banner
	printBanner()

	var cfg *config.Config
	var err error

	// 从curl文件加载
	if curlFile != "" {
		logger.Default.Info("📄 解析curl文件: %s", curlFile)
		cfg, err = config.ParseCurlFile(curlFile)
		if err != nil {
			logger.Default.Fatalf("❌ 解析curl文件失败: %v", err)
		}
		// 如果命令行指定了并发数和请求数，覆盖curl配置
		if concurrency > 0 {
			cfg.Concurrency = concurrency
		}
		if requests > 0 {
			cfg.Requests = requests
		}
		if timeout > 0 {
			cfg.Timeout = timeout
		}
	} else if configFile != "" {
		// 从配置文件加载
		logger.Default.Info("📄 加载配置文件: %s", configFile)
		loader := config.NewLoader()
		cfg, err = loader.LoadFromFile(configFile)
		if err != nil {
			logger.Default.Fatalf("❌ 加载配置文件失败: %v", err)
		}
		// 设置运行模式为配置文件模式
		cfg.RunMode = types.RunModeConfig
	} else {
		// 使用命令行参数
		cfg = buildConfigFromFlags()
		cfg.RunMode = types.RunModeCLI
	}

	// 验证配置
	if err := validateConfig(cfg); err != nil {
		logger.Default.Errorf("❌ 配置验证失败: %v\n", err)
		printUsage()
		os.Exit(1)
	}

	// 创建执行器（根据存储模式选择）
	var exec *executor.Executor

	switch storageMode {
	case types.StorageModeMemory:
		// 内存模式：高速、无限制、不持久化
		logger.Default.Info("💾 存储模式: 内存 (高速、无限制、不持久化)")
		exec, err = executor.NewExecutorWithMemoryStorage(cfg)

	case types.StorageModeSQLite:
		// SQLite 模式：持久化、无限制、可查询
		reportDir := filepath.Join(reportPrefix, fmt.Sprintf("%d", time.Now().Unix()))
		if err := os.MkdirAll(reportDir, os.ModePerm); err != nil {
			logger.Default.Fatalf("❌ 创建报告目录失败: %v", err)
		}
		dbPath := filepath.Join(reportDir, "details.db")
		logger.Default.Info("💾 存储模式: SQLite (持久化、无限制、可查询)")
		logger.Default.Info("💾 数据库路径: %s", dbPath)
		exec, err = executor.NewExecutorWithSQLiteStorage(cfg, dbPath)

	default:
		logger.Default.Fatalf("❌ 未知的存储模式: %s (支持: %s, %s)",
			storageMode, types.StorageModeMemory, types.StorageModeSQLite)
	}

	if err != nil {
		logger.Default.Fatalf("❌ 创建执行器失败: %v", err)
	}

	// 创建context，支持Ctrl+C中断
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Default.Warn("\n\n⚠️  收到中断信号，正在停止...")
		cancel()
	}()

	// 启动内存监控（如果配置了阈值）
	if maxMemory != "" {
		threshold, err := units.ParseBytes(maxMemory)
		if err != nil {
			logger.Default.Warnf("⚠️  内存阈值格式错误: %v,将忽略内存监控", err)
		} else {
			logger.Default.Infof("🔍 启动内存监控，阈值: %s (%d MB)", maxMemory, threshold/(1024*1024))

			// 使用高级内存监控器
			monitor := osx.NewAdvancedMonitor().
				AddThreshold(osx.LevelWarning, threshold*80/100). // 80% 警告
				AddThreshold(osx.LevelCritical, threshold).       // 100% 严重
				SetMetricType(osx.MetricAlloc).
				SetCheckOnce(false).
				SetMaxHistory(200).
				EnableGrowthCheck(20.0, 30*time.Second). // 30秒内增长超过20%告警
				OnWarning(func(snapshot osx.Snapshot) {
					logger.Default.Warnf("[⚠️  警告] 内存使用: %s / %s (%.1f%%), Goroutines: %d",
						units.FormatBytes(snapshot.Alloc),
						maxMemory,
						float64(snapshot.Alloc)/float64(threshold)*100,
						snapshot.Goroutines)
				}).
				OnCritical(func(snapshot osx.Snapshot) {
					logger.Default.Warnf("\n[🚨 严重] 内存使用超过阈值: %s / %s (%.1f%%)",
						units.FormatBytes(snapshot.Alloc),
						maxMemory,
						float64(snapshot.Alloc)/float64(threshold)*100)
					logger.Default.Warnf("  GC次数: %d, Goroutines: %d", snapshot.NumGC, snapshot.Goroutines)
					logger.Default.Warn("🛑 自动停止测试任务...")
					cancel()
				}).
				OnGrowthAlert(func(rate osx.GrowthRate, snapshot osx.Snapshot) {
					logger.Default.Warnf("[📈 增长告警] 增长率: %.2f%%, 绝对增长: %s, 时间窗口: %v",
						rate.Percentage,
						units.FormatBytes(uint64(rate.Absolute)),
						rate.Duration)
				}).
				OnCheck(func(snapshot osx.Snapshot) {
					logger.Default.Debugf("📊 内存监控 - Alloc: %s, Sys: %s, Goroutines: %d, GC: %d",
						units.FormatBytes(snapshot.Alloc),
						units.FormatBytes(snapshot.Sys),
						snapshot.Goroutines,
						snapshot.NumGC)
				})

			go monitor.Start(ctx, 5*time.Second)
		}
	}

	// 执行压测
	report, err := exec.Run(ctx)
	if err != nil {
		// 如果是用户中断（context canceled），不视为错误
		if err.Error() == "执行压测失败: context canceled" ||
			strings.Contains(err.Error(), "context canceled") {
			logger.Default.Warn("⚠️  用户已中断压测")
			// 继续执行后续的报告生成和数据保存
		} else {
			// 其他错误直接退出
			logger.Default.Fatalf("❌ 压测执行失败: %v", err)
		}
	}

	// 打印报告
	if report != nil {
		report.Print()
	}

	// 清理旧报告（保留最近10个）
	cleanOldReports(10)

	// 创建报告目录
	reportDir := filepath.Join(reportPrefix, fmt.Sprintf("%d", time.Now().Unix()))

	if err := os.MkdirAll(reportDir, os.ModePerm); err != nil {
		logger.Default.Warnf("⚠️  创建报告目录失败: %v", err)
		// 即使创建目录失败，也要关闭存储
		if err := exec.GetCollector().Close(); err != nil {
			logger.Default.Warnf("⚠️  关闭存储失败: %v", err)
		}
		return
	}

	// 生成并保存HTML报告（会自动生成配套的 JSON 文件）
	htmlReportFile := filepath.Join(reportDir, "index.html")
	totalDuration := time.Duration(0)
	if report != nil {
		totalDuration = report.TotalTime
	}
	if err := exec.GetCollector().GenerateHTMLReport(totalDuration, htmlReportFile); err != nil {
		logger.Default.Warnf("⚠️  生成HTML报告失败: %v", err)
	} else {
		logger.Default.Info("🌐 在浏览器中打开查看详细图表: file:///%s", htmlReportFile)
	}

	// 确保所有数据都写入存储
	if err := exec.GetCollector().Close(); err != nil {
		logger.Default.Warnf("⚠️  关闭存储失败: %v", err)
	}

	// 获取实时报告服务器端口
	realtimePort := 8088 // 默认端口
	if realtimeServer := exec.GetRealtimeServer(); realtimeServer != nil {
		realtimePort = realtimeServer.GetPort()
	}

	// 等待用户查看报告后手动退出
	logger.Default.Info("\n💡 提示: 实时报告服务器仍在运行")
	logger.Default.Info("   访问 http://localhost:%d 查看实时报告", realtimePort)
	logger.Default.Info("   按 Ctrl+C 退出程序")

	// 阻塞等待中断信号
	<-sigCh
	logger.Default.Info("\n👋 程序已退出")
}

// cleanOldReports 清理旧的报告文件，保留最近的N个
func cleanOldReports(keepCount int) {
	// 获取所有报告文件
	files, err := os.ReadDir(".")
	if err != nil {
		return
	}

	var jsonReports []reportFile
	var htmlReports []reportFile

	// 收集所有报告文件
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		// 匹配报告文件（使用配置的前缀）
		prefix := reportPrefix + "-"
		if strings.HasPrefix(name, prefix) {
			info, err := file.Info()
			if err != nil {
				continue
			}

			rf := reportFile{
				name:    name,
				modTime: info.ModTime(),
			}

			if strings.HasSuffix(name, ".json") {
				jsonReports = append(jsonReports, rf)
			} else if strings.HasSuffix(name, ".html") {
				htmlReports = append(htmlReports, rf)
			}
		}
	}

	// 清理JSON报告
	cleanReportFiles(jsonReports, keepCount)
	// 清理HTML报告
	cleanReportFiles(htmlReports, keepCount)
}

// cleanReportFiles 清理指定类型的报告文件
func cleanReportFiles(files []reportFile, keepCount int) {
	if len(files) <= keepCount {
		return
	}

	// 按修改时间排序（新的在前）
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	// 删除超出保留数量的文件
	for i := keepCount; i < len(files); i++ {
		if err := os.Remove(files[i].name); err != nil {
			logger.Default.Debugf("删除旧报告失败: %s, %v", files[i].name, err)
		} else {
			logger.Default.Debugf("🗑️  已删除旧报告: %s", files[i].name)
		}
	}
}

// buildConfigFromFlags 从命令行参数构建配置
func buildConfigFromFlags() *config.Config {
	cfg := config.DefaultConfig()

	cfg.Protocol = types.ProtocolType(protocol)
	cfg.Concurrency = concurrency
	cfg.Requests = requests
	cfg.URL = url
	cfg.Method = method
	cfg.Timeout = timeout
	cfg.Body = body

	// 解析Headers
	cfg.Headers = make(map[string]string)
	for _, h := range headers {
		parseHeader(h, cfg.Headers)
	}

	// HTTP配置
	if cfg.Protocol == types.ProtocolHTTP {
		cfg.HTTP = &config.HTTPConfig{
			HTTP2:           http2,
			KeepAlive:       keepalive,
			FollowRedirects: true,
			MaxConnsPerHost: 100,
		}
	}

	// gRPC配置
	if cfg.Protocol == types.ProtocolGRPC {
		cfg.GRPC = &config.GRPCConfig{
			UseReflection: grpcReflection,
			Service:       grpcService,
			Method:        grpcMethod,
			Metadata:      make(map[string]string),
		}
	}

	return cfg
}

// validateConfig 验证配置
func validateConfig(cfg *config.Config) error {
	// 多API模式下，URL已经在config.Loader中验证过了
	if len(cfg.APIs) == 0 {
		// 单API模式才检查URL
		if cfg.URL == "" {
			return fmt.Errorf("URL不能为空")
		}
	}

	if cfg.Concurrency == 0 {
		return fmt.Errorf("并发数不能为0")
	}

	if cfg.Requests == 0 {
		return fmt.Errorf("请求数不能为0")
	}

	// gRPC特定验证
	if cfg.Protocol == types.ProtocolGRPC {
		if cfg.GRPC == nil {
			return fmt.Errorf("gRPC配置不能为空")
		}
		if cfg.GRPC.UseReflection {
			if cfg.GRPC.Service == "" {
				return fmt.Errorf("gRPC服务名不能为空")
			}
			if cfg.GRPC.Method == "" {
				return fmt.Errorf("gRPC方法名不能为空")
			}
		}
	}

	return nil
}

// initLogger 初始化日志器
func initLogger() {
	config := logger.DefaultConfig()

	// 优先级：verbose > quiet > logLevel
	if verbose {
		config = config.WithLevel(logger.DEBUG).WithShowCaller(true).WithTimeFormat("2006-01-02 15:04:05.000")
	} else if quiet {
		config = config.WithLevel(logger.ERROR)
	} else {
		config = config.WithLevel(logger.ParseLogLevel(logLevel))
	}

	// 配置输出
	if logFile != "" {
		// 使用轮转文件日志（最大100MB，保留5个备份）
		rotateWriter := logger.NewRotateWriter(logFile, 100*1024*1024, 5)
		config = config.WithOutput(rotateWriter).WithColorful(false)
	}

	logger.SetDefault(logger.New(config))
}

// parseHeader 解析请求头字符串
func parseHeader(header string, headers map[string]string) {
	for i := 0; i < len(header); i++ {
		if header[i] == ':' {
			key := header[:i]
			value := header[i+1:]
			// 去除前后空格
			for len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
			headers[key] = value
			return
		}
	}
}

// printBanner 打印启动banner
func printBanner() {
	logger.Default.Info(`
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║     ⚡ Go Stress Testing Tool ⚡                         ║
║                                                          ║
║     🚀 高性能压测工具                                     ║
║     🔧 支持 HTTP / gRPC / WebSocket                      ║
║     ⚙️  基于 go-toolbox 工具库                           ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
`)
}

// printUsage 打印使用说明
func printUsage() {
	resolver := config.NewVariableResolver()

	printHeader("使用方法:")
	flag.Usage()

	printHeader("基本示例:")
	printExamples()

	printHeader("变量功能:")
	printVariableExamples(resolver)

	printHeader("可用变量函数:")
	printAvailableFunctions(resolver)

	printHeader("配置文件示例 (config.yaml):")
	printConfigExample()
}

func printHeader(title string) {
	fmt.Println("\n" + title)
}

func printExamples() {
	examples := []string{
		"# 简单HTTP压测",
		"go-stress -url https://example.com -c 10 -n 100",
		"",
		"# POST请求",
		"go-stress -url https://api.example.com/users -method POST -data '{\"name\":\"test\"}' -H \"Content-Type: application/json\" -c 5 -n 50",
		"",
		"# 使用配置文件",
		"go-stress -config config.yaml",
		"",
		"# 使用curl文件",
		"go-stress -curl requests.txt -c 10 -n 100",
		"",
		"# 自定义报告前缀",
		"go-stress -url https://example.com -c 10 -n 100 -report-prefix my-test",
		"",
		"# gRPC压测",
		"go-stress -protocol grpc -url localhost:50051 -grpc-reflection -grpc-service myservice -grpc-method MyMethod -c 5 -n 50",
		"",
		"# 实时监控",
		"运行后自动打开浏览器查看实时报告（默认端口: 8088，可通过配置文件的 realtime_port 修改）",
		"测试完成后生成静态HTML报告: stress-report-{时间戳}.html",
	}
	for _, example := range examples {
		fmt.Println(example)
	}
}

func printVariableExamples(resolver *config.VariableResolver) {
	seqExample, _ := resolver.Resolve("{{seq}}")
	unixExample, _ := resolver.Resolve("{{unix}}")

	fmt.Println("  支持在 URL、请求体、请求头中使用变量，使用 {{variable}} 或 {{function}} 语法")
	fmt.Println("  go-stress -url 'https://api.example.com/user/{{seq}}' -c 10 -n 100")
	fmt.Printf("    实际示例: https://api.example.com/user/%s\n", seqExample)
	fmt.Println("  go-stress -data '{\"timestamp\": {{unix}}, \"id\": {{seq}}}' ...")
	fmt.Printf("    实际示例: {\"timestamp\": %s, \"id\": %s}\n", unixExample, seqExample)

	printRandomExamples(resolver)
	printEnvironmentExamples(resolver)
}

func printRandomExamples(resolver *config.VariableResolver) {
	randomStr, _ := resolver.Resolve("{{randomString 8}}")
	randomInt, _ := resolver.Resolve("{{randomInt 18 60}}")
	randomUUID, _ := resolver.Resolve("{{randomUUID}}")

	fmt.Println("  # 随机值")
	fmt.Println("  go-stress -data '{\"username\": \"user_{{randomString 8}}\", \"age\": {{randomInt 18 60}}}' ...")
	fmt.Printf("    实际示例: {\"username\": \"user_%s\", \"age\": %s}\n", randomStr, randomInt)
	fmt.Println("  go-stress -H 'X-Request-ID: {{randomUUID}}' ...")
	fmt.Printf("    实际示例: X-Request-ID: %s\n", randomUUID)
}

func printEnvironmentExamples(resolver *config.VariableResolver) {
	hostname, _ := resolver.Resolve("{{hostname}}")
	dateExample, _ := resolver.Resolve("{{date \"2006-01-02 15:04:05\"}}")

	fmt.Println("  # 环境变量和其他")
	fmt.Println("  go-stress -H 'X-Hostname: {{hostname}}' ...")
	fmt.Printf("    实际示例: X-Hostname: %s\n", hostname)
	fmt.Println("  go-stress -data '{\"date\": \"{{date \"2006-01-02 15:04:05\"}}\"}' ...")
	fmt.Printf("    实际示例: {\"date\": \"%s\"}\n", dateExample)
}

func printAvailableFunctions(resolver *config.VariableResolver) {
	seqExample, _ := resolver.Resolve("{{seq}}")
	unixExample, _ := resolver.Resolve("{{unix}}")
	unixNano, _ := resolver.Resolve("{{unixNano}}")
	timestamp, _ := resolver.Resolve("{{timestamp}}")
	randomInt, _ := resolver.Resolve("{{randomInt 1 100}}")
	randomFloat, _ := resolver.Resolve("{{randomFloat 0.0 1.0}}")
	randomStr, _ := resolver.Resolve("{{randomString 10}}")
	hostname, _ := resolver.Resolve("{{hostname}}")
	localIP, _ := resolver.Resolve("{{localIP}}")
	md5Ex, _ := resolver.Resolve("{{md5 \"test\"}}")
	sha1Ex, _ := resolver.Resolve("{{sha1 \"test\"}}")
	base64Ex, _ := resolver.Resolve("{{base64 \"hello\"}}")
	urlEncodeEx, _ := resolver.Resolve("{{urlEncode \"a b c\"}}")

	fmt.Println("  环境变量:")
	fmt.Println("    {{env \"VAR_NAME\"}}           - 获取环境变量")
	fmt.Printf("    {{hostname}}                  - 主机名 (示例: %s)\n", hostname)
	fmt.Printf("    {{localIP}}                   - 本机IP (示例: %s)\n", localIP)

	fmt.Println("  序列号:")
	fmt.Printf("    {{seq}}                       - 自增序列号 (示例: %s)\n", seqExample)

	fmt.Println("  时间函数:")
	fmt.Printf("    {{unix}}                      - Unix时间戳/秒 (示例: %s)\n", unixExample)
	fmt.Printf("    {{unixNano}}                  - Unix纳秒时间戳 (示例: %s)\n", unixNano)
	fmt.Printf("    {{timestamp}}                 - Unix毫秒时间戳 (示例: %s)\n", timestamp)

	fmt.Println("  随机函数:")
	fmt.Printf("    {{randomInt 1 100}}           - 随机整数 (示例: %s)\n", randomInt)
	fmt.Printf("    {{randomFloat 0.0 1.0}}       - 随机浮点数 (示例: %s)\n", randomFloat)
	fmt.Printf("    {{randomString 10}}           - 随机字符串 (示例: %s)\n", randomStr)

	fmt.Println("  加密/编码:")
	fmt.Printf("    {{md5 \"text\"}}               - MD5 (示例: %s)\n", md5Ex)
	fmt.Printf("    {{sha1 \"text\"}}              - SHA1 (示例: %s...)\n", sha1Ex[:16])
	fmt.Printf("    {{base64 \"text\"}}            - Base64 (示例: %s)\n", base64Ex)
	fmt.Printf("    {{urlEncode \"a b\"}}          - URL编码 (示例: %s)\n", urlEncodeEx)
}

func printConfigExample() {
	fmt.Println("protocol: http")
	fmt.Println("url: https://api.example.com/users")
	fmt.Println("method: POST")
	fmt.Println("concurrency: 10")
	fmt.Println("requests: 100")
	fmt.Println("headers:")
	fmt.Println("  Content-Type: application/json")
	fmt.Println("  X-Request-ID: \"{{randomUUID}}\"")
	fmt.Println("  X-Trace-ID: \"{{md5 (print (seq) (timestamp))}}\"")
	fmt.Println("  Authorization: \"Bearer {{env \"API_TOKEN\"}}\"")
	fmt.Println("body: |")
	fmt.Println("  {")
	fmt.Println("    \"id\": {{seq}},")
	fmt.Println("    \"username\": \"user_{{randomString 8}}\",")
	fmt.Println("    \"email\": \"{{randomEmail}}\",")
	fmt.Println("    \"phone\": \"{{randomPhone}}\",")
	fmt.Println("    \"timestamp\": {{timestamp}},")
	fmt.Println("    \"client_ip\": \"{{randomIP}}\",")
	fmt.Println("    \"token\": \"{{base64 (randomString 16)}}\"")
	fmt.Println("  }")
}
