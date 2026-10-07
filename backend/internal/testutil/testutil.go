// Package testutil holds the DB helpers every DB-backed test file used to
// copy-paste. Keeping one copy means a fix (a new env var, a better skip
// message) lands in one place instead of four.
package testutil

import (
	"database/sql"
	"os"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// OpenTestDB opens a Postgres pool from DB_URL and ping-verifies it. It
// registers a t.Cleanup to close the pool, so callers do not need a defer.
// DB_URL is loaded from the environment, falling back to ../../.env.
func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if url := os.Getenv("DB_URL"); url != "" {
		return openDB(t, url)
	}
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")
	url := os.Getenv("DB_URL")
	if url == "" {
		RequireDB(t)
	}
	return openDB(t, url)
}

func openDB(t *testing.T, url string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// RequireDB handles a missing DB_URL. It always skips so a DB-less run stays
// usable locally, but REQUIRE_DB=1 turns the skip into a hard failure so CI
// cannot report green while silently exercising nothing.
func RequireDB(t *testing.T) {
	t.Helper()
	if os.Getenv("REQUIRE_DB") == "1" {
		t.Fatalf("DB_URL not set and REQUIRE_DB=1: DB-backed tests must run")
	}
	t.Skip("DB_URL not set (set DB_URL, or REQUIRE_DB=1 to fail loudly)")
}

// IsContractTest skips unless CONTRACT_TESTS=1. Rows in a table-driven test
// that need the live Postgres database mark themselves with `isContract: true`
// and call this guard, so unit rows run offline while contract rows only run
// when explicitly opted in.
func IsContractTest(t *testing.T) {
	t.Helper()
	if os.Getenv("CONTRACT_TESTS") != "1" {
		t.Skip("contract test: set CONTRACT_TESTS=1 to run against the live database")
	}
}
