# Rate Limiter Go Package

A production-ready rate limiting library for Go, designed for distributed systems and high-throughput applications.

## Features

- **Multiple Algorithms**:
  - **Token Bucket**: Efficient in-memory implementation allowing for traffic bursts.
  - **Sliding Window**: Smoother rate limiting implementation using weighted counters.
- **Distributed Support**: Fully atomic Redis-backed rate limiting using Lua scripts.
- **HTTP Middleware**: Flexible middleware compatible with standard `net/http` and easily adaptable to other frameworks.
- **Dynamic Configuration**: Configure limits per-request (e.g., based on User Tier, IP, or Endpoint).

## Installation

```bash
go get github.com/alibaba/rate-limiter-go
```

## Usage

### 1. In-Memory Token Bucket

```go
import (
    "context"
    "fmt"
    "time"
    "github.com/alibaba/rate-limiter-go/limiter"
)

// Define rule: 100 requests per minute, burst of 20
limit := limiter.Limit{
    Rate:   100,
    Period: time.Minute,
    Burst:  20,
}

tb := limiter.NewTokenBucket()

// Check if request is allowed
res, _ := tb.Allow(context.Background(), "user-123", limit)

if res.Allowed {
    fmt.Println("Request allowed", res.Remaining, "tokens remaining")
} else {
    fmt.Printf("Rate limit exceeded. Retry after %v\n", res.ResetAfter)
}
```

### 2. Distributed Redis Limiter

Use `RedisTokenBucket` for distributed applications. It uses Lua scripts to ensure atomicity across multiple instances.

```go
import (
    "github.com/alibaba/rate-limiter-go/limiter"
    "github.com/redis/go-redis/v9"
)

// Initialize Redis
rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Create Strategy
redisLimiter := limiter.NewRedisTokenBucket(rdb)

// Use exactly like local limiter
res, err := redisLimiter.Allow(ctx, "api-key-xyz", limit)
```

### 3. HTTP Middleware

The package provides a `middleware` subpackage for easy integration.

```go
import "github.com/alibaba/rate-limiter-go/middleware"

// Configure Middleware
cfg := middleware.Config{
    Limiter: limiter.NewTokenBucket(), // Swap with RedisLimiter for distributed

    // Define how to identify the client (IP, API Key, User ID)
    KeyFunc: func(r *http.Request) string {
        return r.RemoteAddr
    },

    // Define limits dynamically
    LimitFunc: func(r *http.Request) limiter.Limit {
        // Logic to determine limit based on user tier, etc.
        return limiter.Limit{Rate: 10, Period: time.Second, Burst: 5}
    },
}

// Apply to your handler
handler := middleware.New(cfg)(myHandler)
http.ListenAndServe(":8080", handler)
```

## Testing

Run tests (requires Redis for integration tests):

```bash
go test ./...
```

## License

MIT
