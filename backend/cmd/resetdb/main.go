package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"ai-student-diagnostic/backend/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Connect to DB
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}
	defer db.Close()

	fmt.Println("Dropping public schema to reset database...")
	_, err = db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	if err != nil {
		log.Fatalf("Failed to drop schema: %v", err)
	}

	fmt.Println("Schema reset. Running migrations UP...")

	// 2. Run migrations
	m, err := migrate.New("file://migrations", cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("Migrations completed successfully. Database is fresh!")
	os.Exit(0)
}
