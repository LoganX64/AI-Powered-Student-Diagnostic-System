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
	ipWindow  = time.Minute
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

// redisRateLimit enforces a fixed-window per-IP+route limit shared across
// instances. It checks the current count first and only increments on allowed
// requests, so rejected requests do not consume a counter slot.
func redisRateLimit(rdb *redis.Client, limit int) gin.HandlerFunc {
	fallback := newIPLimiter(limit)
	tooMany := func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error":   "too many requests",
			"message": "rate limit exceeded, please try again later",
		})
	}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "rl:" + ip + ":" + c.FullPath()
		ctx := context.Background()
		cur, err := rdb.Get(ctx, key).Int64()
		if err != nil && err != redis.Nil {
			// Redis unavailable: don't bypass the control. Fall back to the
			// per-instance limiter so requests are still throttled.
			log.Printf("[RATELIMIT] redis GET failed (ip=%s key=%s): %v", ip, key, err)
			if !fallback.get(ip).Allow() {
				tooMany(c)
				return
			}
			c.Next()
			return
		}
		if err == nil && cur >= int64(limit) {
			tooMany(c)
			return
		}
		n, err := rdb.Incr(ctx, key).Result()
		if err != nil && err != redis.Nil {
			// Cannot record the request; deny rather than allow unlimited.
			log.Printf("[RATELIMIT] redis INCR failed (ip=%s key=%s): %v", ip, key, err)
			tooMany(c)
			return
		}
		if err == nil && n == 1 {
			if e := rdb.Expire(ctx, key, ipWindow).Err(); e != nil {
				log.Printf("[RATELIMIT] redis EXPIRE failed (key=%s): %v", key, e)
			}
		}
		c.Next()
	}
}
