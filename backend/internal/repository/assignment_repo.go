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
	ID         int    `json:"id"`
	TestID     int    `json:"test_id"`
	TestTitle  string `json:"test_title"`
	Status     string `json:"status"`
	AssignedAt string `json:"assigned_at"`
	Submitted  bool   `json:"submitted"`
}

type AssignmentDetailRow struct {
	ID          int    `json:"id"`
	StudentID   int    `json:"student_id"`
	StudentName string `json:"student_name"`
	StudentCode string `json:"student_code"`
	TestID      int    `json:"test_id"`
	TestTitle   string `json:"test_title"`
	CoachID     int    `json:"coach_id"`
	Status      string `json:"status"`
	AssignedAt  string `json:"assigned_at"`
}

func (r *AssignmentRepo) Create(studentID, testID, coachID int) (int, error) {
	var id int
	err := r.DB.QueryRow(`
		INSERT INTO assignments (student_id, test_id, coach_id)
		VALUES ($1,$2,$3) RETURNING id
	`, studentID, testID, coachID).Scan(&id)
	return id, err
}

func (r *AssignmentRepo) ListByStudent(studentID int, coachID *int, limit, offset int) ([]AssignmentRow, int, error) {
	var total int
	var dataQuery string
	var args []interface{}

	if coachID != nil {
		err := r.DB.QueryRow("SELECT COUNT(*) FROM assignments WHERE student_id=$1 AND coach_id=$2", studentID, *coachID).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		dataQuery = `SELECT a.id, a.test_id, t.title, a.status, a.assigned_at,
		       (a.status = 'submitted') AS submitted
		FROM assignments a JOIN tests t ON a.test_id = t.id
		WHERE a.student_id = $1 AND a.coach_id = $2
		ORDER BY a.id DESC LIMIT $3 OFFSET $4`
		args = []interface{}{studentID, *coachID, limit, offset}
	} else {
		err := r.DB.QueryRow("SELECT COUNT(*) FROM assignments WHERE student_id=$1", studentID).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		dataQuery = `SELECT a.id, a.test_id, t.title, a.status, a.assigned_at,
		       (a.status = 'submitted') AS submitted
		FROM assignments a JOIN tests t ON a.test_id = t.id
		WHERE a.student_id = $1
		ORDER BY a.id DESC LIMIT $2 OFFSET $3`
		args = []interface{}{studentID, limit, offset}
	}

	rows, err := r.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assignments []AssignmentRow
	for rows.Next() {
		var a AssignmentRow
		if err := rows.Scan(&a.ID, &a.TestID, &a.TestTitle, &a.Status, &a.AssignedAt, &a.Submitted); err != nil {
			return nil, 0, err
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return assignments, total, nil
}

func (r *AssignmentRepo) ListAll(tenantID int, coachID *int, testIDStr string, limit, offset int) ([]AssignmentDetailRow, int, error) {
	where := "s.tenant_id=$1 AND s.deleted_at IS NULL"
	args := []interface{}{tenantID}

	if coachID != nil {
		where += " AND a.coach_id=$" + strconv.Itoa(len(args)+1)
		args = append(args, *coachID)
	}
	if testIDStr != "" {
		where += " AND a.test_id=$" + strconv.Itoa(len(args)+1)
		args = append(args, testIDStr)
	}

	baseQuery := "FROM assignments a JOIN students s ON a.student_id = s.id JOIN tests t ON a.test_id = t.id WHERE " + where

	var total int
	err := r.DB.QueryRow("SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery := "SELECT a.id, a.student_id, s.name, s.student_code, a.test_id, t.title, a.coach_id, a.status, a.assigned_at " + baseQuery
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
		if err := rows.Scan(&a.ID, &a.StudentID, &a.StudentName, &a.StudentCode, &a.TestID, &a.TestTitle, &a.CoachID, &a.Status, &a.AssignedAt); err != nil {
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

func (r *AssignmentRepo) MarkSubmittedTx(tx *sql.Tx, assignmentID int) error {
	_, err := tx.Exec("UPDATE assignments SET status = 'submitted' WHERE id = $1", assignmentID)
	return err
}
