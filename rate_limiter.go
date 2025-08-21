package main

import (
	"sync"
	"time"
)

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	capacity    int64     // 桶容量
	tokens      int64     // 当前令牌数
	refillRate  int64     // 每秒补充令牌数
	lastRefill  time.Time // 上次补充时间
	mu          sync.Mutex
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求通过
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	
	// 计算需要补充的令牌数
	tokensToAdd := int64(elapsed.Seconds()) * tb.refillRate
	if tokensToAdd > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	// 检查是否有可用令牌
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	
	return false
}

// UpdateRate 更新补充速率
func (tb *TokenBucket) UpdateRate(newRate int64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refillRate = newRate
}

// GetStatus 获取当前状态
func (tb *TokenBucket) GetStatus() (int64, int64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens, tb.capacity
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// SlidingWindowRateLimiter 滑动窗口限流器
type SlidingWindowRateLimiter struct {
	limit      int64
	window     time.Duration
	requests   []time.Time
	mu         sync.Mutex
}

// NewSlidingWindowRateLimiter 创建滑动窗口限流器
func NewSlidingWindowRateLimiter(limit int64, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		limit:    limit,
		window:   window,
		requests: make([]time.Time, 0),
	}
}

// Allow 检查是否允许请求
func (sw *SlidingWindowRateLimiter) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	// 清理过期请求
	validRequests := make([]time.Time, 0)
	for _, req := range sw.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	sw.requests = validRequests

	// 检查是否超过限制
	if int64(len(sw.requests)) >= sw.limit {
		return false
	}

	// 记录当前请求
	sw.requests = append(sw.requests, now)
	return true
}

// UpdateLimit 更新限制
func (sw *SlidingWindowRateLimiter) UpdateLimit(newLimit int64) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.limit = newLimit
}
