package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	metrics map[string]interface{}
	mu      sync.RWMutex
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]interface{}),
	}
}

// SetMetric 设置指标
func (mc *MetricsCollector) SetMetric(name string, value interface{}) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics[name] = value
}

// GetMetric 获取指标
func (mc *MetricsCollector) GetMetric(name string) (interface{}, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	value, exists := mc.metrics[name]
	return value, exists
}

// GetAllMetrics 获取所有指标
func (mc *MetricsCollector) GetAllMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make(map[string]interface{})
	for k, v := range mc.metrics {
		result[k] = v
	}
	return result
}

// UpdateMetrics 批量更新指标
func (mc *MetricsCollector) UpdateMetrics(metrics map[string]interface{}) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	for k, v := range metrics {
		mc.metrics[k] = v
	}
}

// PrometheusExporter Prometheus指标导出器
type PrometheusExporter struct {
	collector *MetricsCollector
}

// NewPrometheusExporter 创建Prometheus导出器
func NewPrometheusExporter(collector *MetricsCollector) *PrometheusExporter {
	return &PrometheusExporter{
		collector: collector,
	}
}

// ExportMetrics 导出Prometheus格式指标
func (pe *PrometheusExporter) ExportMetrics() string {
	metrics := pe.collector.GetAllMetrics()
	var result string
	
	// 系统指标
	if systemMetrics, ok := metrics["system_metrics"].(SystemMetrics); ok {
		result += fmt.Sprintf("# HELP cpu_usage_percent CPU usage percentage\n")
		result += fmt.Sprintf("# TYPE cpu_usage_percent gauge\n")
		result += fmt.Sprintf("cpu_usage_percent %.2f\n", systemMetrics.CPUUsage)
		
		result += fmt.Sprintf("# HELP memory_usage_percent Memory usage percentage\n")
		result += fmt.Sprintf("# TYPE memory_usage_percent gauge\n")
		result += fmt.Sprintf("memory_usage_percent %.2f\n", systemMetrics.MemoryUsage)
		
		result += fmt.Sprintf("# HELP goroutine_count Number of goroutines\n")
		result += fmt.Sprintf("# TYPE goroutine_count gauge\n")
		result += fmt.Sprintf("goroutine_count %d\n", systemMetrics.GoroutineNum)
		
		result += fmt.Sprintf("# HELP response_time_ms Average response time in milliseconds\n")
		result += fmt.Sprintf("# TYPE response_time_ms gauge\n")
		result += fmt.Sprintf("response_time_ms %.2f\n", systemMetrics.ResponseTime)
		
		result += fmt.Sprintf("# HELP error_rate_percent Error rate percentage\n")
		result += fmt.Sprintf("# TYPE error_rate_percent gauge\n")
		result += fmt.Sprintf("error_rate_percent %.2f\n", systemMetrics.ErrorRate)
	}
	
	// 限流指标
	if currentLimit, ok := metrics["current_limit"].(int64); ok {
		result += fmt.Sprintf("# HELP rate_limit_current Current rate limit\n")
		result += fmt.Sprintf("# TYPE rate_limit_current gauge\n")
		result += fmt.Sprintf("rate_limit_current %d\n", currentLimit)
	}
	
	// 令牌桶指标
	if tokenBucket, ok := metrics["token_bucket"].(map[string]interface{}); ok {
		if tokens, ok := tokenBucket["tokens"].(int64); ok {
			result += fmt.Sprintf("# HELP token_bucket_tokens Current tokens in bucket\n")
			result += fmt.Sprintf("# TYPE token_bucket_tokens gauge\n")
			result += fmt.Sprintf("token_bucket_tokens %d\n", tokens)
		}
		
		if capacity, ok := tokenBucket["capacity"].(int64); ok {
			result += fmt.Sprintf("# HELP token_bucket_capacity Token bucket capacity\n")
			result += fmt.Sprintf("# TYPE token_bucket_capacity gauge\n")
			result += fmt.Sprintf("token_bucket_capacity %d\n", capacity)
		}
	}
	
	return result
}

