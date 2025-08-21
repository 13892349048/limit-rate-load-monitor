package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// EnhancedRateLimiterApp 增强的限流应用
type EnhancedRateLimiterApp struct {
	monitor       *EnhancedSystemLoadMonitor
	configManager *ConfigManager
	metricsServer *HTTPMetricsServer
	collector     *MetricsCollector
	profiler      *PerformanceProfiler

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewEnhancedRateLimiterApp 创建增强的限流应用
func NewEnhancedRateLimiterApp(configPath string) (*EnhancedRateLimiterApp, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 初始化配置管理器
	configManager := NewConfigManager(configPath)
	config, err := configManager.LoadConfig()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	// 验证配置
	if err := configManager.ValidateConfig(config); err != nil {
		cancel()
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	// 初始化指标收集器
	collector := NewMetricsCollector()

	// 初始化性能分析器
	profiler := NewPerformanceProfiler()

	// 初始化增强监控器
	monitor := NewEnhancedSystemLoadMonitor()

	// 应用配置到监控器
	if err := applyConfigToMonitor(monitor, config); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to apply config: %v", err)
	}

	// 初始化指标服务器
	var metricsServer *HTTPMetricsServer
	if config.Metrics.EnablePrometheus {
		metricsServer = NewHTTPMetricsServer(config.Metrics.PrometheusPort, collector)
	}

	return &EnhancedRateLimiterApp{
		monitor:       monitor,
		configManager: configManager,
		metricsServer: metricsServer,
		collector:     collector,
		profiler:      profiler,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// applyConfigToMonitor 应用配置到监控器
func applyConfigToMonitor(monitor *EnhancedSystemLoadMonitor, config *Config) error {
	// 解析时间间隔
	monitorInterval, err := ParseDuration(config.Monitor.MonitorInterval)
	if err != nil {
		return fmt.Errorf("invalid monitor interval: %v", err)
	}

	adjustInterval, err := ParseDuration(config.Monitor.AdjustInterval)
	if err != nil {
		return fmt.Errorf("invalid adjust interval: %v", err)
	}

	// 应用配置
	monitor.monitorInterval = monitorInterval
	monitor.adjustInterval = adjustInterval
	monitor.rateLimitConf = &config.RateLimit
	monitor.threshold = &config.Threshold

	return nil
}

// Start 启动应用
func (app *EnhancedRateLimiterApp) Start() error {
	fmt.Println("启动增强动态限流器...")

	// 启动监控器
	app.monitor.Start()

	// 启动指标服务器
	if app.metricsServer != nil {
		app.wg.Add(1)
		go func() {
			defer app.wg.Done()
			if err := app.metricsServer.Start(); err != nil {
				log.Printf("Metrics server error: %v", err)
			}
		}()
		fmt.Printf("指标服务器启动在端口: %d\n", app.configManager.GetConfig().Metrics.PrometheusPort)
	}

	// 启动指标收集循环
	app.wg.Add(1)
	go app.metricsCollectionLoop()

	// 启动性能分析循环
	app.wg.Add(1)
	go app.performanceAnalysisLoop()

	// 启动模拟请求处理
	app.wg.Add(1)
	go app.simulateRequestHandling()

	return nil
}

// Stop 停止应用
func (app *EnhancedRateLimiterApp) Stop() {
	fmt.Println("停止增强动态限流器...")

	// 停止监控器
	app.monitor.Stop()

	// 停止指标服务器
	if app.metricsServer != nil {
		app.metricsServer.Stop()
	}

	// 取消上下文
	app.cancel()

	// 等待所有goroutine结束
	app.wg.Wait()

	fmt.Println("应用已停止")
}

// metricsCollectionLoop 指标收集循环
func (app *EnhancedRateLimiterApp) metricsCollectionLoop() {
	defer app.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			// 收集详细指标
			detailedMetrics := app.monitor.GetDetailedMetrics()
			app.collector.UpdateMetrics(detailedMetrics)
		}
	}
}

// performanceAnalysisLoop 性能分析循环
func (app *EnhancedRateLimiterApp) performanceAnalysisLoop() {
	defer app.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			// 创建性能样本
			metrics := app.monitor.GetSystemMetrics()
			sample := PerformanceSample{
				Timestamp:    time.Now(),
				CPUUsage:     metrics.CPUUsage,
				MemoryUsage:  metrics.MemoryUsage,
				GoroutineNum: metrics.GoroutineNum,
				ResponseTime: metrics.ResponseTime,
				ErrorRate:    metrics.ErrorRate,
				RateLimit:    app.monitor.GetCurrentLimit(),
			}

			app.profiler.AddSample(sample)

			// 每分钟输出平均指标
			if time.Now().Second() == 0 {
				avgMetrics := app.profiler.GetAverageMetrics()
				fmt.Printf("平均性能指标 - CPU: %.1f%%, 内存: %.1f%%, 响应时间: %.1fms, 限流: %d\n",
					avgMetrics.CPUUsage, avgMetrics.MemoryUsage, avgMetrics.ResponseTime, avgMetrics.RateLimit)
			}
		}
	}
}

