package repository

import (
	"context"
	"database/sql"
	"log"
	"time"

	"ai-student-diagnostic/backend/internal/config"

	_ "github.com/lib/pq"
)

func InitDB(cfg *config.Config) *sql.DB {
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.DBMaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns >= 0 {
		db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	}

	// Ping with a hard deadline so Avast (or an unreachable DB) cannot block
	// the server from starting indefinitely.
	pingCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = db.PingContext(pingCtx); err != nil {
		log.Fatal("DB connection failed:", err)
	}

	return db
}
