package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisTokenBucket implements the Strategy interface using a Redis-backed token bucket.
type RedisTokenBucket struct {
	client *redis.Client
}

// NewRedisTokenBucket creates a new instance of RedisTokenBucket.
func NewRedisTokenBucket(client *redis.Client) *RedisTokenBucket {
	return &RedisTokenBucket{
		client: client,
	}
}

// Lua script for token bucket
// Keys: [1] bucket_key
// Args: [1] rate (tokens/sec), [2] capacity, [3] now (unixtime float), [4] requested (tokens)
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local rate = tonumber(ARGS[1])
local capacity = tonumber(ARGS[2])
local now = tonumber(ARGS[3])
local requested = tonumber(ARGS[4])

local last_tokens = tonumber(redis.call("HGET", key, "tokens"))
local last_updated = tonumber(redis.call("HGET", key, "last_updated"))

if last_tokens == nil then
    last_tokens = capacity
    last_updated = now
end

local delta = math.max(0, now - last_updated)
local filled_tokens = math.min(capacity, last_tokens + (delta * rate))

local allowed = 0
local remaining = filled_tokens
local reset_after = 0

if filled_tokens >= requested then
    allowed = 1
    remaining = filled_tokens - requested
    redis.call("HSET", key, "tokens", remaining, "last_updated", now)
    redis.call("PEXPIRE", key, 60000) -- Expire idle keys (1 min)
else
    allowed = 0
    remaining = filled_tokens
    reset_after = (requested - filled_tokens) / rate
end

return {allowed, remaining, reset_after}
`)

func (r *RedisTokenBucket) Allow(ctx context.Context, key string, limit Limit) (*Result, error) {
	// Rate is requests per period.
	ratePerSec := float64(limit.Rate) / limit.Period.Seconds()
	
	// Use microsecond precision for smoother updates
	now := float64(time.Now().UnixMicro()) / 1e6

	keys := []string{key}
	args := []interface{}{ratePerSec, limit.Burst, now, 1}

	// Helper to cast interface{} to float64 safely
	toFloat := func(v interface{}) float64 {
		switch t := v.(type) {
		case float64:
			return t
		case int64:
			return float64(t)
		default:
			return 0
		}
	}

	res, err := tokenBucketScript.Run(ctx, r.client, keys, args...).Result()
	if err != nil {
		return nil, err
	}

	vals := res.([]interface{})
	allowedVal := vals[0].(int64)
	remainingVal := toFloat(vals[1]) // Lua might return int or float
	resetAfterVal := toFloat(vals[2])

	result := &Result{
		Allowed:   allowedVal == 1,
		Remaining: int(remainingVal),
	}
	
	if resetAfterVal > 0 {
		result.ResetAfter = time.Duration(resetAfterVal * float64(time.Second))
	}

	return result, nil
}
