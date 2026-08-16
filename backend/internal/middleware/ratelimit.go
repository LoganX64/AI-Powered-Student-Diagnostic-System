package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

const (
	ipLimit  = 10
	ipBurst  = 10
	ipWindow = time.Minute
)

type ipLimiter struct {
	limiters *sync.Map
}

func newIPLimiter() *ipLimiter {
	return &ipLimiter{limiters: &sync.Map{}}
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	val, ok := l.limiters.Load(ip)
	if !ok {
		limiter := rate.NewLimiter(rate.Limit(ipLimit)/rate.Limit(ipWindow/time.Second), ipBurst)
		loaded, _ := l.limiters.LoadOrStore(ip, limiter)
		return loaded.(*rate.Limiter)
	}
	return val.(*rate.Limiter)
}

// RateLimit throttles each client IP to ipLimit requests per minute (in-memory,
// per-instance). Retained for auth/login routes.
func RateLimit() gin.HandlerFunc {
	return NewRateLimiter(nil)
}

// NewRateLimiter returns a rate-limit middleware. When rdb is non-nil, limits
// are shared across all API instances via Redis (fixed window keyed by IP+route)
// so a multi-instance deployment enforces a single global budget.
func NewRateLimiter(rdb *redis.Client) gin.HandlerFunc {
	if rdb != nil {
		return redisRateLimit(rdb)
	}
	return inMemoryRateLimit()
}

func inMemoryRateLimit() gin.HandlerFunc {
	limiter := newIPLimiter()
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

func redisRateLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "rl:" + ip + ":" + c.FullPath()
		ctx := context.Background()
		n, err := rdb.Incr(ctx, key).Result()
		if err == nil {
			if n == 1 {
				_ = rdb.Expire(ctx, key, ipWindow).Err()
			}
			if n > int64(ipLimit) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":   "too many requests",
					"message": "rate limit exceeded, please try again later",
				})
				return
			}
		}
		c.Next()
	}
}
