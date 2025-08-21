package main

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// CPUMonitor CPU监控器
type CPUMonitor struct {
	lastCPUTime int64
	lastSysTime time.Time
	cpuUsage    float64
	mu          sync.RWMutex
}

// NewCPUMonitor 创建CPU监控器
func NewCPUMonitor() *CPUMonitor {
	return &CPUMonitor{
		lastSysTime: time.Now(),
	}
}

// GetCPUUsage 获取CPU使用率（简化版本）
func (c *CPUMonitor) GetCPUUsage() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cpuUsage
}

// UpdateCPUUsage 更新CPU使用率
func (c *CPUMonitor) UpdateCPUUsage() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 基于goroutine数量和系统负载的简化CPU估算
	goroutines := float64(runtime.NumGoroutine())
	cpuCount := float64(runtime.NumCPU())

	// 更精确的估算算法
	baseLoad := (goroutines / (cpuCount * 10)) * 100
	if baseLoad > 100 {
		baseLoad = 100
	}

	// 添加一些随机波动来模拟真实情况
	variation := (float64(time.Now().UnixNano()%100) - 50) / 10
	c.cpuUsage = baseLoad + variation

	if c.cpuUsage < 0 {
		c.cpuUsage = 0
	} else if c.cpuUsage > 100 {
		c.cpuUsage = 100
	}
}

// MemoryMonitor 内存监控器
type MemoryMonitor struct {
	mu sync.RWMutex
}

// GetMemoryStats 获取内存统计信息
func (m *MemoryMonitor) GetMemoryStats() (float64, uint64, uint64) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 计算内存使用率
	usage := float64(memStats.Alloc) / float64(memStats.Sys) * 100

	return usage, memStats.Alloc, memStats.Sys
}

// ResponseTimeTracker 响应时间跟踪器
type ResponseTimeTracker struct {
	samples    []float64
	maxSamples int
	mu         sync.RWMutex
}

// NewResponseTimeTracker 创建响应时间跟踪器
func NewResponseTimeTracker(maxSamples int) *ResponseTimeTracker {
	return &ResponseTimeTracker{
		samples:    make([]float64, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// AddSample 添加响应时间样本
func (rt *ResponseTimeTracker) AddSample(responseTime float64) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.samples = append(rt.samples, responseTime)

	// 保持样本数量在限制内
	if len(rt.samples) > rt.maxSamples {
		rt.samples = rt.samples[1:]
	}
}

// GetAverageResponseTime 获取平均响应时间
func (rt *ResponseTimeTracker) GetAverageResponseTime() float64 {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if len(rt.samples) == 0 {
		return 0
	}

	total := 0.0
	for _, sample := range rt.samples {
		total += sample
	}

	return total / float64(len(rt.samples))
}

// GetPercentile 获取百分位数
func (rt *ResponseTimeTracker) GetPercentile(percentile float64) float64 {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if len(rt.samples) == 0 {
		return 0
	}

	// 简单排序获取百分位数
	sorted := make([]float64, len(rt.samples))
	copy(sorted, rt.samples)

	// 简单冒泡排序
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)) * percentile / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// ErrorRateTracker 错误率跟踪器
type ErrorRateTracker struct {
	totalRequests int64
	errorRequests int64
	windowStart   time.Time
	windowSize    time.Duration
	mu            sync.RWMutex
}

// NewErrorRateTracker 创建错误率跟踪器
func NewErrorRateTracker(windowSize time.Duration) *ErrorRateTracker {
	return &ErrorRateTracker{
		windowStart: time.Now(),
		windowSize:  windowSize,
	}
}

// RecordRequest 记录请求
func (er *ErrorRateTracker) RecordRequest(isError bool) {
	er.mu.Lock()
	defer er.mu.Unlock()

	// 检查是否需要重置窗口
	if time.Since(er.windowStart) > er.windowSize {
		er.totalRequests = 0
		er.errorRequests = 0
		er.windowStart = time.Now()
	}

	atomic.AddInt64(&er.totalRequests, 1)
	if isError {
		atomic.AddInt64(&er.errorRequests, 1)
	}
}

// GetErrorRate 获取错误率
func (er *ErrorRateTracker) GetErrorRate() float64 {
	er.mu.RLock()
	defer er.mu.RUnlock()

	total := atomic.LoadInt64(&er.totalRequests)
	if total == 0 {
		return 0
	}

	errors := atomic.LoadInt64(&er.errorRequests)
	return float64(errors) / float64(total) * 100
}

// EnhancedSystemLoadMonitor 增强的系统负载监控器
type EnhancedSystemLoadMonitor struct {
	metrics       *SystemMetrics
	threshold     *LoadThreshold
	rateLimitConf *RateLimitConfig

	// 性能统计
	requestCount int64
	errorCount   int64
	totalRT      int64
	rtSamples    int64

	// 监控配置
	monitorInterval time.Duration
	adjustInterval  time.Duration

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// 增强组件
	cpuMonitor    *CPUMonitor
	memoryMonitor *MemoryMonitor
	rtTracker     *ResponseTimeTracker
	errorTracker  *ErrorRateTracker
	tokenBucket   *TokenBucket
	slidingWindow *SlidingWindowRateLimiter

	// 配置
	enableTokenBucket   bool
	enableSlidingWindow bool
}

// NewEnhancedSystemLoadMonitor 创建增强的系统负载监控器
func NewEnhancedSystemLoadMonitor() *EnhancedSystemLoadMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	rateLimitConf := &RateLimitConfig{
		BaseLimit:    1000,
		CurrentLimit: 1000,
		MinLimit:     100,
		MaxLimit:     5000,
		AdjustFactor: 1.0,
	}

	enhanced := &EnhancedSystemLoadMonitor{
		metrics: &SystemMetrics{},
		threshold: &LoadThreshold{
			CPUHigh:       70.0,
			CPUCritical:   85.0,
			MemHigh:       80.0,
			MemCritical:   90.0,
			RTHigh:        100.0,
			RTCritical:    300.0,
			ErrorHigh:     5.0,
			ErrorCritical: 10.0,
		},
		rateLimitConf:       rateLimitConf,
		monitorInterval:     5 * time.Second,
		adjustInterval:      10 * time.Second,
		ctx:                 ctx,
		cancel:              cancel,
		cpuMonitor:          NewCPUMonitor(),
		memoryMonitor:       &MemoryMonitor{},
		rtTracker:           NewResponseTimeTracker(1000),
		errorTracker:        NewErrorRateTracker(time.Minute),
		enableTokenBucket:   true,
		enableSlidingWindow: false,
	}

	// 初始化令牌桶
	enhanced.tokenBucket = NewTokenBucket(
		rateLimitConf.CurrentLimit,
		rateLimitConf.CurrentLimit,
	)

	// 初始化滑动窗口
	enhanced.slidingWindow = NewSlidingWindowRateLimiter(
		rateLimitConf.CurrentLimit,
		time.Minute,
	)

	return enhanced
}

