package main

import "time"

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
