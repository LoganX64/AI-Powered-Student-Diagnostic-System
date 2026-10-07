package services

import (
	"context"
	"database/sql"
	"testing"

	"ai-student-diagnostic/backend/internal/queue"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"
)

// recordingQueue captures what the sweeper enqueues. queue.Queue is an
// interface, so this is a few lines and needs no Redis.
type recordingQueue struct {
	compute  []int
	finalize []queue.FinalizePayload
}

func (q *recordingQueue) EnqueueCompute(jobID, tenantID int) error {
	q.compute = append(q.compute, jobID)
	return nil
}
func (q *recordingQueue) EnqueueFinalize(p queue.FinalizePayload) error {
	q.finalize = append(q.finalize, p)
	return nil
}
func (q *recordingQueue) Start(computeHandler func(int, int) error, finalizeHandler func(queue.FinalizePayload) error) {
}
func (q *recordingQueue) Stop() {}

// sweeperGraph builds a tenant + test + student + assignment + in-progress
// attempt. durationMinutes lets a test control whether the attempt looks expired.
func sweeperGraph(t *testing.T, db *sql.DB, durationMinutes int) (tenantID, studentID, assignmentID, attemptID int) {
	t.Helper()
	tenantID = testutil.CreateTenant(t, db)
	_, coachID := testutil.CreateCoach(t, db, tenantID)
	subjectID := testutil.CreateSubject(t, db, tenantID, "Sweep")
	testID := testutil.CreateTest(t, db, tenantID, subjectID, coachID, durationMinutes)
	studentID = testutil.CreateStudent(t, db, tenantID, coachID, testutil.UniqueCode(t, "swp"), "Sweep Student")
	assignmentID = testutil.CreateAssignment(t, db, studentID, testID, coachID)
	attemptID, _, err := repository.NewAttemptRepo(db).CreateInProgressAttempt(assignmentID)
	if err != nil {
		t.Fatalf("create in-progress attempt: %v", err)
	}
	return tenantID, studentID, assignmentID, attemptID
}

// TestSweeperEnqueuesExpiredAttempt: a negative grace makes any in-progress
// attempt look expired, so RunOnce must claim it and enqueue a finalize with the
// right ids and nil answers.
func TestSweeperEnqueuesExpiredAttempt(t *testing.T) {
	db := testutil.OpenTestDB(t)
	_, studentID, assignmentID, attemptID := sweeperGraph(t, db, 60)

	q := &recordingQueue{}
	s := NewSweeper(repository.NewAttemptRepo(db), q, nil, -3600, nil)
	s.RunOnce(context.Background())

	if len(q.finalize) != 1 {
		t.Fatalf("finalize count=%d, want 1 (%+v)", len(q.finalize), q.finalize)
	}
	p := q.finalize[0]
	if p.AttemptID != attemptID || p.AssignmentID != assignmentID || p.StudentID != studentID {
		t.Fatalf("payload=%+v, want attempt=%d assignment=%d student=%d",
			p, attemptID, assignmentID, studentID)
	}
	// The sweeper passes no answers; it relies on the autosave buffer having
	// already flushed them.
	if p.Answers != nil {
		t.Fatalf("payload.Answers=%v, want nil", p.Answers)
	}
}

// TestSweeperSkipsUnexpiredAttempt is the negative path: an attempt well inside
// its window must not be enqueued. The assertion targets our own attempt id, so
// it is immune to any in-progress attempts left in dev data.
func TestSweeperSkipsUnexpiredAttempt(t *testing.T) {
	db := testutil.OpenTestDB(t)
	_, _, _, attemptID := sweeperGraph(t, db, 60)

	q := &recordingQueue{}
	// Duration 60 min, grace 30s, attempt just started => not expired.
	s := NewSweeper(repository.NewAttemptRepo(db), q, nil, 30, nil)
	s.RunOnce(context.Background())

	for _, p := range q.finalize {
		if p.AttemptID == attemptID {
			t.Fatalf("attempt %d is inside its window and must not be finalized: %+v", attemptID, p)
		}
	}
}

// TestSweeperNilNotificationService is safe: a nil notification service must not
// panic during RunOnce.
func TestSweeperNilNotificationService(t *testing.T) {
	db := testutil.OpenTestDB(t)
	sweeperGraph(t, db, 60)

	q := &recordingQueue{}
	s := NewSweeper(repository.NewAttemptRepo(db), q, nil, -3600, nil)
	s.RunOnce(context.Background()) // must not panic
}

func TestSweeperNilAutosaveBufferIsSafe(t *testing.T) {
	db := testutil.OpenTestDB(t)
	sweeperGraph(t, db, 60)

	q := &recordingQueue{}
	s := NewSweeper(repository.NewAttemptRepo(db), q, nil, -3600, nil)
	s.RunOnce(context.Background()) // nil autosave must not panic
}
