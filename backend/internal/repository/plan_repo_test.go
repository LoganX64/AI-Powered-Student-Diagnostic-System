package repository

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func testDB(t *testing.T) *sql.DB {
	url := os.Getenv("DB_URL")
	if url == "" {
		url = "postgres://postgres:9908@localhost:5432/sqi_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return db
}

func TestPlanRepoCRUD(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	pr := NewPlanRepo(db)

	plans, err := pr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(plans) < 4 {
		t.Fatalf("expected >=4 seeded plans, got %d", len(plans))
	}

	free, err := pr.GetBySlug("free")
	if err != nil {
		t.Fatalf("GetBySlug(free): %v", err)
	}
	if free.ID == 0 || free.SQIAccess {
		t.Fatalf("free plan sanity failed: id=%d sqi=%v", free.ID, free.SQIAccess)
	}

	// nil features must not violate NOT NULL (regression for ISSUE-6 path)
	id, err := pr.Create(PlanRow{
		Name:         "TestPlan",
		Slug:         "test-plan-x",
		StudentLimit: 5,
		PriceMonthly: 100,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer pr.Delete(id)

	got, err := pr.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "TestPlan" || string(got.Features) != "[]" {
		t.Fatalf("Create did not persist correctly: %+v", got)
	}

	got.Name = "TestPlan2"
	if err := pr.Update(*got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, _ := pr.GetByID(id)
	if got2.Name != "TestPlan2" {
		t.Fatalf("Update did not persist: %+v", got2)
	}
}
