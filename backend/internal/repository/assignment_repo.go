package repository

import (
	"database/sql"
	"strconv"
)

type AssignmentRepo struct {
	DB *sql.DB
}

func NewAssignmentRepo(db *sql.DB) *AssignmentRepo {
	return &AssignmentRepo{DB: db}
}

type AssignmentRow struct {
	ID                int    `json:"id"`
	TestID            int    `json:"test_id"`
	TestTitle         string `json:"test_title"`
	Status            string `json:"status"`
	AssignedAt        string `json:"assigned_at"`
	Submitted         bool   `json:"submitted"`
	AttemptInProgress bool   `json:"attempt_in_progress"`
}

type AssignmentDetailRow struct {
	ID          int    `json:"id"`
	StudentID   int    `json:"student_id"`
	StudentName string `json:"student_name"`
	StudentCode string `json:"student_code"`
	TestID      int    `json:"test_id"`
	TestTitle   string `json:"test_title"`
	CoachID     int    `json:"coach_id"`
	CoachName   string `json:"coach_name"`
	Status      string `json:"status"`
	AssignedAt  string `json:"assigned_at"`
	SubjectName string `json:"subject_name"`
}

// HasActiveAssignment reports whether a non-submitted (active) assignment
// already exists for the given student and test. This is used to prevent
// re-assigning a test while the student still has an unattempted version.
func (r *AssignmentRepo) HasActiveAssignment(studentID, testID int) (bool, error) {
	var n int
	err := r.DB.QueryRow(
		"SELECT 1 FROM assignments WHERE student_id=$1 AND test_id=$2 AND status <> 'submitted' LIMIT 1",
		studentID, testID,
	).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *AssignmentRepo) Create(studentID, testID, coachID int, integrityPolicy []byte, estimatedCost float64, deliveryMode string) (int, error) {
	if len(integrityPolicy) == 0 {
		integrityPolicy = []byte("{}")
	}
	if deliveryMode == "" {
		deliveryMode = "standard"
	}
	var id int
	err := r.DB.QueryRow(`
		INSERT INTO assignments (student_id, test_id, coach_id, integrity_policy, estimated_cost, delivery_mode)
		VALUES ($1,$2,$3, COALESCE($4, '{}'::jsonb), $5, $6) RETURNING id
	`, studentID, testID, coachID, integrityPolicy, estimatedCost, deliveryMode).Scan(&id)
	return id, err
}

// CreateBatch assigns a test to many students in one call. Students with an
// active (non-submitted) assignment for the same test are skipped.
// Returns the number of assignments actually created and skipped.
func (r *AssignmentRepo) CreateBatch(studentIDs []int, testID, coachID int, integrityPolicy []byte, estimatedCost float64, deliveryMode string) (created int, skipped int, err error) {
	if len(integrityPolicy) == 0 {
		integrityPolicy = []byte("{}")
	}
	if deliveryMode == "" {
		deliveryMode = "standard"
	}
	for _, studentID := range studentIDs {
		// Check for an active (non-submitted) assignment first.
		var n int
		qErr := r.DB.QueryRow(
			"SELECT 1 FROM assignments WHERE student_id=$1 AND test_id=$2 AND status <> 'submitted' LIMIT 1",
			studentID, testID,
		).Scan(&n)
		if qErr == nil {
			skipped++
			continue
		}
		if qErr != sql.ErrNoRows {
			return created, skipped, qErr
		}

		var id int
		err := r.DB.QueryRow(`
			INSERT INTO assignments (student_id, test_id, coach_id, integrity_policy, estimated_cost, delivery_mode)
			VALUES ($1,$2,$3, COALESCE($4, '{}'::jsonb), $5, $6) RETURNING id
		`, studentID, testID, coachID, integrityPolicy, estimatedCost, deliveryMode).Scan(&id)
		if err != nil {
			return created, skipped, err
		}
		created++
	}
	return created, skipped, nil
}

// GetPolicy returns the raw integrity_policy JSONB for an assignment.
func (r *AssignmentRepo) GetPolicy(assignmentID int) ([]byte, error) {
	var policy []byte
	err := r.DB.QueryRow(
		`SELECT integrity_policy FROM assignments WHERE id = $1`,
		assignmentID,
	).Scan(&policy)
	return policy, err
}

