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
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(dbURL string) {
	log.Println("[MIGRATE] Connecting to database...")

	// Append connect_timeout to the DSN so the TCP dial itself has a hard
	// deadline. Without this, Avast (or a slow/unreachable DB) can block the
	// process indefinitely during startup.
	timedURL := dbURL
	if !strings.Contains(dbURL, "connect_timeout") {
		sep := "?"
		if strings.Contains(dbURL, "?") {
			sep = "&"
		}
		timedURL = dbURL + sep + "connect_timeout=15"
	}

	m, err := migrate.New(
		"file://migrations",
		timedURL,
	)
	if err != nil {
		log.Fatalf("[MIGRATE] failed to initialise: %v", err)
	}

	if err := m.Up(); err != nil {
		if err.Error() == "no change" {
			log.Println("[MIGRATE] No new migrations")
		} else {
			log.Fatalf("[MIGRATE] failed to apply: %v", err)
		}
	}

	log.Println("[MIGRATE] Migrations applied successfully")
}

func main() {
	cfg := config.LoadConfig()
	utils.InitJWTConfigWithVideoSecret(cfg.JWTSecret, cfg.VideoTokenSecret, cfg.JWTExpiry, cfg.JWTIssuer)

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
