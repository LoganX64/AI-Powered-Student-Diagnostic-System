package services

import (
	"database/sql"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

// releasePair builds two students in one tenant, each with their own computed
// result, and returns their graphs.
func releasePair(t *testing.T, db *sql.DB) (mine, theirs viewGraph) {
	t.Helper()
	mine = setupView(t, db, "Mine", false)
	theirs = setupView(t, db, "Theirs", false)
	// Move theirs into mine's tenant so the tenant check cannot be what rejects
	// the release; the student binding is what's under test.
	if _, err := db.Exec(`UPDATE students SET tenant_id = $1 WHERE id = $2`, mine.tid, theirs.studentID); err != nil {
		t.Fatalf("move second student into first tenant: %v", err)
	}
	return mine, theirs
}

// A caller authorised for student A must not be able to release student B's
// result by pairing A's id with B's assignment id. The handler's ownership check
// only proves the caller owns A; the query must then bind A, so B's row is
// unreachable.
func TestReleaseRefusesMismatchedStudentAndAssignment(t *testing.T) {
	db := testutil.OpenTestDB(t)
	mine, theirs := releasePair(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, mine.tid)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, theirs.tid)

	svc, _, _ := releaseServiceFor(db)

	// Authorised student, foreign assignment.
	if _, err := svc.ReleaseResult(mine.tid, mine.studentID, theirs.assignmentID, 1); err != ErrResultNotComputed {
		t.Fatalf("err=%v, want ErrResultNotComputed for a mismatched student/assignment pair", err)
	}

	var releasedAt sql.NullTime
	if err := db.QueryRow(`SELECT released_at FROM attempt_results WHERE attempt_id = $1`, theirs.attemptID).
		Scan(&releasedAt); err != nil {
		t.Fatalf("read released_at: %v", err)
	}
	if releasedAt.Valid {
		t.Fatal("SECURITY: another student's result was released")
	}

	var tokens int
	if err := db.QueryRow(`SELECT COUNT(*) FROM result_tokens WHERE attempt_id = $1`, theirs.attemptID).Scan(&tokens); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if tokens != 0 {
		t.Fatal("SECURITY: a token was minted for another student's result")
	}
}

// The same test across tenants: an admin of one tenant must not reach another
// tenant's result even with a valid assignment id.
func TestReleaseRefusesCrossTenant(t *testing.T) {
	db := testutil.OpenTestDB(t)
	mine := setupView(t, db, "Mine", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, mine.tid)
	theirs := setupView(t, db, "Theirs", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, theirs.tid)

	svc, _, _ := releaseServiceFor(db)

	if _, err := svc.ReleaseResult(mine.tid, mine.studentID, theirs.assignmentID, 1); err != ErrResultNotComputed {
		t.Fatalf("err=%v, want ErrResultNotComputed across tenants", err)
	}

	var releasedAt sql.NullTime
	if err := db.QueryRow(`SELECT released_at FROM attempt_results WHERE attempt_id = $1`, theirs.attemptID).
		Scan(&releasedAt); err != nil {
		t.Fatalf("read released_at: %v", err)
	}
	if releasedAt.Valid {
		t.Fatal("SECURITY: a result in another tenant was released")
	}
}

// The authenticated read path must not cross students either.
func TestResultViewListRefusesCrossTenant(t *testing.T) {
	db := testutil.OpenTestDB(t)
	mine := setupView(t, db, "Mine", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, mine.tid)
	theirs := setupView(t, db, "Theirs", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, theirs.tid)

	releaseAndToken(t, db, theirs)

	svc, _ := viewServiceFor(db)
	results, err := svc.ListForStudent(mine.studentID, mine.tid)
	if err != nil {
		t.Fatalf("ListForStudent: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("SECURITY: %d results from another tenant returned", len(results))
	}
}

var _ = testutil.CreateTenant