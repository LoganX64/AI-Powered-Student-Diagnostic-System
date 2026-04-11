package main

import (
	"ai-student-diagnostic/backend/internal/config"
	db "ai-student-diagnostic/backend/internal/repository"
	routes "ai-student-diagnostic/backend/internal/routes"
	"log"

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

	runMigrations(cfg.DBURL)

	conn := db.InitDB(cfg.DBURL)

	r := routes.SetupRouter(conn)
	r.Run(":8080")
}
