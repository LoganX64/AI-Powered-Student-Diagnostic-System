package services

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"
	"ai-student-diagnostic/backend/utils"
)

func hashOf(token string) string { return utils.HashSHA256Hex(token) }

type viewGraph struct {
	tid, studentID, assignmentID, attemptID int
}

// setupView builds a student with a computed result, optionally released.
func setupView(t *testing.T, db *sql.DB, name string, release bool) viewGraph {
	t.Helper()
	tid := testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tid)
	subjectID := testutil.CreateSubject(t, db, tid, "View Subject")
	studentID := testutil.CreateStudent(t, db, tid, coachID, testutil.UniqueCode(t, "vw"), name)
	testID := testutil.CreateTest(t, db, tid, subjectID, coachID, 30)
	assignmentID := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	attemptID := testutil.CreateAttempt(t, db, assignmentID)

	summary := []byte(`{"overall_sqi":41.5,"exam_summary":{"correct":6,"wrong":4,"skipped":0,` +
		`"total_questions":10,"total_marks_earned":12,"max_possible_score":40,"net_score":5,"score_percent":30}}`)
	if _, err := db.Exec(
		`INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json)
		 VALUES ($1, 41.5, 12, $2)`, attemptID, summary); err != nil {
		t.Fatalf("insert result: %v", err)
	}
	if release {
		if _, err := db.Exec(
			`UPDATE attempt_results SET released_at = NOW() WHERE attempt_id = $1`, attemptID); err != nil {
			t.Fatalf("mark released: %v", err)
		}
	}
	return viewGraph{tid, studentID, assignmentID, attemptID}
}

func viewServiceFor(db *sql.DB) (*ResultViewService, *repository.ResultTokenRepo) {
	ar := repository.NewAttemptRepo(db)
	tr := repository.NewResultTokenRepo(db)
	return NewResultViewService(ar, tr), tr
}

// releaseAndToken performs the real release so the token is a genuine one.
func releaseAndToken(t *testing.T, db *sql.DB, g viewGraph) string {
	t.Helper()
	svc, _, _ := releaseServiceFor(db)
	out, err := svc.ReleaseResult(g.tid, g.studentID, g.assignmentID, 1)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	i := strings.LastIndex(out.Link, "/")
	if i < 0 {
		t.Fatalf("link carries no token: %q", out.Link)
	}
	return out.Link[i+1:]
}

