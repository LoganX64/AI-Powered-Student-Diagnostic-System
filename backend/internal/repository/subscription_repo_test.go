package repository

import (
	"database/sql"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
	_ "github.com/lib/pq"
)

// ensureTenant returns the lowest existing tenant id, skipping the test if none
// exist. Ordered so the storage tests below always meter the same tenant.
func ensureTenant(t *testing.T, db *sql.DB) int {
	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants ORDER BY id LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant available:", err)
	}
	return tid
}

// missingTenantID is far above any real id, so GetStorageLimitBytes takes the
// no-subscription fallback path. Verified absent rather than assumed.
func missingTenantID(t *testing.T, db *sql.DB) int {
	const id = 99999999
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)`, id).Scan(&exists); err != nil {
		t.Fatalf("probe tenant %d: %v", id, err)
	}
	if exists {
		t.Fatalf("tenant id %d unexpectedly exists; pick another", id)
	}
	return id
}

// ensureAssignmentRow returns an existing assignment id for the tenant, skipping
// if the tenant has no assignments. Reusing a real row avoids the
// (student_id, test_id) unique constraint on inserts.
func ensureAssignmentRow(t *testing.T, db *sql.DB, tenantID int) int {
	var aid int
	err := db.QueryRow(`
		SELECT a.id FROM assignments a
		JOIN students s ON s.id = a.student_id
		WHERE s.tenant_id = $1
		LIMIT 1
	`, tenantID).Scan(&aid)
	if err != nil {
		t.Skip("no assignment for tenant:", err)
	}
	return aid
}

// TestGetStorageLimitBytes asserts the tenant's cap resolves to whatever plan
// the tenant is actually on. The earlier version of this test asserted every
// tenant was on Free, which is false in dev data (Demo Academy is starter, Demo
// Institute is enterprise) -- so it only ever passed against an empty database.
// Each tenant is checked against its own joined plan, so the assertion holds
// for Free, paid, and future plans alike.
func TestGetStorageLimitBytes(t *testing.T) {
	db := testutil.OpenTestDB(t)
	sub := NewSubscriptionRepo(db)

	rows, err := db.Query(`
		SELECT ts.tenant_id, COALESCE(sp.storage_limit_bytes, 0)
		FROM tenant_subscriptions ts
		JOIN subscription_plans sp ON sp.id = ts.plan_id
		ORDER BY ts.tenant_id
	`)
	if err != nil {
		t.Fatalf("query tenants: %v", err)
	}
	defer rows.Close()

	var checked int
	for rows.Next() {
		var tid, want int64
		if err := rows.Scan(&tid, &want); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got, err := sub.GetStorageLimitBytes(int(tid))
		if err != nil {
			t.Fatalf("GetStorageLimitBytes(%d): %v", tid, err)
		}
		if got != want {
			t.Errorf("GetStorageLimitBytes(%d)=%d, want that tenant's plan limit %d", tid, got, want)
		}
		checked++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if checked == 0 {
		t.Skip("no tenant subscriptions to verify")
	}
}

// TestGetStorageLimitBytesDefaultsToFree covers the ErrNoRows fallback at
// subscription_repo.go:124 -- a tenant with no subscription row must get the
// Free plan's cap, never an unlimited 0.
func TestGetStorageLimitBytesDefaultsToFree(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := missingTenantID(t, db)

	free, err := NewPlanRepo(db).GetBySlug("free")
	if err != nil {
		t.Fatalf("GetBySlug(free): %v", err)
	}

	got, err := NewSubscriptionRepo(db).GetStorageLimitBytes(tid)
	if err != nil {
		t.Fatalf("GetStorageLimitBytes: %v", err)
	}
	if got != free.StorageLimitBytes {
		t.Fatalf("GetStorageLimitBytes=%d for tenant with no subscription, want free plan limit %d", got, free.StorageLimitBytes)
	}
}

// TestIncrementStorageUsageIdempotent verifies the same (assignment, chunk) is
// metered exactly once, and that overage is recomputed past the cap.
func TestIncrementStorageUsageIdempotent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := ensureTenant(t, db)
	assignmentID := ensureAssignmentRow(t, db, tid)

	sub := NewSubscriptionRepo(db)

	// Reset metering baseline for a clean assertion.
	if _, err := db.Exec(`UPDATE storage_usage SET used_bytes = 0, overage_bytes = 0 WHERE tenant_id = $1`, tid); err != nil {
		t.Fatalf("reset storage_usage: %v", err)
	}
	defer db.Exec(`DELETE FROM storage_chunks WHERE assignment_id = $1`, assignmentID)
	defer db.Exec(`UPDATE storage_usage SET used_bytes = 0, overage_bytes = 0 WHERE tenant_id = $1`, tid)

	const chunkBytes int64 = 1000
	if err := sub.IncrementStorageUsage(tid, chunkBytes, assignmentID, "0"); err != nil {
		t.Fatalf("Increment #1: %v", err)
	}
	// Retry with the same index must be a no-op (idempotent).
	if err := sub.IncrementStorageUsage(tid, chunkBytes, assignmentID, "0"); err != nil {
		t.Fatalf("Increment #2: %v", err)
	}

	var used int64
	if err := db.QueryRow(`SELECT used_bytes FROM storage_usage WHERE tenant_id = $1`, tid).Scan(&used); err != nil {
		t.Fatalf("read used_bytes: %v", err)
	}
	if used != chunkBytes {
		t.Fatalf("used_bytes=%d after idempotent increments, want %d", used, chunkBytes)
	}
}

// TestReleaseStorageForAssignment verifies metered bytes are returned on release.
func TestReleaseStorageForAssignment(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := ensureTenant(t, db)
	assignmentID := ensureAssignmentRow(t, db, tid)

	sub := NewSubscriptionRepo(db)
	if _, err := db.Exec(`UPDATE storage_usage SET used_bytes = 0, overage_bytes = 0 WHERE tenant_id = $1`, tid); err != nil {
		t.Fatalf("reset storage_usage: %v", err)
	}
	defer db.Exec(`UPDATE storage_usage SET used_bytes = 0, overage_bytes = 0 WHERE tenant_id = $1`, tid)

	const chunkBytes int64 = 2500
	if err := sub.IncrementStorageUsage(tid, chunkBytes, assignmentID, "0"); err != nil {
		t.Fatalf("Increment: %v", err)
	}
	if err := sub.IncrementStorageUsage(tid, chunkBytes, assignmentID, "1"); err != nil {
		t.Fatalf("Increment #2: %v", err)
	}

	if err := sub.ReleaseStorageForAssignment(tid, assignmentID); err != nil {
		t.Fatalf("Release: %v", err)
	}

	var used int64
	if err := db.QueryRow(`SELECT used_bytes FROM storage_usage WHERE tenant_id = $1`, tid).Scan(&used); err != nil {
		t.Fatalf("read used_bytes: %v", err)
	}
	if used != 0 {
		t.Fatalf("used_bytes after release = %d, want 0", used)
	}
	var chunks int
	if err := db.QueryRow(`SELECT COUNT(*) FROM storage_chunks WHERE assignment_id = $1`, assignmentID).Scan(&chunks); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if chunks != 0 {
		t.Fatalf("storage_chunks after release = %d, want 0", chunks)
	}
}

// TestOverageComputedPastCap verifies overage_bytes becomes > 0 once used_bytes
// exceeds the plan's storage_limit_bytes.
func TestOverageComputedPastCap(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := ensureTenant(t, db)
	assignmentID := ensureAssignmentRow(t, db, tid)

	sub := NewSubscriptionRepo(db)
	limit, err := sub.GetStorageLimitBytes(tid)
	if err != nil {
		t.Fatalf("GetStorageLimitBytes: %v", err)
	}

	if _, err := db.Exec(`UPDATE storage_usage SET used_bytes = 0, overage_bytes = 0 WHERE tenant_id = $1`, tid); err != nil {
		t.Fatalf("reset storage_usage: %v", err)
	}
	defer db.Exec(`DELETE FROM storage_chunks WHERE assignment_id = $1`, assignmentID)
	defer db.Exec(`UPDATE storage_usage SET used_bytes = 0, overage_bytes = 0 WHERE tenant_id = $1`, tid)

	// Push usage just past the cap.
	over := int64(1024)
	if err := sub.IncrementStorageUsage(tid, limit+over, assignmentID, "0"); err != nil {
		t.Fatalf("Increment past cap: %v", err)
	}

	var overage int64
	if err := db.QueryRow(`SELECT overage_bytes FROM storage_usage WHERE tenant_id = $1`, tid).Scan(&overage); err != nil {
		t.Fatalf("read overage_bytes: %v", err)
	}
	if overage <= 0 {
		t.Fatalf("overage_bytes=%d, want > 0 once over cap", overage)
	}
}
