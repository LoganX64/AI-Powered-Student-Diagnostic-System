package main

import (
	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/internal/repository"
	routes "ai-student-diagnostic/backend/internal/routes"
	"ai-student-diagnostic/backend/utils"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(dbURL string) {
	m, err := migrate.New(
		"file://migrations",
		dbURL,
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := m.Up(); err != nil {
		if err.Error() == "no change" {
			log.Println("No new migrations")
		} else {
			log.Fatal(err)
		}
	}

	log.Println("Migrations applied successfully")
}

func main() {
	cfg := config.LoadConfig()
	utils.InitJWTConfig(cfg.JWTSecret, cfg.JWTExpiry, cfg.JWTIssuer)

	runMigrations(cfg.DBURL)

	conn := repository.InitDB(cfg)

	r, shutdown := routes.SetupRouter(conn, cfg, cfg.AllowedOrigins, cfg.TrustedProxies)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Drain buffered answers and stop background workers before closing the DB
	// so no student input is lost on shutdown.
	if err := shutdown(); err != nil {
		log.Printf("Error during shutdown drain: %v", err)
	}

	if err := conn.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Server stopped")
}
