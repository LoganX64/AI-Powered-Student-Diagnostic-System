package config

import (
	"log"
	"os"
)

type Config struct {
	DBURL string
}

func LoadConfig() *Config {
	dbURL := os.Getenv("DB_URL")
	log.Println("DB_URL from env:", os.Getenv("DB_URL"))

	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	return &Config{
		DBURL: dbURL,
	}
}
