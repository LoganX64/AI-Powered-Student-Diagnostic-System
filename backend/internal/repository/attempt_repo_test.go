package repository

import (
	"database/sql"
	"errors"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

func TestSubmitAnswersTxInsertsAnswersAndComputesCorrectness(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	tenantID, _, _, testID, assignmentID := attemptFixture(t, db)
	q1 := questionFixture(t, db, testID)

	correctMap, err := r.GetCorrectAnswers(testID)
	if err != nil {
		t.Fatalf("GetCorrectAnswers: %v", err)
	}
	if correctMap[q1] == "" {
		t.Fatal("correctMap missing fixture question")
	}

	seen := true
	res, err := r.SubmitAnswersTx(assignmentID, correctMap, []AnswerInput{
		{QuestionID: q1, SelectedAnswer: "A", TimeSpent: 10, Seen: &seen},
	}, func(tx *sql.Tx) error { return nil })
	if err != nil {
		t.Fatalf("SubmitAnswersTx: %v", err)
	}
	if res.AttemptID == 0 || res.TotalTimeSpent != 10 {
		t.Fatalf("res=%+v", res)
	}

	if locked, _ := r.HasSubmittedAttempt(assignmentID); !locked {
		t.Fatal("HasSubmittedAttempt must be true after submit")
	}
	answers, err := r.GetAnswerDetails(res.AttemptID)
	if err != nil {
		t.Fatalf("GetAnswerDetails: %v", err)
	}
	if len(answers) != 1 || answers[0].SelectedAnswer != "A" || !answers[0].IsCorrect {
		t.Fatalf("answers=%+v", answers)
	}
	_ = tenantID
}

func TestSubmitAnswersTxRejectsUnknownQuestionID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, testID, assignmentID := attemptFixture(t, db)
	questionFixture(t, db, testID)

	seen := true
	_, err := r.SubmitAnswersTx(assignmentID, map[int]string{1: "A"}, []AnswerInput{
		{QuestionID: 999999, SelectedAnswer: "A", Seen: &seen},
	}, func(tx *sql.Tx) error { return nil })
	if err == nil {
		t.Fatal("unknown question id must be rejected")
	}
}

func TestSubmitAnswersTxRejectsDuplicateQuestionID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, testID, assignmentID := attemptFixture(t, db)
	q1 := questionFixture(t, db, testID)

	seen := true
	_, err := r.SubmitAnswersTx(assignmentID, map[int]string{q1: "A"}, []AnswerInput{
		{QuestionID: q1, SelectedAnswer: "A", Seen: &seen},
		{QuestionID: q1, SelectedAnswer: "B", Seen: &seen},
	}, func(tx *sql.Tx) error { return nil })
	if err == nil {
		t.Fatal("duplicate question id must be rejected")
	}
}

func TestStoreResultAndGetAverageSQI(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, studentID, testID, assignmentID := attemptFixture(t, db)
	questionFixture(t, db, testID)
	res, err := r.SubmitAnswersTx(assignmentID, map[int]string{}, []AnswerInput{}, func(tx *sql.Tx) error { return nil })
	if err != nil {
		t.Fatalf("SubmitAnswersTx: %v", err)
	}
	if err := r.StoreResult(res.AttemptID, 80.0, 40.0, []byte(`{"ok":true}`), "v2"); err != nil {
		t.Fatalf("StoreResult: %v", err)
	}

	score, analysis, err := r.GetSQIResult(res.AttemptID)
	if err != nil {
		t.Fatalf("GetSQIResult: %v", err)
	}
	if !score.Valid || score.Float64 != 80.0 {
		t.Fatalf("score=%v", score)
	}
	if len(analysis) == 0 {
		t.Fatal("analysis empty")
	}

	avg, err := r.GetAverageSQI(studentID)
	if err != nil {
		t.Fatalf("GetAverageSQI: %v", err)
	}
	if avg != 80.0 {
		t.Fatalf("avg=%v want 80", avg)
	}
}

