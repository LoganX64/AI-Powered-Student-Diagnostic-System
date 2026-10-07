package config

import (
	"strings"
	"testing"
	"time"
)

func TestIntEnv(t *testing.T) {
	t.Setenv("EXISTING_INT", "42")
	t.Setenv("BAD_INT", "not-a-number")
	if got := intEnv("MISSING_INT", 7); got != 7 {
		t.Fatalf("missing: got %d", got)
	}
	if got := intEnv("EXISTING_INT", 7); got != 42 {
		t.Fatalf("existing: got %d", got)
	}
	if got := intEnv("BAD_INT", 7); got != 7 {
		t.Fatalf("invalid: got %d", got)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@h/db")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("JWT_EXPIRY", "4h")
	t.Setenv("JWT_ISSUER", "eduquant")
	t.Setenv("PORT", "8080")
	t.Setenv("VIDEO_TOKEN_SECRET", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("QUEUE_MODE", "")
	t.Setenv("REDIS_ENABLED", "")
	t.Setenv("UPLOAD_DIR", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_APP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("FRONTEND_URL", "")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("TRUSTED_PROXIES", "")
	t.Setenv("SCALE_BAND_C", "")
	t.Setenv("COMPUTE_CHUNK_SIZE", "")
	t.Setenv("SUBMIT_GRACE_SECONDS", "")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME", "")

	c := LoadConfig()
	if c.DBMaxOpenConns != 25 || c.DBMaxIdleConns != 25 || c.DBConnMaxLifetime != 5*time.Minute {
		t.Fatalf("pool defaults = %d/%d/%v", c.DBMaxOpenConns, c.DBMaxIdleConns, c.DBConnMaxLifetime)
	}
	if c.UploadDir != "./uploads" {
		t.Fatalf("UploadDir = %q", c.UploadDir)
	}
	if c.SMTPHost != "smtp.gmail.com" || c.SMTPPort != 587 {
		t.Fatalf("smtp = %s:%d", c.SMTPHost, c.SMTPPort)
	}
	if c.FrontendURL != "http://localhost:5173" {
		t.Fatalf("FrontendURL = %q", c.FrontendURL)
	}
	if c.RedisEnabled {
		t.Fatal("Redis should be disabled with empty REDIS_URL")
	}
	if c.VideoTokenSecret != c.JWTSecret {
		t.Fatal("empty VIDEO_TOKEN_SECRET must fall back to JWT_SECRET")
	}
	if c.ScaleBandC != 50000 || c.ComputeChunkSize != 100 || c.SubmitGraceSeconds != 30 {
		t.Fatalf("scale defaults = %d/%d/%d", c.ScaleBandC, c.ComputeChunkSize, c.SubmitGraceSeconds)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@h/db")
	t.Setenv("JWT_SECRET", strings.Repeat("y", 40))
	t.Setenv("JWT_EXPIRY", "12h")
	t.Setenv("JWT_ISSUER", "issuer-x")
	t.Setenv("PORT", "9090")
	t.Setenv("VIDEO_TOKEN_SECRET", strings.Repeat("z", 40))
	t.Setenv("DB_MAX_OPEN_CONNS", "50")
	t.Setenv("DB_MAX_IDLE_CONNS", "10")
	t.Setenv("DB_CONN_MAX_LIFETIME", "1h")
	t.Setenv("ALLOWED_ORIGINS", " https://a.com , https://b.com , ")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1,10.0.0.0/8")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("REDIS_ENABLED", "true")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_USER", "u@example.com")
	t.Setenv("SMTP_APP_PASSWORD", "pw")

	c := LoadConfig()
	if c.DBMaxOpenConns != 50 || c.DBMaxIdleConns != 10 || c.DBConnMaxLifetime != time.Hour {
		t.Fatalf("pool = %d/%d/%v", c.DBMaxOpenConns, c.DBMaxIdleConns, c.DBConnMaxLifetime)
	}
	if len(c.AllowedOrigins) != 2 || c.AllowedOrigins[0] != "https://a.com" || c.AllowedOrigins[1] != "https://b.com" {
		t.Fatalf("AllowedOrigins = %v", c.AllowedOrigins)
	}
	if len(c.TrustedProxies) != 2 || c.TrustedProxies[0] != "127.0.0.1" {
		t.Fatalf("TrustedProxies = %v", c.TrustedProxies)
	}
	if !c.RedisEnabled || c.RedisURL == "" {
		t.Fatalf("redis = %v %q", c.RedisEnabled, c.RedisURL)
	}
	if c.SMTPPort != 2525 || c.SMTPFrom != "u@example.com" {
		t.Fatalf("smtp = %d %q", c.SMTPPort, c.SMTPFrom)
	}
	if c.VideoTokenSecret != strings.Repeat("z", 40) {
		t.Fatal("VIDEO_TOKEN_SECRET should not be overridden")
	}
}

func TestLoadConfigRedisEnabledRequiresURLOrFlag(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@h/db")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("JWT_EXPIRY", "4h")
	t.Setenv("JWT_ISSUER", "eduquant")
	t.Setenv("PORT", "8080")
	t.Setenv("VIDEO_TOKEN_SECRET", "")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("TRUSTED_PROXIES", "")
	t.Setenv("UPLOAD_DIR", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_APP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("FRONTEND_URL", "")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("REDIS_ENABLED", "")
	t.Setenv("QUEUE_MODE", "")
	if c := LoadConfig(); c.RedisEnabled {
		t.Fatal("REDIS_URL alone must not enable Redis in standard mode")
	}
	t.Setenv("QUEUE_MODE", "scale")
	if c := LoadConfig(); !c.RedisEnabled {
		t.Fatal("scale mode must enable Redis when REDIS_URL is set")
	}
	t.Setenv("QUEUE_MODE", "standard")
	t.Setenv("REDIS_ENABLED", "true")
	if c := LoadConfig(); !c.RedisEnabled {
		t.Fatal("REDIS_ENABLED=true must enable Redis")
	}
}
