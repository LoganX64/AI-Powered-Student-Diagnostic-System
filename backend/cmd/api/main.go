package main

import (
	"ai-student-diagnostic/backend/internal/config"
	db "ai-student-diagnostic/backend/internal/repository"
	routes "ai-student-diagnostic/backend/internal/routes"
)

func main() {
	cfg := config.LoadConfig()
	db.InitDB(cfg.DBURL)

	r := routes.SetupRouter()
	r.Run(":8080")
}
