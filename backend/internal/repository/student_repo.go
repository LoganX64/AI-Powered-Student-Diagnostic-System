package repository

import (
	"database/sql"
	"strconv"
)

type StudentRepo struct {
	DB *sql.DB
}

func NewStudentRepo(db *sql.DB) *StudentRepo {
	return &StudentRepo{DB: db}
}

type StudentRow struct {
	StudentID   int     `json:"student_id"`
	Name        string  `json:"name"`
	StudentCode string  `json:"student_code"`
	CoachID     int     `json:"coach_id"`
	BatchID     *int    `json:"batch_id"`
	DeletedAt   *string `json:"deleted_at"`
}

	type StudentDetailRow struct {
	StudentID      int     `json:"student_id"`
	Name           string  `json:"name"`
	StudentCode    string  `json:"student_code"`
	CoachID        int     `json:"coach_id"`
	BatchID        *int    `json:"batch_id"`
	CoachName      string  `json:"coach_name"`
	CreatedAt      string  `json:"created_at"`
	DeletedAt      *string `json:"deleted_at"`
	DeletedByName  *string `json:"deleted_by_name"`
	DeletedByEmail *string `json:"deleted_by_email"`
	DeletedByRole  *string `json:"deleted_by_role"`
}

func (r *StudentRepo) Create(tenantID int, name, studentCode string, coachID int) (int, error) {
	var id int
	err := r.DB.QueryRow(`
		INSERT INTO students (tenant_id, name, student_code, coach_id)
		VALUES ($1,$2,$3,$4) RETURNING id
	`, tenantID, name, studentCode, coachID).Scan(&id)
	return id, err
}

func (r *StudentRepo) Exists(studentID, tenantID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND tenant_id=$2)",
		studentID, tenantID,
	).Scan(&exists)
	return exists, err
}

func (r *StudentRepo) ExistsActive(studentID, tenantID, coachID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND tenant_id=$2 AND coach_id=$3)",
		studentID, tenantID, coachID,
	).Scan(&exists)
	return exists, err
}

func (r *StudentRepo) GetName(studentID, tenantID int) (string, error) {
	var name string
	err := r.DB.QueryRow(
		"SELECT name FROM students WHERE id=$1 AND tenant_id=$2",
		studentID, tenantID,
	).Scan(&name)
	return name, err
}

func (r *StudentRepo) GetNameCode(studentID, tenantID int) (string, string, error) {
	var name, code string
	err := r.DB.QueryRow(
		"SELECT name, student_code FROM students WHERE id=$1 AND tenant_id=$2",
		studentID, tenantID,
	).Scan(&name, &code)
	return name, code, err
}

func (r *StudentRepo) GetCoachIDAndTenantID(studentID int) (int, int, error) {
	var coachID, tenantID int
	err := r.DB.QueryRow(
		"SELECT coach_id, tenant_id FROM students WHERE id=$1 AND deleted_at IS NULL",
		studentID,
	).Scan(&coachID, &tenantID)
	return coachID, tenantID, err
}

func (r *StudentRepo) GetCoachID(studentID, tenantID int) (int, error) {
	var coachID int
	err := r.DB.QueryRow(
		"SELECT coach_id FROM students WHERE id=$1 AND tenant_id=$2",
		studentID, tenantID,
	).Scan(&coachID)
	return coachID, err
}

