package cache

import (
	"testing"

	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/internal/testutil"
)

// NewRedis returns nil when Redis is unconfigured, and every caller
// (hub, autosave buffer, rate limiter) branches on that nil. These cases pin the
// two nil-returning paths so a future refactor cannot start handing out an
// unusable client.

func TestNewRedisNilConfig(t *testing.T) {
	if c := NewRedis(nil); c != nil {
		t.Fatal("NewRedis(nil) must return nil")
	}
}

func TestNewRedisEmptyURL(t *testing.T) {
	if c := NewRedis(&config.Config{}); c != nil {
		t.Fatal("NewRedis with empty RedisURL must return nil")
	}
}

func TestNewRedisInvalidURL(t *testing.T) {
	// An unparseable URL must degrade to nil rather than panicking.
	if c := NewRedis(&config.Config{RedisURL: "://not a url"}); c != nil {
		t.Fatal("NewRedis with unparseable RedisURL must return nil")
	}
}

func TestNewRedisLiveClient(t *testing.T) {
	url := testutil.RequireRedis(t)
	c := NewRedis(&config.Config{RedisURL: url})
	if c == nil {
		t.Fatal("NewRedis must return a client when Redis is configured and reachable")
	}
	defer c.Close()

	// NewRedis logs a ping failure but still returns the client; against a live
	// Redis the ping must actually succeed.
	if err := c.Ping(t.Context()).Err(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
