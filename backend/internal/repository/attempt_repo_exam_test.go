package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"ai-student-diagnostic/backend/internal/testutil"
)

// attemptFixture builds the minimal graph attempt queries need:
// tenant → coach + user, subject, test, student, assignment.
func attemptFixture(t *testing.T, db *sql.DB) (tenantID, coachID, studentID, testID, assignmentID int) {
	t.Helper()
	tenantID = createTenant(t, db)
	_, coachID = createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Math")
	testID = createTest(t, db, tenantID, subjectID, coachID, 60)
	studentID = createStudent(t, db, tenantID, coachID, "fixture-code-1", "Fixture Student")
	assignmentID = createAssignment(t, db, studentID, testID, coachID)
	return
}

func TestCreateAndGetInProgressAttempt(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, _, assignmentID := attemptFixture(t, db)

	id, startedAt, err := r.CreateInProgressAttempt(assignmentID)
	if err != nil {
		t.Fatalf("CreateInProgressAttempt: %v", err)
	}
	if id == 0 || startedAt.IsZero() {
		t.Fatalf("id=%d startedAt=%v", id, startedAt)
	}

	gotID, gotStarted, err := r.GetInProgressAttempt(assignmentID)
	if err != nil {
		t.Fatalf("GetInProgressAttempt: %v", err)
	}
	if gotID != id {
		t.Fatalf("GetInProgressAttempt id=%d, want %d", gotID, id)
	}
	if gotStarted.Sub(startedAt).Abs() > time.Second {
		t.Fatalf("startedAt drift %v", gotStarted.Sub(startedAt))
	}
}

func TestGetInProgressAttemptNoneExists(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, _, assignmentID := attemptFixture(t, db)
	if _, _, err := r.GetInProgressAttempt(assignmentID); err != sql.ErrNoRows {
		t.Fatalf("got %v, want sql.ErrNoRows", err)
	}
}

func TestGetTestIDForAttempt(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, testID, assignmentID := attemptFixture(t, db)
	attemptID, _, err := r.CreateInProgressAttempt(assignmentID)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	got, err := r.GetTestIDForAttempt(attemptID)
	if err != nil {
		t.Fatalf("GetTestIDForAttempt: %v", err)
	}
	if got != testID {
		t.Fatalf("got %d, want %d", got, testID)
	}
}

func TestAttemptBelongsToTenant(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	tenantID, _, _, _, assignmentID := attemptFixture(t, db)
	attemptID, _, _ := r.CreateInProgressAttempt(assignmentID)

	ok, err := r.AttemptBelongsToTenant(attemptID, tenantID)
	if err != nil || !ok {
		t.Fatalf("same tenant: ok=%v err=%v", ok, err)
	}
	ok, err = r.AttemptBelongsToTenant(attemptID, tenantID+99999)
	if err != nil || ok {
		t.Fatalf("other tenant: ok=%v err=%v", ok, err)
	}
}

