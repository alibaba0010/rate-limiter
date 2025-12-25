package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/alibaba/rate-limiter-go/limiter"
	"github.com/alibaba/rate-limiter-go/middleware"
	// "github.com/redis/go-redis/v9"
)

func main() {
	// 1. In-memory Token Bucket Strategy
	tb := limiter.NewTokenBucket()

	// 2. Configure Middleware
	cfg := middleware.Config{
		Limiter: tb,
		KeyFunc: func(r *http.Request) string {
			// Identify users by IP (or API key header)
			return r.RemoteAddr
		},
		LimitFunc: func(r *http.Request) limiter.Limit {
			// Dynamic limits:
			// e.g. Free tier: 5 req/min, Paid: 100 req/min
			return limiter.Limit{
				Rate:   5,
				Period: 10 * time.Second, // 5 requests every 10 seconds
				Burst:  10,
			}
		},
	}

	mw := middleware.New(cfg)

	// 3. Setup Server
	mux := http.NewServeMux()
	mux.Handle("/", mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Request Allowed!\n"))
	})))

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Server failed:", err)
	}
}
