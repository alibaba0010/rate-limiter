package limiter

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements the Strategy interface using the token bucket algorithm.
type TokenBucket struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens     float64
	lastUpdate time.Time
}

// NewTokenBucket creates a new instance of TokenBucket strategy.
// Note: In a real production system, you would want a mechanism to clean up old keys.
func NewTokenBucket() *TokenBucket {
	return &TokenBucket{
		buckets: make(map[string]*bucket),
	}
}

// Allow checks if the request is allowed based on the token bucket algorithm.
func (tb *TokenBucket) Allow(ctx context.Context, key string, limit Limit) (*Result, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	b, exists := tb.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     float64(limit.Burst),
			lastUpdate: now,
		}
		tb.buckets[key] = b
	}

	// Calculate tokens to add
	// Rate is requests per Period.
	tokensPerSec := float64(limit.Rate) / limit.Period.Seconds()
	elapsed := now.Sub(b.lastUpdate).Seconds()

	b.tokens += elapsed * tokensPerSec
	if b.tokens > float64(limit.Burst) {
		b.tokens = float64(limit.Burst)
	}
	b.lastUpdate = now

	result := &Result{}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		result.Allowed = true
		result.Remaining = int(b.tokens)
		result.ResetAfter = 0
	} else {
		result.Allowed = false
		result.Remaining = 0
		// Time to wait for enough tokens for 1 request
		waitSec := (1.0 - b.tokens) / tokensPerSec
		result.ResetAfter = time.Duration(waitSec * float64(time.Second))
	}

	return result, nil
}
