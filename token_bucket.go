package goshield

import (
	"sync"
	"time"
)

// TokenBucket 定义令牌桶结构
type TokenBucket struct {
	rate       int        // 每秒生成的令牌数
	capacity   int        // 桶的容量
	tokens     int        // 当前令牌数
	lastFilled time.Time  // 上次填充令牌的时间
	mu         sync.Mutex // 互斥锁保护令牌桶的并发访问
}

// NewTokenBucket 创建一个新的令牌桶
func NewTokenBucket(rate, capacity int) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastFilled: time.Now(),
	}
}

func DefaultTokenBucket() *TokenBucket {
	return NewTokenBucket(5, 10)
}

// AllowRequest 尝试从令牌桶中获取一个令牌，如果成功则返回 true，否则返回 false
func (tb *TokenBucket) AllowRequest() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 计算需要添加多少令牌
	now := time.Now()
	elapsed := now.Sub(tb.lastFilled).Seconds()
	tb.tokens += int(elapsed * float64(tb.rate))
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastFilled = now

	// 检查是否有足够的令牌处理请求
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}