// simulateRequestHandling 模拟请求处理
func (app *EnhancedRateLimiterApp) simulateRequestHandling() {
	defer app.wg.Done()

	requestCount := 0

	for {
		select {
		case <-app.ctx.Done():
			return
		default:
			// 检查是否允许请求
			if !app.monitor.AllowRequest() {
				fmt.Printf("请求 #%d 被限流\n", requestCount)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			start := time.Now()

			// 模拟请求处理时间（根据系统负载动态调整）
			processingTime := app.calculateProcessingTime()
			time.Sleep(processingTime)

			responseTime := time.Since(start)
			isError := requestCount%25 == 0 // 4% 错误率

			// 记录请求
			app.monitor.RecordEnhancedRequest(responseTime, isError)

			requestCount++

			// 每100个请求输出一次状态
			if requestCount%100 == 0 {
				currentLimit := app.monitor.GetCurrentLimit()
				fmt.Printf("处理了 %d 个请求，当前限流值: %d\n", requestCount, currentLimit)
			}

			// 控制请求频率
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// calculateProcessingTime 根据系统负载计算处理时间
func (app *EnhancedRateLimiterApp) calculateProcessingTime() time.Duration {
	metrics := app.monitor.GetSystemMetrics()

	// 基础处理时间
	baseTime := 10 * time.Millisecond

	// 根据CPU使用率调整
	cpuFactor := 1.0 + (metrics.CPUUsage / 100.0)

	// 根据内存使用率调整
	memFactor := 1.0 + (metrics.MemoryUsage / 200.0)

	// 计算最终处理时间
	finalTime := time.Duration(float64(baseTime) * cpuFactor * memFactor)

	// 限制在合理范围内
	if finalTime < 5*time.Millisecond {
		finalTime = 5 * time.Millisecond
	} else if finalTime > 200*time.Millisecond {
		finalTime = 200 * time.Millisecond
	}

	return finalTime
}

// GetStatus 获取应用状态
func (app *EnhancedRateLimiterApp) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"config":           app.configManager.GetConfig(),
		"system_metrics":   app.monitor.GetSystemMetrics(),
		"detailed_metrics": app.monitor.GetDetailedMetrics(),
		"average_metrics":  app.profiler.GetAverageMetrics(),
		"uptime":           time.Since(app.profiler.startTime).String(),
	}
}

// ExportPerformanceData 导出性能数据
func (app *EnhancedRateLimiterApp) ExportPerformanceData() ([]byte, error) {
	return app.profiler.ExportToJSON()
}

// enhancedMain 增强版主函数
func enhancedMain() {
	// 设置配置文件路径
	configPath := "./config/config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// 创建应用
	app, err := NewEnhancedRateLimiterApp(configPath)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动应用
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("增强动态限流器已启动，按 Ctrl+C 停止")
	fmt.Println("访问 http://localhost:9090/metrics 查看指标")
	fmt.Println("访问 http://localhost:9090/status 查看状态")

	// 等待信号
	<-sigChan

	// 优雅停止
	app.Stop()

	// 导出性能数据
	if data, err := app.ExportPerformanceData(); err == nil {
		fmt.Printf("性能数据已导出: %d 字节\n", len(data))
		// 可以选择保存到文件
		// ioutil.WriteFile("performance_data.json", data, 0644)
	}
}
func main() {
	// 创建增强限流器
	app, err := NewEnhancedRateLimiterApp("./config.json")
	if err != nil {
		log.Fatal(err)
	}

	// 启动监控
	app.Start()
	time.Sleep(100 * time.Second)
	app.Stop()
}
