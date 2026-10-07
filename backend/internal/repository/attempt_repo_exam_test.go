package repository

import (
	"database/sql"
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
