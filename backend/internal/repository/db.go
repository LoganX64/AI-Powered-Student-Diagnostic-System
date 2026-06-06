package repository

import (
	"database/sql"
	"log"

	"ai-student-diagnostic/backend/internal/config"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(cfg *config.Config) *sql.DB {
	var err error

	DB, err = sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}

	// Apply pool settings from config
	if cfg.DBMaxOpenConns > 0 {
		DB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns >= 0 {
		DB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetime > 0 {
		DB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	}

	// Verify connectivity
	err = DB.Ping()
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	return DB
}

func GetDB() *sql.DB {
	return DB
}
