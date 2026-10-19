package repository

import (
	"database/sql"
	"errors"
	"time"
)

// MaxAttemptsPerAssignment is the hard cap on sittings for one assignment,
// including the first attempt. Retakes are granted by admin/coach one at a time.
const MaxAttemptsPerAssignment = 3

var (
	// ErrAssignmentNotFound means the assignment does not exist in the tenant
	// (or is not owned by the coach) the caller is scoped to.
	ErrAssignmentNotFound = errors.New("assignment not found")
	// ErrRetakeNotAllowed means the assignment is not in a retakable state:
	// it has not been submitted yet, or an attempt is still in progress.
	ErrRetakeNotAllowed = errors.New("assignment is not eligible for a retake")
	// ErrRetakeLimitReached means every allowed attempt has been used.
	ErrRetakeLimitReached = errors.New("maximum attempts reached")
)

// CreateInProgressAttempt inserts a new 'in_progress' attempt for an assignment
// (server-authoritative timing). Returns the attempt id and started_at.
//
// The insert is guarded so a submitted assignment or an assignment at the
// attempt cap cannot silently gain another attempt. It returns sql.ErrNoRows in
// that case; the unique partial index on in-progress attempts handles
// concurrent starts.
func (r *AttemptRepo) CreateInProgressAttempt(assignmentID int) (int, time.Time, error) {
	var id int
	var startedAt time.Time
	err := r.DB.QueryRow(`
		INSERT INTO attempts (assignment_id, status, started_at)
		SELECT $1, 'in_progress', NOW()
		FROM assignments a
		WHERE a.id = $1
		  AND a.status <> 'submitted'
		  AND (SELECT COUNT(*) FROM attempts WHERE assignment_id = $1) < $2
		RETURNING id, started_at
	`, assignmentID, MaxAttemptsPerAssignment).Scan(&id, &startedAt)
	return id, startedAt, err
}

// CountAttempts returns how many attempts (any status) an assignment has.
func (r *AttemptRepo) CountAttempts(assignmentID int) (int, error) {
	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM attempts WHERE assignment_id = $1", assignmentID,
	).Scan(&count)
	return count, err
}

// GrantRetake reopens a submitted assignment for one more sitting. It is
// transactional and row-locks the assignment so concurrent grants cannot push
// the attempt count past MaxAttemptsPerAssignment or grant while an attempt is
// still in progress.
//
// grantedBy is the user id recorded in the audit columns; coachID, when set,
// additionally requires the assignment to belong to that coach.
func (r *AttemptRepo) GrantRetake(assignmentID, tenantID, grantedBy int, coachID *int) (attemptsUsed, maxAttempts int, err error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return 0, MaxAttemptsPerAssignment, err
	}
	defer tx.Rollback()

	query := `
		SELECT a.status
		FROM assignments a
		JOIN students s ON s.id = a.student_id
		WHERE a.id = $1 AND s.tenant_id = $2`
	args := []interface{}{assignmentID, tenantID}
	if coachID != nil {
		query += " AND a.coach_id = $3"
		args = append(args, *coachID)
	}
	query += " FOR UPDATE OF a"

	var status string
	if err := tx.QueryRow(query, args...).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return 0, MaxAttemptsPerAssignment, ErrAssignmentNotFound
		}
		return 0, MaxAttemptsPerAssignment, err
	}
	if status != "submitted" {
		return 0, MaxAttemptsPerAssignment, ErrRetakeNotAllowed
	}

	var inProgress int
	if err := tx.QueryRow(
		"SELECT COUNT(*) FROM attempts WHERE assignment_id = $1 AND status = 'in_progress'",
		assignmentID,
	).Scan(&inProgress); err != nil {
		return 0, MaxAttemptsPerAssignment, err
	}
	if inProgress > 0 {
		return 0, MaxAttemptsPerAssignment, ErrRetakeNotAllowed
	}

	if err := tx.QueryRow(
		"SELECT COUNT(*) FROM attempts WHERE assignment_id = $1", assignmentID,
	).Scan(&attemptsUsed); err != nil {
		return 0, MaxAttemptsPerAssignment, err
	}
	if attemptsUsed >= MaxAttemptsPerAssignment {
		return attemptsUsed, MaxAttemptsPerAssignment, ErrRetakeLimitReached
	}

	if _, err := tx.Exec(`
		UPDATE assignments
		SET status = 'assigned', retake_granted_at = NOW(), retake_granted_by = $2
		WHERE id = $1
	`, assignmentID, grantedBy); err != nil {
		return attemptsUsed, MaxAttemptsPerAssignment, err
	}

	if err := tx.Commit(); err != nil {
		return attemptsUsed, MaxAttemptsPerAssignment, err
	}
	return attemptsUsed, MaxAttemptsPerAssignment, nil
}

// GetInProgressAttempt returns the active (in_progress) attempt for an assignment.
func (r *AttemptRepo) GetInProgressAttempt(assignmentID int) (int, time.Time, error) {
	var id int
	var startedAt time.Time
	err := r.DB.QueryRow(`
		SELECT id, started_at FROM attempts
		WHERE assignment_id = $1 AND status = 'in_progress'
		ORDER BY id DESC LIMIT 1
	`, assignmentID).Scan(&id, &startedAt)
	return id, startedAt, err
}

