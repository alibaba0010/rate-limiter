package limiter

import (
	"context"
	"time"
)

// Result represents the result of a rate limit check
type Result struct {
	Allowed    bool
	Remaining  int
	ResetAfter time.Duration
}

// Strategy defines the interface for different rate limiting algorithms
type Strategy interface {
	// Allow checks if the request is allowed
	Allow(ctx context.Context, key string, limit Limit) (*Result, error)
}

// Limit defines the rate limiting rules
type Limit struct {
	Rate   int           // How many requests
	Period time.Duration // Time window (e.g., Per Second, Per Minute)
	Burst  int           // Maximum burst size (e.g. for Token Bucket)
}