// A released result is visible to its owner through the token.
func TestResultViewByToken(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupView(t, db, "Token Owner", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	token := releaseAndToken(t, db, g)
	svc, _ := viewServiceFor(db)

	res, err := svc.GetByToken(token)
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if res.StudentName != "Token Owner" {
		t.Fatalf("student_name=%q", res.StudentName)
	}
	if res.MarksEarned != 12 || res.MaxMarks != 40 || res.Percentage != 30 {
		t.Fatalf("score mismatch: %+v", res)
	}
	if res.Correct != 6 || res.Wrong != 4 {
		t.Fatalf("counts mismatch: correct=%d wrong=%d", res.Correct, res.Wrong)
	}
	if res.ReleasedAt == nil {
		t.Fatal("released_at missing")
	}
}

// The payload must not carry coach-facing diagnostics. A student who could read
// sqi_score could reverse-engineer the scoring model between sittings.
func TestResultViewPayloadOmitsCoachingSignals(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupView(t, db, "Payload Check", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	token := releaseAndToken(t, db, g)
	svc, _ := viewServiceFor(db)
	res, err := svc.GetByToken(token)
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}

	blob, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, banned := range []string{"sqi", "SQI", "overall", "rank", "percentile", "grade", "risk", "dimension"} {
		if strings.Contains(string(blob), banned) {
			t.Errorf("payload leaked %q: %s", banned, blob)
		}
	}
}

// Unknown, expired, empty, and revoked tokens are indistinguishable: one error,
// so the endpoint cannot be used to probe which is which.
func TestResultViewTokenFailuresAreIndistinguishable(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupView(t, db, "Expiry", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	svc, tr := viewServiceFor(db)
	token := releaseAndToken(t, db, g)

	// Valid before expiry.
	if _, err := svc.GetByToken(token); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}

	unknown, err := tr.Mint(g.attemptID, time.Hour)
	if err != nil {
		t.Fatalf("mint unknown: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM result_tokens WHERE token_hash = $1`, hashOf(unknown)); err != nil {
		t.Fatalf("delete unknown token row: %v", err)
	}

	expired, err := tr.Mint(g.attemptID, -time.Minute)
	if err != nil {
		t.Fatalf("mint expired: %v", err)
	}

cases := map[string]string{
		"empty":   "",
		"garbage": "not-a-real-token",
		"unknown": unknown,
		"expired": expired,
	}
	for name, tok := range cases {
		if _, err := svc.GetByToken(tok); err != ErrResultNotFound {
			t.Errorf("%s token: err=%v, want ErrResultNotFound", name, err)
		}
	}

	// Revocation is checked separately, after the row is actually deleted.

	if err := tr.DeleteForAttempt(g.attemptID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := svc.GetByToken(token); err != ErrResultNotFound {
		t.Errorf("revoked token: err=%v, want ErrResultNotFound", err)
	}
}

// A token resolves only to its own result, never another student's.
func TestResultViewTokenIsScopedToOneResult(t *testing.T) {
	db := testutil.OpenTestDB(t)
	mine := setupView(t, db, "Mine", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, mine.tid)

	other := setupView(t, db, "Theirs", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, other.tid)

	mineToken := releaseAndToken(t, db, mine)
	otherToken := releaseAndToken(t, db, other)

	svc, _ := viewServiceFor(db)

	res, err := svc.GetByToken(mineToken)
	if err != nil {
		t.Fatalf("my token: %v", err)
	}
	if res.StudentName != "Mine" {
		t.Fatalf("my token returned %q", res.StudentName)
	}
	if res.StudentName == "Theirs" {
		t.Fatal("SECURITY: my token returned another student's result")
	}

	// And the other token must not be interchangeable.
	otherRes, err := svc.GetByToken(otherToken)
	if err != nil {
		t.Fatalf("their token: %v", err)
	}
	if otherRes.StudentName != "Theirs" {
		t.Fatalf("their token returned %q", otherRes.StudentName)
	}
}

// The authenticated list returns only the caller's own released results.
func TestResultViewListIsScopedToStudent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	mine := setupView(t, db, "Mine", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, mine.tid)
	other := setupView(t, db, "Theirs", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, other.tid)

	releaseAndToken(t, db, mine)
	releaseAndToken(t, db, other)

	svc, _ := viewServiceFor(db)
	results, err := svc.ListForStudent(mine.studentID, mine.tid)
	if err != nil {
		t.Fatalf("ListForStudent: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("%d results returned, want only the caller's own", len(results))
	}
	if results[0].StudentName != "Mine" {
		t.Fatalf("returned %q", results[0].StudentName)
	}
	for _, r := range results {
		if r.StudentName == "Theirs" {
			t.Fatal("SECURITY: another student's result appeared in the list")
		}
	}
}

// A submitted-but-unreleased result must be invisible on the authenticated path,
// so a student never sees a score their coach has not published.
func TestResultViewListExcludesUnreleased(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupView(t, db, "Withheld", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	svc, _ := viewServiceFor(db)
	results, err := svc.ListForStudent(g.studentID, g.tid)
	if err != nil {
		t.Fatalf("ListForStudent: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("an unreleased result was listed: %+v", results)
	}
}

// A token must not resolve after its release is withdrawn. The token alone is
// not sufficient; the row must still be released.
func TestResultViewTokenRequiresRelease(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupView(t, db, "Withdrawn", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	token := releaseAndToken(t, db, g)
	svc, _ := viewServiceFor(db)

	if _, err := svc.GetByToken(token); err != nil {
		t.Fatalf("released result should be visible: %v", err)
	}

	// Withdraw the release, as though a coach had rolled it back.
	if _, err := db.Exec(`UPDATE attempt_results SET released_at = NULL WHERE attempt_id = $1`, g.attemptID); err != nil {
		t.Fatalf("withdraw release: %v", err)
	}
	if _, err := svc.GetByToken(token); err != ErrResultNotFound {
		t.Fatalf("err=%v, want ErrResultNotFound once the release is withdrawn", err)
	}
}

// A released result appears in the owner's list.
func TestResultViewListIncludesReleased(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupView(t, db, "Listed", false)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	releaseAndToken(t, db, g)
	svc, _ := viewServiceFor(db)

	results, err := svc.ListForStudent(g.studentID, g.tid)
	if err != nil {
		t.Fatalf("ListForStudent: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("%d results, want 1", len(results))
	}
	if results[0].TestTitle == "" {
		t.Error("test title missing from the list payload")
	}
}

// Retests: both sittings are independently releasable and both are listed.
func TestResultViewListsRetestsSeparately(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tid)
	subjectID := testutil.CreateSubject(t, db, tid, "Retest View")
	studentID := testutil.CreateStudent(t, db, tid, coachID, testutil.UniqueCode(t, "rv"), "Retester")
	testID := testutil.CreateTest(t, db, tid, subjectID, coachID, 30)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	summary := []byte(`{"exam_summary":{"correct":3,"max_possible_score":40,"score_percent":30}}`)
	for i := range 2 {
		aID := testutil.CreateAssignment(t, db, studentID, testID, coachID)
		attID := testutil.CreateAttempt(t, db, aID)
		if _, err := db.Exec(
			`INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json)
			 VALUES ($1, 40, 12, $2)`, attID, summary); err != nil {
			t.Fatalf("insert result %d: %v", i, err)
		}
		svc, _, _ := releaseServiceFor(db)
		if _, err := svc.ReleaseResult(tid, studentID, aID, 1); err != nil {
			t.Fatalf("release sitting %d: %v", i, err)
		}
	}

	vs, _ := viewServiceFor(db)
	results, err := vs.ListForStudent(studentID, tid)
	if err != nil {
		t.Fatalf("ListForStudent: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("%d results, want one per released sitting", len(results))
	}
	// Both point at the same test but are distinct attempts.
	if results[0].AttemptID == results[1].AttemptID {
		t.Fatal("both sittings resolved to the same attempt")
	}
}