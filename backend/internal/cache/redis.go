package cache

import (
	"context"
	"log"

	"ai-student-diagnostic/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedis builds a Redis client from config. Returns nil when Redis is not
// configured, so callers can fall back to in-process behavior gracefully.
func NewRedis(cfg *config.Config) *redis.Client {
	if cfg == nil || cfg.RedisURL == "" {
		return nil
	}
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("[CACHE] invalid REDIS_URL %q: %v", cfg.RedisURL, err)
		return nil
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Printf("[CACHE] REDIS_URL %q reachable check failed: %v", cfg.RedisURL, err)
	}
	return client
}