func TestGetUncomputedAttempts(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, studentID, testID, assignmentID := attemptFixture(t, db)
	questionFixture(t, db, testID)
	res, err := r.SubmitAnswersTx(assignmentID, map[int]string{}, []AnswerInput{}, func(tx *sql.Tx) error { return nil })
	if err != nil {
		t.Fatalf("SubmitAnswersTx: %v", err)
	}

	uncomputed, err := r.GetUncomputedAttempts(studentID)
	if err != nil {
		t.Fatalf("GetUncomputedAttempts: %v", err)
	}
	if len(uncomputed) != 1 || uncomputed[0][0] != res.AttemptID || uncomputed[0][1] != testID {
		t.Fatalf("uncomputed=%v", uncomputed)
	}

	if err := r.StoreResult(res.AttemptID, 50, 20, []byte(`{}`), "v2"); err != nil {
		t.Fatalf("StoreResult: %v", err)
	}
	uncomputed, err = r.GetUncomputedAttempts(studentID)
	if err != nil {
		t.Fatalf("GetUncomputedAttempts after: %v", err)
	}
	if len(uncomputed) != 0 {
		t.Fatalf("uncomputed after StoreResult=%v", uncomputed)
	}
}

func TestGetByAssignment(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, _, assignmentID := attemptFixture(t, db)
	res, err := r.SubmitAnswersTx(assignmentID, map[int]string{}, []AnswerInput{}, func(tx *sql.Tx) error { return nil })
	if err != nil {
		t.Fatalf("SubmitAnswersTx: %v", err)
	}
	id, submittedAt, err := r.GetByAssignment(assignmentID)
	if err != nil {
		t.Fatalf("GetByAssignment: %v", err)
	}
	if id != res.AttemptID || !submittedAt.Valid {
		t.Fatalf("GetByAssignment=(%d,%v)", id, submittedAt)
	}
}

// markSubmitted mirrors what AttemptService does inside the submit transaction.
func markSubmitted(assignmentID int) func(*sql.Tx) error {
	return func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE assignments SET status = 'submitted' WHERE id = $1", assignmentID)
		return err
	}
}

func TestSubmitAnswersTxRejectsSecondSubmission(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	_, _, _, _, assignmentID := attemptFixture(t, db)

	if _, err := r.SubmitAnswersTx(assignmentID, map[int]string{}, []AnswerInput{}, markSubmitted(assignmentID)); err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if _, err := r.SubmitAnswersTx(assignmentID, map[int]string{}, []AnswerInput{}, markSubmitted(assignmentID)); !errors.Is(err, ErrRetakeNotAllowed) {
		t.Fatalf("second submit: err=%v, want ErrRetakeNotAllowed", err)
	}

	count, err := r.CountAttempts(assignmentID)
	if err != nil {
		t.Fatalf("CountAttempts: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d, want 1 (no duplicate attempt)", count)
	}
}

func TestRetakeLatestAttemptDrivesResultsAndAverage(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAttemptRepo(db)
	tenantID, _, studentID, _, assignmentID := attemptFixture(t, db)

	var grantedBy int
	if err := db.QueryRow("SELECT id FROM users WHERE tenant_id = $1 LIMIT 1", tenantID).Scan(&grantedBy); err != nil {
		t.Fatalf("pick granting user: %v", err)
	}

	first, err := r.SubmitAnswersTx(assignmentID, map[int]string{}, []AnswerInput{}, markSubmitted(assignmentID))
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if err := r.StoreResult(first.AttemptID, 40, 10, []byte(`{"attempt":1}`), "v2"); err != nil {
		t.Fatalf("store first result: %v", err)
	}

	if _, _, err := r.GrantRetake(assignmentID, tenantID, grantedBy, nil); err != nil {
		t.Fatalf("grant retake: %v", err)
	}
	second, err := r.SubmitAnswersTx(assignmentID, map[int]string{}, []AnswerInput{}, markSubmitted(assignmentID))
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}
	if second.AttemptID == first.AttemptID {
		t.Fatalf("second attempt reused id %d", first.AttemptID)
	}
	if err := r.StoreResult(second.AttemptID, 90, 30, []byte(`{"attempt":2}`), "v2"); err != nil {
		t.Fatalf("store second result: %v", err)
	}

	id, _, err := r.GetByAssignment(assignmentID)
	if err != nil {
		t.Fatalf("GetByAssignment: %v", err)
	}
	if id != second.AttemptID {
		t.Fatalf("latest attempt id=%d, want %d", id, second.AttemptID)
	}

	results, err := r.GetResults(studentID, false)
	if err != nil {
		t.Fatalf("GetResults: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results)=%d, want 1 (latest attempt only)", len(results))
	}
	if results[0].AttemptID != second.AttemptID || !results[0].SQI.Valid || results[0].SQI.Float64 != 90 {
		t.Fatalf("result=%+v", results[0])
	}

	avg, err := r.GetAverageSQI(studentID)
	if err != nil {
		t.Fatalf("GetAverageSQI: %v", err)
	}
	if avg != 90 {
		t.Fatalf("avg=%v, want 90 (latest attempt only)", avg)
	}
}
