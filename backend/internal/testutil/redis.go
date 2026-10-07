package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// RequireRedis resolves a Redis connection for tests, skipping when none is
// reachable. REQUIRE_REDIS=1 turns the skip into a hard failure so a run can
// never report green while silently skipping every Redis-backed test — the
// same guarantee RequireDB gives for Postgres.
func RequireRedis(t *testing.T) string {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://127.0.0.1:6379"
	}
	if !redisReachable(url) {
		if os.Getenv("REQUIRE_REDIS") == "1" {
			t.Fatalf("Redis not reachable at %s and REQUIRE_REDIS=1: Redis-backed tests must run", url)
		}
		t.Skipf("Redis not reachable at %s (start it with: docker compose up -d redis, or set REQUIRE_REDIS=1 to fail loudly)", url)
	}
	return url
}

// redisReachable dials with a short timeout so an absent Redis fails fast
// instead of adding connection-refused latency to every test.
func redisReachable(url string) bool {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return false
	}
	opts.DialTimeout = 2 * time.Second
	opts.ReadTimeout = 2 * time.Second
	opts.WriteTimeout = 2 * time.Second
	c := redis.NewClient(opts)
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return c.Ping(ctx).Err() == nil
}

// OpenRedisClient returns a live client for a test, skipping when Redis is not
// reachable. The client is closed on cleanup.
func OpenRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	url := RequireRedis(t)
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse REDIS_URL %q: %v", url, err)
	}
	c := redis.NewClient(opts)
	t.Cleanup(func() { c.Close() })
	return c
}

// FlushTestKeys removes the given keys so Redis-backed tests do not leak state
// into each other.
func FlushTestKeys(t *testing.T, c *redis.Client, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if err := c.Del(t.Context(), k).Err(); err != nil && err != redis.Nil {
			t.Fatalf("flush key %s: %v", k, err)
		}
	}
}
