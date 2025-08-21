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

// SystemMetrics 系统指标
type SystemMetrics struct {
	CPUUsage     float64   `json:"cpu_usage"`     // CPU使用率 (0-100)
	MemoryUsage  float64   `json:"memory_usage"`  // 内存使用率 (0-100)
	GoroutineNum int       `json:"goroutine_num"` // Goroutine数量
	ResponseTime float64   `json:"response_time"` // 平均响应时间(ms)
	ErrorRate    float64   `json:"error_rate"`    // 错误率 (0-100)
	Timestamp    time.Time `json:"timestamp"`     // 时间戳
}

// LoadThreshold 负载阈值配置
type LoadThreshold struct {
	CPUHigh       float64 `json:"cpu_high"`       // CPU高负载阈值
	CPUCritical   float64 `json:"cpu_critical"`   // CPU临界阈值
	MemHigh       float64 `json:"mem_high"`       // 内存高负载阈值
	MemCritical   float64 `json:"mem_critical"`   // 内存临界阈值
	RTHigh        float64 `json:"rt_high"`        // 响应时间高阈值(ms)
	RTCritical    float64 `json:"rt_critical"`    // 响应时间临界阈值(ms)
	ErrorHigh     float64 `json:"error_high"`     // 错误率高阈值
	ErrorCritical float64 `json:"error_critical"` // 错误率临界阈值
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	BaseLimit    int64   `json:"base_limit"`    // 基础限流值
	CurrentLimit int64   `json:"current_limit"` // 当前限流值
	MinLimit     int64   `json:"min_limit"`     // 最小限流值
	MaxLimit     int64   `json:"max_limit"`     // 最大限流值
	AdjustFactor float64 `json:"adjust_factor"` // 调整因子 (0.5-2.0)
}

// SystemLoadMonitor 系统负载监控器
type SystemLoadMonitor struct {
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
}