// applyStatusFilter appends an assignment status predicate to the WHERE clause
// and appends the corresponding argument. "submitted" matches submitted rows;
// "active" matches everything except submitted (assigned/in-progress/etc.).
func applyStatusFilter(where string, args []interface{}, status string) (string, []interface{}) {
	switch status {
	case "submitted":
		where += " AND a.status = $" + strconv.Itoa(len(args)+1)
		args = append(args, "submitted")
	case "active":
		where += " AND a.status <> $" + strconv.Itoa(len(args)+1)
		args = append(args, "submitted")
	}
	return where, args
}

func (r *AssignmentRepo) ListByStudent(studentID int, coachID *int, status string, limit, offset int) ([]AssignmentRow, int, error) {
	where := "a.student_id = $1"
	args := []interface{}{studentID}
	if coachID != nil {
		where += " AND a.coach_id = $" + strconv.Itoa(len(args)+1)
		args = append(args, *coachID)
	}
	where, args = applyStatusFilter(where, args, status)

	var total int
	if err := r.DB.QueryRow("SELECT COUNT(*) FROM assignments a WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQuery := `SELECT a.id, a.test_id, t.title, a.status, a.assigned_at,
		       (a.status = 'submitted') AS submitted,
		       EXISTS(SELECT 1 FROM attempts att WHERE att.assignment_id = a.id AND att.status = 'in_progress') AS attempt_in_progress
		FROM assignments a JOIN tests t ON a.test_id = t.id
		WHERE ` + where + `
		ORDER BY a.id DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assignments []AssignmentRow
	for rows.Next() {
		var a AssignmentRow
		if err := rows.Scan(&a.ID, &a.TestID, &a.TestTitle, &a.Status, &a.AssignedAt, &a.Submitted, &a.AttemptInProgress); err != nil {
			return nil, 0, err
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return assignments, total, nil
}

type AssignmentFilters struct {
	Status     string
	TestID     string
	Search     string
	Year       string
	SubjectID  string
	CoachIDStr string
}

func (r *AssignmentRepo) ListAll(tenantID int, coachID *int, filters AssignmentFilters, limit, offset int) ([]AssignmentDetailRow, int, error) {
	where := "s.tenant_id=$1 AND s.deleted_at IS NULL"
	args := []interface{}{tenantID}

	if coachID != nil {
		where += " AND a.coach_id=$" + strconv.Itoa(len(args)+1)
		args = append(args, *coachID)
	}
	if filters.CoachIDStr != "" && coachID == nil {
		cid, err := strconv.Atoi(filters.CoachIDStr)
		if err != nil {
			return nil, 0, err
		}
		where += " AND a.coach_id=$" + strconv.Itoa(len(args)+1)
		args = append(args, cid)
	}
	if filters.TestID != "" {
		testID, err := strconv.Atoi(filters.TestID)
		if err != nil {
			return nil, 0, err
		}
		where += " AND a.test_id=$" + strconv.Itoa(len(args)+1)
		args = append(args, testID)
	}
	if filters.SubjectID != "" {
		sid, err := strconv.Atoi(filters.SubjectID)
		if err != nil {
			return nil, 0, err
		}
		where += " AND t.subject_id=$" + strconv.Itoa(len(args)+1)
		args = append(args, sid)
	}
	if filters.Year != "" {
		where += " AND EXTRACT(YEAR FROM a.assigned_at)::text=$" + strconv.Itoa(len(args)+1)
		args = append(args, filters.Year)
	}
	if filters.Search != "" {
		where += " AND (s.name ILIKE $" + strconv.Itoa(len(args)+1) + " OR t.title ILIKE $" + strconv.Itoa(len(args)+1) + " OR t.subject_name ILIKE $" + strconv.Itoa(len(args)+1) + ")"
		args = append(args, "%"+filters.Search+"%")
	}
	where, args = applyStatusFilter(where, args, filters.Status)

	baseQuery := "FROM assignments a JOIN students s ON a.student_id = s.id JOIN tests t ON a.test_id = t.id JOIN coaches c ON a.coach_id = c.id WHERE " + where

	var total int
	err := r.DB.QueryRow("SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery := "SELECT a.id, a.student_id, s.name, s.student_code, a.test_id, t.title, a.coach_id, c.name, a.status, a.assigned_at, t.subject_name " + baseQuery
	dataQuery += " ORDER BY a.id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assignments []AssignmentDetailRow
	for rows.Next() {
		var a AssignmentDetailRow
		if err := rows.Scan(&a.ID, &a.StudentID, &a.StudentName, &a.StudentCode, &a.TestID, &a.TestTitle, &a.CoachID, &a.CoachName, &a.Status, &a.AssignedAt, &a.SubjectName); err != nil {
			return nil, 0, err
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return assignments, total, nil
}

func (r *AssignmentRepo) GetByID(assignmentID, studentID int) (int, int, string, string, string, error) {
	var assignmentStudentID, testID int
	var status, assignedAt, testTitle string
	err := r.DB.QueryRow(`
		SELECT a.student_id, a.test_id, a.status, a.assigned_at, t.title
		FROM assignments a JOIN tests t ON a.test_id = t.id
		WHERE a.id = $1 AND a.student_id = $2
	`, assignmentID, studentID).Scan(&assignmentStudentID, &testID, &status, &assignedAt, &testTitle)
	return assignmentStudentID, testID, status, assignedAt, testTitle, err
}

func (r *AssignmentRepo) GetByIDForCoach(assignmentID, studentID, coachID int) (int, string, string, string, error) {
	var testID int
	var status, assignedAt, testTitle string
	err := r.DB.QueryRow(`
		SELECT a.test_id, a.status, a.assigned_at, t.title
		FROM assignments a JOIN tests t ON a.test_id = t.id
		WHERE a.id = $1 AND a.student_id = $2 AND a.coach_id = $3
	`, assignmentID, studentID, coachID).Scan(&testID, &status, &assignedAt, &testTitle)
	return testID, status, assignedAt, testTitle, err
}

type AssignmentOwnerRow struct {
	OwnerID  int
	TestID   int
	Duration int
}

func (r *AssignmentRepo) GetOwnerAndTest(assignmentID int) (AssignmentOwnerRow, error) {
	var row AssignmentOwnerRow
	err := r.DB.QueryRow(`
		SELECT ass.student_id, ass.test_id, COALESCE(t.duration, 0)
		FROM assignments ass JOIN tests t ON ass.test_id = t.id
		WHERE ass.id = $1
	`, assignmentID).Scan(&row.OwnerID, &row.TestID, &row.Duration)
	return row, err
}

type AssignmentStudentDetailRow struct {
	OwnerID  int
	TestID   int
	TestTitle string
	Duration int
	ExamDate sql.NullTime
}

func (r *AssignmentRepo) GetDetailForStudent(assignmentID int) (AssignmentStudentDetailRow, error) {
	var row AssignmentStudentDetailRow
	err := r.DB.QueryRow(`
		SELECT ass.student_id, ass.test_id, t.title, COALESCE(t.duration, 0), t.exam_date
		FROM assignments ass JOIN tests t ON ass.test_id = t.id
		WHERE ass.id = $1
	`, assignmentID).Scan(&row.OwnerID, &row.TestID, &row.TestTitle, &row.Duration, &row.ExamDate)
	return row, err
}

func (r *AssignmentRepo) MarkSubmittedTx(tx *sql.Tx, assignmentID, studentID int) error {
	_, err := tx.Exec("UPDATE assignments SET status = 'submitted' WHERE id = $1 AND student_id = $2", assignmentID, studentID)
	return err
}

// Delete removes an assignment (and its attempts via ON DELETE CASCADE),
// scoped to the tenant (and optionally the coach) for safety.
func (r *AssignmentRepo) Delete(assignmentID, tenantID int, coachID *int) (bool, error) {
	query := `DELETE FROM assignments a USING students s
		WHERE a.id = $1 AND a.student_id = s.id AND s.tenant_id = $2`
	args := []interface{}{assignmentID, tenantID}
	if coachID != nil {
		query += " AND a.coach_id = $3"
		args = append(args, *coachID)
	}
	result, err := r.DB.Exec(query, args...)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}
