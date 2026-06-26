package repository

import "database/sql"

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
	DeletedAt   *string `json:"deleted_at"`
}

type StudentDetailRow struct {
	StudentID      int     `json:"student_id"`
	Name           string  `json:"name"`
	StudentCode    string  `json:"student_code"`
	CoachID        int     `json:"coach_id"`
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

func (r *StudentRepo) GetTestCoachAndTenant(testID int) (int, int, error) {
	var coachID, tenantID int
	err := r.DB.QueryRow(
		"SELECT coach_id, tenant_id FROM tests WHERE id=$1",
		testID,
	).Scan(&coachID, &tenantID)
	return coachID, tenantID, err
}

func (r *StudentRepo) List(tenantID int, coachID *int, includeDeactivated bool, limit, offset int) ([]StudentRow, int, error) {
	var total int
	var countArgs []interface{}
	var query string
	var countQuery string

	if coachID != nil {
		if includeDeactivated {
			countQuery = "SELECT COUNT(*) FROM students WHERE tenant_id=$1 AND coach_id=$2"
			query = "SELECT id, name, student_code, coach_id, deleted_at FROM students WHERE tenant_id=$1 AND coach_id=$2"
		} else {
			countQuery = "SELECT COUNT(*) FROM students WHERE tenant_id=$1 AND coach_id=$2 AND deleted_at IS NULL"
			query = "SELECT id, name, student_code, coach_id, deleted_at FROM students WHERE tenant_id=$1 AND coach_id=$2 AND deleted_at IS NULL"
		}
		countArgs = []interface{}{tenantID, *coachID}
	} else {
		if includeDeactivated {
			countQuery = "SELECT COUNT(*) FROM students WHERE tenant_id=$1"
			query = "SELECT id, name, student_code, coach_id, deleted_at FROM students WHERE tenant_id=$1"
		} else {
			countQuery = "SELECT COUNT(*) FROM students WHERE tenant_id=$1 AND deleted_at IS NULL"
			query = "SELECT id, name, student_code, coach_id, deleted_at FROM students WHERE tenant_id=$1 AND deleted_at IS NULL"
		}
		countArgs = []interface{}{tenantID}
	}

	err := r.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += " ORDER BY id DESC"
	args := countArgs
	if coachID != nil {
		query += " LIMIT $3 OFFSET $4"
	} else {
		query += " LIMIT $2 OFFSET $3"
	}
	args = append(args, limit, offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var students []StudentRow
	for rows.Next() {
		var s StudentRow
		if err := rows.Scan(&s.StudentID, &s.Name, &s.StudentCode, &s.CoachID, &s.DeletedAt); err != nil {
			return nil, 0, err
		}
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (r *StudentRepo) GetDetail(studentID, tenantID int, coachID *int) (*StudentDetailRow, error) {
	query := `
		SELECT st.id, st.name, st.student_code, st.coach_id, COALESCE(c.name, ''),
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
		&s.StudentID, &s.Name, &s.StudentCode, &s.CoachID, &s.CoachName,
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
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
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
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}
