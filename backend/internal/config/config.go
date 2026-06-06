package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL             string
	JWTSecret         string
	JWTExpiry         string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
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

	maxOpenStr := os.Getenv("DB_MAX_OPEN_CONNS")
	maxIdleStr := os.Getenv("DB_MAX_IDLE_CONNS")
	maxLifetimeStr := os.Getenv("DB_CONN_MAX_LIFETIME")

	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	if jwtExpiry == "" {
		jwtExpiry = "4h" // default to 4 hours
	}

	// Defaults
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

	return &Config{
		DBURL:             dbURL,
		JWTSecret:         jwtSecret,
		JWTExpiry:         jwtExpiry,
		DBMaxOpenConns:    maxOpen,
		DBMaxIdleConns:    maxIdle,
		DBConnMaxLifetime: maxLifetime,
	}
}
