package testutil

import (
	"testing"
)

// TestOpenRedisClientRoundTrip proves the helper hands back a working client, so
// every Redis-backed test built on it is not silently vacuous. It also exercises
// the Redis gate: RequireRedis skips when Redis is unreachable, so this test
// either runs against a live Redis or skips with the reason.
func TestOpenRedisClientRoundTrip(t *testing.T) {
	c := OpenRedisClient(t)

	key := "testutil:probe"
	FlushTestKeys(t, c, key)

	if err := c.Set(t.Context(), key, "v1", 0).Err(); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Get(t.Context(), key).Result()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "v1" {
		t.Fatalf("got %q, want v1", got)
	}
	FlushTestKeys(t, c, key)
	if _, err := c.Get(t.Context(), key).Result(); err == nil {
		t.Fatal("key survived flush")
	}
}
