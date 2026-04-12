package repository

import "database/sql"

func Exists(db *sql.DB, query string, args ...interface{}) (bool, error) {
	var exists bool
	err := db.QueryRow(query, args...).Scan(&exists)
	return exists, err
}
