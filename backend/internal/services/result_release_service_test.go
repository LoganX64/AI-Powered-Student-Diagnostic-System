package services

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"
)

// releaseGraph builds a tenant with one submitted attempt and an optional
// result row, matching the two states release has to distinguish.
type releaseGraph struct {
	tid, assignmentID, attemptID, studentID int
}

func setupRelease(t *testing.T, db *sql.DB, withResult bool, email string, score float64) releaseGraph {
	t.Helper()
	tid := testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tid)
	subjectID := testutil.CreateSubject(t, db, tid, "Release Subject")
	studentID := testutil.CreateStudent(t, db, tid, coachID, testutil.UniqueCode(t, "rel"), "Release Student")
	testID := testutil.CreateTest(t, db, tid, subjectID, coachID, 30)
	assignmentID := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	attemptID := testutil.CreateAttempt(t, db, assignmentID)

	if email != "" {
		if _, err := db.Exec(`UPDATE students SET email = $1 WHERE id = $2`, email, studentID); err != nil {
			t.Fatalf("set student email: %v", err)
		}
	}
	if withResult {
		var summary = []byte(`{"exam_summary":{"correct":3,"wrong":7,"skipped":0,"total_questions":10,` +
			`"total_marks_earned":12,"max_possible_score":40,"net_score":5,"score_percent":12.5}}`)
		if _, err := db.Exec(
			`INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json) VALUES ($1, $2, $3, $4)`,
			attemptID, score, score, summary); err != nil {
			t.Fatalf("insert result: %v", err)
		}
	}

	return releaseGraph{tid, assignmentID, attemptID, studentID}
}

func releaseServiceFor(db *sql.DB) (*ResultReleaseService, *repository.AttemptRepo, *repository.ResultTokenRepo) {
	ar := repository.NewAttemptRepo(db)
	tr := repository.NewResultTokenRepo(db)
	// SMTP unconfigured, so senders log and return nil without dialling.
	m := NewMailer("smtp.example", 587, "", "", "from@example.com", "https://app.example")
	return NewResultReleaseService(ar, tr, m), ar, tr
}

// The happy path: a computed result is released, marked, and given a token.
func TestReleaseResultSuccess(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupRelease(t, db, true, "student@example.com", 5)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	svc, ar, tr := releaseServiceFor(db)
	out, err := svc.ReleaseResult(g.assignmentID, 1)
	if err != nil {
		t.Fatalf("ReleaseResult: %v", err)
	}
	if out.AlreadyRel {
		t.Fatal("first release reported already_released")
	}
	if !out.TokenMinted {
		t.Fatal("no token minted")
	}
	if out.AttemptID != g.attemptID {
		t.Fatalf("attempt_id=%d, want %d", out.AttemptID, g.attemptID)
	}
	if out.Link == "" {
		t.Fatal("no link returned to the coach")
	}

	var releasedAt sql.NullTime
	var releasedBy sql.NullInt64
	if err := db.QueryRow(`SELECT released_at, released_by FROM attempt_results WHERE attempt_id = $1`, g.attemptID).
		Scan(&releasedAt, &releasedBy); err != nil {
		t.Fatalf("read release state: %v", err)
	}
	if !releasedAt.Valid {
		t.Fatal("released_at was not set")
	}
	if releasedBy.Valid && int(releasedBy.Int64) != 1 {
		t.Fatalf("released_by=%d, want 1", releasedBy.Int64)
	}

	// A token must exist for the attempt, and the returned link must carry one.
	var tokens int
	if err := db.QueryRow(`SELECT COUNT(*) FROM result_tokens WHERE attempt_id = $1`, g.attemptID).Scan(&tokens); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if tokens != 1 {
		t.Fatalf("%d tokens minted, want 1", tokens)
	}
	_ = ar
	_ = tr
}