func (r *StudentRepo) List(tenantID int, coachID *int, includeDeactivated bool, search string, limit, offset int) ([]StudentRow, int, error) {
	where := "tenant_id=$1"
	args := []interface{}{tenantID}

	if coachID != nil {
		where += " AND coach_id=$" + strconv.Itoa(len(args)+1)
		args = append(args, *coachID)
	}
	if includeDeactivated {
		where += " AND deleted_at IS NOT NULL"
	} else {
		where += " AND deleted_at IS NULL"
	}
	if search != "" {
		where += " AND (name ILIKE $" + strconv.Itoa(len(args)+1) + " OR student_code ILIKE $" + strconv.Itoa(len(args)+1) + ")"
		args = append(args, "%"+search+"%")
	}

	countQuery := "SELECT COUNT(*) FROM students WHERE " + where
	dataQuery := "SELECT id, name, student_code, coach_id, batch_id, deleted_at FROM students WHERE " + where

	var total int
	err := r.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery += " ORDER BY id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var students []StudentRow
	for rows.Next() {
		var s StudentRow
		if err := rows.Scan(&s.StudentID, &s.Name, &s.StudentCode, &s.CoachID, &s.BatchID, &s.DeletedAt); err != nil {
			return nil, 0, err
		}
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (r *StudentRepo) GetIDByStudentCode(studentCode string) (int, int, error) {
	var id, tenantID int
	err := r.DB.QueryRow("SELECT id, tenant_id FROM students WHERE student_code = $1 AND deleted_at IS NULL", studentCode).Scan(&id, &tenantID)
	return id, tenantID, err
}

func (r *StudentRepo) ExistsByID(studentID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM students WHERE id = $1)", studentID).Scan(&exists)
	return exists, err
}

func (r *StudentRepo) GetDetail(studentID, tenantID int, coachID *int) (*StudentDetailRow, error) {
	query := `
		SELECT st.id, st.name, st.student_code, st.coach_id, st.batch_id, COALESCE(c.name, ''),
		       st.created_at, st.deleted_at, u.email, dco.name, u.role
		FROM students st
		LEFT JOIN coaches c ON st.coach_id = c.id
		LEFT JOIN users u ON st.deleted_by = u.id
		LEFT JOIN coaches dco ON dco.user_id = u.id
		WHERE st.id = $1 AND st.tenant_id = $2`

	args := []interface{}{studentID, tenantID}
	if coachID != nil {
		query += " AND st.coach_id = $3"
		args = append(args, *coachID)
	}

	var s StudentDetailRow
	err := r.DB.QueryRow(query, args...).Scan(
		&s.StudentID, &s.Name, &s.StudentCode, &s.CoachID, &s.BatchID, &s.CoachName,
		&s.CreatedAt, &s.DeletedAt, &s.DeletedByEmail, &s.DeletedByName, &s.DeletedByRole,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *StudentRepo) SoftDelete(studentID, tenantID, deletedBy int, coachID *int) (bool, error) {
	query := `UPDATE students SET deleted_at = NOW(), deleted_by = $1
		 WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`
	args := []interface{}{deletedBy, studentID, tenantID}
	if coachID != nil {
		query += " AND coach_id = $4"
		args = append(args, *coachID)
	}

	result, err := r.DB.Exec(query, args...)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *StudentRepo) UpdateStudent(tenantID, studentID int, name, studentCode string, coachID int, batchID *int) error {
	_, err := r.DB.Exec(`
		UPDATE students
		SET name = $1, student_code = $2, coach_id = $3, batch_id = $4
		WHERE id = $5 AND tenant_id = $6 AND deleted_at IS NULL
	`, name, studentCode, coachID, batchID, studentID, tenantID)
	return err
}

func (r *StudentRepo) StudentCodeExists(tenantID int, studentCode string, exceptStudentID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM students WHERE tenant_id=$1 AND student_code=$2 AND id<>$3 AND deleted_at IS NULL)",
		tenantID, studentCode, exceptStudentID,
	).Scan(&exists)
	return exists, err
}

func (r *StudentRepo) Reactivate(studentID, tenantID int, coachID *int) (bool, error) {
	query := `UPDATE students SET deleted_at = NULL, deleted_by = NULL
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NOT NULL`
	args := []interface{}{studentID, tenantID}
	if coachID != nil {
		query += " AND coach_id = $3"
		args = append(args, *coachID)
	}

	result, err := r.DB.Exec(query, args...)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}
