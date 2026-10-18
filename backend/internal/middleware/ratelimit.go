package middleware

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

const (
	// DefaultLimit is the per-IP budget for normal routes (requests/min).
	DefaultLimit = 10
	// LoginLimit is a stricter per-IP budget for auth/login routes.
	LoginLimit = 5
	ipWindow   = time.Minute
)

type ipLimiter struct {
	limit    int
	limiters *sync.Map
}

func newIPLimiter(limit int) *ipLimiter {
	return &ipLimiter{limit: limit, limiters: &sync.Map{}}
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	val, ok := l.limiters.Load(ip)
	if !ok {
		limiter := rate.NewLimiter(rate.Limit(l.limit)/rate.Limit(ipWindow/time.Second), l.limit)
		loaded, _ := l.limiters.LoadOrStore(ip, limiter)
		return loaded.(*rate.Limiter)
	}
	return val.(*rate.Limiter)
}

// NewRateLimiter returns a rate-limit middleware. When rdb is non-nil, limits
// are shared across all API instances via Redis (fixed window keyed by IP+route)
// so a multi-instance deployment enforces a single global budget. The limit
// (requests per minute per IP) is configurable per route.
func NewRateLimiter(rdb *redis.Client, limit int) gin.HandlerFunc {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if rdb != nil {
		return redisRateLimit(rdb, limit)
	}
	return inMemoryRateLimit(limit)
}

func inMemoryRateLimit(limit int) gin.HandlerFunc {
	limiter := newIPLimiter(limit)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.get(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "too many requests",
				"message": "rate limit exceeded, please try again later",
			})
			return
		}
		c.Next()
	}
}

// rateLimitScript atomically increments the fixed-window counter for a key,
// sets the TTL on first use, and returns the new count.
//
// If the new count exceeds the limit the script decrements the counter before
// returning, so rejected requests do not consume a budget slot. All three
// operations (INCR, EXPIRE, conditional DECR) execute as a single atomic unit
// on the Redis server, eliminating the TOCTOU gap in the previous GET→INCR
// design where concurrent requests from multiple instances could all read the
// same sub-limit count and all be admitted.
//
// KEYS[1] = rate-limit key  (e.g. "rl:1.2.3.4:/auth/login")
// ARGV[1] = limit           (integer, max allowed requests per window)
// ARGV[2] = window seconds  (integer, TTL of the key)
//
// Returns the post-increment count, or 0 when the request is denied.
var rateLimitScript = redis.NewScript(`
local n = redis.call("INCR", KEYS[1])
if n == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[2])
end
if n > tonumber(ARGV[1]) then
  redis.call("DECR", KEYS[1])
  return 0
end
return n
`)

// redisRateLimit enforces a fixed-window per-IP+route limit shared across
// instances. The counter is incremented atomically via a Lua script, so
// concurrent requests from multiple instances cannot race past the limit.
// Rejected requests do not consume a budget slot (the Lua script decrements
// before returning 0).
func redisRateLimit(rdb *redis.Client, limit int) gin.HandlerFunc {
	fallback := newIPLimiter(limit)
	tooMany := func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error":   "too many requests",
			"message": "rate limit exceeded, please try again later",
		})
	}
	windowSecs := int(ipWindow.Seconds())
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "rl:" + ip + ":" + c.FullPath()
		ctx := context.Background()

		n, err := rateLimitScript.Run(ctx, rdb, []string{key}, limit, windowSecs).Int64()
		if err != nil {
			// Redis unavailable: don't bypass the control. Fall back to the
			// per-instance limiter so requests are still throttled.
			log.Printf("[RATELIMIT] lua script failed (ip=%s key=%s): %v", ip, key, err)
			if !fallback.get(ip).Allow() {
				tooMany(c)
				return
			}
			c.Next()
			return
		}

		// The script returns 0 when the new count exceeds the limit.
		if n == 0 {
			tooMany(c)
			return
		}

		c.Next()
	}
}
