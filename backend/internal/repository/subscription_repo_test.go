package repository

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

// ensureTenant returns an existing tenant id, skipping the test if none exist.
func ensureTenant(t *testing.T, db *sql.DB) int {
	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant available:", err)
	}
	return tid
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

func TestGetStorageLimitBytes(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	tid := ensureTenant(t, db)

	pr := NewPlanRepo(db)
	sub := NewSubscriptionRepo(db)

	free, err := pr.GetBySlug("free")
	if err != nil {
		t.Fatalf("GetBySlug(free): %v", err)
	}

	got, err := sub.GetStorageLimitBytes(tid)
	if err != nil {
		t.Fatalf("GetStorageLimitBytes: %v", err)
	}
	// Every seeded tenant has a Free subscription, so the limit must match Free.
	if got != free.StorageLimitBytes {
		t.Fatalf("GetStorageLimitBytes=%d, want free plan limit %d", got, free.StorageLimitBytes)
	}
}

// TestIncrementStorageUsageIdempotent verifies the same (assignment, chunk) is
// metered exactly once, and that overage is recomputed past the cap.
func TestIncrementStorageUsageIdempotent(t *testing.T) {
	db := testDB(t)
	defer db.Close()
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
	db := testDB(t)
	defer db.Close()
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
	db := testDB(t)
	defer db.Close()
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