// NewSystemLoadMonitor 创建系统负载监控器
func NewSystemLoadMonitor() *SystemLoadMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &SystemLoadMonitor{
		metrics: &SystemMetrics{},
		threshold: &LoadThreshold{
			CPUHigh:       70.0,
			CPUCritical:   85.0,
			MemHigh:       80.0,
			MemCritical:   90.0,
			RTHigh:        100.0, // 100ms
			RTCritical:    300.0, // 300ms
			ErrorHigh:     5.0,   // 5%
			ErrorCritical: 10.0,  // 10%
		},
		rateLimitConf: &RateLimitConfig{
			BaseLimit:    1000,
			CurrentLimit: 1000,
			MinLimit:     100,
			MaxLimit:     5000,
			AdjustFactor: 1.0,
		},
		monitorInterval: 5 * time.Second,  // 每5秒收集一次指标
		adjustInterval:  10 * time.Second, // 每10秒调整一次限流参数
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start 启动监控器
func (m *SystemLoadMonitor) Start() {
	go m.collectMetricsLoop()
	go m.adjustRateLimitLoop()
}

// Stop 停止监控器
func (m *SystemLoadMonitor) Stop() {
	m.cancel()
}

// collectMetricsLoop 收集系统指标循环
func (m *SystemLoadMonitor) collectMetricsLoop() {
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

// collectSystemMetrics 收集系统指标
func (m *SystemLoadMonitor) collectSystemMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 计算内存使用率（简化计算）
	memUsage := float64(memStats.Alloc) / float64(memStats.Sys) * 100

	// 获取Goroutine数量
	goroutineNum := runtime.NumGoroutine()

	// 计算平均响应时间
	avgRT := m.calculateAverageRT()

	// 计算错误率
	errorRate := m.calculateErrorRate()

	// CPU使用率需要额外实现，这里模拟
	cpuUsage := m.estimateCPUUsage()

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

	fmt.Printf("系统指标 - CPU: %.1f%%, 内存: %.1f%%, Goroutine: %d, 响应时间: %.1fms, 错误率: %.1f%%\n",
		cpuUsage, memUsage, goroutineNum, avgRT, errorRate)
}

// estimateCPUUsage 估算CPU使用率（简化实现）
func (m *SystemLoadMonitor) estimateCPUUsage() float64 {
	// 这里可以集成第三方库如 gopsutil 来获取真实的CPU使用率
	// 为了演示，这里基于Goroutine数量做简单估算
	goroutines := float64(runtime.NumGoroutine())
	// cpuCount := float64(runtime.NumCPU())

	// 简单估算：每100个goroutine约占用10%的CPU
	estimatedCPU := (goroutines / 100) * 10
	if estimatedCPU > 100 {
		estimatedCPU = 100
	}

	return estimatedCPU
}

// calculateAverageRT 计算平均响应时间
func (m *SystemLoadMonitor) calculateAverageRT() float64 {
	samples := atomic.LoadInt64(&m.rtSamples)
	if samples == 0 {
		return 0
	}

	totalRT := atomic.LoadInt64(&m.totalRT)
	return float64(totalRT) / float64(samples)
}

// calculateErrorRate 计算错误率
func (m *SystemLoadMonitor) calculateErrorRate() float64 {
	totalReq := atomic.LoadInt64(&m.requestCount)
	if totalReq == 0 {
		return 0
	}

	errorCount := atomic.LoadInt64(&m.errorCount)
	return float64(errorCount) / float64(totalReq) * 100
}

// adjustRateLimitLoop 调整限流参数循环
func (m *SystemLoadMonitor) adjustRateLimitLoop() {
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

// adjustRateLimit 根据系统负载调整限流参数
func (m *SystemLoadMonitor) adjustRateLimit() {
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

	if oldLimit != newLimit {
		fmt.Printf("限流调整 - 负载权重: %.2f, 限流: %d -> %d\n",
			loadWeight, oldLimit, newLimit)
	}
}

// calculateLoadWeight 计算负载权重
func (m *SystemLoadMonitor) calculateLoadWeight(metrics SystemMetrics) float64 {
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
func (m *SystemLoadMonitor) calculateMetricWeight(value, high, critical float64) float64 {
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
func (m *SystemLoadMonitor) calculateNewLimit(loadWeight float64) int64 {
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

// RecordRequest 记录请求（用于统计）
func (m *SystemLoadMonitor) RecordRequest(responseTime time.Duration, isError bool) {
	atomic.AddInt64(&m.requestCount, 1)
	atomic.AddInt64(&m.totalRT, responseTime.Nanoseconds()/int64(time.Millisecond))
	atomic.AddInt64(&m.rtSamples, 1)

	if isError {
		atomic.AddInt64(&m.errorCount, 1)
	}
}

// GetCurrentLimit 获取当前限流值
func (m *SystemLoadMonitor) GetCurrentLimit() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rateLimitConf.CurrentLimit
}

// GetSystemMetrics 获取系统指标
func (m *SystemLoadMonitor) GetSystemMetrics() SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.metrics
}

// ResetStats 重置统计数据
func (m *SystemLoadMonitor) ResetStats() {
	atomic.StoreInt64(&m.requestCount, 0)
	atomic.StoreInt64(&m.errorCount, 0)
	atomic.StoreInt64(&m.totalRT, 0)
	atomic.StoreInt64(&m.rtSamples, 0)
}

// 使用示例
func main() {
	monitor := NewSystemLoadMonitor()
	monitor.Start()
	defer monitor.Stop()

	// 模拟请求处理
	go func() {
		for i := 0; i < 1000; i++ {
			start := time.Now()

			// 模拟请求处理时间
			time.Sleep(time.Duration(10+i%100) * time.Millisecond)

			responseTime := time.Since(start)
			isError := i%20 == 0 // 5% 错误率

			monitor.RecordRequest(responseTime, isError)

			currentLimit := monitor.GetCurrentLimit()
			if int64(i)%currentLimit == 0 {
				fmt.Printf("当前限流值: %d\n", currentLimit)
			}

			time.Sleep(50 * time.Millisecond)
		}
	}()

	// 运行30秒
	time.Sleep(30 * time.Second)

	// 打印最终指标
	metrics := monitor.GetSystemMetrics()
	fmt.Printf("\n最终系统指标: %+v\n", metrics)
	fmt.Printf("最终限流值: %d\n", monitor.GetCurrentLimit())
}
