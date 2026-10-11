package repository

import (
	"database/sql"
	"testing"
	"time"

	"ai-student-diagnostic/backend/internal/testutil"
	"ai-student-diagnostic/backend/utils"
)

// resultTokenGraph builds a throwaway tenant with one submitted attempt to hang
// tokens off. Deleted via the tenant, which cascades to attempts and tokens.
func resultTokenGraph(t *testing.T, db *sql.DB) (tenantID, attemptID int) {
	t.Helper()
	tenantID = testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tenantID)
	subjectID := testutil.CreateSubject(t, db, tenantID, "Token Subject")
	studentID := testutil.CreateStudent(t, db, tenantID, coachID, testutil.UniqueCode(t, "tok"), "Token Student")
	testID := testutil.CreateTest(t, db, tenantID, subjectID, coachID, 30)
	assignmentID := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	return tenantID, testutil.CreateAttempt(t, db, assignmentID)
}

// A minted token resolves back to its attempt.
func TestResultTokenResolveRoundTrip(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, attemptID := resultTokenGraph(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	tr := NewResultTokenRepo(db)
	token, err := tr.Mint(attemptID, time.Hour)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if token == "" {
		t.Fatal("Mint returned an empty token")
	}

	got, err := tr.Resolve(token)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != attemptID {
		t.Fatalf("Resolve returned attempt %d, want %d", got, attemptID)
	}
}

// The raw token must never be persisted. If the stored column ever contains the
// value that was emailed, a database read would yield a working link.
func TestResultTokenStoresOnlyHash(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, attemptID := resultTokenGraph(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	tr := NewResultTokenRepo(db)
	token, err := tr.Mint(attemptID, time.Hour)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	var stored string
	if err := db.QueryRow(`SELECT token_hash FROM result_tokens WHERE attempt_id = $1`, attemptID).Scan(&stored); err != nil {
		t.Fatalf("read stored hash: %v", err)
	}
	if stored == token {
		t.Fatal("SECURITY: the raw token was persisted")
	}
	if stored != utils.HashSHA256Hex(token) {
		t.Fatalf("stored value is not the SHA-256 of the token")
	}

	// The reverse must not be derivable: knowing the hash must not yield a token.
	if utils.HashSHA256Hex(stored) == stored {
		t.Fatal("stored value is not a hash of a hash")
	}
}

// Expired and unknown tokens are indistinguishable. Both must return
// sql.ErrNoRows so the HTTP layer can render a single 404 and cannot be used to
// probe for whether an attempt exists.
func TestResultTokenExpiredIndistinguishableFromUnknown(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, attemptID := resultTokenGraph(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	tr := NewResultTokenRepo(db)

	expired, err := tr.Mint(attemptID, -time.Minute)
	if err != nil {
		t.Fatalf("Mint expired: %v", err)
	}
	valid, err := tr.Mint(attemptID, time.Hour)
	if err != nil {
		t.Fatalf("Mint valid: %v", err)
	}

	_, errExpired := tr.Resolve(expired)
	if errExpired != sql.ErrNoRows {
		t.Fatalf("expired token error = %v, want sql.ErrNoRows", errExpired)
	}

	unknown, err := utils.RandomTokenHex(32)
	if err != nil {
		t.Fatalf("generate unknown token: %v", err)
	}
	_, errUnknown := tr.Resolve(unknown)
	if errUnknown != sql.ErrNoRows {
		t.Fatalf("unknown token error = %v, want sql.ErrNoRows", errUnknown)
	}
	if errExpired != errUnknown {
		t.Fatal("expired and unknown tokens must fail identically")
	}

	// Empty input is also a plain miss, not an error the caller must special-case.
	if _, err := tr.Resolve(""); err != sql.ErrNoRows {
		t.Fatalf("empty token error = %v, want sql.ErrNoRows", err)
	}

	// The valid sibling still resolves, so the negative cases are not passing
	// because everything is broken.
	if got, err := tr.Resolve(valid); err != nil || got != attemptID {
		t.Fatalf("valid sibling failed: attempt=%d err=%v", got, err)
	}
}

// Tokens are per attempt and unguessable. Two mints never collide, and a token
// minted for one attempt does not resolve to another.
func TestResultTokenPerAttempt(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, attemptID := resultTokenGraph(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	_, coachID := testutil.CreateCoach(t, db, tid)
	subjectID := testutil.CreateSubject(t, db, tid, "Second Subject")
	studentID := testutil.CreateStudent(t, db, tid, coachID, testutil.UniqueCode(t, "tok2"), "Second Student")
	testID := testutil.CreateTest(t, db, tid, subjectID, coachID, 30)
	otherAssignment := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	otherAttempt := testutil.CreateAttempt(t, db, otherAssignment)

	tr := NewResultTokenRepo(db)
	first, err := tr.Mint(attemptID, time.Hour)
	if err != nil {
		t.Fatalf("Mint first: %v", err)
	}
	second, err := tr.Mint(attemptID, time.Hour)
	if err != nil {
		t.Fatalf("Mint second: %v", err)
	}
	if first == second {
		t.Fatal("two mints for the same attempt produced identical tokens")
	}
	third, err := tr.Mint(otherAttempt, time.Hour)
	if err != nil {
		t.Fatalf("Mint third: %v", err)
	}

	if got, err := tr.Resolve(first); err != nil || got != attemptID {
		t.Fatalf("first token resolved to attempt=%d err=%v, want %d", got, err, attemptID)
	}
	if got, err := tr.Resolve(third); err != nil || got != otherAttempt {
		t.Fatalf("third token resolved to attempt=%d err=%v, want %d", got, err, otherAttempt)
	}
	if got, err := tr.Resolve(third); err == nil && got == attemptID {
		t.Fatal("a token for one attempt resolved to another attempt")
	}
}

// DeleteForAttempt revokes every token for an attempt, so re-releasing does not
// leave an older link working alongside a newer one.
func TestResultTokenDeleteForAttempt(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, attemptID := resultTokenGraph(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	tr := NewResultTokenRepo(db)
	first, err := tr.Mint(attemptID, time.Hour)
	if err != nil {
		t.Fatalf("Mint first: %v", err)
	}
	second, err := tr.Mint(attemptID, time.Hour)
	if err != nil {
		t.Fatalf("Mint second: %v", err)
	}

	if err := tr.DeleteForAttempt(attemptID); err != nil {
		t.Fatalf("DeleteForAttempt: %v", err)
	}

	for _, tok := range []string{first, second} {
		if _, err := tr.Resolve(tok); err != sql.ErrNoRows {
			t.Fatalf("revoked token still resolved (err=%v)", err)
		}
	}

	var left int
	if err := db.QueryRow(`SELECT COUNT(*) FROM result_tokens WHERE attempt_id = $1`, attemptID).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left != 0 {
		t.Fatalf("%d token rows survived revocation", left)
	}
}

// Deleting the attempt cascades to its tokens, so a deleted attempt cannot leave
// a dangling credential behind.
func TestResultTokenCascadesOnAttemptDelete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, attemptID := resultTokenGraph(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	tr := NewResultTokenRepo(db)
	token, err := tr.Mint(attemptID, time.Hour)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM attempts WHERE id = $1`, attemptID); err != nil {
		t.Fatalf("delete attempt: %v", err)
	}

	if _, err := tr.Resolve(token); err != sql.ErrNoRows {
		t.Fatalf("token for a deleted attempt still resolved (err=%v)", err)
	}
	var left int
	if err := db.QueryRow(`SELECT COUNT(*) FROM result_tokens WHERE attempt_id = $1`, attemptID).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left != 0 {
		t.Fatalf("%d token rows survived the attempt deletion", left)
	}
}