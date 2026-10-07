package services

import (
	"encoding/json"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"
)

// --- Pure helper extracted in Phase 6 Step 1 ---

func TestNormalizeAnswerUnseenZeroesEverything(t *testing.T) {
	seen := false
	in := AnswerInput{
		QuestionID:        5,
		SelectedAnswer:    "B",
		TimeSpent:         42,
		Seen:              &seen,
		MarkedForReview:   true,
		Revisited:         true,
		ChangedAnswer:     true,
		WasInitiallyWrong: true,
	}
	out, gotSeen := normalizeAnswer(in)
	if gotSeen {
		t.Fatal("explicit Seen=false must win over a non-empty selection")
	}
	if out.TimeSpent != 0 || out.SelectedAnswer != "" || out.MarkedForReview ||
		out.Revisited || out.ChangedAnswer || out.WasInitiallyWrong {
		t.Fatalf("unseen answer not zeroed: %+v", out)
	}
	if out.QuestionID != 5 {
		t.Fatalf("QuestionID must survive normalization: %+v", out)
	}
}

func TestNormalizeAnswerSeenKeepsData(t *testing.T) {
	seen := true
	out, gotSeen := normalizeAnswer(AnswerInput{SelectedAnswer: "C", TimeSpent: 10, Seen: &seen, Revisited: true})
	if !gotSeen {
		t.Fatal("explicit Seen=true must report seen")
	}
	if out.SelectedAnswer != "C" || out.TimeSpent != 10 || !out.Revisited {
		t.Fatalf("seen answer was altered: %+v", out)
	}
}

// TestNormalizeAnswerInferredSeen: without an explicit Seen pointer, a non-empty
// selection implies seen; an empty selection implies unseen.
func TestNormalizeAnswerInferredSeen(t *testing.T) {
	if _, seen := normalizeAnswer(AnswerInput{SelectedAnswer: "A"}); !seen {
		t.Fatal("non-empty selection must infer seen")
	}
	if _, seen := normalizeAnswer(AnswerInput{SelectedAnswer: ""}); seen {
		t.Fatal("empty selection must infer unseen")
	}
}

// TestAutosaveKeyFormat pins the Redis key layout the flush scan relies on.
func TestAutosaveKeyFormat(t *testing.T) {
	if got := autosaveKey(77); got != "autosave:77" {
		t.Fatalf("autosaveKey(77)=%q", got)
	}
}

// --- Redis-backed ---

