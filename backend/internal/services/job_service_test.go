package services

import (
	"database/sql"
	"encoding/json"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"
)

// --- Pure helpers extracted in Phase 6 Step 1 ---

func TestTerminalStatus(t *testing.T) {
	cases := []struct {
		done, failed int
		want         string
	}{
		{0, 0, "completed"},   // nothing failed
		{5, 0, "completed"},   // all succeeded
		{0, 3, "failed"},      // none succeeded
		{2, 3, "partial"},     // mixed
		{1, 1, "partial"},     // one of each
	}
	for _, c := range cases {
		if got := terminalStatus(c.done, c.failed); got != c.want {
			t.Errorf("terminalStatus(%d,%d)=%q, want %q", c.done, c.failed, got, c.want)
		}
	}
}

func TestChunkSizeDefaultsTo100(t *testing.T) {
	if got := (&JobService{}).chunkSize(); got != 100 {
		t.Fatalf("nil-config chunkSize=%d, want 100", got)
	}
	if got := (&JobService{ChunkSize: 0}).chunkSize(); got != 100 {
		t.Fatalf("zero chunkSize=%d, want 100", got)
	}
	if got := (&JobService{ChunkSize: -5}).chunkSize(); got != 100 {
		t.Fatalf("negative chunkSize=%d, want 100", got)
	}
	if got := (&JobService{ChunkSize: 25}).chunkSize(); got != 25 {
		t.Fatalf("explicit chunkSize=%d, want 25", got)
	}
}

// --- DB-backed Process ---

func jobGraph(t *testing.T, db *sql.DB) (tenantID int, attemptIDs []int) {
	t.Helper()
	tenantID = testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tenantID)
	subjectID := testutil.CreateSubject(t, db, tenantID, "Job")
	testID := testutil.CreateTest(t, db, tenantID, subjectID, coachID, 60)
	q := testutil.CreateQuestionWithAnswer(t, db, testID, "A")

	// One student, one submitted attempt with an answer log, so ComputeAndStore
	// has something real to score.
	studentID := testutil.CreateStudent(t, db, tenantID, coachID, testutil.UniqueCode(t, "job"), "Job Student")
	assignmentID := testutil.CreateAssignment(t, db, studentID, testID, coachID)
	seen := true
	res, err := repository.NewAttemptRepo(db).SubmitAnswersTx(assignmentID, map[int]string{q: "A"},
		[]repository.AnswerInput{{QuestionID: q, SelectedAnswer: "A", TimeSpent: 5, Seen: &seen}},
		func(tx *sql.Tx) error { return nil })
	if err != nil {
		t.Fatalf("submit answers: %v", err)
	}
	return tenantID, []int{res.AttemptID}
}

// TestComputeAndStoreAttemptDirectly surfaces the underlying error that
// JobService.Process swallows into a failure counter.
func TestComputeAndStoreAttemptDirectly(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID, attemptIDs := jobGraph(t, db)
	_ = tenantID

	attemptService := NewAttemptService(repository.NewAttemptRepo(db), repository.NewAssignmentRepo(db),
		repository.NewStudentRepo(db), repository.NewTestPaperRepo(db))

	testID, err := attemptService.TestIDForAttempt(attemptIDs[0])
	if err != nil {
		t.Fatalf("TestIDForAttempt: %v", err)
	}
	if err := attemptService.ComputeAndStoreAttempt(attemptIDs[0], testID); err != nil {
		t.Fatalf("ComputeAndStoreAttempt: %v", err)
	}
}