// Releasing an uncomputed result must be refused, so "compute SQI first" is
// enforced by the system rather than left to the coach to remember.
func TestReleaseRefusesUncomputedResult(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupRelease(t, db, false, "student@example.com", 0)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	svc, _, _ := releaseServiceFor(db)
	if _, err := svc.ReleaseResult(g.assignmentID, 1); err != ErrResultNotComputed {
		t.Fatalf("err=%v, want ErrResultNotComputed", err)
	}

	var tokens int
	if err := db.QueryRow(`SELECT COUNT(*) FROM result_tokens WHERE attempt_id = $1`, g.attemptID).Scan(&tokens); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if tokens != 0 {
		t.Fatal("a token was minted for an uncomputed result")
	}
}

// A genuinely zero score is a real result and must be releasable. This is why
// the service checks for the presence of the row rather than for a zero score.
func TestReleaseAllowsGenuineZeroScore(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupRelease(t, db, true, "student@example.com", 0)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	svc, _, _ := releaseServiceFor(db)
	out, err := svc.ReleaseResult(g.assignmentID, 1)
	if err != nil {
		t.Fatalf("a zero score must be releasable, got %v", err)
	}
	if out.AlreadyRel || !out.TokenMinted {
		t.Fatalf("unexpected outcome: %+v", out)
	}
}

// Re-releasing is a no-op, so a double-click cannot mint a second token and
// invalidate the link the student already holds.
func TestReleaseIsIdempotent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupRelease(t, db, true, "student@example.com", 5)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	svc, _, _ := releaseServiceFor(db)
	if _, err := svc.ReleaseResult(g.assignmentID, 1); err != nil {
		t.Fatalf("first release: %v", err)
	}

	out, err := svc.ReleaseResult(g.assignmentID, 2)
	if err != nil {
		t.Fatalf("second release: %v", err)
	}
	if !out.AlreadyRel {
		t.Fatal("second release was not reported as already released")
	}
	// The second call must not hand back a different link.
	if out.Link != "" {
		t.Fatalf("second release minted a new link: %q", out.Link)
	}

	var tokens int
	if err := db.QueryRow(`SELECT COUNT(*) FROM result_tokens WHERE attempt_id = $1`, g.attemptID).Scan(&tokens); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if tokens != 1 {
		t.Fatalf("%d tokens after two releases, want 1", tokens)
	}
}

// A student with no email on file still gets released; the link is returned to
// the coach instead. The release must not be blocked by a missing address.
func TestReleaseSucceedsWithoutEmail(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupRelease(t, db, true, "", 5)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	svc, _, _ := releaseServiceFor(db)
	out, err := svc.ReleaseResult(g.assignmentID, 1)
	if err != nil {
		t.Fatalf("release without an email must still succeed, got %v", err)
	}
	if out.EmailSent {
		t.Fatal("reported an email as sent when the student has no address")
	}
	if out.Link == "" {
		t.Fatal("no link returned for manual delivery")
	}

	var releasedAt sql.NullTime
	if err := db.QueryRow(`SELECT released_at FROM attempt_results WHERE attempt_id = $1`, g.attemptID).Scan(&releasedAt); err != nil {
		t.Fatalf("read released_at: %v", err)
	}
	if !releasedAt.Valid {
		t.Fatal("release was not recorded")
	}
}

// An assignment with no submitted attempt has nothing to release.
func TestReleaseRefusesWhenNoAttempt(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tid)
	subjectID := testutil.CreateSubject(t, db, tid, "No Attempt")
	studentID := testutil.CreateStudent(t, db, tid, coachID, testutil.UniqueCode(t, "na"), "No Attempt Student")
	testID := testutil.CreateTest(t, db, tid, subjectID, coachID, 30)
	assignmentID := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	svc, _, _ := releaseServiceFor(db)
	if _, err := svc.ReleaseResult(assignmentID, 1); err != ErrResultNotComputed {
		t.Fatalf("err=%v, want ErrResultNotComputed", err)
	}
}