// TestAutosaveFlushAttemptPersistsAnswer is the core autosave guarantee: a
// buffered answer must reach answer_logs, and an unseen one must arrive zeroed.
func TestAutosaveFlushAttemptPersistsAnswer(t *testing.T) {
	rdb := testutil.OpenRedisClient(t)
	db := testutil.OpenTestDB(t)

	tenantID := testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tenantID)
	subjectID := testutil.CreateSubject(t, db, tenantID, "Auto")
	testID := testutil.CreateTest(t, db, tenantID, subjectID, coachID, 60)
	q1 := testutil.CreateQuestionWithAnswer(t, db, testID, "A")
	studentID := testutil.CreateStudent(t, db, tenantID, coachID, testutil.UniqueCode(t, "autos"), "Auto Student")
	assignmentID := testutil.CreateAssignment(t, db, studentID, testID, coachID)

	attemptRepo := repository.NewAttemptRepo(db)
	attemptID, _, err := attemptRepo.CreateInProgressAttempt(assignmentID)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	b := NewAutosaveBuffer(rdb, attemptRepo)
	key := autosaveKey(attemptID)
	testutil.FlushTestKeys(t, rdb, key)

	// Buffer a seen answer and an unseen one directly as raw list items.
	seen, unseen := true, false
	seenItem, _ := json.Marshal(bufferedAnswer{AttemptID: attemptID, Answer: AnswerInput{QuestionID: q1, SelectedAnswer: "A", TimeSpent: 20, Seen: &seen}})
	unseenItem, _ := json.Marshal(bufferedAnswer{AttemptID: attemptID, Answer: AnswerInput{QuestionID: q1, SelectedAnswer: "", TimeSpent: 99, Seen: &unseen}})

	// Same question twice is not representative; use one seen and, for the
	// unseen case, a second question.
	q2 := testutil.CreateQuestionWithAnswer(t, db, testID, "B")
	unseenItem, _ = json.Marshal(bufferedAnswer{AttemptID: attemptID, Answer: AnswerInput{QuestionID: q2, SelectedAnswer: "", TimeSpent: 99, Seen: &unseen}})

	if err := rdb.RPush(t.Context(), key, string(seenItem), string(unseenItem)).Err(); err != nil {
		t.Fatalf("RPush: %v", err)
	}
	if err := b.FlushAttempt(attemptID); err != nil {
		t.Fatalf("FlushAttempt: %v", err)
	}

	details, err := attemptRepo.GetAnswerDetails(attemptID)
	if err != nil {
		t.Fatalf("GetAnswerDetails: %v", err)
	}
	if len(details) != 2 {
		t.Fatalf("details=%d, want 2: %+v", len(details), details)
	}
	seenOK, unseenOK := false, false
	for _, d := range details {
		if d.QuestionID == q1 {
			seenOK = d.SelectedAnswer == "A" && d.TimeSpent == 20
		}
		if d.QuestionID == q2 {
			// Unseen must have no selection and no time.
			unseenOK = d.SelectedAnswer == "" && d.TimeSpent == 0
		}
	}
	if !seenOK {
		t.Error("seen answer not persisted with its data")
	}
	if !unseenOK {
		t.Error("unseen answer must persist with selection and time zeroed")
	}

	// The list must be drained after a successful flush.
	if n, err := rdb.LLen(t.Context(), key).Result(); err != nil || n != 0 {
		t.Errorf("autosave list not drained: len=%d err=%v", n, err)
	}
}

// TestAutosaveNilRedisIsNoOp pins the documented nil-Redis behaviour the handler
// fixtures rely on.
func TestAutosaveNilRedisIsNoOp(t *testing.T) {
	b := NewAutosaveBuffer(nil, nil)
	if err := b.Push(1, []AnswerInput{{QuestionID: 1}}); err != nil {
		t.Fatalf("Push with nil redis must be a no-op, got %v", err)
	}
	if err := b.FlushAttempt(1); err != nil {
		t.Fatalf("FlushAttempt with nil redis must be a no-op, got %v", err)
	}
}

// TestAutosavePushBuffersThenFlushes covers Push -> FlushAttempt round trip.
func TestAutosavePushBuffersThenFlushes(t *testing.T) {
	rdb := testutil.OpenRedisClient(t)
	db := testutil.OpenTestDB(t)

	tenantID := testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tenantID)
	subjectID := testutil.CreateSubject(t, db, tenantID, "Push")
	testID := testutil.CreateTest(t, db, tenantID, subjectID, coachID, 60)
	q1 := testutil.CreateQuestionWithAnswer(t, db, testID, "D")
	studentID := testutil.CreateStudent(t, db, tenantID, coachID, testutil.UniqueCode(t, "push"), "Push Student")
	assignmentID := testutil.CreateAssignment(t, db, studentID, testID, coachID)

	attemptRepo := repository.NewAttemptRepo(db)
	attemptID, _, _ := attemptRepo.CreateInProgressAttempt(assignmentID)

	b := NewAutosaveBuffer(rdb, attemptRepo)
	key := autosaveKey(attemptID)
	testutil.FlushTestKeys(t, rdb, key)

	seen := true
	if err := b.Push(attemptID, []AnswerInput{{QuestionID: q1, SelectedAnswer: "D", TimeSpent: 7, Seen: &seen}}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if n, _ := rdb.LLen(t.Context(), key).Result(); n != 1 {
		t.Fatalf("buffer length=%d, want 1", n)
	}
	if err := b.FlushAttempt(attemptID); err != nil {
		t.Fatalf("FlushAttempt: %v", err)
	}
	details, _ := attemptRepo.GetAnswerDetails(attemptID)
	if len(details) != 1 || details[0].SelectedAnswer != "D" || details[0].TimeSpent != 7 {
		t.Fatalf("details=%+v", details)
	}
}