// TestJobProcessComputesAndCompletes is the happy path: a job with one valid
// attempt must score it and land in a terminal completed status.
func TestJobProcessComputesAndCompletes(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID, attemptIDs := jobGraph(t, db)

	jobRepo := repository.NewJobRepo(db)
	payload, _ := json.Marshal(computeJobPayload{AttemptIDs: attemptIDs})
	jobID, err := jobRepo.Create(tenantID, "compute_sqi", payload, len(attemptIDs))
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	attemptService := NewAttemptService(repository.NewAttemptRepo(db), repository.NewAssignmentRepo(db),
		repository.NewStudentRepo(db), repository.NewTestPaperRepo(db))
	js := NewJobService(jobRepo, attemptService, 100, nil)

	if err := js.Process(jobID, tenantID); err != nil {
		t.Fatalf("Process: %v", err)
	}
	job, err := jobRepo.Get(jobID, tenantID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if job.Status != "completed" {
		t.Fatalf("status=%q done=%d failed=%d, want completed", job.Status, job.Done, job.Failed)
	}
	if job.Done != 1 || job.Failed != 0 {
		t.Fatalf("done=%d failed=%d, want 1/0", job.Done, job.Failed)
	}

	// The attempt must now carry a stored result.
	score, _, err := repository.NewAttemptRepo(db).GetSQIResult(attemptIDs[0])
	if err != nil {
		t.Fatalf("GetSQIResult: %v", err)
	}
	if !score.Valid {
		t.Fatal("attempt result was not stored")
	}
}

// TestJobProcessBadPayloadIsTerminalFailed: a payload that is valid jsonb but
// does not fit computeJobPayload is acked as terminal (nil error) rather than
// replayed forever. Note the column is jsonb, so a raw non-JSON payload cannot
// even be stored — the malformed case that actually reaches Process is a
// well-formed value of the wrong shape.
func TestJobProcessBadPayloadIsTerminalFailed(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)

	jobRepo := repository.NewJobRepo(db)
	// attempt_ids as a string, not an array -> unmarshal error.
	jobID, err := jobRepo.Create(tenantID, "compute_sqi", []byte(`{"attempt_ids":"nope"}`), 1)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	js := NewJobService(jobRepo, nil, 100, nil)
	if err := js.Process(jobID, tenantID); err != nil {
		t.Fatalf("Process with bad payload must be acked (nil), got %v", err)
	}
	job, _ := jobRepo.Get(jobID, tenantID)
	if job.Status != "failed" {
		t.Fatalf("status=%q, want failed", job.Status)
	}
}

// TestJobProcessEmptyPayloadCompletes: zero attempts is a no-op success.
func TestJobProcessEmptyPayloadCompletes(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)

	jobRepo := repository.NewJobRepo(db)
	payload, _ := json.Marshal(computeJobPayload{AttemptIDs: []int{}})
	jobID, _ := jobRepo.Create(tenantID, "compute_sqi", payload, 0)

	js := NewJobService(jobRepo, nil, 100, nil)
	if err := js.Process(jobID, tenantID); err != nil {
		t.Fatalf("Process: %v", err)
	}
	job, _ := jobRepo.Get(jobID, tenantID)
	if job.Status != "completed" {
		t.Fatalf("status=%q, want completed", job.Status)
	}
}

// TestJobProcessUnknownAttemptFails: a payload pointing at a nonexistent attempt
// counts as a failure (done=0), so the job lands in failed.
func TestJobProcessUnknownAttemptFails(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)

	jobRepo := repository.NewJobRepo(db)
	payload, _ := json.Marshal(computeJobPayload{AttemptIDs: []int{999999999}})
	jobID, _ := jobRepo.Create(tenantID, "compute_sqi", payload, 1)

	attemptService := NewAttemptService(repository.NewAttemptRepo(db), repository.NewAssignmentRepo(db),
		repository.NewStudentRepo(db), repository.NewTestPaperRepo(db))
	js := NewJobService(jobRepo, attemptService, 100, nil)
	if err := js.Process(jobID, tenantID); err != nil {
		t.Fatalf("Process must not error on per-attempt failure: %v", err)
	}
	job, _ := jobRepo.Get(jobID, tenantID)
	if job.Status != "failed" {
		t.Fatalf("status=%q done=%d failed=%d, want failed", job.Status, job.Done, job.Failed)
	}
}

// TestJobProcessAlreadyFinishedIsNoOp: a terminal job is acked without reprocessing.
func TestJobProcessAlreadyFinishedIsNoOp(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)

	jobRepo := repository.NewJobRepo(db)
	jobID, err := jobRepo.Create(tenantID, "compute_sqi", []byte(`{"attempt_ids":[]}`), 0)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := jobRepo.SetStatus(jobID, tenantID, "completed"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	js := NewJobService(jobRepo, nil, 100, nil)
	if err := js.Process(jobID, tenantID); err != nil {
		t.Fatalf("Process on finished job: %v", err)
	}
	job, err := jobRepo.Get(jobID, tenantID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	// Status must be untouched by a reprocess attempt.
	if job.Status != "completed" {
		t.Fatalf("status=%q, want completed", job.Status)
	}
}
