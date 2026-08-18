package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL             string
	JWTSecret         string
	JWTExpiry         string
	JWTIssuer         string
	Port              string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	AllowedOrigins    []string
	TrustedProxies    []string

	ScaleBandB            int
	ScaleBandC            int
	ComputeChunkSize      int
	QueueMode             string
	UploadDir             string
	PricingTimingFlat     float64
	PricingAutosaveFlat   float64
	PricingTabFlat        float64
	PricingVideoPerMinute float64
	SubmitGraceSeconds    int

	RedisURL      string
	RedisEnabled  bool
	CloudinaryURL string
}

func LoadConfig() *Config {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found (this is fine in production)")
	}

	dbURL := os.Getenv("DB_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	jwtExpiry := os.Getenv("JWT_EXPIRY")
	port := os.Getenv("PORT")

	maxOpenStr := os.Getenv("DB_MAX_OPEN_CONNS")
	maxIdleStr := os.Getenv("DB_MAX_IDLE_CONNS")
	maxLifetimeStr := os.Getenv("DB_CONN_MAX_LIFETIME")

	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	if port == "" {
		log.Fatal("PORT is not set")
	}

	if jwtExpiry == "" {
		log.Fatal("JWT_EXPIRY is not set")
	}

	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		log.Fatal("JWT_ISSUER is not set")
	}

	// Defaults (only if not provided)
	maxOpen := 25
	maxIdle := 25
	maxLifetime := 5 * time.Minute

	if maxOpenStr != "" {
		if v, err := strconv.Atoi(maxOpenStr); err == nil {
			maxOpen = v
		}
	}
	if maxIdleStr != "" {
		if v, err := strconv.Atoi(maxIdleStr); err == nil {
			maxIdle = v
		}
	}
	if maxLifetimeStr != "" {
		if d, err := time.ParseDuration(maxLifetimeStr); err == nil {
			maxLifetime = d
		}
	}

	var allowedOrigins []string
	if originsStr := os.Getenv("ALLOWED_ORIGINS"); originsStr != "" {
		for _, o := range strings.Split(originsStr, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}

	var trustedProxies []string
	if proxyStr := os.Getenv("TRUSTED_PROXIES"); proxyStr != "" {
		for _, p := range strings.Split(proxyStr, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				trustedProxies = append(trustedProxies, trimmed)
			}
		}
	}

	scaleBandB := intEnv("SCALE_BAND_B", 10000)
	scaleBandC := intEnv("SCALE_BAND_C", 50000)
	computeChunkSize := intEnv("COMPUTE_CHUNK_SIZE", 100)
	submitGrace := intEnv("SUBMIT_GRACE_SECONDS", 30)

	queueMode := os.Getenv("QUEUE_MODE")
	if queueMode == "" {
		queueMode = "standard"
	}

	redisURL := os.Getenv("REDIS_URL")
	redisEnabled := redisURL != "" && (queueMode == "scale" || os.Getenv("REDIS_ENABLED") == "true")

	cloudinaryURL := os.Getenv("CLOUDINARY_URL")

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	pricingTiming := floatEnv("PRICING_TIMING_FLAT", 0)
	pricingAutosave := floatEnv("PRICING_AUTOSAVE_FLAT", 0)
	pricingTab := floatEnv("PRICING_TAB_FLAT", 0)
	pricingVideo := floatEnv("PRICING_VIDEO_PER_MINUTE", 0)

	return &Config{
		DBURL:             dbURL,
		JWTSecret:         jwtSecret,
		JWTExpiry:         jwtExpiry,
		JWTIssuer:         jwtIssuer,
		Port:              port,
		DBMaxOpenConns:    maxOpen,
		DBMaxIdleConns:    maxIdle,
		DBConnMaxLifetime: maxLifetime,
		AllowedOrigins:    allowedOrigins,
		TrustedProxies:    trustedProxies,

		ScaleBandB:            scaleBandB,
		ScaleBandC:            scaleBandC,
		ComputeChunkSize:      computeChunkSize,
		QueueMode:             queueMode,
		UploadDir:             uploadDir,
		PricingTimingFlat:     pricingTiming,
		PricingAutosaveFlat:   pricingAutosave,
		PricingTabFlat:        pricingTab,
		PricingVideoPerMinute: pricingVideo,
		SubmitGraceSeconds:    submitGrace,

		RedisURL:      redisURL,
		RedisEnabled:  redisEnabled,
		CloudinaryURL: cloudinaryURL,
	}
}

func intEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		} else {
			log.Printf("[CONFIG] invalid int for %s=%q, using default %d: %v", key, v, def, err)
		}
	}
	return def
}

func floatEnv(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		} else {
			log.Printf("[CONFIG] invalid float for %s=%q, using default %v: %v", key, v, def, err)
		}
	}
	return def
}