// HTTPMetricsServer HTTP指标服务器
type HTTPMetricsServer struct {
	collector *MetricsCollector
	exporter  *PrometheusExporter
	server    *http.Server
}

// NewHTTPMetricsServer 创建HTTP指标服务器
func NewHTTPMetricsServer(port int, collector *MetricsCollector) *HTTPMetricsServer {
	exporter := NewPrometheusExporter(collector)
	
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	
	metricsServer := &HTTPMetricsServer{
		collector: collector,
		exporter:  exporter,
		server:    server,
	}
	
	// 注册路由
	mux.HandleFunc("/metrics", metricsServer.handleMetrics)
	mux.HandleFunc("/health", metricsServer.handleHealth)
	mux.HandleFunc("/status", metricsServer.handleStatus)
	
	return metricsServer
}

// Start 启动指标服务器
func (hms *HTTPMetricsServer) Start() error {
	return hms.server.ListenAndServe()
}

// Stop 停止指标服务器
func (hms *HTTPMetricsServer) Stop() error {
	return hms.server.Close()
}

// handleMetrics 处理指标请求
func (hms *HTTPMetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	
	switch r.URL.Query().Get("format") {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		metrics := hms.collector.GetAllMetrics()
		json.NewEncoder(w).Encode(metrics)
	default:
		// Prometheus格式
		fmt.Fprint(w, hms.exporter.ExportMetrics())
	}
}

// handleHealth 处理健康检查
func (hms *HTTPMetricsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleStatus 处理状态请求
func (hms *HTTPMetricsServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	metrics := hms.collector.GetAllMetrics()
	
	status := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"metrics":   metrics,
	}
	
	json.NewEncoder(w).Encode(status)
}

// PerformanceProfiler 性能分析器
type PerformanceProfiler struct {
	startTime time.Time
	samples   []PerformanceSample
	mu        sync.RWMutex
}

// PerformanceSample 性能样本
type PerformanceSample struct {
	Timestamp    time.Time `json:"timestamp"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  float64   `json:"memory_usage"`
	GoroutineNum int       `json:"goroutine_num"`
	ResponseTime float64   `json:"response_time"`
	ErrorRate    float64   `json:"error_rate"`
	RateLimit    int64     `json:"rate_limit"`
}

// NewPerformanceProfiler 创建性能分析器
func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		startTime: time.Now(),
		samples:   make([]PerformanceSample, 0),
	}
}

// AddSample 添加性能样本
func (pp *PerformanceProfiler) AddSample(sample PerformanceSample) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	
	pp.samples = append(pp.samples, sample)
	
	// 保持最近1000个样本
	if len(pp.samples) > 1000 {
		pp.samples = pp.samples[1:]
	}
}

// GetSamples 获取性能样本
func (pp *PerformanceProfiler) GetSamples() []PerformanceSample {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	
	result := make([]PerformanceSample, len(pp.samples))
	copy(result, pp.samples)
	return result
}

// GetAverageMetrics 获取平均指标
func (pp *PerformanceProfiler) GetAverageMetrics() PerformanceSample {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	
	if len(pp.samples) == 0 {
		return PerformanceSample{}
	}
	
	var avg PerformanceSample
	for _, sample := range pp.samples {
		avg.CPUUsage += sample.CPUUsage
		avg.MemoryUsage += sample.MemoryUsage
		avg.GoroutineNum += sample.GoroutineNum
		avg.ResponseTime += sample.ResponseTime
		avg.ErrorRate += sample.ErrorRate
		avg.RateLimit += sample.RateLimit
	}
	
	count := float64(len(pp.samples))
	avg.CPUUsage /= count
	avg.MemoryUsage /= count
	avg.GoroutineNum = int(float64(avg.GoroutineNum) / count)
	avg.ResponseTime /= count
	avg.ErrorRate /= count
	avg.RateLimit = int64(float64(avg.RateLimit) / count)
	avg.Timestamp = time.Now()
	
	return avg
}

// ExportToJSON 导出为JSON
func (pp *PerformanceProfiler) ExportToJSON() ([]byte, error) {
	samples := pp.GetSamples()
	return json.MarshalIndent(samples, "", "  ")
}
