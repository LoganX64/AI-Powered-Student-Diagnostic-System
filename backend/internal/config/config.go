package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL      string
	JWTSecret  string
	JWTExpiry  string
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

	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	if jwtExpiry == "" {
		jwtExpiry = "4h" // default to 4 hours
	}

	return &Config{
		DBURL:      dbURL,
		JWTSecret:  jwtSecret,
		JWTExpiry:  jwtExpiry,
	}
}
