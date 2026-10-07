package services

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

// requireDB handles a missing DB_URL. It always skips so a DB-less run stays
// usable locally, but REQUIRE_DB=1 turns the skip into a hard failure so CI
// cannot report green while silently exercising nothing.
func requireDB(t *testing.T) {
	if os.Getenv("REQUIRE_DB") == "1" {
		t.Fatalf("DB_URL not set and REQUIRE_DB=1: DB-backed tests must run")
	}
	t.Skip("DB_URL not set (set DB_URL, or REQUIRE_DB=1 to fail loudly)")
}

func svcTestDB(t *testing.T) *sql.DB {
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")
	url := os.Getenv("DB_URL")
	if url == "" {
		requireDB(t)
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

func TestGuardStorageNoProctoringPlan(t *testing.T) {
	db := svcTestDB(t)
	defer db.Close()

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant:", err)
	}
	var testID int
	if err := db.QueryRow(`SELECT id FROM tests WHERE tenant_id = $1 LIMIT 1`, tid).Scan(&testID); err != nil {
		t.Skip("no test for tenant:", err)
	}

	svc := &AssignmentService{
		TestPaperRepo:    repository.NewTestPaperRepo(db),
		SubscriptionRepo: repository.NewSubscriptionRepo(db),
	}

	// Free plan does not include proctoring → guard is a no-op (nil).
	if err := svc.guardStorageForProctoring(tid, testID, 1); err != nil {
		t.Fatalf("guard on non-proctored plan returned error: %v", err)
	}
}

func TestGuardStorageProctoringOverLimit(t *testing.T) {
	db := svcTestDB(t)
	defer db.Close()

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant:", err)
	}
	var testID int
	if err := db.QueryRow(`SELECT id FROM tests WHERE tenant_id = $1 LIMIT 1`, tid).Scan(&testID); err != nil {
		t.Skip("no test for tenant:", err)
	}

	// Switch tenant to a proctored plan (professional = id 3) for the test.
	var origPlan int
	db.QueryRow(`SELECT plan_id FROM tenant_subscriptions WHERE tenant_id = $1`, tid).Scan(&origPlan)
	defer db.Exec(`UPDATE tenant_subscriptions SET plan_id = $1, updated_at = NOW() WHERE tenant_id = $2`, origPlan, tid)
	if _, err := db.Exec(`
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, status)
		VALUES ($1, 3, 'active')
		ON CONFLICT (tenant_id) DO UPDATE SET plan_id = 3, status = 'active', updated_at = NOW()
	`, tid); err != nil {
		t.Fatalf("assign pro plan: %v", err)
	}

	svc := &AssignmentService{
		TestPaperRepo:    repository.NewTestPaperRepo(db),
		SubscriptionRepo: repository.NewSubscriptionRepo(db),
	}

	// A huge cohort pushes the projection past the 50GB professional cap.
	err := svc.guardStorageForProctoring(tid, testID, 1_000_000)
	if err == nil {
		t.Fatalf("expected 402 over-limit error, got nil")
	}
	var svcErr *CreateAssignmentError
	if !errors.As(err, &svcErr) || svcErr.Status != http.StatusPaymentRequired {
		t.Fatalf("expected PaymentRequired, got %v", err)
	}
}

func TestGuardStorageProctoringWithinLimit(t *testing.T) {
	db := svcTestDB(t)
	defer db.Close()

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant:", err)
	}
	var testID int
	if err := db.QueryRow(`SELECT id FROM tests WHERE tenant_id = $1 LIMIT 1`, tid).Scan(&testID); err != nil {
		t.Skip("no test for tenant:", err)
	}

	var origPlan int
	db.QueryRow(`SELECT plan_id FROM tenant_subscriptions WHERE tenant_id = $1`, tid).Scan(&origPlan)
	defer db.Exec(`UPDATE tenant_subscriptions SET plan_id = $1, updated_at = NOW() WHERE tenant_id = $2`, origPlan, tid)
	if _, err := db.Exec(`
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, status)
		VALUES ($1, 3, 'active')
		ON CONFLICT (tenant_id) DO UPDATE SET plan_id = 3, status = 'active', updated_at = NOW()
	`, tid); err != nil {
		t.Fatalf("assign pro plan: %v", err)
	}

	svc := &AssignmentService{
		TestPaperRepo:    repository.NewTestPaperRepo(db),
		SubscriptionRepo: repository.NewSubscriptionRepo(db),
	}

	// A single student is well within the cap → no error.
	if err := svc.guardStorageForProctoring(tid, testID, 1); err != nil {
		t.Fatalf("guard within limit returned error: %v", err)
	}
}