// AllowRequest 检查是否允许请求
func (m *EnhancedSystemLoadMonitor) AllowRequest() bool {
	if m.enableTokenBucket {
		return m.tokenBucket.Allow()
	}

	if m.enableSlidingWindow {
		return m.slidingWindow.Allow()
	}

	// 默认允许所有请求
	return true
}

// Start 启动监控器
func (m *EnhancedSystemLoadMonitor) Start() {
	go m.collectMetricsLoop()
	go m.adjustRateLimitLoop()
}

// Stop 停止监控器
func (m *EnhancedSystemLoadMonitor) Stop() {
	m.cancel()
}

// collectMetricsLoop 收集系统指标循环
func (m *EnhancedSystemLoadMonitor) collectMetricsLoop() {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collectSystemMetrics()
		}
	}
}

// adjustRateLimitLoop 调整限流参数循环
func (m *EnhancedSystemLoadMonitor) adjustRateLimitLoop() {
	ticker := time.NewTicker(m.adjustInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.adjustRateLimit()
		}
	}
}

// collectSystemMetrics 重写收集系统指标方法
func (m *EnhancedSystemLoadMonitor) collectSystemMetrics() {
	// 更新CPU使用率
	m.cpuMonitor.UpdateCPUUsage()
	cpuUsage := m.cpuMonitor.GetCPUUsage()

	// 获取内存使用率
	memUsage, _, _ := m.memoryMonitor.GetMemoryStats()

	// 获取Goroutine数量
	goroutineNum := runtime.NumGoroutine()

	// 获取平均响应时间
	avgRT := m.rtTracker.GetAverageResponseTime()

	// 获取错误率
	errorRate := m.errorTracker.GetErrorRate()

	m.mu.Lock()
	m.metrics = &SystemMetrics{
		CPUUsage:     cpuUsage,
		MemoryUsage:  memUsage,
		GoroutineNum: goroutineNum,
		ResponseTime: avgRT,
		ErrorRate:    errorRate,
		Timestamp:    time.Now(),
	}
	m.mu.Unlock()

	fmt.Printf("增强指标 - CPU: %.1f%%, 内存: %.1f%%, Goroutine: %d, 响应时间: %.1fms, 错误率: %.1f%%\n",
		cpuUsage, memUsage, goroutineNum, avgRT, errorRate)
}

// adjustRateLimit 调整限流参数
func (m *EnhancedSystemLoadMonitor) adjustRateLimit() {
	m.mu.RLock()
	metrics := *m.metrics
	m.mu.RUnlock()

	// 计算负载权重
	loadWeight := m.calculateLoadWeight(metrics)

	// 根据负载权重调整限流参数
	newLimit := m.calculateNewLimit(loadWeight)

	m.mu.Lock()
	oldLimit := m.rateLimitConf.CurrentLimit
	m.rateLimitConf.CurrentLimit = newLimit
	m.rateLimitConf.AdjustFactor = loadWeight
	m.mu.Unlock()

	// 更新令牌桶和滑动窗口的限制
	if m.enableTokenBucket {
		m.tokenBucket.UpdateRate(newLimit)
	}

	if m.enableSlidingWindow {
		m.slidingWindow.UpdateLimit(newLimit)
	}

	if oldLimit != newLimit {
		fmt.Printf("限流调整 - 负载权重: %.2f, 限流: %d -> %d\n",
			loadWeight, oldLimit, newLimit)
	}
}

