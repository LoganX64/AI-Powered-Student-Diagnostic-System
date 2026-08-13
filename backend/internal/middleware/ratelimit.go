package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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

// RateLimit throttles each client IP to ipLimit requests per minute.
func RateLimit() gin.HandlerFunc {
	limiter := newIPLimiter()
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.get(ip).Allow() {
			c.AbortWithStatusJSON(429, gin.H{
				"error":   "too many requests",
				"message": "rate limit exceeded, please try again later",
			})
			return
		}
		c.Next()
	}
}