// UpsertAnswer writes (or updates) an answer_log for a question within an attempt.
func (r *AttemptRepo) UpsertAnswer(
	attemptID, questionID int,
	selectedAnswer string,
	isCorrect bool,
	timeSpent float64,
	markedForReview, revisited, changedAnswer, wasInitiallyWrong, seen bool,
) error {
	_, err := r.DB.Exec(`
		INSERT INTO answer_logs
			(attempt_id, question_id, selected_answer, is_correct, time_spent,
			 marked_for_review, revisited, changed_answer, was_initially_wrong, seen)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (attempt_id, question_id) DO UPDATE SET
			selected_answer = EXCLUDED.selected_answer,
			is_correct = EXCLUDED.is_correct,
			time_spent = EXCLUDED.time_spent,
			marked_for_review = EXCLUDED.marked_for_review,
			revisited = EXCLUDED.revisited,
			changed_answer = EXCLUDED.changed_answer,
			was_initially_wrong = EXCLUDED.was_initially_wrong,
			seen = EXCLUDED.seen
	`, attemptID, questionID, selectedAnswer, isCorrect, timeSpent,
		markedForReview, revisited, changedAnswer, wasInitiallyWrong, seen)
	return err
}

// FinalizeAttemptTx marks the attempt submitted and flips the assignment status,
// all in one transaction. No attempt_results write happens here.
func (r *AttemptRepo) FinalizeAttemptTx(assignmentID, attemptID, studentID int) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE attempts SET submitted_at = NOW(), status = 'submitted' WHERE id = $1`,
		attemptID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE assignments SET status = 'submitted' WHERE id = $1 AND student_id = $2`,
		assignmentID, studentID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

type SavedAnswer struct {
	QuestionID      int     `json:"question_id"`
	SelectedAnswer  string  `json:"selected_answer"`
	TimeSpent       float64 `json:"time_spent"`
	MarkedForReview bool    `json:"marked_for_review"`
	Seen            bool    `json:"seen"`
	Revisited       bool    `json:"revisited"`
	ChangedAnswer   bool    `json:"changed_answer"`
}

// GetSavedAnswers returns persisted answers for resume (autosave tier).
func (r *AttemptRepo) GetSavedAnswers(attemptID int) ([]SavedAnswer, error) {
	rows, err := r.DB.Query(`
		SELECT question_id, COALESCE(selected_answer, ''), COALESCE(time_spent, 0),
		       COALESCE(marked_for_review, false), COALESCE(seen, true),
		       COALESCE(revisited, false), COALESCE(changed_answer, false)
		FROM answer_logs WHERE attempt_id = $1 ORDER BY question_id
	`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SavedAnswer
	for rows.Next() {
		var a SavedAnswer
		if err := rows.Scan(&a.QuestionID, &a.SelectedAnswer, &a.TimeSpent, &a.MarkedForReview, &a.Seen, &a.Revisited, &a.ChangedAnswer); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetTestIDForAttempt returns the test_id backing an attempt (for compute jobs).
func (r *AttemptRepo) GetTestIDForAttempt(attemptID int) (int, error) {
	var testID int
	err := r.DB.QueryRow(`
		SELECT ass.test_id FROM attempts a
		JOIN assignments ass ON a.assignment_id = ass.id
		WHERE a.id = $1
	`, attemptID).Scan(&testID)
	return testID, err
}

// AttemptIDsByTest returns all attempt ids for a test within a tenant.
func (r *AttemptRepo) AttemptIDsByTest(testID, tenantID int) ([]int, error) {
	rows, err := r.DB.Query(`
		SELECT a.id FROM attempts a
		JOIN assignments ass ON a.assignment_id = ass.id
		JOIN students s ON ass.student_id = s.id
		WHERE ass.test_id = $1 AND s.tenant_id = $2
		ORDER BY a.id
	`, testID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// AttemptBelongsToTenant reports whether an attempt falls under the tenant.
func (r *AttemptRepo) AttemptBelongsToTenant(attemptID, tenantID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM attempts a
			JOIN assignments ass ON a.assignment_id = ass.id
			JOIN students s ON ass.student_id = s.id
			WHERE a.id = $1 AND s.tenant_id = $2
		)
	`, attemptID, tenantID).Scan(&exists)
	return exists, err
}

// ExpiredInProgressAttempt attempts past their deadline+grace that are still
// in_progress. Claimed with SKIP LOCKED so concurrent sweepers don't double-finalize.
type ExpiredAttempt struct {
	AssignmentID int
	AttemptID    int
	StudentID    int
	TenantID     int
	StudentName  string
}

func (r *AttemptRepo) ExpiredInProgressAttempts(graceSeconds int) ([]ExpiredAttempt, error) {
	rows, err := r.DB.Query(`
		SELECT a.id, a.assignment_id, ass.student_id, s.tenant_id, s.name FROM attempts a
		JOIN assignments ass ON a.assignment_id = ass.id
		JOIN tests t ON ass.test_id = t.id
		JOIN students s ON ass.student_id = s.id
		WHERE a.status = 'in_progress'
		  AND a.started_at + (COALESCE(t.duration, 0) || ' minutes')::interval
		      + ($1 || ' seconds')::interval < NOW()
		ORDER BY a.id
		FOR UPDATE SKIP LOCKED
	`, graceSeconds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExpiredAttempt
	for rows.Next() {
		var e ExpiredAttempt
		if err := rows.Scan(&e.AttemptID, &e.AssignmentID, &e.StudentID, &e.TenantID, &e.StudentName); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