func TestUpsertAnswerRoundtrip(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, testID, assignmentID := attemptFixture(t, db)
	q1 := questionFixture(t, db, testID)
	attemptID, _, _ := r.CreateInProgressAttempt(assignmentID)

	if err := r.UpsertAnswer(attemptID, q1, "A", true, 12.5, true, true, false, false, true); err != nil {
		t.Fatalf("UpsertAnswer insert: %v", err)
	}
	// Second upsert with different values must not error (ON CONFLICT update).
	if err := r.UpsertAnswer(attemptID, q1, "B", false, 20, false, false, true, true, true); err != nil {
		t.Fatalf("UpsertAnswer update: %v", err)
	}

	saved, err := r.GetSavedAnswers(attemptID)
	if err != nil {
		t.Fatalf("GetSavedAnswers: %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("len(saved)=%d, want 1", len(saved))
	}
	if saved[0].SelectedAnswer != "B" || saved[0].TimeSpent != 20 || !saved[0].ChangedAnswer {
		t.Fatalf("saved=%+v", saved[0])
	}
}

func TestFinalizeAttemptTxFlipsStatus(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, studentID, _, assignmentID := attemptFixture(t, db)
	attemptID, _, _ := r.CreateInProgressAttempt(assignmentID)

	if err := r.FinalizeAttemptTx(assignmentID, attemptID, studentID); err != nil {
		t.Fatalf("FinalizeAttemptTx: %v", err)
	}
	// Assignment is submitted → no more in-progress attempt.
	if _, _, err := r.GetInProgressAttempt(assignmentID); err != sql.ErrNoRows {
		t.Fatalf("GetInProgressAttempt after finalize: %v", err)
	}
}

func TestAttemptIDsByTest(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	tenantID, _, _, testID, assignmentID := attemptFixture(t, db)
	a1, _, _ := r.CreateInProgressAttempt(assignmentID)

	ids, err := r.AttemptIDsByTest(testID, tenantID)
	if err != nil {
		t.Fatalf("AttemptIDsByTest: %v", err)
	}
	if len(ids) != 1 || ids[0] != a1 {
		t.Fatalf("ids=%v want [%d]", ids, a1)
	}
	// Different tenant → empty.
	ids, _ = r.AttemptIDsByTest(testID, tenantID+99999)
	if len(ids) != 0 {
		t.Fatalf("other tenant ids=%v", ids)
	}
}

func TestExpiredInProgressAttempts(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, _, assignmentID := attemptFixture(t, db)
	attemptID, _, _ := r.CreateInProgressAttempt(assignmentID)

	// Duration 60, grace -3600 → anything started > 0s ago is expired.
	expired, err := r.ExpiredInProgressAttempts(-3600)
	if err != nil {
		t.Fatalf("ExpiredInProgressAttempts: %v", err)
	}
	found := false
	for _, e := range expired {
		if e.AttemptID == attemptID {
			found = true
		}
	}
	if !found {
		t.Fatalf("attempt %d not in expired %v", attemptID, expired)
	}
}

// submitFirstAttempt creates and finalizes attempt #1 so the assignment is in
// the 'submitted' state retakes are granted from.
func submitFirstAttempt(t *testing.T, r *AttemptRepo, studentID, assignmentID int) int {
	t.Helper()
	attemptID, _, err := r.CreateInProgressAttempt(assignmentID)
	if err != nil {
		t.Fatalf("create first attempt: %v", err)
	}
	if err := r.FinalizeAttemptTx(assignmentID, attemptID, studentID); err != nil {
		t.Fatalf("finalize first attempt: %v", err)
	}
	return attemptID
}

func TestCreateInProgressAttemptBlockedAfterSubmit(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, studentID, _, assignmentID := attemptFixture(t, db)
	submitFirstAttempt(t, r, studentID, assignmentID)

	if _, _, err := r.CreateInProgressAttempt(assignmentID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("CreateInProgressAttempt after submit: err=%v, want sql.ErrNoRows", err)
	}
	count, err := r.CountAttempts(assignmentID)
	if err != nil {
		t.Fatalf("CountAttempts: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d, want 1", count)
	}
}

func TestGrantRetakeAllowsAttemptsUpToCap(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	tenantID, _, studentID, _, assignmentID := attemptFixture(t, db)

	var grantedBy int
	if err := db.QueryRow("SELECT id FROM users WHERE tenant_id = $1 LIMIT 1", tenantID).Scan(&grantedBy); err != nil {
		t.Fatalf("pick granting user: %v", err)
	}

	firstID := submitFirstAttempt(t, r, studentID, assignmentID)

	used, max, err := r.GrantRetake(assignmentID, tenantID, grantedBy, nil)
	if err != nil {
		t.Fatalf("GrantRetake #1: %v", err)
	}
	if used != 1 || max != MaxAttemptsPerAssignment {
		t.Fatalf("GrantRetake #1 used=%d max=%d", used, max)
	}

	var status string
	var grantedByCol sql.NullInt64
	if err := db.QueryRow(
		"SELECT status, retake_granted_by FROM assignments WHERE id = $1", assignmentID,
	).Scan(&status, &grantedByCol); err != nil {
		t.Fatalf("read assignment: %v", err)
	}
	if status != "assigned" {
		t.Fatalf("status=%q, want assigned", status)
	}
	if !grantedByCol.Valid || int(grantedByCol.Int64) != grantedBy {
		t.Fatalf("retake_granted_by=%v, want %d", grantedByCol, grantedBy)
	}

	secondID, _, err := r.CreateInProgressAttempt(assignmentID)
	if err != nil {
		t.Fatalf("create second attempt: %v", err)
	}
	if secondID == firstID {
		t.Fatalf("second attempt reused id %d", firstID)
	}
	if err := r.FinalizeAttemptTx(assignmentID, secondID, studentID); err != nil {
		t.Fatalf("finalize second attempt: %v", err)
	}

	if used, _, err := r.GrantRetake(assignmentID, tenantID, grantedBy, nil); err != nil || used != 2 {
		t.Fatalf("GrantRetake #2 used=%d err=%v", used, err)
	}
	thirdID, _, err := r.CreateInProgressAttempt(assignmentID)
	if err != nil {
		t.Fatalf("create third attempt: %v", err)
	}
	if err := r.FinalizeAttemptTx(assignmentID, thirdID, studentID); err != nil {
		t.Fatalf("finalize third attempt: %v", err)
	}

	used, _, err = r.GrantRetake(assignmentID, tenantID, grantedBy, nil)
	if !errors.Is(err, ErrRetakeLimitReached) {
		t.Fatalf("GrantRetake #3 err=%v, want ErrRetakeLimitReached", err)
	}
	if used != MaxAttemptsPerAssignment {
		t.Fatalf("used=%d, want %d", used, MaxAttemptsPerAssignment)
	}

	count, err := r.CountAttempts(assignmentID)
	if err != nil {
		t.Fatalf("CountAttempts: %v", err)
	}
	if count != MaxAttemptsPerAssignment {
		t.Fatalf("count=%d, want %d", count, MaxAttemptsPerAssignment)
	}
}

func TestGrantRetakeRejectsUnsubmittedAndInProgress(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	tenantID, _, _, _, assignmentID := attemptFixture(t, db)

	var grantedBy int
	if err := db.QueryRow("SELECT id FROM users WHERE tenant_id = $1 LIMIT 1", tenantID).Scan(&grantedBy); err != nil {
		t.Fatalf("pick granting user: %v", err)
	}

	// Never attempted and still 'assigned' → no retake to grant.
	if _, _, err := r.GrantRetake(assignmentID, tenantID, grantedBy, nil); !errors.Is(err, ErrRetakeNotAllowed) {
		t.Fatalf("GrantRetake on assigned assignment: err=%v, want ErrRetakeNotAllowed", err)
	}

	if _, _, err := r.CreateInProgressAttempt(assignmentID); err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	if _, _, err := r.GrantRetake(assignmentID, tenantID, grantedBy, nil); !errors.Is(err, ErrRetakeNotAllowed) {
		t.Fatalf("GrantRetake with in-progress attempt: err=%v, want ErrRetakeNotAllowed", err)
	}
}

func TestGrantRetakeScopedToTenantAndCoach(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	tenantID, coachID, studentID, _, assignmentID := attemptFixture(t, db)
	var grantedBy int
	if err := db.QueryRow("SELECT id FROM users WHERE tenant_id = $1 LIMIT 1", tenantID).Scan(&grantedBy); err != nil {
		t.Fatalf("pick granting user: %v", err)
	}
	submitFirstAttempt(t, r, studentID, assignmentID)

	if _, _, err := r.GrantRetake(assignmentID, tenantID+99999, grantedBy, nil); !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf("foreign tenant err=%v, want ErrAssignmentNotFound", err)
	}

	_, otherCoachID := createCoach(t, db, tenantID)
	if _, _, err := r.GrantRetake(assignmentID, tenantID, grantedBy, &otherCoachID); !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf("foreign coach err=%v, want ErrAssignmentNotFound", err)
	}

	if _, _, err := r.GrantRetake(assignmentID, tenantID, grantedBy, &coachID); err != nil {
		t.Fatalf("owning coach: %v", err)
	}
}
