package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alibaba/rate-limiter-go/limiter"
)

// Config defines the configuration for the rate limiter middleware
type Config struct {
	Limiter limiter.Strategy
	// KeyFunc computes the rate limit key from the request.
	// Common examples: IP address, User ID (from context), API Key.
	KeyFunc func(r *http.Request) string
	// LimitFunc returns the limit configuration for the request.
	// This allows dynamic limits per user/endpoint.
	LimitFunc func(r *http.Request) limiter.Limit
	// ErrorHandler handles internal errors from the limiter (e.g. Redis down).
	// Default: Log and continue (Fail Open) or returns 500? Often Fail Open is safer.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
	// RateLimitHandler handles requests allowed/denied logic customization.
	// If nil, default 429 response is used when denied.
	RateLimitHandler func(w http.ResponseWriter, r *http.Request, res *limiter.Result)
}

// New creates a new HTTP middleware handler
func New(cfg Config) func(http.Handler) http.Handler {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(r *http.Request) string {
			return r.RemoteAddr // Simple default, usually you want X-Forwarded-For or similar
		}
	}
	if cfg.LimitFunc == nil {
		// Default strict limit
		cfg.LimitFunc = func(r *http.Request) limiter.Limit {
			return limiter.Limit{
				Rate:   10,
				Period: time.Minute,
				Burst:  10,
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := cfg.KeyFunc(r)
			limit := cfg.LimitFunc(r)

			res, err := cfg.Limiter.Allow(r.Context(), key, limit)
			if err != nil {
				if cfg.ErrorHandler != nil {
					cfg.ErrorHandler(w, r, err)
					return
				}
				// Default behavior for error: Log and Fail Open (Allow) to avoid blocking traffic on DB failure,
				// OR Fail Closed. Here we'll return 500 for safety of awareness.
				http.Error(w, "Rate Limit Internal Error", http.StatusInternalServerError)
				return
			}

			// Set generic headers if desired (X-RateLimit-Limit, etc)
			// (Optional: standard headers)

			if !res.Allowed {
				if cfg.RateLimitHandler != nil {
					cfg.RateLimitHandler(w, r, res)
					return
				}

				w.Header().Set("Retry-After", strconv.Itoa(int(res.ResetAfter.Seconds())))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
