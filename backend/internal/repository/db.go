package repository

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(dbURL string) *sql.DB {
	var err error

	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	return DB
}

func GetDB() *sql.DB {
	return DB
}
