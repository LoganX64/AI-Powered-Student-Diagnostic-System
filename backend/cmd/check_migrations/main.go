package main

import (
	"database/sql"
	"fmt"
	"log"

	cfgpkg "ai-student-diagnostic/backend/internal/config"

	_ "github.com/lib/pq"
)

func main() {
	cfg := cfgpkg.LoadConfig()

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var version sql.NullInt64
	var dirty sql.NullBool

	rows, err := db.Query("SELECT version, dirty FROM schema_migrations")
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("schema_migrations:")
	found := false
	for rows.Next() {
		if err := rows.Scan(&version, &dirty); err != nil {
			log.Fatalf("scan failed: %v", err)
		}
		found = true
		v := "NULL"
		if version.Valid {
			v = fmt.Sprintf("%d", version.Int64)
		}
		d := "NULL"
		if dirty.Valid {
			d = fmt.Sprintf("%t", dirty.Bool)
		}
		fmt.Printf("version=%s, dirty=%s\n", v, d)
	}

	if !found {
		fmt.Println("(no rows)")
	}
}