// Two sittings of the same test are separate assignments and must release
// independently, each with its own token.
func TestReleaseIsIndependentPerAssignment(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tid)
	subjectID := testutil.CreateSubject(t, db, tid, "Retest Subject")
	studentID := testutil.CreateStudent(t, db, tid, coachID, testutil.UniqueCode(t, "rt"), "Retest Student")
	testID := testutil.CreateTest(t, db, tid, subjectID, coachID, 30)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	first := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	firstAttempt := testutil.CreateAttempt(t, db, first)
	second := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	secondAttempt := testutil.CreateAttempt(t, db, second)
	for _, a := range []int{firstAttempt, secondAttempt} {
		if _, err := db.Exec(
			`INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json)
			 VALUES ($1, $2, $2, '{}'::jsonb)`, a, 5); err != nil {
			t.Fatalf("insert result: %v", err)
		}
	}

	svc, _, _ := releaseServiceFor(db)
	if _, err := svc.ReleaseResult(first, 1); err != nil {
		t.Fatalf("release first: %v", err)
	}
	// The first release must not have released the retest.
	out, err := svc.ReleaseResult(second, 1)
	if err != nil {
		t.Fatalf("release second: %v", err)
	}
	if out.AlreadyRel {
		t.Fatal("releasing the second assignment reported the first as already released")
	}
	if out.AttemptID != secondAttempt {
		t.Fatalf("released attempt %d, want the retest's %d", out.AttemptID, secondAttempt)
	}

	var tokens int
	if err := db.QueryRow(`SELECT COUNT(*) FROM result_tokens`).Scan(&tokens); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if tokens != 2 {
		t.Fatalf("%d tokens, want one per released attempt", tokens)
	}
}

// The release must not depend on the SQI job's own concurrency: a result whose
// summary is missing still releases, using zero for the derived fields rather
// than failing.
func TestReleaseToleratesSummaryWithoutScores(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tid)
	subjectID := testutil.CreateSubject(t, db, tid, "Sparse Subject")
	studentID := testutil.CreateStudent(t, db, tid, coachID, testutil.UniqueCode(t, "sp"), "Sparse Student")
	testID := testutil.CreateTest(t, db, tid, subjectID, coachID, 30)
	assignmentID := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	attemptID := testutil.CreateAttempt(t, db, assignmentID)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	if _, err := db.Exec(
		`INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json)
		 VALUES ($1, 40, 7, '{"exam_summary":{"correct":2}}')`, attemptID); err != nil {
		t.Fatalf("insert result: %v", err)
	}

	svc, ar, _ := releaseServiceFor(db)
	if _, err := svc.ReleaseResult(assignmentID, 1); err != nil {
		t.Fatalf("release with a sparse summary: %v", err)
	}
	res, err := ar.GetReleasableResult(attemptID, assignmentID)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if res.RawScore != 7 {
		t.Fatalf("raw_score=%v, want 7", res.RawScore)
	}
	if res.Percent != 0 || res.MaxScore != 0 {
		t.Fatalf("absent summary fields should fall back to zero, got max=%v percent=%v", res.MaxScore, res.Percent)
	}
}

// The minted token must resolve back to the attempt that was released.
func TestReleasedTokenResolves(t *testing.T) {
	db := testutil.OpenTestDB(t)
	g := setupRelease(t, db, true, "student@example.com", 5)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, g.tid)

	svc, _, tr := releaseServiceFor(db)
	out, err := svc.ReleaseResult(g.assignmentID, 1)
	if err != nil {
		t.Fatalf("ReleaseResult: %v", err)
	}

	// Recover the raw token from the minted link, the way the student would.
	token := out.Link[strings.LastIndex(out.Link, "/")+1:]
	if token == "" {
		t.Fatalf("link carries no token: %q", out.Link)
	}
	attemptID, err := tr.Resolve(token)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if attemptID != g.attemptID {
		t.Fatalf("token resolved to attempt %d, want %d", attemptID, g.attemptID)
	}

	// And the token must expire on the documented window, not sooner.
	var expires time.Time
	if err := db.QueryRow(`SELECT expires_at FROM result_tokens WHERE attempt_id = $1`, g.attemptID).Scan(&expires); err != nil {
		t.Fatalf("read expires_at: %v", err)
	}
	days := time.Until(expires).Hours() / 24
	if days < 14.5 || days > 15.5 {
		t.Fatalf("token lifetime is %.2f days, want ~15", days)
	}
}