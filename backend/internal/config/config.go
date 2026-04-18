package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL string
}

func LoadConfig() *Config {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found (this is fine in production)")
	}

	dbURL := os.Getenv("DB_URL")
	log.Println("DB_URL from env:", dbURL)

	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	return &Config{
		DBURL: dbURL,
	}
}
