package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(dbURL string) {
	var err error
	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Error pinging the database:", err)
	}

	log.Println("Successfully connected to the database")
}

func GetDB() *sql.DB {
	return DB
}