// RecordRequest 记录请求（用于统计）
func (m *EnhancedSystemLoadMonitor) RecordRequest(responseTime time.Duration, isError bool) {
	atomic.AddInt64(&m.requestCount, 1)
	atomic.AddInt64(&m.totalRT, responseTime.Nanoseconds()/int64(time.Millisecond))
	atomic.AddInt64(&m.rtSamples, 1)

	if isError {
		atomic.AddInt64(&m.errorCount, 1)
	}
}

// RecordEnhancedRequest 增强的请求记录
func (m *EnhancedSystemLoadMonitor) RecordEnhancedRequest(responseTime time.Duration, isError bool) {
	// 调用基础记录方法
	m.RecordRequest(responseTime, isError)

	// 记录到增强跟踪器
	rtMs := float64(responseTime.Nanoseconds()) / float64(time.Millisecond)
	m.rtTracker.AddSample(rtMs)
	m.errorTracker.RecordRequest(isError)
}

// GetCurrentLimit 获取当前限流值
func (m *EnhancedSystemLoadMonitor) GetCurrentLimit() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rateLimitConf.CurrentLimit
}

// GetSystemMetrics 获取系统指标
func (m *EnhancedSystemLoadMonitor) GetSystemMetrics() SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.metrics
}

// calculateLoadWeight 计算负载权重
func (m *EnhancedSystemLoadMonitor) calculateLoadWeight(metrics SystemMetrics) float64 {
	var weights []float64

	// CPU负载权重 (权重: 0.3)
	cpuWeight := m.calculateMetricWeight(metrics.CPUUsage,
		m.threshold.CPUHigh, m.threshold.CPUCritical) * 0.3

	// 内存负载权重 (权重: 0.2)
	memWeight := m.calculateMetricWeight(metrics.MemoryUsage,
		m.threshold.MemHigh, m.threshold.MemCritical) * 0.2

	// 响应时间权重 (权重: 0.3)
	rtWeight := m.calculateMetricWeight(metrics.ResponseTime,
		m.threshold.RTHigh, m.threshold.RTCritical) * 0.3

	// 错误率权重 (权重: 0.2)
	errorWeight := m.calculateMetricWeight(metrics.ErrorRate,
		m.threshold.ErrorHigh, m.threshold.ErrorCritical) * 0.2

	weights = append(weights, cpuWeight, memWeight, rtWeight, errorWeight)

	// 计算加权平均
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	// 限制权重范围在 0.1 - 2.0 之间
	if totalWeight < 0.1 {
		totalWeight = 0.1
	} else if totalWeight > 2.0 {
		totalWeight = 2.0
	}

	return totalWeight
}

// calculateMetricWeight 计算单个指标的权重
func (m *EnhancedSystemLoadMonitor) calculateMetricWeight(value, high, critical float64) float64 {
	if value <= high {
		// 正常范围，权重为1.0
		return 1.0
	} else if value <= critical {
		// 高负载范围，线性递减权重 1.0 -> 0.5
		ratio := (value - high) / (critical - high)
		return 1.0 - ratio*0.5
	} else {
		// 临界范围，权重为0.1-0.5
		excessRatio := math.Min((value-critical)/critical, 1.0)
		return 0.5 - excessRatio*0.4 // 0.5 -> 0.1
	}
}

// calculateNewLimit 计算新的限流值
func (m *EnhancedSystemLoadMonitor) calculateNewLimit(loadWeight float64) int64 {
	baseLimit := float64(m.rateLimitConf.BaseLimit)
	newLimit := int64(baseLimit * loadWeight)

	// 确保在最小和最大限制范围内
	if newLimit < m.rateLimitConf.MinLimit {
		newLimit = m.rateLimitConf.MinLimit
	} else if newLimit > m.rateLimitConf.MaxLimit {
		newLimit = m.rateLimitConf.MaxLimit
	}

	return newLimit
}

// GetDetailedMetrics 获取详细指标
func (m *EnhancedSystemLoadMonitor) GetDetailedMetrics() map[string]interface{} {
	metrics := m.GetSystemMetrics()
	tokens, capacity := m.tokenBucket.GetStatus()

	return map[string]interface{}{
		"basic_metrics": metrics,
		"token_bucket": map[string]interface{}{
			"tokens":   tokens,
			"capacity": capacity,
		},
		"response_time_percentiles": map[string]interface{}{
			"p50": m.rtTracker.GetPercentile(50),
			"p90": m.rtTracker.GetPercentile(90),
			"p95": m.rtTracker.GetPercentile(95),
			"p99": m.rtTracker.GetPercentile(99),
		},
		"current_limit": m.GetCurrentLimit(),
	}
}
